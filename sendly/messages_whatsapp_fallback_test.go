package sendly

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestMessagesSendWhatsApp_FallbackToSmsField(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name    string
		value   *bool
		present bool
		want    bool
	}{
		{"true", &yes, true, true},
		{"false", &no, true, false},
		{"nil", nil, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode request: %v", err)
				}
				got, ok := body["fallbackToSms"]
				if ok != tt.present {
					t.Fatalf("expected fallbackToSms present=%v, got present=%v (body %v)", tt.present, ok, body)
				}
				if ok && got != tt.want {
					t.Errorf("expected fallbackToSms to be %v, got %v", tt.want, got)
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"id":"msg_wa","channel":"whatsapp","message_format":"whatsapp","to":"+15551234567","from":"+15559876543","text":"Hi","status":"sent","segments":1,"creditsUsed":1,"whatsapp":{"kind":"text","messageId":"wamid.1"},"createdAt":"2026-10-17T00:00:00Z"}`))
			}))
			defer server.Close()

			client := NewClient("test-api-key", WithBaseURL(server.URL))
			_, err := client.Messages.SendWhatsApp(context.Background(), &SendWhatsAppMessageRequest{
				To:            "+15551234567",
				From:          "+15559876543",
				Text:          "Hi",
				FallbackToSms: tt.value,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestMessagesSendWhatsApp_FallbackSent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"id": "msg_wa",
			"channel": "whatsapp",
			"message_format": "whatsapp",
			"messageFormat": "whatsapp",
			"to": "+15551234567",
			"from": "+15559876543",
			"text": null,
			"status": "failed",
			"direction": "outbound",
			"segments": 1,
			"creditsUsed": 0,
			"whatsapp": {"kind": "template", "template": {"name": "order_shipped", "language": "en_US", "category": "utility"}, "messageId": null},
			"createdAt": "2026-10-17T00:00:00Z",
			"metadata": {},
			"fallback": {
				"channel": "sms",
				"status": "sent",
				"reason": "window_closed",
				"messageId": "msg_sms",
				"creditsUsed": 2,
				"contentDropped": ["quick_replies"]
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	yes := true
	msg, err := client.Messages.SendWhatsApp(context.Background(), &SendWhatsAppMessageRequest{
		To:            "+15551234567",
		From:          "+15559876543",
		Template:      &WhatsAppTemplateSendParams{Name: "order_shipped", Language: "en_US"},
		FallbackToSms: &yes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Status != MessageStatusFailed {
		t.Errorf("expected Status 'failed', got '%s'", msg.Status)
	}
	if msg.Fallback == nil {
		t.Fatal("expected Fallback to be set")
	}
	want := WhatsAppSmsFallback{
		Channel:        "sms",
		Status:         WhatsAppSmsFallbackStatusSent,
		Reason:         WhatsAppSmsFallbackReasonWindowClosed,
		MessageID:      "msg_sms",
		CreditsUsed:    2,
		ContentDropped: []string{"quick_replies"},
	}
	if !reflect.DeepEqual(*msg.Fallback, want) {
		t.Errorf("expected Fallback %+v, got %+v", want, *msg.Fallback)
	}
	if !msg.FellBack() {
		t.Error("expected FellBack to be true")
	}
}

func TestMessagesSendWhatsApp_FallbackNotSent(t *testing.T) {
	statuses := []WhatsAppSmsFallbackStatus{
		WhatsAppSmsFallbackStatusPending,
		WhatsAppSmsFallbackStatusFailed,
		WhatsAppSmsFallbackStatusUnavailable,
		WhatsAppSmsFallbackStatusSkipped,
		"something_new",
	}
	for _, status := range statuses {
		var msg WhatsAppMessage
		raw := `{"id":"msg_wa","status":"sent","fallback":{"channel":"sms","status":"` + string(status) + `","error":"Out of credits."}}`
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if msg.Fallback == nil || msg.Fallback.Status != status {
			t.Fatalf("expected Fallback.Status %q, got %+v", status, msg.Fallback)
		}
		if msg.Fallback.Error != "Out of credits." {
			t.Errorf("expected Fallback.Error, got %q", msg.Fallback.Error)
		}
		if msg.FellBack() {
			t.Errorf("expected FellBack to be false for status %q", status)
		}
	}
}

func TestMessagesSendWhatsApp_NoFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"msg_wa","channel":"whatsapp","message_format":"whatsapp","to":"+15551234567","from":"+15559876543","text":"Hi","status":"sent","segments":1,"creditsUsed":1,"whatsapp":{"kind":"text","messageId":"wamid.1"},"createdAt":"2026-10-17T00:00:00Z"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	msg, err := client.Messages.SendWhatsApp(context.Background(), &SendWhatsAppMessageRequest{
		To:   "+15551234567",
		From: "+15559876543",
		Text: "Hi",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Fallback != nil {
		t.Errorf("expected Fallback to be nil, got %+v", msg.Fallback)
	}
	if msg.FellBack() {
		t.Error("expected FellBack to be false")
	}
	var nilMsg *WhatsAppMessage
	if nilMsg.FellBack() {
		t.Error("expected FellBack on a nil message to be false")
	}
}

func TestMessagesGet_WhatsAppFallbackAndFallbackOf(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/messages/msg_wa":
			w.Write([]byte(`{"id":"msg_wa","to":"+15551234567","from":"+15559876543","text":"Hi","status":"failed","direction":"outbound","messageFormat":"whatsapp","fallback":{"channel":"sms","status":"sent","reason":"whatsapp_failed","messageId":"msg_sms","creditsUsed":2}}`))
		case "/messages/msg_sms":
			w.Write([]byte(`{"id":"msg_sms","to":"+15551234567","from":"+15559876543","text":"Hi","status":"delivered","direction":"outbound","messageFormat":"sms","fallbackOf":"msg_wa"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	wa, err := client.Messages.Get(ctx, "msg_wa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wa.Fallback == nil || wa.Fallback.Status != WhatsAppSmsFallbackStatusSent || wa.Fallback.Reason != WhatsAppSmsFallbackReasonWhatsAppFailed || wa.Fallback.MessageID != "msg_sms" {
		t.Fatalf("unexpected Fallback %+v", wa.Fallback)
	}
	if wa.FallbackOf != "" {
		t.Errorf("expected FallbackOf to be empty on the WhatsApp message, got %q", wa.FallbackOf)
	}

	sms, err := client.Messages.Get(ctx, wa.Fallback.MessageID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sms.FallbackOf != "msg_wa" {
		t.Errorf("expected FallbackOf 'msg_wa', got %q", sms.FallbackOf)
	}
	if sms.Fallback != nil {
		t.Errorf("expected Fallback to be nil on the SMS, got %+v", sms.Fallback)
	}
}

func TestMessagesSendWhatsApp_SmsFallbackUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"sms_fallback_unavailable","message":"SMS fallback reaches US and Canada numbers only.","suggestion":"Send without fallbackToSms, or make sure this number can send SMS to this recipient first."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	yes := true
	_, err := client.Messages.SendWhatsApp(context.Background(), &SendWhatsAppMessageRequest{
		To:            "+447700900123",
		From:          "+15559876543",
		Text:          "Hi",
		FallbackToSms: &yes,
	})
	var sendlyErr *SendlyError
	if !errors.As(err, &sendlyErr) {
		t.Fatalf("expected *SendlyError, got %T: %v", err, err)
	}
	if sendlyErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected StatusCode 403, got %d", sendlyErr.StatusCode)
	}
	if sendlyErr.Code != WhatsAppErrorCodeSmsFallbackUnavailable {
		t.Errorf("expected Code %q, got %q", WhatsAppErrorCodeSmsFallbackUnavailable, sendlyErr.Code)
	}
	if sendlyErr.Message != "SMS fallback reaches US and Canada numbers only." {
		t.Errorf("unexpected Message %q", sendlyErr.Message)
	}
	if _, ok := sendlyErr.Extra["suggestion"]; !ok {
		t.Error("expected suggestion in Extra")
	}
}

func TestMessagesSendWhatsApp_SmsFallbackNotSupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error":"sms_fallback_not_supported","reason":"marketing_template","message":"Marketing templates can't fall back to SMS: SMS marketing needs its own consent. Send without fallbackToSms."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	yes := true
	_, err := client.Messages.SendWhatsApp(context.Background(), &SendWhatsAppMessageRequest{
		To:            "+15551234567",
		From:          "+15559876543",
		Template:      &WhatsAppTemplateSendParams{Name: "spring_sale", Language: "en_US"},
		FallbackToSms: &yes,
	})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if validationErr.Code != WhatsAppErrorCodeSmsFallbackNotSupported {
		t.Errorf("expected Code %q, got %q", WhatsAppErrorCodeSmsFallbackNotSupported, validationErr.Code)
	}
	var reason string
	if err := json.Unmarshal(validationErr.Extra["reason"], &reason); err != nil || reason != "marketing_template" {
		t.Errorf("expected reason 'marketing_template' in Extra, got %q (%v)", reason, err)
	}
}

func TestMessagesSendWhatsApp_WindowClosedFallbackFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error":"whatsapp_window_closed","message":"The 24-hour window is closed.","suggestion":"Send an approved template.","id":"msg_wa","fallback":{"channel":"sms","status":"failed","reason":"window_closed","error":"Insufficient credits for the SMS."}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	yes := true
	_, err := client.Messages.SendWhatsApp(context.Background(), &SendWhatsAppMessageRequest{
		To:            "+15551234567",
		From:          "+15559876543",
		Text:          "Hi",
		FallbackToSms: &yes,
	})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if validationErr.Code != "whatsapp_window_closed" {
		t.Errorf("expected Code 'whatsapp_window_closed', got %q", validationErr.Code)
	}
	var fallback WhatsAppSmsFallback
	if err := json.Unmarshal(validationErr.Extra["fallback"], &fallback); err != nil {
		t.Fatalf("expected fallback in Extra: %v", err)
	}
	if fallback.Status != WhatsAppSmsFallbackStatusFailed || fallback.Error != "Insufficient credits for the SMS." {
		t.Errorf("unexpected fallback %+v", fallback)
	}
	var id string
	if err := json.Unmarshal(validationErr.Extra["id"], &id); err != nil || id != "msg_wa" {
		t.Errorf("expected id 'msg_wa' in Extra, got %q (%v)", id, err)
	}
}
