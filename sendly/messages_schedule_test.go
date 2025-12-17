package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMessagesSchedule_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/messages/schedule" {
			t.Errorf("expected path '/messages/schedule', got '%s'", r.URL.Path)
		}

		var req ScheduleMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.To != "+1234567890" {
			t.Errorf("expected To to be '+1234567890', got '%s'", req.To)
		}
		if req.Text != "Scheduled message" {
			t.Errorf("expected Text to be 'Scheduled message', got '%s'", req.Text)
		}
		if req.ScheduledAt != "2024-12-31T23:59:59Z" {
			t.Errorf("expected ScheduledAt to be '2024-12-31T23:59:59Z', got '%s'", req.ScheduledAt)
		}

		resp := ScheduledMessage{
			ID:              "sched_123",
			To:              req.To,
			Text:            req.Text,
			ScheduledAt:     req.ScheduledAt,
			Status:          ScheduledMessageStatusScheduled,
			CreditsReserved: 1,
			CreatedAt:       "2024-01-01T00:00:00Z",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := client.Messages.Schedule(ctx, &ScheduleMessageRequest{
		To:          "+1234567890",
		Text:        "Scheduled message",
		ScheduledAt: "2024-12-31T23:59:59Z",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.ID != "sched_123" {
		t.Errorf("expected ID to be 'sched_123', got '%s'", msg.ID)
	}
	if msg.Status != ScheduledMessageStatusScheduled {
		t.Errorf("expected Status to be 'scheduled', got '%s'", msg.Status)
	}
	if msg.CreditsReserved != 1 {
		t.Errorf("expected CreditsReserved to be 1, got %d", msg.CreditsReserved)
	}
}

func TestMessagesSchedule_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	tests := []struct {
		name        string
		req         *ScheduleMessageRequest
		expectedErr string
	}{
		{
			name:        "nil request",
			req:         nil,
			expectedErr: "request is required",
		},
		{
			name: "empty to",
			req: &ScheduleMessageRequest{
				To:          "",
				Text:        "Test",
				ScheduledAt: "2024-12-31T23:59:59Z",
			},
			expectedErr: "to is required",
		},
		{
			name: "empty text",
			req: &ScheduleMessageRequest{
				To:          "+1234567890",
				Text:        "",
				ScheduledAt: "2024-12-31T23:59:59Z",
			},
			expectedErr: "text is required",
		},
		{
			name: "empty scheduledAt",
			req: &ScheduleMessageRequest{
				To:          "+1234567890",
				Text:        "Test",
				ScheduledAt: "",
			},
			expectedErr: "scheduledAt is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Messages.Schedule(ctx, tt.req)
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

func TestMessagesSchedule_InvalidPhoneFormat(t *testing.T) {
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

	_, err := client.Messages.Schedule(ctx, &ScheduleMessageRequest{
		To:          "invalid-phone",
		Text:        "Test message",
		ScheduledAt: "2024-12-31T23:59:59Z",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestMessagesSchedule_AuthenticationError(t *testing.T) {
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

	_, err := client.Messages.Schedule(ctx, &ScheduleMessageRequest{
		To:          "+1234567890",
		Text:        "Test message",
		ScheduledAt: "2024-12-31T23:59:59Z",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestMessagesSchedule_InsufficientCredits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(APIError{
			Code:    "INSUFFICIENT_CREDITS",
			Message: "Not enough credits to schedule message",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.Schedule(ctx, &ScheduleMessageRequest{
		To:          "+1234567890",
		Text:        "Test message",
		ScheduledAt: "2024-12-31T23:59:59Z",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsInsufficientCreditsError(err) {
		t.Errorf("expected InsufficientCreditsError, got %T", err)
	}
}

func TestMessagesListScheduled_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/messages/scheduled" {
			t.Errorf("expected path '/messages/scheduled', got '%s'", r.URL.Path)
		}

		query := r.URL.Query()
		if limit := query.Get("limit"); limit != "10" {
			t.Errorf("expected limit to be '10', got '%s'", limit)
		}
		if offset := query.Get("offset"); offset != "5" {
			t.Errorf("expected offset to be '5', got '%s'", offset)
		}
		if status := query.Get("status"); status != "scheduled" {
			t.Errorf("expected status to be 'scheduled', got '%s'", status)
		}

		resp := ListScheduledMessagesResponse{
			Data: []ScheduledMessage{
				{
					ID:          "sched_1",
					To:          "+1234567890",
					Text:        "Message 1",
					ScheduledAt: "2024-12-31T23:59:59Z",
					Status:      ScheduledMessageStatusScheduled,
				},
				{
					ID:          "sched_2",
					To:          "+1987654321",
					Text:        "Message 2",
					ScheduledAt: "2024-12-31T23:59:59Z",
					Status:      ScheduledMessageStatusScheduled,
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

	result, err := client.Messages.ListScheduled(ctx, &ListScheduledMessagesRequest{
		Limit:  10,
		Offset: 5,
		Status: ScheduledMessageStatusScheduled,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("expected count to be 2, got %d", result.Count)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 scheduled messages, got %d", len(result.Data))
	}
}

func TestMessagesListScheduled_NoParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query parameters, got '%s'", r.URL.RawQuery)
		}

		resp := ListScheduledMessagesResponse{
			Data:  []ScheduledMessage{},
			Count: 0,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.ListScheduled(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMessagesListScheduled_AuthenticationError(t *testing.T) {
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

	_, err := client.Messages.ListScheduled(ctx, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestMessagesGetScheduled_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/messages/scheduled/sched_123" {
			t.Errorf("expected path '/messages/scheduled/sched_123', got '%s'", r.URL.Path)
		}

		resp := ScheduledMessage{
			ID:              "sched_123",
			To:              "+1234567890",
			Text:            "Scheduled message",
			ScheduledAt:     "2024-12-31T23:59:59Z",
			Status:          ScheduledMessageStatusScheduled,
			CreditsReserved: 1,
			CreatedAt:       "2024-01-01T00:00:00Z",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := client.Messages.GetScheduled(ctx, "sched_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.ID != "sched_123" {
		t.Errorf("expected ID to be 'sched_123', got '%s'", msg.ID)
	}
	if msg.Status != ScheduledMessageStatusScheduled {
		t.Errorf("expected Status to be 'scheduled', got '%s'", msg.Status)
	}
}

func TestMessagesGetScheduled_EmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with empty ID")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.GetScheduled(ctx, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestMessagesGetScheduled_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Code:    "SCHEDULED_MESSAGE_NOT_FOUND",
			Message: "Scheduled message not found",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.GetScheduled(ctx, "sched_nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestMessagesGetScheduled_AuthenticationError(t *testing.T) {
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

	_, err := client.Messages.GetScheduled(ctx, "sched_123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestMessagesCancelScheduled_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/messages/scheduled/sched_123" {
			t.Errorf("expected path '/messages/scheduled/sched_123', got '%s'", r.URL.Path)
		}

		resp := CancelScheduledMessageResponse{
			ID:              "sched_123",
			Status:          ScheduledMessageStatusCancelled,
			CreditsRefunded: 1,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	result, err := client.Messages.CancelScheduled(ctx, "sched_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "sched_123" {
		t.Errorf("expected ID to be 'sched_123', got '%s'", result.ID)
	}
	if result.Status != ScheduledMessageStatusCancelled {
		t.Errorf("expected Status to be 'cancelled', got '%s'", result.Status)
	}
	if result.CreditsRefunded != 1 {
		t.Errorf("expected CreditsRefunded to be 1, got %d", result.CreditsRefunded)
	}
}

func TestMessagesCancelScheduled_EmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with empty ID")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.CancelScheduled(ctx, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestMessagesCancelScheduled_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Code:    "SCHEDULED_MESSAGE_NOT_FOUND",
			Message: "Scheduled message not found",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.CancelScheduled(ctx, "sched_nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestMessagesCancelScheduled_AuthenticationError(t *testing.T) {
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

	_, err := client.Messages.CancelScheduled(ctx, "sched_123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestMessagesCancelScheduled_RateLimitError(t *testing.T) {
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

	_, err := client.Messages.CancelScheduled(ctx, "sched_123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T", err)
	}
}

func TestMessagesCancelScheduled_ServerError(t *testing.T) {
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

	_, err := client.Messages.CancelScheduled(ctx, "sched_123")
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
