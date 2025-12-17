package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMessagesSendBatch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/messages/batch" {
			t.Errorf("expected path '/messages/batch', got '%s'", r.URL.Path)
		}

		var req SendBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if len(req.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(req.Messages))
		}

		resp := BatchMessageResponse{
			BatchID:     "batch_123",
			Status:      BatchStatusProcessing,
			Total:       2,
			Queued:      2,
			Sent:        0,
			Failed:      0,
			CreditsUsed: 0,
			CreatedAt:   "2024-01-01T00:00:00Z",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	result, err := client.Messages.SendBatch(ctx, &SendBatchRequest{
		Messages: []BatchMessageItem{
			{To: "+1234567890", Text: "Message 1"},
			{To: "+1987654321", Text: "Message 2"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.BatchID != "batch_123" {
		t.Errorf("expected BatchID to be 'batch_123', got '%s'", result.BatchID)
	}
	if result.Status != BatchStatusProcessing {
		t.Errorf("expected Status to be 'processing', got '%s'", result.Status)
	}
	if result.Total != 2 {
		t.Errorf("expected Total to be 2, got %d", result.Total)
	}
}

func TestMessagesSendBatch_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	tests := []struct {
		name        string
		req         *SendBatchRequest
		expectedErr string
	}{
		{
			name:        "nil request",
			req:         nil,
			expectedErr: "request is required",
		},
		{
			name:        "empty messages",
			req:         &SendBatchRequest{Messages: []BatchMessageItem{}},
			expectedErr: "messages are required",
		},
		{
			name: "message with empty to",
			req: &SendBatchRequest{
				Messages: []BatchMessageItem{
					{To: "", Text: "Test"},
				},
			},
			expectedErr: "to is required for message at index 0",
		},
		{
			name: "message with empty text",
			req: &SendBatchRequest{
				Messages: []BatchMessageItem{
					{To: "+1234567890", Text: ""},
				},
			},
			expectedErr: "text is required for message at index 0",
		},
		{
			name: "second message with validation error",
			req: &SendBatchRequest{
				Messages: []BatchMessageItem{
					{To: "+1234567890", Text: "Valid"},
					{To: "", Text: "Invalid"},
				},
			},
			expectedErr: "to is required for message at index 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Messages.SendBatch(ctx, tt.req)
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

func TestMessagesSendBatch_InvalidPhoneFormat(t *testing.T) {
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

	_, err := client.Messages.SendBatch(ctx, &SendBatchRequest{
		Messages: []BatchMessageItem{
			{To: "invalid-phone", Text: "Test message"},
		},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestMessagesSendBatch_TextTooLong(t *testing.T) {
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
	_, err := client.Messages.SendBatch(ctx, &SendBatchRequest{
		Messages: []BatchMessageItem{
			{To: "+1234567890", Text: longText},
		},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestMessagesSendBatch_AuthenticationError(t *testing.T) {
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

	_, err := client.Messages.SendBatch(ctx, &SendBatchRequest{
		Messages: []BatchMessageItem{
			{To: "+1234567890", Text: "Test message"},
		},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestMessagesSendBatch_InsufficientCredits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(APIError{
			Code:    "INSUFFICIENT_CREDITS",
			Message: "Not enough credits to send batch",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.SendBatch(ctx, &SendBatchRequest{
		Messages: []BatchMessageItem{
			{To: "+1234567890", Text: "Test message"},
		},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsInsufficientCreditsError(err) {
		t.Errorf("expected InsufficientCreditsError, got %T", err)
	}
}

func TestMessagesSendBatch_RateLimitError(t *testing.T) {
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

	_, err := client.Messages.SendBatch(ctx, &SendBatchRequest{
		Messages: []BatchMessageItem{
			{To: "+1234567890", Text: "Test message"},
		},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T", err)
	}
}

func TestMessagesSendBatch_ServerError(t *testing.T) {
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

	_, err := client.Messages.SendBatch(ctx, &SendBatchRequest{
		Messages: []BatchMessageItem{
			{To: "+1234567890", Text: "Test message"},
		},
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

func TestMessagesGetBatch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/messages/batch/batch_123" {
			t.Errorf("expected path '/messages/batch/batch_123', got '%s'", r.URL.Path)
		}

		msgID1 := "msg_1"
		msgID2 := "msg_2"
		errMsg := "Failed to deliver"

		resp := BatchMessageResponse{
			BatchID:     "batch_123",
			Status:      BatchStatusCompleted,
			Total:       2,
			Queued:      0,
			Sent:        1,
			Failed:      1,
			CreditsUsed: 1,
			CreatedAt:   "2024-01-01T00:00:00Z",
			Messages: []BatchMessageResult{
				{
					To:        "+1234567890",
					MessageID: &msgID1,
					Status:    "delivered",
					Error:     nil,
				},
				{
					To:        "+1987654321",
					MessageID: &msgID2,
					Status:    "failed",
					Error:     &errMsg,
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	result, err := client.Messages.GetBatch(ctx, "batch_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.BatchID != "batch_123" {
		t.Errorf("expected BatchID to be 'batch_123', got '%s'", result.BatchID)
	}
	if result.Status != BatchStatusCompleted {
		t.Errorf("expected Status to be 'completed', got '%s'", result.Status)
	}
	if result.Total != 2 {
		t.Errorf("expected Total to be 2, got %d", result.Total)
	}
	if result.Sent != 1 {
		t.Errorf("expected Sent to be 1, got %d", result.Sent)
	}
	if result.Failed != 1 {
		t.Errorf("expected Failed to be 1, got %d", result.Failed)
	}
	if len(result.Messages) != 2 {
		t.Errorf("expected 2 message results, got %d", len(result.Messages))
	}
}

func TestMessagesGetBatch_EmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with empty ID")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.GetBatch(ctx, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestMessagesGetBatch_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Code:    "BATCH_NOT_FOUND",
			Message: "Batch not found",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.GetBatch(ctx, "batch_nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestMessagesGetBatch_AuthenticationError(t *testing.T) {
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

	_, err := client.Messages.GetBatch(ctx, "batch_123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestMessagesListBatches_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/messages/batches" {
			t.Errorf("expected path '/messages/batches', got '%s'", r.URL.Path)
		}

		query := r.URL.Query()
		if limit := query.Get("limit"); limit != "10" {
			t.Errorf("expected limit to be '10', got '%s'", limit)
		}
		if offset := query.Get("offset"); offset != "5" {
			t.Errorf("expected offset to be '5', got '%s'", offset)
		}
		if status := query.Get("status"); status != "completed" {
			t.Errorf("expected status to be 'completed', got '%s'", status)
		}

		resp := ListBatchesResponse{
			Data: []BatchMessageResponse{
				{
					BatchID:     "batch_1",
					Status:      BatchStatusCompleted,
					Total:       5,
					Sent:        5,
					Failed:      0,
					CreditsUsed: 5,
					CreatedAt:   "2024-01-01T00:00:00Z",
				},
				{
					BatchID:     "batch_2",
					Status:      BatchStatusCompleted,
					Total:       3,
					Sent:        2,
					Failed:      1,
					CreditsUsed: 2,
					CreatedAt:   "2024-01-02T00:00:00Z",
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

	result, err := client.Messages.ListBatches(ctx, &ListBatchesRequest{
		Limit:  10,
		Offset: 5,
		Status: BatchStatusCompleted,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("expected count to be 2, got %d", result.Count)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 batches, got %d", len(result.Data))
	}
}

func TestMessagesListBatches_NoParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query parameters, got '%s'", r.URL.RawQuery)
		}

		resp := ListBatchesResponse{
			Data:  []BatchMessageResponse{},
			Count: 0,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.ListBatches(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMessagesListBatches_AuthenticationError(t *testing.T) {
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

	_, err := client.Messages.ListBatches(ctx, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestMessagesListBatches_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Code:    "NOT_FOUND",
			Message: "Resource not found",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.ListBatches(ctx, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestMessagesListBatches_RateLimitError(t *testing.T) {
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

	_, err := client.Messages.ListBatches(ctx, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T", err)
	}
}

func TestMessagesListBatches_ServerError(t *testing.T) {
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

	_, err := client.Messages.ListBatches(ctx, nil)
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
