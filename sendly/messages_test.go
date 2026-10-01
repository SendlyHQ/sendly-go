package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestMessagesSend_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("expected path '/messages', got '%s'", r.URL.Path)
		}

		var req SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.To != "+1234567890" {
			t.Errorf("expected To to be '+1234567890', got '%s'", req.To)
		}
		if req.Text != "Test message" {
			t.Errorf("expected Text to be 'Test message', got '%s'", req.Text)
		}

		resp := Message{
			ID:          "msg_123",
			To:          req.To,
			Text:        req.Text,
			Status:      MessageStatusQueued,
			Segments:    1,
			CreditsUsed: 1,
			CreatedAt:   "2024-01-01T00:00:00Z",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := client.Messages.Send(ctx, &SendMessageRequest{
		To:   "+1234567890",
		Text: "Test message",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.ID != "msg_123" {
		t.Errorf("expected ID to be 'msg_123', got '%s'", msg.ID)
	}
	if msg.Status != MessageStatusQueued {
		t.Errorf("expected Status to be 'queued', got '%s'", msg.Status)
	}
}

func TestMessagesSend_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	tests := []struct {
		name        string
		req         *SendMessageRequest
		expectedErr string
	}{
		{
			name:        "nil request",
			req:         nil,
			expectedErr: "request is required",
		},
		{
			name:        "empty to",
			req:         &SendMessageRequest{To: "", Text: "Test"},
			expectedErr: "to is required",
		},
		{
			name:        "empty text",
			req:         &SendMessageRequest{To: "+1234567890", Text: ""},
			expectedErr: "either text or media_urls is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Messages.Send(ctx, tt.req)
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

func TestMessagesSend_InvalidPhoneFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{
			Code:    "INVALID_PHONE_NUMBER",
			Message: "Phone number must be in E.164 format",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.Send(ctx, &SendMessageRequest{
		To:   "invalid-phone",
		Text: "Test message",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestMessagesSend_TextTooLong(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{
			Code:    "TEXT_TOO_LONG",
			Message: "Message text exceeds maximum length",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	longText := strings.Repeat("a", 2000)
	_, err := client.Messages.Send(ctx, &SendMessageRequest{
		To:   "+1234567890",
		Text: longText,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestMessagesSend_AuthenticationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(APIError{
			Code:    "UNAUTHORIZED",
			Message: "Invalid API key",
		})
	}))
	defer server.Close()

	client := NewClient("invalid-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.Send(ctx, &SendMessageRequest{
		To:   "+1234567890",
		Text: "Test message",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestMessagesSend_InsufficientCredits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(APIError{
			Code:    "INSUFFICIENT_CREDITS",
			Message: "Not enough credits to send message",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.Send(ctx, &SendMessageRequest{
		To:   "+1234567890",
		Text: "Test message",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsInsufficientCreditsError(err) {
		t.Errorf("expected InsufficientCreditsError, got %T", err)
	}
}

func TestMessagesSend_RateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(APIError{
			Code:    "RATE_LIMIT_EXCEEDED",
			Message: "Too many requests",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))
	ctx := context.Background()

	_, err := client.Messages.Send(ctx, &SendMessageRequest{
		To:   "+1234567890",
		Text: "Test message",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T", err)
	}

	rateLimitErr := err.(*RateLimitError)
	if rateLimitErr.RetryAfter != 1 {
		t.Errorf("expected RetryAfter to be 1, got %d", rateLimitErr.RetryAfter)
	}
}

func TestMessagesSend_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIError{
			Code:    "INTERNAL_ERROR",
			Message: "Internal server error",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))
	ctx := context.Background()

	_, err := client.Messages.Send(ctx, &SendMessageRequest{
		To:   "+1234567890",
		Text: "Test message",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	sendlyErr, ok := err.(*SendlyError)
	if !ok {
		t.Errorf("expected SendlyError, got %T", err)
	}
	if sendlyErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status code 500, got %d", sendlyErr.StatusCode)
	}
}

func TestMessagesList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("expected path '/messages', got '%s'", r.URL.Path)
		}

		// Check query parameters
		query := r.URL.Query()
		if limit := query.Get("limit"); limit != "10" {
			t.Errorf("expected limit to be '10', got '%s'", limit)
		}
		if offset := query.Get("offset"); offset != "20" {
			t.Errorf("expected offset to be '20', got '%s'", offset)
		}
		if status := query.Get("status"); status != "delivered" {
			t.Errorf("expected status to be 'delivered', got '%s'", status)
		}

		resp := ListMessagesResponse{
			Data: []Message{
				{
					ID:     "msg_1",
					To:     "+1234567890",
					Text:   "Message 1",
					Status: MessageStatusDelivered,
				},
				{
					ID:     "msg_2",
					To:     "+1987654321",
					Text:   "Message 2",
					Status: MessageStatusDelivered,
				},
			},
			Count: 2,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	result, err := client.Messages.List(ctx, &ListMessagesRequest{
		Limit:  10,
		Offset: 20,
		Status: MessageStatusDelivered,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("expected count to be 2, got %d", result.Count)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result.Data))
	}
}

func TestMessagesList_NoParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query parameters, got '%s'", r.URL.RawQuery)
		}

		resp := ListMessagesResponse{
			Data:  []Message{},
			Count: 0,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.List(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMessagesList_AuthenticationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(APIError{
			Code:    "UNAUTHORIZED",
			Message: "Invalid API key",
		})
	}))
	defer server.Close()

	client := NewClient("invalid-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.List(ctx, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestMessagesGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/messages/msg_123" {
			t.Errorf("expected path '/messages/msg_123', got '%s'", r.URL.Path)
		}

		resp := Message{
			ID:          "msg_123",
			To:          "+1234567890",
			Text:        "Test message",
			Status:      MessageStatusDelivered,
			Segments:    1,
			CreditsUsed: 1,
			CreatedAt:   "2024-01-01T00:00:00Z",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := client.Messages.Get(ctx, "msg_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.ID != "msg_123" {
		t.Errorf("expected ID to be 'msg_123', got '%s'", msg.ID)
	}
	if msg.Status != MessageStatusDelivered {
		t.Errorf("expected Status to be 'delivered', got '%s'", msg.Status)
	}
}

func TestMessagesGet_EmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with empty ID")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.Get(ctx, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestMessagesGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Code:    "MESSAGE_NOT_FOUND",
			Message: "Message not found",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.Get(ctx, "msg_nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestMessagesGet_AuthenticationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(APIError{
			Code:    "UNAUTHORIZED",
			Message: "Invalid API key",
		})
	}))
	defer server.Close()

	client := NewClient("invalid-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.Get(ctx, "msg_123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestMessagesGet_URLEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The ID with special characters should be URL encoded in the request
		// but r.URL.Path is automatically decoded by the HTTP server
		// Check RawPath or EscapedPath for the encoded version
		expectedRawPath := "/messages/msg%2F123%2Ftest"
		if r.URL.EscapedPath() != expectedRawPath {
			t.Errorf("expected escaped path '%s', got '%s'", expectedRawPath, r.URL.EscapedPath())
		}

		resp := Message{
			ID:     "msg/123/test",
			To:     "+1234567890",
			Text:   "Test message",
			Status: MessageStatusDelivered,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.Get(ctx, "msg/123/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type legacyMessagesSender interface {
	Send(context.Context, *SendMessageRequest) (*Message, error)
	SendWhatsApp(context.Context, *SendWhatsAppMessageRequest) (*WhatsAppMessage, error)
	SendRcs(context.Context, *SendRcsMessageRequest) (*RcsMessage, error)
	SendGroup(context.Context, *SendGroupMessageRequest) (*GroupMessageResponse, error)
	Schedule(context.Context, *ScheduleMessageRequest) (*ScheduledMessage, error)
	SendBatch(context.Context, *SendBatchRequest) (*BatchMessageResponse, error)
}

var _ legacyMessagesSender = (*MessagesService)(nil)

func TestMessagesService_LegacyMethodValues(t *testing.T) {
	client := NewClient("test-api-key")

	send := client.Messages.Send
	sendWhatsApp := client.Messages.SendWhatsApp
	sendRcs := client.Messages.SendRcs
	sendGroup := client.Messages.SendGroup
	schedule := client.Messages.Schedule
	sendBatch := client.Messages.SendBatch

	var legacy legacyMessagesSender = client.Messages
	if legacy == nil {
		t.Fatal("expected MessagesService to satisfy the legacy send surface")
	}
	if send == nil || sendWhatsApp == nil || sendRcs == nil || sendGroup == nil || schedule == nil || sendBatch == nil {
		t.Fatal("expected legacy method values to be non-nil")
	}
}

func TestMessagesSendGroup_LiveRecipients(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/messages/group" {
			t.Errorf("expected path '/messages/group', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"msg_1","status":"sent","to":[{"phoneNumber":"+15551230001","status":"queued"},{"phoneNumber":"+15551230002","status":"queued"}],"group_message_id":"grp_1"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.Messages.SendGroup(context.Background(), &SendGroupMessageRequest{
		To:   []string{"+15551230001", "+15551230002"},
		Text: "Hello, group",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 1 {
		t.Errorf("expected 1 request, got %d", requests)
	}
	if resp.ID != "msg_1" || resp.Status != MessageStatusSent || resp.GroupMessageID != "grp_1" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if len(resp.To) != 2 || resp.To[0] != "+15551230001" || resp.To[1] != "+15551230002" {
		t.Errorf("expected To to list the recipients' numbers, got %v", resp.To)
	}
	if len(resp.Recipients) != 2 || resp.Recipients[0].PhoneNumber != "+15551230001" || resp.Recipients[0].Status != "queued" {
		t.Errorf("expected Recipients to carry each recipient's status, got %+v", resp.Recipients)
	}
}

func TestGroupMessageResponse_RoundTrip(t *testing.T) {
	original := GroupMessageResponse{
		ID:             "msg_1",
		Status:         MessageStatusSent,
		To:             []string{"+15551230001", "+15551230002"},
		GroupMessageID: "grp_1",
		Recipients: []GroupRecipient{
			{PhoneNumber: "+15551230001", Status: "queued"},
			{PhoneNumber: "+15551230002", Status: "queued"},
		},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded GroupMessageResponse
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip changed the value:\n before %+v\n after  %+v", original, decoded)
	}
}

func TestMessagesSendGroup_SimulatedRecipients(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"msg_2","status":"delivered","to":["+15551230001","+15551230002"],"simulated":true,"message":"Group message simulated (test key or verification pending)."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.Messages.SendGroup(context.Background(), &SendGroupMessageRequest{
		To:   []string{"+15551230001", "+15551230002"},
		Text: "Hello, group",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Simulated || len(resp.To) != 2 || resp.To[1] != "+15551230002" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGroupMessageResponse_WithoutRecipients(t *testing.T) {
	bodies := map[string]string{
		"null to":    `{"id":"msg_3","status":"delivered","to":null,"simulated":true}`,
		"missing to": `{"id":"msg_3","status":"delivered","simulated":true}`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			var resp GroupMessageResponse
			if err := json.Unmarshal([]byte(body), &resp); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.ID != "msg_3" || !resp.Simulated || resp.To != nil || resp.Recipients != nil {
				t.Errorf("unexpected response: %+v", resp)
			}
		})
	}
}

func TestGroupMessageResponse_RejectsWhatDoesNotDecode(t *testing.T) {
	bodies := map[string]string{
		"to that is neither numbers nor recipients": `{"id":"msg_4","status":"sent","to":"+15551230001"}`,
		"id that is not a string":                   `{"id":5,"status":"sent","to":["+15551230001"]}`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			var resp GroupMessageResponse
			if err := json.Unmarshal([]byte(body), &resp); err == nil {
				t.Errorf("expected an error, got %+v", resp)
			}
		})
	}
}

func TestMessagesList_ReadsPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[{"id":"msg_1","to":"+15551230001","from":"+18885550100","text":"Hi","status":"delivered","direction":"outbound","error":null,"errorCode":null,"retryCount":0,"segments":1,"creditsUsed":2,"isSandbox":false,"createdAt":"2026-09-25T10:00:00.000Z","deliveredAt":"2026-09-25T10:00:03.000Z","message_format":"sms","messageFormat":"sms"},{"id":"msg_2","to":"+15551230002","from":"+18885550100","text":"Hi","status":"received","direction":"inbound","error":null,"errorCode":null,"retryCount":0,"segments":1,"creditsUsed":0,"isSandbox":false,"createdAt":"2026-09-25T10:01:00.000Z","deliveredAt":null,"message_format":"sms","messageFormat":"sms"}],"pagination":{"total":57,"limit":2,"offset":0,"page":1,"totalPages":29,"hasMore":true},"count":2}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.Messages.List(context.Background(), &ListMessagesRequest{Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 || resp.Count != 2 {
		t.Errorf("expected 2 messages and Count 2, got %d and %d", len(resp.Data), resp.Count)
	}
	if resp.Pagination == nil {
		t.Fatalf("expected Pagination")
	}
	if resp.Pagination.Total != 57 || !resp.Pagination.HasMore || resp.Pagination.TotalPages != 29 || resp.Pagination.Page != 1 || resp.Pagination.Limit != 2 {
		t.Errorf("unexpected Pagination: %+v", resp.Pagination)
	}
}
