package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMessagesSendRcs_Text(t *testing.T) {
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
		if body["channel"] != "rcs" {
			t.Errorf("expected channel to be 'rcs', got '%v'", body["channel"])
		}
		if body["to"] != "+15551234567" {
			t.Errorf("expected to to be '+15551234567', got '%v'", body["to"])
		}
		if body["text"] != "Your order has shipped!" {
			t.Errorf("expected text to be 'Your order has shipped!', got '%v'", body["text"])
		}
		if _, ok := body["card"]; ok {
			t.Error("expected card to be omitted")
		}
		if _, ok := body["suggestions"]; ok {
			t.Error("expected suggestions to be omitted")
		}
		if _, ok := body["fallbackToSms"]; ok {
			t.Error("expected fallbackToSms to be omitted")
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"id": "msg_201",
			"channel": "rcs",
			"message_format": "rcs",
			"to": "+15551234567",
			"from": "Acme Inc",
			"text": "Your order has shipped!",
			"status": "sent",
			"segments": 1,
			"creditsUsed": 2,
			"rcs": {"kind": "text", "agentId": "rcsa_1", "agentName": "Acme Inc"},
			"createdAt": "2026-08-01T00:00:00Z",
			"metadata": {}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := client.Messages.SendRcs(ctx, &SendRcsMessageRequest{
		To:   "+15551234567",
		Text: "Your order has shipped!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.ID != "msg_201" {
		t.Errorf("expected ID to be 'msg_201', got '%s'", msg.ID)
	}
	if msg.Channel != "rcs" {
		t.Errorf("expected Channel to be 'rcs', got '%s'", msg.Channel)
	}
	if msg.FellBackTo != "" {
		t.Errorf("expected FellBackTo to be empty, got '%s'", msg.FellBackTo)
	}
	if msg.MessageFormat != "rcs" {
		t.Errorf("expected MessageFormat to be 'rcs', got '%s'", msg.MessageFormat)
	}
	if msg.Text == nil || *msg.Text != "Your order has shipped!" {
		t.Errorf("expected Text to be 'Your order has shipped!', got %v", msg.Text)
	}
	if msg.RCS.Kind != "text" {
		t.Errorf("expected Kind to be 'text', got '%s'", msg.RCS.Kind)
	}
	if msg.RCS.AgentID != "rcsa_1" {
		t.Errorf("expected AgentID to be 'rcsa_1', got '%s'", msg.RCS.AgentID)
	}
	if msg.RCS.AgentName != "Acme Inc" {
		t.Errorf("expected AgentName to be 'Acme Inc', got '%s'", msg.RCS.AgentName)
	}
	if msg.RCS.SuggestionsDropped {
		t.Error("expected SuggestionsDropped to be false")
	}
}

func TestMessagesSendRcs_TextWithSuggestions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		suggestions, ok := body["suggestions"].([]interface{})
		if !ok || len(suggestions) != 2 {
			t.Fatalf("expected 2 suggestions, got %v", body["suggestions"])
		}
		reply, ok := suggestions[0].(map[string]interface{})["reply"].(map[string]interface{})
		if !ok || reply["text"] != "Track order" || reply["postbackData"] != "track_4821" {
			t.Errorf("expected reply suggestion to round-trip, got %v", suggestions[0])
		}
		if _, hasAction := suggestions[0].(map[string]interface{})["action"]; hasAction {
			t.Error("expected reply suggestion to omit action")
		}
		action, ok := suggestions[1].(map[string]interface{})["action"].(map[string]interface{})
		if !ok || action["text"] != "View site" || action["postbackData"] != "site" || action["url"] != "https://example.com/orders" {
			t.Errorf("expected action suggestion to round-trip, got %v", suggestions[1])
		}
		if body["agentId"] != "rcsa_1" {
			t.Errorf("expected agentId to be 'rcsa_1', got '%v'", body["agentId"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"id": "msg_202",
			"channel": "rcs",
			"message_format": "rcs",
			"to": "+15551234567",
			"from": "Acme Inc",
			"text": "Your order has shipped!",
			"status": "sent",
			"segments": 1,
			"creditsUsed": 2,
			"rcs": {"kind": "text", "agentId": "rcsa_1", "agentName": "Acme Inc"},
			"createdAt": "2026-08-01T00:00:00Z",
			"metadata": {}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := client.Messages.SendRcs(ctx, &SendRcsMessageRequest{
		To:      "+15551234567",
		AgentID: "rcsa_1",
		Text:    "Your order has shipped!",
		Suggestions: []RcsSuggestion{
			{Reply: &RcsSuggestedReply{Text: "Track order", PostbackData: "track_4821"}},
			{Action: &RcsSuggestedAction{Text: "View site", PostbackData: "site", URL: "https://example.com/orders"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.RCS.Kind != "text" {
		t.Errorf("expected Kind to be 'text', got '%s'", msg.RCS.Kind)
	}
}

func TestMessagesSendRcs_Card(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		card, ok := body["card"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected card object, got %v", body["card"])
		}
		if card["title"] != "Spring sale" {
			t.Errorf("expected card title to be 'Spring sale', got '%v'", card["title"])
		}
		if card["description"] != "20% off everything this weekend." {
			t.Errorf("expected card description to round-trip, got '%v'", card["description"])
		}
		if card["mediaUrl"] != "https://example.com/sale.jpg" {
			t.Errorf("expected card mediaUrl to round-trip, got '%v'", card["mediaUrl"])
		}
		if card["orientation"] != "horizontal" {
			t.Errorf("expected card orientation to be 'horizontal', got '%v'", card["orientation"])
		}
		cardSuggestions, ok := card["suggestions"].([]interface{})
		if !ok || len(cardSuggestions) != 1 {
			t.Fatalf("expected 1 card suggestion, got %v", card["suggestions"])
		}
		if _, ok := body["text"]; ok {
			t.Error("expected text to be omitted")
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"id": "msg_203",
			"channel": "rcs",
			"message_format": "rcs",
			"to": "+15551234567",
			"from": "Acme Inc",
			"text": null,
			"status": "sent",
			"segments": 1,
			"creditsUsed": 2,
			"rcs": {"kind": "card", "agentId": "rcsa_1", "agentName": "Acme Inc"},
			"createdAt": "2026-08-01T00:00:00Z",
			"metadata": {}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := client.Messages.SendRcs(ctx, &SendRcsMessageRequest{
		To: "+15551234567",
		Card: &RcsCard{
			Title:       "Spring sale",
			Description: "20% off everything this weekend.",
			MediaURL:    "https://example.com/sale.jpg",
			Orientation: "horizontal",
			Suggestions: []RcsSuggestion{
				{Action: &RcsSuggestedAction{Text: "Shop now", PostbackData: "shop", URL: "https://example.com/sale"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.RCS.Kind != "card" {
		t.Errorf("expected Kind to be 'card', got '%s'", msg.RCS.Kind)
	}
	if msg.Text != nil {
		t.Errorf("expected Text to be nil, got '%s'", *msg.Text)
	}
}

func TestMessagesSendRcs_SmsFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["channel"] != "rcs" {
			t.Errorf("expected channel to be 'rcs', got '%v'", body["channel"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"id": "msg_204",
			"channel": "sms",
			"fellBackTo": "sms",
			"message_format": "sms",
			"to": "+15551234567",
			"from": "+18885550101",
			"text": "Your order has shipped!",
			"status": "sent",
			"segments": 1,
			"creditsUsed": 2,
			"rcs": {"requestedChannel": "rcs", "agentId": "rcsa_1", "suggestionsDropped": true},
			"createdAt": "2026-08-01T00:00:00Z",
			"metadata": {}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := client.Messages.SendRcs(ctx, &SendRcsMessageRequest{
		To:   "+15551234567",
		Text: "Your order has shipped!",
		Suggestions: []RcsSuggestion{
			{Reply: &RcsSuggestedReply{Text: "Track order", PostbackData: "track_4821"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Channel != "sms" {
		t.Errorf("expected Channel to be 'sms', got '%s'", msg.Channel)
	}
	if msg.FellBackTo != "sms" {
		t.Errorf("expected FellBackTo to be 'sms', got '%s'", msg.FellBackTo)
	}
	if msg.MessageFormat != "sms" {
		t.Errorf("expected MessageFormat to be 'sms', got '%s'", msg.MessageFormat)
	}
	if msg.RCS.Kind != "" {
		t.Errorf("expected Kind to be empty on a fallback, got '%s'", msg.RCS.Kind)
	}
	if msg.RCS.RequestedChannel != "rcs" {
		t.Errorf("expected RequestedChannel to be 'rcs', got '%s'", msg.RCS.RequestedChannel)
	}
	if msg.RCS.AgentID != "rcsa_1" {
		t.Errorf("expected AgentID to be 'rcsa_1', got '%s'", msg.RCS.AgentID)
	}
	if !msg.RCS.SuggestionsDropped {
		t.Error("expected SuggestionsDropped to be true")
	}
}

func TestMessagesSendRcs_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	tests := []struct {
		name        string
		req         *SendRcsMessageRequest
		expectedErr string
	}{
		{
			name:        "nil request",
			req:         nil,
			expectedErr: "request is required",
		},
		{
			name:        "empty to",
			req:         &SendRcsMessageRequest{Text: "Hi"},
			expectedErr: "to is required",
		},
		{
			name:        "no content",
			req:         &SendRcsMessageRequest{To: "+15551234567"},
			expectedErr: "exactly one of text or card is required",
		},
		{
			name: "both text and card",
			req: &SendRcsMessageRequest{
				To:   "+15551234567",
				Text: "Hi",
				Card: &RcsCard{Title: "Hi", Description: "There"},
			},
			expectedErr: "exactly one of text or card is required",
		},
		{
			name: "suggestions alongside a card",
			req: &SendRcsMessageRequest{
				To:   "+15551234567",
				Card: &RcsCard{Title: "Hi", Description: "There"},
				Suggestions: []RcsSuggestion{
					{Reply: &RcsSuggestedReply{Text: "Track it", PostbackData: "track_4821"}},
				},
			},
			expectedErr: "suggestions ride on text messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Messages.SendRcs(ctx, tt.req)
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

func TestMessagesSendRcs_NotSupportedForRecipient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(APIError{
			Code:    "rcs_not_supported_for_recipient",
			Message: "This recipient's device or network doesn't support RCS, and a rich card has no SMS form.",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.SendRcs(ctx, &SendRcsMessageRequest{
		To:   "+15551234567",
		Card: &RcsCard{Title: "Spring sale", Description: "20% off everything this weekend."},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	validationErr := err.(*ValidationError)
	if validationErr.Code != "rcs_not_supported_for_recipient" {
		t.Errorf("expected Code to be 'rcs_not_supported_for_recipient', got '%s'", validationErr.Code)
	}
}

func TestMessagesSendRcs_ChannelForced(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["channel"] != "rcs" {
			t.Errorf("expected channel to be 'rcs', got '%v'", body["channel"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"id": "msg_205",
			"channel": "rcs",
			"message_format": "rcs",
			"to": "+15551234567",
			"from": "Acme Inc",
			"text": "Hi",
			"status": "sent",
			"segments": 1,
			"creditsUsed": 2,
			"rcs": {"kind": "text", "agentId": "rcsa_1", "agentName": "Acme Inc"},
			"createdAt": "2026-08-01T00:00:00Z",
			"metadata": {}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	req := &SendRcsMessageRequest{
		Channel: "sms",
		To:      "+15551234567",
		Text:    "Hi",
	}
	_, err := client.Messages.SendRcs(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Channel != "sms" {
		t.Errorf("expected caller's request to be unmodified, got '%s'", req.Channel)
	}
}
