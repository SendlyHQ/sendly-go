package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWhatsAppSignupCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/whatsapp/signup" {
			t.Errorf("expected path '/whatsapp/signup', got '%s'", r.URL.Path)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["phoneNumber"] != "+15559876543" {
			t.Errorf("expected phoneNumber to be '+15559876543', got '%s'", body["phoneNumber"])
		}

		resp := WhatsAppSignupSession{
			ID:         "was_123",
			ConnectURL: "https://sendly.live/whatsapp/connect/abc",
			Status:     "initiated",
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	signup, err := client.WhatsApp.Signup.Create(ctx, "+15559876543")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signup.ID != "was_123" {
		t.Errorf("expected ID to be 'was_123', got '%s'", signup.ID)
	}
	if signup.ConnectURL != "https://sendly.live/whatsapp/connect/abc" {
		t.Errorf("expected ConnectURL to be set, got '%s'", signup.ConnectURL)
	}
	if signup.Status != "initiated" {
		t.Errorf("expected Status to be 'initiated', got '%s'", signup.Status)
	}
}

func TestWhatsAppSignupCreate_EmptyPhoneNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with empty phone number")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.WhatsApp.Signup.Create(ctx, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(err.Error(), "phoneNumber is required") {
		t.Errorf("expected error to contain 'phoneNumber is required', got '%s'", err.Error())
	}
}

func TestWhatsAppSignupGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/whatsapp/signup/was_123" {
			t.Errorf("expected path '/whatsapp/signup/was_123', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "was_123",
			"status": "initiated",
			"phoneNumber": "+15559876543",
			"businessAccountId": null,
			"failureReasons": null,
			"updatedAt": "2026-07-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	signup, err := client.WhatsApp.Signup.Get(ctx, "was_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signup.ID != "was_123" {
		t.Errorf("expected ID to be 'was_123', got '%s'", signup.ID)
	}
	if signup.PhoneNumber != "+15559876543" {
		t.Errorf("expected PhoneNumber to be '+15559876543', got '%s'", signup.PhoneNumber)
	}
	if signup.BusinessAccountID != nil {
		t.Errorf("expected BusinessAccountID to be nil, got '%s'", *signup.BusinessAccountID)
	}
	if signup.FailureReasons != nil {
		t.Errorf("expected FailureReasons to be nil, got %v", signup.FailureReasons)
	}
}

func TestWhatsAppSignupGet_Failed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "was_123",
			"status": "failed",
			"phoneNumber": "+15559876543",
			"businessAccountId": "1234567890",
			"failureReasons": ["display_name_rejected"],
			"updatedAt": "2026-07-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	signup, err := client.WhatsApp.Signup.Get(ctx, "was_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signup.Status != "failed" {
		t.Errorf("expected Status to be 'failed', got '%s'", signup.Status)
	}
	if signup.BusinessAccountID == nil || *signup.BusinessAccountID != "1234567890" {
		t.Errorf("expected BusinessAccountID to be '1234567890', got %v", signup.BusinessAccountID)
	}
	if len(signup.FailureReasons) != 1 || signup.FailureReasons[0] != "display_name_rejected" {
		t.Errorf("expected FailureReasons to be ['display_name_rejected'], got %v", signup.FailureReasons)
	}
}

func TestWhatsAppSignupGet_EmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with empty ID")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.WhatsApp.Signup.Get(ctx, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestWhatsAppSignupGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Code:    "NOT_FOUND",
			Message: "Signup not found",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.WhatsApp.Signup.Get(ctx, "was_nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestWhatsAppSendersList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/whatsapp/senders" {
			t.Errorf("expected path '/whatsapp/senders', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"senders": [
				{
					"phoneNumber": "+15559876543",
					"displayName": "Acme Inc",
					"status": "active",
					"qualityRating": "GREEN",
					"createdAt": "2026-07-01T00:00:00Z"
				},
				{
					"phoneNumber": "+15551112222",
					"displayName": null,
					"status": "pending",
					"qualityRating": null,
					"createdAt": "2026-07-02T00:00:00Z"
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.WhatsApp.Senders.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Senders) != 2 {
		t.Fatalf("expected 2 senders, got %d", len(resp.Senders))
	}
	if resp.Senders[0].PhoneNumber != "+15559876543" {
		t.Errorf("expected PhoneNumber to be '+15559876543', got '%s'", resp.Senders[0].PhoneNumber)
	}
	if resp.Senders[0].DisplayName == nil || *resp.Senders[0].DisplayName != "Acme Inc" {
		t.Errorf("expected DisplayName to be 'Acme Inc', got %v", resp.Senders[0].DisplayName)
	}
	if resp.Senders[0].QualityRating == nil || *resp.Senders[0].QualityRating != "GREEN" {
		t.Errorf("expected QualityRating to be 'GREEN', got %v", resp.Senders[0].QualityRating)
	}
	if resp.Senders[1].DisplayName != nil {
		t.Errorf("expected DisplayName to be nil, got '%s'", *resp.Senders[1].DisplayName)
	}
	if resp.Senders[1].Status != "pending" {
		t.Errorf("expected Status to be 'pending', got '%s'", resp.Senders[1].Status)
	}
	if resp.Senders[1].QualityRating != nil {
		t.Errorf("expected QualityRating to be nil, got '%s'", *resp.Senders[1].QualityRating)
	}
}

func TestWhatsAppSendersGetProfile_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/whatsapp/senders/+15559876543/profile" {
			t.Errorf("expected path '/whatsapp/senders/+15559876543/profile', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"phoneNumber": "+15559876543",
			"displayName": "Acme Inc",
			"profilePhotoUrl": "https://example.com/logo.png",
			"category": "Retail",
			"about": "Family-run since 1998",
			"description": "Order updates and support over WhatsApp.",
			"email": "support@example.com",
			"website": "https://example.com",
			"address": null
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	profile, err := client.WhatsApp.Senders.GetProfile(ctx, "+15559876543")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.PhoneNumber != "+15559876543" {
		t.Errorf("expected PhoneNumber to be '+15559876543', got '%s'", profile.PhoneNumber)
	}
	if profile.DisplayName == nil || *profile.DisplayName != "Acme Inc" {
		t.Errorf("expected DisplayName to be 'Acme Inc', got %v", profile.DisplayName)
	}
	if profile.ProfilePhotoURL == nil || *profile.ProfilePhotoURL != "https://example.com/logo.png" {
		t.Errorf("expected ProfilePhotoURL to be set, got %v", profile.ProfilePhotoURL)
	}
	if profile.About == nil || *profile.About != "Family-run since 1998" {
		t.Errorf("expected About to be 'Family-run since 1998', got %v", profile.About)
	}
	if profile.Address != nil {
		t.Errorf("expected Address to be nil, got '%s'", *profile.Address)
	}
}

func TestWhatsAppSendersGetProfile_EmptyPhoneNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with empty phone number")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.WhatsApp.Senders.GetProfile(ctx, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(err.Error(), "phoneNumber is required") {
		t.Errorf("expected error to contain 'phoneNumber is required', got '%s'", err.Error())
	}
}

func TestWhatsAppSendersGetProfile_NotConnected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Code:    "whatsapp_sender_not_connected",
			Message: "This number isn't connected to WhatsApp yet.",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.WhatsApp.Senders.GetProfile(ctx, "+15559876543")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestWhatsAppSendersUpdateProfile_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH request, got %s", r.Method)
		}
		if r.URL.Path != "/whatsapp/senders/+15559876543/profile" {
			t.Errorf("expected path '/whatsapp/senders/+15559876543/profile', got '%s'", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["about"] != "Family-run since 1998" {
			t.Errorf("expected about to round-trip, got '%v'", body["about"])
		}
		if body["website"] != "https://example.com" {
			t.Errorf("expected website to round-trip, got '%v'", body["website"])
		}
		if _, ok := body["displayName"]; ok {
			t.Error("expected displayName to be omitted")
		}
		if _, ok := body["description"]; ok {
			t.Error("expected description to be omitted")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"phoneNumber": "+15559876543",
			"displayName": "Acme Inc",
			"profilePhotoUrl": null,
			"category": null,
			"about": "Family-run since 1998",
			"description": null,
			"email": null,
			"website": "https://example.com",
			"address": null
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	profile, err := client.WhatsApp.Senders.UpdateProfile(ctx, "+15559876543", &UpdateWhatsAppSenderProfileRequest{
		About:   "Family-run since 1998",
		Website: "https://example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.About == nil || *profile.About != "Family-run since 1998" {
		t.Errorf("expected About to be 'Family-run since 1998', got %v", profile.About)
	}
	if profile.Website == nil || *profile.Website != "https://example.com" {
		t.Errorf("expected Website to be 'https://example.com', got %v", profile.Website)
	}
	if profile.Description != nil {
		t.Errorf("expected Description to be nil, got '%s'", *profile.Description)
	}
}

func TestWhatsAppSendersUpdateProfile_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.WhatsApp.Senders.UpdateProfile(ctx, "", &UpdateWhatsAppSenderProfileRequest{About: "Hi"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(err.Error(), "phoneNumber is required") {
		t.Errorf("expected error to contain 'phoneNumber is required', got '%s'", err.Error())
	}

	_, err = client.WhatsApp.Senders.UpdateProfile(ctx, "+15559876543", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(err.Error(), "request is required") {
		t.Errorf("expected error to contain 'request is required', got '%s'", err.Error())
	}
}

func TestWhatsAppTemplatesList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/whatsapp/templates" {
			t.Errorf("expected path '/whatsapp/templates', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"templates": [
				{
					"id": "wat_1",
					"name": "order_shipped",
					"language": "en_US",
					"category": "UTILITY",
					"status": "APPROVED",
					"qualityRating": "GREEN",
					"rejectionReason": null,
					"createdAt": "2026-07-01T00:00:00Z",
					"updatedAt": "2026-07-02T00:00:00Z"
				},
				{
					"id": "wat_2",
					"name": "spring_sale",
					"language": "en_US",
					"category": "MARKETING",
					"status": "REJECTED",
					"qualityRating": null,
					"rejectionReason": "INVALID_FORMAT",
					"createdAt": "2026-07-03T00:00:00Z",
					"updatedAt": "2026-07-04T00:00:00Z"
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.WhatsApp.Templates.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(resp.Templates))
	}
	if resp.Templates[0].Name != "order_shipped" {
		t.Errorf("expected Name to be 'order_shipped', got '%s'", resp.Templates[0].Name)
	}
	if resp.Templates[0].Status != "APPROVED" {
		t.Errorf("expected Status to be 'APPROVED', got '%s'", resp.Templates[0].Status)
	}
	if resp.Templates[0].RejectionReason != nil {
		t.Errorf("expected RejectionReason to be nil, got '%s'", *resp.Templates[0].RejectionReason)
	}
	if resp.Templates[1].RejectionReason == nil || *resp.Templates[1].RejectionReason != "INVALID_FORMAT" {
		t.Errorf("expected RejectionReason to be 'INVALID_FORMAT', got %v", resp.Templates[1].RejectionReason)
	}
}

func TestWhatsAppTemplatesCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/whatsapp/templates" {
			t.Errorf("expected path '/whatsapp/templates', got '%s'", r.URL.Path)
		}

		var req CreateWhatsAppTemplateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Sender != "+15559876543" {
			t.Errorf("expected sender to be '+15559876543', got '%s'", req.Sender)
		}
		if req.Name != "order_shipped" {
			t.Errorf("expected name to be 'order_shipped', got '%s'", req.Name)
		}
		if req.Category != "UTILITY" {
			t.Errorf("expected category to be 'UTILITY', got '%s'", req.Category)
		}
		if req.Examples["1"] != "Sam" || req.Examples["2"] != "#4821" {
			t.Errorf("expected examples to round-trip, got %v", req.Examples)
		}
		if len(req.Buttons) != 1 || req.Buttons[0].Type != "quick_reply" {
			t.Errorf("expected 1 quick_reply button, got %v", req.Buttons)
		}

		resp := WhatsAppTemplate{
			ID:        "wat_1",
			Name:      req.Name,
			Language:  req.Language,
			Category:  req.Category,
			Status:    "PENDING",
			CreatedAt: "2026-07-01T00:00:00Z",
			UpdatedAt: "2026-07-01T00:00:00Z",
			Warnings:  []string{"display_name_not_approved"},
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	template, err := client.WhatsApp.Templates.Create(ctx, &CreateWhatsAppTemplateRequest{
		Sender:   "+15559876543",
		Name:     "order_shipped",
		Language: "en_US",
		Category: "UTILITY",
		Body:     "Hi {{1}}, your order {{2}} has shipped!",
		Buttons: []WhatsAppTemplateButton{
			{Type: "quick_reply", Text: "Stop promotions"},
		},
		Examples: map[string]string{"1": "Sam", "2": "#4821"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if template.ID != "wat_1" {
		t.Errorf("expected ID to be 'wat_1', got '%s'", template.ID)
	}
	if template.Status != "PENDING" {
		t.Errorf("expected Status to be 'PENDING', got '%s'", template.Status)
	}
	if len(template.Warnings) != 1 || template.Warnings[0] != "display_name_not_approved" {
		t.Errorf("expected Warnings to be ['display_name_not_approved'], got %v", template.Warnings)
	}
}

func TestWhatsAppTemplatesCreate_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	valid := CreateWhatsAppTemplateRequest{
		Sender:   "+15559876543",
		Name:     "order_shipped",
		Language: "en_US",
		Category: "UTILITY",
		Body:     "Your order has shipped!",
	}

	tests := []struct {
		name        string
		mutate      func(req *CreateWhatsAppTemplateRequest)
		nilReq      bool
		expectedErr string
	}{
		{
			name:        "nil request",
			nilReq:      true,
			expectedErr: "request is required",
		},
		{
			name:        "empty sender",
			mutate:      func(req *CreateWhatsAppTemplateRequest) { req.Sender = "" },
			expectedErr: "sender is required",
		},
		{
			name:        "empty name",
			mutate:      func(req *CreateWhatsAppTemplateRequest) { req.Name = "" },
			expectedErr: "name is required",
		},
		{
			name:        "empty language",
			mutate:      func(req *CreateWhatsAppTemplateRequest) { req.Language = "" },
			expectedErr: "language is required",
		},
		{
			name:        "empty category",
			mutate:      func(req *CreateWhatsAppTemplateRequest) { req.Category = "" },
			expectedErr: "category is required",
		},
		{
			name:        "empty body",
			mutate:      func(req *CreateWhatsAppTemplateRequest) { req.Body = "" },
			expectedErr: "body is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *CreateWhatsAppTemplateRequest
			if !tt.nilReq {
				r := valid
				tt.mutate(&r)
				req = &r
			}

			_, err := client.WhatsApp.Templates.Create(ctx, req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error to contain '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestWhatsAppTemplatesUpdate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH request, got %s", r.Method)
		}
		if r.URL.Path != "/whatsapp/templates/wat_1" {
			t.Errorf("expected path '/whatsapp/templates/wat_1', got '%s'", r.URL.Path)
		}

		var req UpdateWhatsAppTemplateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Body != "Hi {{1}}, your order {{2}} is on its way!" {
			t.Errorf("unexpected body: '%s'", req.Body)
		}

		resp := WhatsAppTemplate{
			ID:        "wat_1",
			Name:      "order_shipped",
			Language:  "en_US",
			Category:  "UTILITY",
			Status:    "PENDING",
			CreatedAt: "2026-07-01T00:00:00Z",
			UpdatedAt: "2026-07-05T00:00:00Z",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	template, err := client.WhatsApp.Templates.Update(ctx, "wat_1", &UpdateWhatsAppTemplateRequest{
		Body:     "Hi {{1}}, your order {{2}} is on its way!",
		Examples: map[string]string{"1": "Sam", "2": "#4821"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if template.Status != "PENDING" {
		t.Errorf("expected Status to be 'PENDING', got '%s'", template.Status)
	}
}

func TestWhatsAppTemplatesUpdate_EmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with empty ID")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.WhatsApp.Templates.Update(ctx, "", &UpdateWhatsAppTemplateRequest{Body: "New body"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestWhatsAppTemplatesDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/whatsapp/templates/wat_1" {
			t.Errorf("expected path '/whatsapp/templates/wat_1', got '%s'", r.URL.Path)
		}

		resp := WhatsAppTemplateDeletedResponse{
			ID:      "wat_1",
			Deleted: true,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	result, err := client.WhatsApp.Templates.Delete(ctx, "wat_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "wat_1" {
		t.Errorf("expected ID to be 'wat_1', got '%s'", result.ID)
	}
	if !result.Deleted {
		t.Error("expected Deleted to be true")
	}
}

func TestWhatsAppTemplatesDelete_EmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with empty ID")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.WhatsApp.Templates.Delete(ctx, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestWhatsAppWindow_Open(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/whatsapp/window" {
			t.Errorf("expected path '/whatsapp/window', got '%s'", r.URL.Path)
		}

		query := r.URL.Query()
		if from := query.Get("from"); from != "+15559876543" {
			t.Errorf("expected from to be '+15559876543', got '%s'", from)
		}
		if to := query.Get("to"); to != "+15551234567" {
			t.Errorf("expected to to be '+15551234567', got '%s'", to)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"open": true, "expiresAt": "2026-07-01T12:00:00Z"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	window, err := client.WhatsApp.Window(ctx, "+15559876543", "+15551234567")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !window.Open {
		t.Error("expected Open to be true")
	}
	if window.ExpiresAt == nil || *window.ExpiresAt != "2026-07-01T12:00:00Z" {
		t.Errorf("expected ExpiresAt to be '2026-07-01T12:00:00Z', got %v", window.ExpiresAt)
	}
}

func TestWhatsAppWindow_Closed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"open": false, "expiresAt": null}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	window, err := client.WhatsApp.Window(ctx, "+15559876543", "+15551234567")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if window.Open {
		t.Error("expected Open to be false")
	}
	if window.ExpiresAt != nil {
		t.Errorf("expected ExpiresAt to be nil, got '%s'", *window.ExpiresAt)
	}
}

func TestWhatsAppWindow_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	tests := []struct {
		name        string
		from        string
		to          string
		expectedErr string
	}{
		{
			name:        "empty from",
			from:        "",
			to:          "+15551234567",
			expectedErr: "from is required",
		},
		{
			name:        "empty to",
			from:        "+15559876543",
			to:          "",
			expectedErr: "to is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.WhatsApp.Window(ctx, tt.from, tt.to)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error to contain '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}
