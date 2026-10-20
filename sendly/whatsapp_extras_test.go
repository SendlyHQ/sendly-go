package sendly

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const waProfileFixture = `{
	"phoneNumber": "+15125550123",
	"displayName": "Acme Bakery",
	"profilePhotoUrl": "https://pps.whatsapp.net/v/acme.jpg",
	"category": "FOOD_GROCERY",
	"about": "Fresh bread, daily.",
	"description": null,
	"email": null,
	"website": null,
	"address": null
}`

var jpegBytes = []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}

func TestWhatsAppSendersUploadProfilePhoto_Multipart(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/whatsapp/senders/+15125550123/profile/photo" {
			t.Errorf("unexpected path %q", r.URL.EscapedPath())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-api-key" {
			t.Errorf("unexpected Authorization %q", got)
		}
		if got := r.Header.Get("X-Organization-Id"); got != "org_1" {
			t.Errorf("expected X-Organization-Id org_1, got %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart/form-data, got %q", r.Header.Get("Content-Type"))
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("expected a multipart field named file: %v", err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if !bytes.Equal(data, jpegBytes) {
			t.Errorf("file bytes changed in transit")
		}
		if header.Filename != "logo.jpg" {
			t.Errorf("expected filename logo.jpg, got %q", header.Filename)
		}
		if got := header.Header.Get("Content-Type"); got != "image/jpeg" {
			t.Errorf("expected part Content-Type image/jpeg, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(waProfileFixture))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithOrganizationID("org_1"))
	profile, err := client.WhatsApp.Senders.UploadProfilePhoto(context.Background(), "+15125550123", "logo.jpg", bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.ProfilePhotoURL == nil || *profile.ProfilePhotoURL != "https://pps.whatsapp.net/v/acme.jpg" {
		t.Errorf("unexpected ProfilePhotoURL %v", profile.ProfilePhotoURL)
	}
	if profile.PhoneNumber != "+15125550123" {
		t.Errorf("unexpected PhoneNumber %q", profile.PhoneNumber)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("expected 1 request, got %d", got)
	}
}

func TestWhatsAppSendersUploadProfilePhoto_ErrorsAreSentOnce(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		check  func(t *testing.T, err error)
	}{
		{
			name:   "too large",
			status: http.StatusRequestEntityTooLarge,
			body:   `{"error":"whatsapp_profile_photo_too_large","message":"The photo must be 5 MB or smaller."}`,
			check: func(t *testing.T, err error) {
				var se *SendlyError
				if !errors.As(err, &se) || se.StatusCode != 413 || se.Code != WhatsAppErrorCodeProfilePhotoTooLarge {
					t.Errorf("expected 413 *SendlyError whatsapp_profile_photo_too_large, got %T %v", err, err)
				}
			},
		},
		{
			name:   "not an image",
			status: http.StatusBadRequest,
			body:   `{"error":"whatsapp_profile_photo_invalid","message":"The photo must be a JPEG or PNG image."}`,
			check: func(t *testing.T, err error) {
				var ve *ValidationError
				if !errors.As(err, &ve) || ve.Code != WhatsAppErrorCodeProfilePhotoInvalid {
					t.Errorf("expected *ValidationError whatsapp_profile_photo_invalid, got %T %v", err, err)
				}
			},
		},
		{
			name:   "carrier refused",
			status: http.StatusBadGateway,
			body:   `{"error":"whatsapp_profile_update_failed","message":"The photo couldn't be uploaded."}`,
			check: func(t *testing.T, err error) {
				var se *SendlyError
				if !errors.As(err, &se) || se.StatusCode != 502 || se.Code != WhatsAppErrorCodeProfileUpdateFailed {
					t.Errorf("expected 502 *SendlyError whatsapp_profile_update_failed, got %T %v", err, err)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var requests int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&requests, 1)
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := NewClient("test-api-key", WithBaseURL(server.URL))
			_, err := client.WhatsApp.Senders.UploadProfilePhoto(context.Background(), "+15125550123", "logo.jpg", bytes.NewReader(jpegBytes))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			tc.check(t, err)
			if got := atomic.LoadInt32(&requests); got != 1 {
				t.Errorf("expected the upload to be sent once, got %d requests", got)
			}
		})
	}
}

func TestWhatsAppSendersUploadProfilePhoto_Validation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make a request")
	}))
	defer server.Close()
	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	if _, err := client.WhatsApp.Senders.UploadProfilePhoto(ctx, "", "logo.jpg", bytes.NewReader(jpegBytes)); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty phoneNumber, got %v", err)
	}
	if _, err := client.WhatsApp.Senders.UploadProfilePhoto(ctx, "+15125550123", "", bytes.NewReader(jpegBytes)); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty filename, got %v", err)
	}
	if _, err := client.WhatsApp.Senders.UploadProfilePhoto(ctx, "+15125550123", "logo.jpg", nil); !IsValidationError(err) {
		t.Errorf("expected ValidationError for nil file, got %v", err)
	}
}

func TestWhatsAppSendersUploadProfilePhoto_RefusesDotSegmentBeforeSending(t *testing.T) {
	for _, phone := range []string{"..", "."} {
		client, requests, _ := newPathSegmentServer(t)
		_, err := client.WhatsApp.Senders.UploadProfilePhoto(context.Background(), phone, "logo.jpg", bytes.NewReader(jpegBytes))
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%q: expected *ValidationError, got %T %v", phone, err, err)
		}
		if n := atomic.LoadInt32(requests); n != 0 {
			t.Errorf("%q: expected no request, got %d", phone, n)
		}
	}
}

func TestWhatsAppSendersDeleteProfilePhoto(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/whatsapp/senders/+15125550123/profile/photo" {
			t.Errorf("unexpected path %q", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"phoneNumber":"+15125550123","displayName":"Acme Bakery","profilePhotoUrl":null,"category":null,"about":null,"description":null,"email":null,"website":null,"address":null}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	profile, err := client.WhatsApp.Senders.DeleteProfilePhoto(context.Background(), "+15125550123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.ProfilePhotoURL != nil {
		t.Errorf("expected ProfilePhotoURL nil, got %q", *profile.ProfilePhotoURL)
	}
	if profile.DisplayName == nil || *profile.DisplayName != "Acme Bakery" {
		t.Errorf("unexpected DisplayName %v", profile.DisplayName)
	}

	if _, err := client.WhatsApp.Senders.DeleteProfilePhoto(context.Background(), ""); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty phoneNumber, got %v", err)
	}
}

func TestWhatsAppSendersGetConversationalComponents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/whatsapp/senders/+15125550123/conversational_components" {
			t.Errorf("unexpected path %q", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"phoneNumber":"+15125550123","iceBreakers":["What are your hours?","Book a table"],"commands":[{"command":"menu","description":"See today's menu"}]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	got, err := client.WhatsApp.Senders.GetConversationalComponents(context.Background(), "+15125550123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.PhoneNumber != "+15125550123" {
		t.Errorf("unexpected PhoneNumber %q", got.PhoneNumber)
	}
	if len(got.IceBreakers) != 2 || got.IceBreakers[1] != "Book a table" {
		t.Errorf("unexpected IceBreakers %v", got.IceBreakers)
	}
	if len(got.Commands) != 1 || got.Commands[0].Command != "menu" || got.Commands[0].Description != "See today's menu" {
		t.Errorf("unexpected Commands %+v", got.Commands)
	}

	if _, err := client.WhatsApp.Senders.GetConversationalComponents(context.Background(), ""); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty phoneNumber, got %v", err)
	}
}

func TestWhatsAppSendersUpdateConversationalComponents_Bodies(t *testing.T) {
	cases := []struct {
		name    string
		req     *UpdateWhatsAppConversationalComponentsRequest
		want    string
		refused bool
	}{
		{
			name: "ice breakers only",
			req:  &UpdateWhatsAppConversationalComponentsRequest{IceBreakers: []string{"What are your hours?"}},
			want: `{"iceBreakers":["What are your hours?"]}`,
		},
		{
			name: "commands only",
			req: &UpdateWhatsAppConversationalComponentsRequest{Commands: []WhatsAppCommand{
				{Command: "menu", Description: "See today's menu"},
			}},
			want: `{"commands":[{"command":"menu","description":"See today's menu"}]}`,
		},
		{
			name: "empty lists clear",
			req:  &UpdateWhatsAppConversationalComponentsRequest{IceBreakers: []string{}, Commands: []WhatsAppCommand{}},
			want: `{"iceBreakers":[],"commands":[]}`,
		},
		{
			name:    "nothing set",
			req:     &UpdateWhatsAppConversationalComponentsRequest{},
			want:    `{}`,
			refused: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PATCH" {
					t.Errorf("expected PATCH, got %s", r.Method)
				}
				if r.URL.EscapedPath() != "/whatsapp/senders/+15125550123/conversational_components" {
					t.Errorf("unexpected path %q", r.URL.EscapedPath())
				}
				body, _ := io.ReadAll(r.Body)
				var got, want interface{}
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("body is not JSON: %s", body)
				}
				json.Unmarshal([]byte(tc.want), &want)
				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(want)
				if string(gotJSON) != string(wantJSON) {
					t.Errorf("body = %s, want %s", gotJSON, wantJSON)
				}
				if tc.refused {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{"error":"invalid_request","message":"Provide iceBreakers, commands, or both."}`))
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"phoneNumber":"+15125550123","iceBreakers":[],"commands":[]}`))
			}))
			defer server.Close()

			client := NewClient("test-api-key", WithBaseURL(server.URL))
			got, err := client.WhatsApp.Senders.UpdateConversationalComponents(context.Background(), "+15125550123", tc.req)
			if tc.refused {
				var ve *ValidationError
				if !errors.As(err, &ve) || ve.Code != "invalid_request" || ve.Message != "Provide iceBreakers, commands, or both." {
					t.Errorf("expected the server's 400 invalid_request, got %T %v", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.IceBreakers == nil || len(got.IceBreakers) != 0 || got.Commands == nil || len(got.Commands) != 0 {
				t.Errorf("expected empty, non-nil lists, got %+v", got)
			}
		})
	}
}

func TestWhatsAppSendersUpdateConversationalComponents_ErrorsAndValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_request","message":"At most 4 ice breakers are allowed."}`))
	}))
	defer server.Close()
	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.WhatsApp.Senders.UpdateConversationalComponents(ctx, "+15125550123", &UpdateWhatsAppConversationalComponentsRequest{
		IceBreakers: []string{"a", "b", "c", "d", "e"},
	})
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Code != "invalid_request" || ve.Message != "At most 4 ice breakers are allowed." {
		t.Errorf("expected the server's 400 invalid_request, got %T %v", err, err)
	}

	if _, err := client.WhatsApp.Senders.UpdateConversationalComponents(ctx, "", &UpdateWhatsAppConversationalComponentsRequest{}); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty phoneNumber, got %v", err)
	}
	if _, err := client.WhatsApp.Senders.UpdateConversationalComponents(ctx, "+15125550123", nil); !IsValidationError(err) {
		t.Errorf("expected ValidationError for nil request, got %v", err)
	}
}

func TestWhatsAppSendersSetCalling(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PATCH" {
				t.Errorf("expected PATCH, got %s", r.Method)
			}
			if r.URL.EscapedPath() != "/whatsapp/senders/+442079460123/calling" {
				t.Errorf("unexpected path %q", r.URL.EscapedPath())
			}
			body, _ := io.ReadAll(r.Body)
			want := `{"enabled":false}`
			if enabled {
				want = `{"enabled":true}`
			}
			if string(body) != want {
				t.Errorf("body = %s, want %s", body, want)
			}
			w.WriteHeader(http.StatusOK)
			if enabled {
				w.Write([]byte(`{"phoneNumber":"+442079460123","callingEnabled":true,"outboundCallingAllowed":true}`))
			} else {
				w.Write([]byte(`{"phoneNumber":"+442079460123","callingEnabled":false,"outboundCallingAllowed":true}`))
			}
		}))

		client := NewClient("test-api-key", WithBaseURL(server.URL))
		got, err := client.WhatsApp.Senders.SetCalling(context.Background(), "+442079460123", enabled)
		server.Close()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.PhoneNumber != "+442079460123" || got.CallingEnabled != enabled || !got.OutboundCallingAllowed {
			t.Errorf("unexpected calling settings %+v", got)
		}
	}
}

func TestWhatsAppSendersSetCalling_Errors(t *testing.T) {
	cases := []struct {
		status int
		body   string
		code   string
		check  func(error) bool
	}{
		{http.StatusConflict, `{"error":"voice_not_enabled","message":"Turn on calls for this number first."}`, CallErrorCodeVoiceNotEnabled, func(err error) bool {
			var se *SendlyError
			return errors.As(err, &se) && se.StatusCode == 409
		}},
		{http.StatusUnprocessableEntity, `{"error":"whatsapp_calling_unavailable","message":"WhatsApp didn't allow calling on this number."}`, WhatsAppErrorCodeCallingUnavailable, IsValidationError},
	}
	for _, tc := range cases {
		var requests int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requests, 1)
			w.WriteHeader(tc.status)
			w.Write([]byte(tc.body))
		}))
		client := NewClient("test-api-key", WithBaseURL(server.URL))
		_, err := client.WhatsApp.Senders.SetCalling(context.Background(), "+442079460123", true)
		server.Close()
		if err == nil || !tc.check(err) {
			t.Errorf("status %d: unexpected error %T %v", tc.status, err, err)
			continue
		}
		if got := errorCode(err); got != tc.code {
			t.Errorf("status %d: expected code %q, got %q", tc.status, tc.code, got)
		}
		if got := atomic.LoadInt32(&requests); got != 1 {
			t.Errorf("status %d: expected 1 request, got %d", tc.status, got)
		}
	}

	client := NewClient("test-api-key")
	if _, err := client.WhatsApp.Senders.SetCalling(context.Background(), "", true); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty phoneNumber, got %v", err)
	}
}

func errorCode(err error) string {
	switch e := err.(type) {
	case *SendlyError:
		return e.Code
	case *ValidationError:
		return e.Code
	case *NotFoundError:
		return e.Code
	case *RateLimitError:
		return e.Code
	case *InsufficientCreditsError:
		return e.Code
	}
	return ""
}

func TestWhatsAppSendersList_ExtrasFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"senders":[
			{"phoneNumber":"+442079460123","displayName":"Acme Bakery","status":"active","qualityRating":"GREEN","businessAccountId":"102938475610293","businessName":"Acme Bakery Ltd","callingEnabled":true,"outboundCallingAllowed":true,"createdAt":"2026-09-30T10:00:00.000Z"},
			{"phoneNumber":"+15125550123","displayName":null,"status":"pending","qualityRating":null,"businessAccountId":null,"businessName":null,"callingEnabled":false,"outboundCallingAllowed":false,"createdAt":"2026-09-30T11:00:00.000Z"}
		]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.WhatsApp.Senders.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	active, pending := resp.Senders[0], resp.Senders[1]
	if active.BusinessAccountID == nil || *active.BusinessAccountID != "102938475610293" {
		t.Errorf("unexpected BusinessAccountID %v", active.BusinessAccountID)
	}
	if active.BusinessName == nil || *active.BusinessName != "Acme Bakery Ltd" {
		t.Errorf("unexpected BusinessName %v", active.BusinessName)
	}
	if !active.CallingEnabled || !active.OutboundCallingAllowed {
		t.Errorf("expected calling on and outbound allowed, got %+v", active)
	}
	if pending.BusinessAccountID != nil || pending.BusinessName != nil {
		t.Errorf("expected nil account fields while pending, got %v %v", pending.BusinessAccountID, pending.BusinessName)
	}
	if pending.CallingEnabled || pending.OutboundCallingAllowed {
		t.Errorf("expected calling off and outbound not allowed, got %+v", pending)
	}
}

func TestWhatsAppSignupCreateWithOptions_FacebookFlowMatchesCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"phoneNumber":"+15125550123"}` {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6","connectUrl":"https://sendly.live/whatsapp/connect?token=abc","status":"initiated"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	signup, err := client.WhatsApp.Signup.CreateWithOptions(context.Background(), &CreateWhatsAppSignupRequest{PhoneNumber: "+15125550123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signup.ConnectURL != "https://sendly.live/whatsapp/connect?token=abc" || signup.Status != "initiated" {
		t.Errorf("unexpected session %+v", signup)
	}
}

func TestWhatsAppSignupCreateWithOptions_AddByCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/whatsapp/signup" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") != "add-0123" {
			t.Errorf("expected caller idempotency key, got %q", r.Header.Get("Idempotency-Key"))
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		want := map[string]interface{}{
			"phoneNumber":        "+15125550123",
			"businessAccountId":  "102938475610293",
			"verificationMethod": "voice",
			"displayName":        "Acme Bakery",
		}
		if len(body) != len(want) {
			t.Errorf("body = %v", body)
		}
		for k, v := range want {
			if body[k] != v {
				t.Errorf("body[%s] = %v, want %v", k, body[k], v)
			}
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6","status":"verifying","phoneNumber":"+15125550123","businessAccountId":"102938475610293","failureReasons":null,"verificationMethod":"voice","verificationAttemptsRemaining":5,"updatedAt":"2026-10-01T09:00:00.000Z"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	signup, err := client.WhatsApp.Signup.CreateWithOptions(context.Background(), &CreateWhatsAppSignupRequest{
		PhoneNumber:        "+15125550123",
		BusinessAccountID:  "102938475610293",
		VerificationMethod: WhatsAppVerificationMethodVoice,
		DisplayName:        "Acme Bakery",
	}, WithIdempotencyKey("add-0123"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signup.Status != WhatsAppSignupStatusVerifying {
		t.Errorf("expected verifying, got %q", signup.Status)
	}
	if signup.ConnectURL != "" {
		t.Errorf("expected no ConnectURL, got %q", signup.ConnectURL)
	}
	if signup.PhoneNumber != "+15125550123" || signup.UpdatedAt != "2026-10-01T09:00:00.000Z" {
		t.Errorf("unexpected session %+v", signup)
	}
	if signup.BusinessAccountID == nil || *signup.BusinessAccountID != "102938475610293" {
		t.Errorf("unexpected BusinessAccountID %v", signup.BusinessAccountID)
	}
	if signup.VerificationMethod != "voice" {
		t.Errorf("unexpected VerificationMethod %q", signup.VerificationMethod)
	}
	if signup.VerificationAttemptsRemaining == nil || *signup.VerificationAttemptsRemaining != 5 {
		t.Errorf("unexpected VerificationAttemptsRemaining %v", signup.VerificationAttemptsRemaining)
	}
}

func TestWhatsAppSignupCreateWithOptions_AddByCodeServerErrorIsNotRetried(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"whatsapp_verification_start_failed","message":"WhatsApp couldn't start verifying this number. Any setup fee is refunded automatically. Please try again shortly."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.WhatsApp.Signup.CreateWithOptions(context.Background(), &CreateWhatsAppSignupRequest{
		PhoneNumber:       "+15125550123",
		BusinessAccountID: "102938475610293",
	})
	var se *SendlyError
	if !errors.As(err, &se) || se.StatusCode != 502 || se.Code != WhatsAppErrorCodeVerificationStartFailed {
		t.Fatalf("expected 502 whatsapp_verification_start_failed, got %T %v", err, err)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("expected 1 request (a retry would start and charge a new session), got %d", got)
	}
}

func TestWhatsAppSignupCreateWithOptions_Validation(t *testing.T) {
	client := NewClient("test-api-key", WithBaseURL("http://127.0.0.1:1"))
	ctx := context.Background()
	if _, err := client.WhatsApp.Signup.CreateWithOptions(ctx, nil); !IsValidationError(err) {
		t.Errorf("expected ValidationError for nil request, got %v", err)
	}
	if _, err := client.WhatsApp.Signup.CreateWithOptions(ctx, &CreateWhatsAppSignupRequest{BusinessAccountID: "102938475610293"}); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty phoneNumber, got %v", err)
	}
	if _, err := client.WhatsApp.Signup.CreateWithOptions(ctx, &CreateWhatsAppSignupRequest{PhoneNumber: "+15125550123", BusinessAccountID: "   "}); !IsValidationError(err) {
		t.Errorf("expected ValidationError for whitespace-only businessAccountId, got %v", err)
	}
}

func TestWhatsAppSignupCreateWithOptions_LeavesCallerOptionsUntouched(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Idempotency-Key") != "add-0123" {
			t.Errorf("expected caller idempotency key, got %q", r.Header.Get("Idempotency-Key"))
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6","status":"verifying","phoneNumber":"+15125550123","businessAccountId":"102938475610293","failureReasons":null,"verificationMethod":"sms","verificationAttemptsRemaining":5,"updatedAt":"2026-10-01T09:00:00.000Z"}`))
	}))
	defer server.Close()

	opts := make([]RequestOption, 1, 2)
	opts[0] = WithIdempotencyKey("add-0123")
	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.WhatsApp.Signup.CreateWithOptions(context.Background(), &CreateWhatsAppSignupRequest{
		PhoneNumber:       "+15125550123",
		BusinessAccountID: "102938475610293",
	}, opts...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts[:2][1] != nil {
		t.Error("CreateWithOptions wrote an option into the spare capacity of the caller's slice")
	}
}

func TestWhatsAppSignupCreateWithOptions_VerificationInProgressCarriesID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":"whatsapp_verification_in_progress","message":"This number is already being added to a connected WhatsApp Business account.","id":"8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.WhatsApp.Signup.Create(context.Background(), "+15125550123")
	var se *SendlyError
	if !errors.As(err, &se) || se.StatusCode != 409 || se.Code != WhatsAppErrorCodeVerificationInProgress {
		t.Fatalf("expected 409 whatsapp_verification_in_progress, got %T %v", err, err)
	}
	var id string
	if err := json.Unmarshal(se.Extra["id"], &id); err != nil || id != "8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6" {
		t.Errorf("expected the session id in Extra, got %q (%v)", id, err)
	}
}

func TestWhatsAppSignupGet_Verifying(t *testing.T) {
	cases := []struct {
		name string
		code string
		want *string
	}{
		{"code arrived", `"482913"`, strPtr("482913")},
		{"no code yet", `null`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id":"8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6","status":"verifying","phoneNumber":"+15125550123","businessAccountId":"102938475610293","failureReasons":null,"verificationMethod":"sms","verificationAttemptsRemaining":4,"updatedAt":"2026-10-01T09:00:00.000Z","verificationCode":` + tc.code + `}`))
			}))
			defer server.Close()

			client := NewClient("test-api-key", WithBaseURL(server.URL))
			signup, err := client.WhatsApp.Signup.Get(context.Background(), "8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if signup.Status != WhatsAppSignupStatusVerifying || signup.VerificationMethod != WhatsAppVerificationMethodSMS {
				t.Errorf("unexpected signup %+v", signup)
			}
			if signup.VerificationAttemptsRemaining == nil || *signup.VerificationAttemptsRemaining != 4 {
				t.Errorf("unexpected VerificationAttemptsRemaining %v", signup.VerificationAttemptsRemaining)
			}
			if (tc.want == nil) != (signup.VerificationCode == nil) || (tc.want != nil && *signup.VerificationCode != *tc.want) {
				t.Errorf("unexpected VerificationCode %v", signup.VerificationCode)
			}
		})
	}
}

func TestWhatsAppSignupGet_NewFailureReasons(t *testing.T) {
	for _, reason := range []string{"verification_start_failed", "verification_failed", "verification_expired"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"s1","status":"failed","phoneNumber":"+15125550123","businessAccountId":null,"failureReasons":["` + reason + `"],"updatedAt":"2026-10-01T09:00:00.000Z"}`))
		}))
		client := NewClient("test-api-key", WithBaseURL(server.URL))
		signup, err := client.WhatsApp.Signup.Get(context.Background(), "s1")
		server.Close()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(signup.FailureReasons) != 1 || signup.FailureReasons[0] != reason {
			t.Errorf("unexpected FailureReasons %v", signup.FailureReasons)
		}
		if signup.VerificationAttemptsRemaining != nil || signup.VerificationCode != nil || signup.VerificationMethod != "" {
			t.Errorf("expected no verification fields on a failed signup, got %+v", signup)
		}
	}
}

func TestWhatsAppSignupVerify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/whatsapp/signup/8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6/verify" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"code":"482-913"}` {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6","status":"active","phoneNumber":"+15125550123","businessAccountId":"102938475610293","failureReasons":null,"updatedAt":"2026-10-01T09:02:00.000Z"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	signup, err := client.WhatsApp.Signup.Verify(context.Background(), "8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6", "482-913")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signup.Status != "active" || signup.BusinessAccountID == nil || *signup.BusinessAccountID != "102938475610293" {
		t.Errorf("unexpected signup %+v", signup)
	}
}

func TestWhatsAppSignupVerify_WrongCodeCarriesAttemptsRemaining(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error":"whatsapp_verification_code_invalid","message":"That code wasn't accepted. Check it, or request a new one.","attemptsRemaining":3}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.WhatsApp.Signup.Verify(context.Background(), "s1", "000000")
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Code != WhatsAppErrorCodeVerificationCodeInvalid {
		t.Fatalf("expected *ValidationError whatsapp_verification_code_invalid, got %T %v", err, err)
	}
	var remaining int
	if err := json.Unmarshal(ve.Extra["attemptsRemaining"], &remaining); err != nil || remaining != 3 {
		t.Errorf("expected attemptsRemaining 3 in Extra, got %d (%v)", remaining, err)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("expected 1 request, got %d", got)
	}
}

func TestWhatsAppSignupVerify_ServerErrorIsNotRetried(t *testing.T) {
	cases := []struct {
		name string
		body string
		code string
	}{
		{"activation pending", `{"error":"whatsapp_activation_pending","message":"WhatsApp accepted the code, but we couldn't finish connecting the number. Our team has been alerted; check back shortly."}`, WhatsAppErrorCodeActivationPending},
		{"verification unavailable", `{"error":"whatsapp_verification_unavailable","message":"WhatsApp couldn't check the code right now. Please try again shortly."}`, WhatsAppErrorCodeVerificationUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var requests int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&requests, 1)
				w.WriteHeader(http.StatusBadGateway)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(1))
			_, err := client.WhatsApp.Signup.Verify(context.Background(), "8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6", "482913")
			var se *SendlyError
			if !errors.As(err, &se) || se.StatusCode != 502 || se.Code != tc.code {
				t.Fatalf("expected 502 %s, got %T %v", tc.code, err, err)
			}
			if got := atomic.LoadInt32(&requests); got != 1 {
				t.Errorf("expected the code to be submitted once, got %d requests", got)
			}
		})
	}
}

func TestWhatsAppSignupVerify_Validation(t *testing.T) {
	client := NewClient("test-api-key", WithBaseURL("http://127.0.0.1:1"))
	ctx := context.Background()
	if _, err := client.WhatsApp.Signup.Verify(ctx, "", "482913"); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty id, got %v", err)
	}
	if _, err := client.WhatsApp.Signup.Verify(ctx, "s1", ""); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty code, got %v", err)
	}
}

func TestWhatsAppSignupResend(t *testing.T) {
	cases := []struct {
		method string
		want   string
	}{
		{"", `{}`},
		{WhatsAppVerificationMethodVoice, `{"verificationMethod":"voice"}`},
	}
	for _, tc := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" || r.URL.Path != "/whatsapp/signup/s1/resend" {
				t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != tc.want {
				t.Errorf("body = %s, want %s", body, tc.want)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"s1","status":"verifying","phoneNumber":"+15125550123","businessAccountId":"102938475610293","failureReasons":null,"verificationMethod":"voice","verificationAttemptsRemaining":5,"updatedAt":"2026-10-01T09:01:00.000Z"}`))
		}))
		client := NewClient("test-api-key", WithBaseURL(server.URL))
		signup, err := client.WhatsApp.Signup.Resend(context.Background(), "s1", tc.method)
		server.Close()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if signup.Status != "verifying" || signup.VerificationMethod != "voice" {
			t.Errorf("unexpected signup %+v", signup)
		}
	}

	client := NewClient("test-api-key", WithBaseURL("http://127.0.0.1:1"))
	if _, err := client.WhatsApp.Signup.Resend(context.Background(), "", ""); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty id, got %v", err)
	}
}

func TestWhatsAppSignupResend_TooSoonIsReturnedAtOnce(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Retry-After", "21")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"whatsapp_verification_resend_too_soon","message":"Wait 21 seconds before requesting another code.","retryAfter":21}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	start := time.Now()
	_, err := client.WhatsApp.Signup.Resend(context.Background(), "s1", "")
	var rl *RateLimitError
	if !errors.As(err, &rl) || rl.Code != WhatsAppErrorCodeVerificationResendTooSoon || rl.RetryAfter != 21 {
		t.Fatalf("expected *RateLimitError whatsapp_verification_resend_too_soon with RetryAfter 21, got %T %v", err, err)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("expected 1 request, got %d", got)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("expected an immediate return, took %s", elapsed)
	}
}

func TestMessagesSendWhatsApp_UnconfirmedIsNotRetried(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":"whatsapp_send_unconfirmed","errorCode":"E024","message":"We couldn't confirm whether WhatsApp accepted this message. It has been marked failed and refunded, but it may still be delivered. Check before sending it again, or it could arrive twice."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.Messages.SendWhatsApp(context.Background(), &SendWhatsAppMessageRequest{
		To:   "+15125550100",
		From: "+15125550123",
		Text: "Your table is ready!",
	})
	var se *SendlyError
	if !errors.As(err, &se) || se.StatusCode != 409 || se.Code != WhatsAppErrorCodeSendUnconfirmed {
		t.Fatalf("expected 409 *SendlyError whatsapp_send_unconfirmed, got %T %v", err, err)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("expected 1 request (an unconfirmed send must not be retried), got %d", got)
	}
	var classifier string
	if err := json.Unmarshal(se.Extra["errorCode"], &classifier); err != nil || classifier != "E024" {
		t.Errorf("expected errorCode E024 in Extra, got %q (%v)", classifier, err)
	}
	if WhatsAppErrorCodeSendFailed != "whatsapp_send_failed" {
		t.Errorf("unexpected WhatsAppErrorCodeSendFailed %q", WhatsAppErrorCodeSendFailed)
	}
}

func TestWhatsAppErrorCodeConstants(t *testing.T) {
	want := map[string]string{
		WhatsAppErrorCodeSendFailed:                     "whatsapp_send_failed",
		WhatsAppErrorCodeSendUnconfirmed:                "whatsapp_send_unconfirmed",
		WhatsAppErrorCodeFileRequired:                   "file_required",
		WhatsAppErrorCodeProfilePhotoInvalid:            "whatsapp_profile_photo_invalid",
		WhatsAppErrorCodeProfilePhotoTooLarge:           "whatsapp_profile_photo_too_large",
		WhatsAppErrorCodeProfileUpdateFailed:            "whatsapp_profile_update_failed",
		WhatsAppErrorCodeConversationalComponentsFetch:  "whatsapp_conversational_components_fetch_failed",
		WhatsAppErrorCodeConversationalComponentsUpdate: "whatsapp_conversational_components_update_failed",
		WhatsAppErrorCodeCallingUnavailable:             "whatsapp_calling_unavailable",
		WhatsAppErrorCodeCallingUpdateFailed:            "whatsapp_calling_update_failed",
		WhatsAppErrorCodeBusinessAccountNotFound:        "whatsapp_business_account_not_found",
		WhatsAppErrorCodeDisplayNameRequired:            "display_name_required",
		WhatsAppErrorCodeSignupInProgress:               "whatsapp_signup_in_progress",
		WhatsAppErrorCodeAlreadyEnabled:                 "whatsapp_already_enabled",
		WhatsAppErrorCodeVerificationStartFailed:        "whatsapp_verification_start_failed",
		WhatsAppErrorCodeVerificationInProgress:         "whatsapp_verification_in_progress",
		WhatsAppErrorCodeInvalidVerificationCode:        "invalid_verification_code",
		WhatsAppErrorCodeVerificationCodeInvalid:        "whatsapp_verification_code_invalid",
		WhatsAppErrorCodeVerificationFailed:             "whatsapp_verification_failed",
		WhatsAppErrorCodeVerificationBusy:               "whatsapp_verification_busy",
		WhatsAppErrorCodeVerificationUnavailable:        "whatsapp_verification_unavailable",
		WhatsAppErrorCodeActivationPending:              "whatsapp_activation_pending",
		WhatsAppErrorCodeSignupNotActive:                "signup_not_active",
		WhatsAppErrorCodeSignupNotFound:                 "signup_not_found",
		WhatsAppErrorCodeVerificationResendTooSoon:      "whatsapp_verification_resend_too_soon",
		WhatsAppErrorCodeVerificationResendFailed:       "whatsapp_verification_resend_failed",
	}
	for got, expected := range want {
		if got != expected {
			t.Errorf("constant %q, want %q", got, expected)
		}
	}
	if len(want) != 26 {
		t.Errorf("expected 26 distinct codes, got %d", len(want))
	}
}

func TestCallChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[
			{"id":"c1","object":"call","kind":"pstn","channel":"whatsapp","direction":"inbound","status":"completed","handledBy":"dashboard","agentId":null,"from":"+442079460199","to":"+442079460123","callerName":null,"calleeName":null,"startedAt":"2026-10-01T09:00:00.000Z","answeredAt":null,"endedAt":null,"durationSecs":0,"creditsCharged":0,"billing":"settled","hangupClass":null,"recordingStatus":null,"metadata":{}},
			{"id":"c2","object":"call","kind":"pstn","channel":"phone","direction":"outbound","status":"completed","handledBy":"agent","agentId":"a1","from":"+15125550123","to":"+15125550100","callerName":null,"calleeName":null,"startedAt":"2026-10-01T09:00:00.000Z","answeredAt":null,"endedAt":null,"durationSecs":0,"creditsCharged":0,"billing":"settled","hangupClass":null,"recordingStatus":null,"metadata":{}},
			{"id":"c3","object":"call","kind":"internal","channel":"browser","direction":"outbound","status":"completed","handledBy":"dashboard","agentId":null,"from":null,"to":null,"callerName":null,"calleeName":null,"startedAt":"2026-10-01T09:00:00.000Z","answeredAt":null,"endedAt":null,"durationSecs":0,"creditsCharged":0,"billing":"unbilled","hangupClass":null,"recordingStatus":null,"metadata":{}},
			{"id":"c4","object":"call","kind":"pstn","channel":"carrier_pigeon","direction":"inbound","status":"completed","handledBy":"dashboard","agentId":null,"from":null,"to":null,"callerName":null,"calleeName":null,"startedAt":"2026-10-01T09:00:00.000Z","answeredAt":null,"endedAt":null,"durationSecs":0,"creditsCharged":0,"billing":"settled","hangupClass":null,"recordingStatus":null,"metadata":{}}
		],"pagination":{"total":4,"limit":50,"offset":0,"hasMore":false}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	calls, err := client.Calls.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []CallChannel{CallChannelWhatsApp, CallChannelPhone, CallChannelBrowser, CallChannel("carrier_pigeon")}
	for i, c := range calls.Data {
		if c.Channel != want[i] {
			t.Errorf("call %d: Channel = %q, want %q", i, c.Channel, want[i])
		}
	}
}

func TestWebhookCallData_Channel(t *testing.T) {
	payload := `{"id":"evt_10","type":"call.started","api_version":"2024-01","created":1759309200,"livemode":true,"data":{"object":{` +
		`"id":"c1","object":"call","kind":"pstn","channel":"whatsapp","direction":"inbound","status":"ringing",` +
		`"handled_by":"dashboard","agent_id":null,"from":"+442079460199","to":"+442079460123","caller_name":null,"callee_name":null,` +
		`"started_at":"2026-10-01T09:00:00.000Z","answered_at":null,"ended_at":null,"duration_secs":0,"credits_charged":0,` +
		`"billing":"metered","hangup_class":null,"recording_status":null,"metadata":{}}}}`
	sig, ts := signed(t, payload)
	event, err := Webhooks{}.ParseEvent(payload, sig, testSecret, ts)
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	var call WebhookCallData
	if err := event.DecodeObject(&call); err != nil {
		t.Fatalf("DecodeObject: %v", err)
	}
	if call.Channel != CallChannelWhatsApp {
		t.Errorf("Channel = %q, want whatsapp", call.Channel)
	}
}

func strPtr(s string) *string { return &s }

func TestWhatsAppUnsafeCalls_DroppedConnectionIsNotRetried(t *testing.T) {
	cases := []struct {
		name string
		call func(c *Client) error
	}{
		{"verify", func(c *Client) error {
			_, err := c.WhatsApp.Signup.Verify(context.Background(), "8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6", "482913")
			return err
		}},
		{"add by code", func(c *Client) error {
			_, err := c.WhatsApp.Signup.CreateWithOptions(context.Background(), &CreateWhatsAppSignupRequest{
				PhoneNumber:       "+15125550123",
				BusinessAccountID: "102938475610293",
			})
			return err
		}},
		{"profile photo", func(c *Client) error {
			_, err := c.WhatsApp.Senders.UploadProfilePhoto(context.Background(), "+15125550123", "logo.jpg", bytes.NewReader(jpegBytes))
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var requests int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&requests, 1)
				hijackAndClose(t, w)
			}))
			defer server.Close()

			client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(1), WithHTTPClient(noKeepAliveClient()))
			err := tc.call(client)
			if !IsNetworkError(err) {
				t.Fatalf("expected *NetworkError, got %T %v", err, err)
			}
			if got := atomic.LoadInt32(&requests); got != 1 {
				t.Errorf("expected 1 request (the outcome is unknown, so it must not be sent again), got %d", got)
			}
		})
	}
}

func newDroppingKeepAliveServer(t *testing.T) (*Client, *int32) {
	t.Helper()
	var posts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Write([]byte(`{"senders":[]}`))
			return
		}
		io.ReadAll(r.Body)
		atomic.AddInt32(&posts, 1)
		hijackAndClose(t, w)
	}))
	t.Cleanup(server.Close)
	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))
	if _, err := client.WhatsApp.Senders.List(context.Background()); err != nil {
		t.Fatalf("warm-up GET: %v", err)
	}
	return client, &posts
}

func TestWhatsAppUnsafeCalls_NotReplayedOnReusedConnection(t *testing.T) {
	cases := []struct {
		name string
		call func(c *Client) error
	}{
		{"verify", func(c *Client) error {
			_, err := c.WhatsApp.Signup.Verify(context.Background(), "s1", "482913")
			return err
		}},
		{"add by code", func(c *Client) error {
			_, err := c.WhatsApp.Signup.CreateWithOptions(context.Background(), &CreateWhatsAppSignupRequest{PhoneNumber: "+15125550123", BusinessAccountID: "102938475610293"})
			return err
		}},
		{"profile photo", func(c *Client) error {
			_, err := c.WhatsApp.Senders.UploadProfilePhoto(context.Background(), "+15125550123", "logo.jpg", bytes.NewReader(jpegBytes))
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, posts := newDroppingKeepAliveServer(t)
			err := tc.call(client)
			if !IsNetworkError(err) {
				t.Fatalf("expected *NetworkError, got %T %v", err, err)
			}
			if got := atomic.LoadInt32(posts); got != 1 {
				t.Errorf("expected the request to reach the server once (outcome unknown), got %d", got)
			}
		})
	}
}

func TestReusedConnectionReplay_KeptWhereNetworkErrorsAreRetried(t *testing.T) {
	cases := []struct {
		name string
		call func(c *Client) error
	}{
		{"emergency address", func(c *Client) error {
			_, err := c.Voice.Numbers.RegisterEmergencyAddress(context.Background(), "+15125550123", &EmergencyAddress{Street: "500 Example Ave", City: "Austin", State: "TX", Zip: "78701"})
			return err
		}},
		{"facebook flow", func(c *Client) error {
			_, err := c.WhatsApp.Signup.CreateWithOptions(context.Background(), &CreateWhatsAppSignupRequest{PhoneNumber: "+15125550123"})
			return err
		}},
		{"resend", func(c *Client) error { _, err := c.WhatsApp.Signup.Resend(context.Background(), "s1", ""); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, posts := newDroppingKeepAliveServer(t)
			if err := tc.call(client); !IsNetworkError(err) {
				t.Fatalf("expected *NetworkError, got %T %v", err, err)
			}
			if got := atomic.LoadInt32(posts); got != 2 {
				t.Errorf("expected net/http to resend the request once on a new connection, as before, got %d", got)
			}
		})
	}
}

func TestWhatsAppSignupCreateWithOptions_FacebookFlowStillRetriesDroppedConnection(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) == 1 {
			hijackAndClose(t, w)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6","connectUrl":"https://sendly.live/whatsapp/connect?token=abc","status":"initiated"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(1), WithHTTPClient(noKeepAliveClient()))
	if _, err := client.WhatsApp.Signup.CreateWithOptions(context.Background(), &CreateWhatsAppSignupRequest{PhoneNumber: "+15125550123"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Errorf("expected the Facebook flow to retry a dropped connection once, got %d requests", got)
	}
}

func TestWhatsAppSignupVerify_RateLimitIsStillRetried(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&requests, 1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Too many requests.","retryAfter":1}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6","status":"active","phoneNumber":"+15125550123","businessAccountId":"102938475610293","failureReasons":null,"updatedAt":"2026-10-01T09:02:00.000Z"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(1))
	signup, err := client.WhatsApp.Signup.Verify(context.Background(), "8d0f1c2a-3b4c-4d5e-8f60-718293a4b5c6", "482913")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signup.Status != "active" || atomic.LoadInt32(&requests) != 2 {
		t.Errorf("expected the rate-limited verify to be retried once, got %d requests, status %q", requests, signup.Status)
	}
}
