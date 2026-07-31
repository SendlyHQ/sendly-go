package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMessagesSendWhatsApp_Text(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("expected path '/messages', got '%s'", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["channel"] != "whatsapp" {
			t.Errorf("expected channel to be 'whatsapp', got '%v'", body["channel"])
		}
		if body["to"] != "+15551234567" {
			t.Errorf("expected to to be '+15551234567', got '%v'", body["to"])
		}
		if body["from"] != "+15559876543" {
			t.Errorf("expected from to be '+15559876543', got '%v'", body["from"])
		}
		if body["text"] != "Your table is ready!" {
			t.Errorf("expected text to be 'Your table is ready!', got '%v'", body["text"])
		}
		if _, ok := body["template"]; ok {
			t.Error("expected template to be omitted")
		}
		if _, ok := body["mediaUrls"]; ok {
			t.Error("expected mediaUrls to be omitted")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_123",
			"channel": "whatsapp",
			"message_format": "whatsapp",
			"to": "+15551234567",
			"from": "+15559876543",
			"text": "Your table is ready!",
			"status": "queued",
			"segments": 1,
			"creditsUsed": 2,
			"whatsapp": {"kind": "text", "messageId": null},
			"createdAt": "2026-07-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := client.Messages.SendWhatsApp(ctx, &SendWhatsAppMessageRequest{
		To:   "+15551234567",
		From: "+15559876543",
		Text: "Your table is ready!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.ID != "msg_123" {
		t.Errorf("expected ID to be 'msg_123', got '%s'", msg.ID)
	}
	if msg.Channel != "whatsapp" {
		t.Errorf("expected Channel to be 'whatsapp', got '%s'", msg.Channel)
	}
	if msg.MessageFormat != "whatsapp" {
		t.Errorf("expected MessageFormat to be 'whatsapp', got '%s'", msg.MessageFormat)
	}
	if msg.Text == nil || *msg.Text != "Your table is ready!" {
		t.Errorf("expected Text to be 'Your table is ready!', got %v", msg.Text)
	}
	if msg.Segments != 1 {
		t.Errorf("expected Segments to be 1, got %d", msg.Segments)
	}
	if msg.WhatsApp.Kind != "text" {
		t.Errorf("expected Kind to be 'text', got '%s'", msg.WhatsApp.Kind)
	}
	if msg.WhatsApp.MessageID != nil {
		t.Errorf("expected MessageID to be nil, got '%s'", *msg.WhatsApp.MessageID)
	}
	if msg.WhatsApp.Template != nil {
		t.Errorf("expected Template to be nil, got %v", msg.WhatsApp.Template)
	}
}

func TestMessagesSendWhatsApp_MediaWithCaption(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["channel"] != "whatsapp" {
			t.Errorf("expected channel to be 'whatsapp', got '%v'", body["channel"])
		}
		media, ok := body["mediaUrls"].([]interface{})
		if !ok || len(media) != 1 || media[0] != "https://example.com/receipt.pdf" {
			t.Errorf("expected one media URL, got %v", body["mediaUrls"])
		}
		if body["text"] != "Your receipt" {
			t.Errorf("expected text (caption) to be 'Your receipt', got '%v'", body["text"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_124",
			"channel": "whatsapp",
			"message_format": "whatsapp",
			"to": "+15551234567",
			"from": "+15559876543",
			"text": null,
			"status": "queued",
			"segments": 1,
			"creditsUsed": 2,
			"whatsapp": {"kind": "media", "messageId": null},
			"createdAt": "2026-07-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := client.Messages.SendWhatsApp(ctx, &SendWhatsAppMessageRequest{
		To:        "+15551234567",
		From:      "+15559876543",
		Text:      "Your receipt",
		MediaUrls: []string{"https://example.com/receipt.pdf"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.WhatsApp.Kind != "media" {
		t.Errorf("expected Kind to be 'media', got '%s'", msg.WhatsApp.Kind)
	}
	if msg.Text != nil {
		t.Errorf("expected Text to be nil, got '%s'", *msg.Text)
	}
}

func TestMessagesSendWhatsApp_Template(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		template, ok := body["template"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected template object, got %v", body["template"])
		}
		if template["name"] != "order_shipped" {
			t.Errorf("expected template name to be 'order_shipped', got '%v'", template["name"])
		}
		if template["language"] != "en_US" {
			t.Errorf("expected template language to be 'en_US', got '%v'", template["language"])
		}
		variables, ok := template["variables"].(map[string]interface{})
		if !ok || variables["1"] != "Acme Inc" || variables["2"] != "#4821" {
			t.Errorf("expected template variables to round-trip, got %v", template["variables"])
		}
		buttons, ok := template["buttons"].([]interface{})
		if !ok || len(buttons) != 1 {
			t.Fatalf("expected 1 template button, got %v", template["buttons"])
		}
		button := buttons[0].(map[string]interface{})
		if button["index"] != float64(0) {
			t.Errorf("expected button index to be 0, got %v", button["index"])
		}
		if _, ok := body["text"]; ok {
			t.Error("expected text to be omitted")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_125",
			"channel": "whatsapp",
			"message_format": "whatsapp",
			"to": "+15551234567",
			"from": "+15559876543",
			"text": null,
			"status": "queued",
			"segments": 1,
			"creditsUsed": 3,
			"whatsapp": {
				"kind": "template",
				"template": {"name": "order_shipped", "language": "en_US", "category": "utility"},
				"messageId": null
			},
			"createdAt": "2026-07-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := client.Messages.SendWhatsApp(ctx, &SendWhatsAppMessageRequest{
		To:   "+15551234567",
		From: "+15559876543",
		Template: &WhatsAppTemplateSendParams{
			Name:      "order_shipped",
			Language:  "en_US",
			Variables: map[string]string{"1": "Acme Inc", "2": "#4821"},
			Buttons: []WhatsAppTemplateButtonVariables{
				{Index: 0, Variables: map[string]string{"1": "4821"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.WhatsApp.Kind != "template" {
		t.Errorf("expected Kind to be 'template', got '%s'", msg.WhatsApp.Kind)
	}
	if msg.WhatsApp.Template == nil {
		t.Fatal("expected Template to be set")
	}
	if msg.WhatsApp.Template.Name != "order_shipped" {
		t.Errorf("expected Template.Name to be 'order_shipped', got '%s'", msg.WhatsApp.Template.Name)
	}
	if msg.WhatsApp.Template.Category != "utility" {
		t.Errorf("expected Template.Category to be 'utility', got '%s'", msg.WhatsApp.Template.Category)
	}
	if msg.CreditsUsed != 3 {
		t.Errorf("expected CreditsUsed to be 3, got %d", msg.CreditsUsed)
	}
}

func TestMessagesSendWhatsApp_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	tests := []struct {
		name        string
		req         *SendWhatsAppMessageRequest
		expectedErr string
	}{
		{
			name:        "nil request",
			req:         nil,
			expectedErr: "request is required",
		},
		{
			name:        "empty to",
			req:         &SendWhatsAppMessageRequest{From: "+15559876543", Text: "Hi"},
			expectedErr: "to is required",
		},
		{
			name:        "empty from",
			req:         &SendWhatsAppMessageRequest{To: "+15551234567", Text: "Hi"},
			expectedErr: "from is required",
		},
		{
			name:        "no content",
			req:         &SendWhatsAppMessageRequest{To: "+15551234567", From: "+15559876543"},
			expectedErr: "either text, media_urls, or template is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Messages.SendWhatsApp(ctx, tt.req)
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

func TestMessagesSendWhatsApp_WindowClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(APIError{
			Code:    "whatsapp_window_closed",
			Message: "No open 24-hour window for this recipient; send an approved template instead",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.SendWhatsApp(ctx, &SendWhatsAppMessageRequest{
		To:   "+15551234567",
		From: "+15559876543",
		Text: "Hello!",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	validationErr := err.(*ValidationError)
	if validationErr.Code != "whatsapp_window_closed" {
		t.Errorf("expected Code to be 'whatsapp_window_closed', got '%s'", validationErr.Code)
	}
}

func TestMessagesSendWhatsApp_ChannelForced(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["channel"] != "whatsapp" {
			t.Errorf("expected channel to be 'whatsapp', got '%v'", body["channel"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_126",
			"channel": "whatsapp",
			"message_format": "whatsapp",
			"to": "+15551234567",
			"from": "+15559876543",
			"text": "Hi",
			"status": "queued",
			"segments": 1,
			"creditsUsed": 2,
			"whatsapp": {"kind": "text", "messageId": null},
			"createdAt": "2026-07-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	req := &SendWhatsAppMessageRequest{
		Channel: "sms",
		To:      "+15551234567",
		From:    "+15559876543",
		Text:    "Hi",
	}
	_, err := client.Messages.SendWhatsApp(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Channel != "sms" {
		t.Errorf("expected caller's request to be unmodified, got '%s'", req.Channel)
	}
}
