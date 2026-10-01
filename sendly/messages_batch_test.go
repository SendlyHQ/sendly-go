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
					To:     "+1234567890",
					ID:     msgID1,
					Status: "delivered",
					Error:  nil,
				},
				{
					To:     "+1987654321",
					ID:     msgID2,
					Status: "failed",
					Error:  &errMsg,
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

const wireBatchRead = `{"id":"batch_1","status":"completed","total":2,"queued":0,"sent":2,"delivered":2,"failed":0,"creditsReserved":4,"creditsUsed":4,"creditsRefunded":0,"createdAt":"2026-09-25T10:00:00.000Z","completedAt":"2026-09-25T10:00:05.000Z"}`

func TestMessagesListBatches_ReadsIDFromWire(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[` + wireBatchRead + `],"count":1}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	result, err := client.Messages.ListBatches(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(result.Data))
	}
	if result.Data[0].BatchID != "batch_1" {
		t.Errorf("expected BatchID 'batch_1', got '%s'", result.Data[0].BatchID)
	}
	if result.Data[0].ID != "batch_1" {
		t.Errorf("expected ID 'batch_1', got '%s'", result.Data[0].ID)
	}
	if result.Data[0].Delivered != 2 {
		t.Errorf("expected Delivered 2, got %d", result.Data[0].Delivered)
	}
	if result.Data[0].CreditsReserved != 4 {
		t.Errorf("expected CreditsReserved 4, got %d", result.Data[0].CreditsReserved)
	}
}

func TestMessagesGetBatch_ReadsIDFromWire(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages/batch/batch_1" {
			t.Errorf("expected path '/messages/batch/batch_1', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(wireBatchRead))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	result, err := client.Messages.GetBatch(context.Background(), "batch_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.BatchID != "batch_1" {
		t.Errorf("expected BatchID 'batch_1', got '%s'", result.BatchID)
	}
	if result.CompletedAt == nil || *result.CompletedAt != "2026-09-25T10:00:05.000Z" {
		t.Errorf("expected CompletedAt to decode, got %v", result.CompletedAt)
	}
}

func TestMessagesSendBatch_FillsIDFromBatchID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"batchId":"batch_2","status":"completed","total":3,"sent":2,"failed":0,"retrying":0,"optedOutSkipped":1,"invalidSkipped":0,"creditsUsed":4,"creditsRefunded":0,"messages":[{"index":0,"id":"msg_1","to":"+15551230001","status":"sent"}]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	result, err := client.Messages.SendBatch(context.Background(), &SendBatchRequest{
		Messages: []BatchMessageItem{
			{To: "+15551230001", Text: "Hi"},
			{To: "+15551230002", Text: "Hi"},
			{To: "+15551230003", Text: "Hi"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.BatchID != "batch_2" || result.ID != "batch_2" {
		t.Errorf("expected BatchID and ID 'batch_2', got '%s' and '%s'", result.BatchID, result.ID)
	}
	if result.OptedOutSkipped != 1 {
		t.Errorf("expected OptedOutSkipped 1, got %d", result.OptedOutSkipped)
	}
	if len(result.Messages) != 1 || result.Messages[0].ID != "msg_1" {
		t.Errorf("expected the per-message result to decode, got %+v", result.Messages)
	}
}

func TestBatchMessageResponse_RoundTrip(t *testing.T) {
	completed := "2026-09-25T10:00:05.000Z"
	original := BatchMessageResponse{
		BatchID:         "batch_1",
		ID:              "batch_1",
		Status:          BatchStatusCompleted,
		Total:           2,
		Sent:            2,
		Delivered:       2,
		CreditsReserved: 4,
		CreditsUsed:     4,
		CreatedAt:       "2026-09-25T10:00:00.000Z",
		CompletedAt:     &completed,
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded BatchMessageResponse
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip changed the value:\n before %+v\n after  %+v", original, decoded)
	}
}

const wirePreviewCountryBlocked = `{"total":3,"sendable":2,"blocked":1,"duplicates":0,"creditsNeeded":4,"creditBalance":100,"hasSufficientCredits":true,"pooled":false,"keyType":"live","keyScopes":["sms:send","sms:read"],"hasWriteScope":true,"messagingProfile":{"id":"mp_1","canSendDomestic":true,"canSendInternational":false,"verificationStatus":"approved","verificationType":"toll_free"},"byCountry":{"US":{"count":2,"credits":4,"tier":"domestic","allowed":true},"GB":{"count":1,"credits":0,"tier":"tier2","allowed":false,"blockedReason":"country_not_allowed"}},"blockedMessages":[{"index":2,"to":"+447700900000","reason":"country_not_allowed"}],"compliance":{"messageType":"marketing","optedOutBlocked":0,"shaftBlocked":0,"quietHoursBlocked":0,"quietHoursRescheduled":0,"shaftBlockedMessages":[],"quietHoursBlockedMessages":[]},"warnings":["Marketing messages are subject to quiet hours enforcement (8pm-8am recipient local time)"]}`

func previewBatchServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages/batch/preview" {
			t.Errorf("expected path '/messages/batch/preview', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
}

var previewRequest = &SendBatchRequest{Messages: []BatchMessageItem{
	{To: "+15551230001", Text: "Hi"},
	{To: "+15551230002", Text: "Hi"},
	{To: "+447700900000", Text: "Hi"},
}}

func TestMessagesPreviewBatch_ReadsTheWire(t *testing.T) {
	server := previewBatchServer(t, wirePreviewCountryBlocked)
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	preview, err := client.Messages.PreviewBatch(context.Background(), previewRequest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if preview.TotalMessages != 3 || preview.WillSend != 2 || preview.Blocked != 1 || preview.CreditsNeeded != 4 {
		t.Errorf("expected counts 3/2/1/4, got %d/%d/%d/%d", preview.TotalMessages, preview.WillSend, preview.Blocked, preview.CreditsNeeded)
	}
	if preview.CurrentBalance != 100 || !preview.HasEnoughCredits {
		t.Errorf("expected balance 100 with enough credits, got %d/%v", preview.CurrentBalance, preview.HasEnoughCredits)
	}
	if preview.BlockReasons["country_not_allowed"] != 1 {
		t.Errorf("expected BlockReasons[country_not_allowed] 1, got %v", preview.BlockReasons)
	}
	if preview.CanSend {
		t.Errorf("expected CanSend false: a live send rejects the whole batch when a message is blocked for anything but an opt-out")
	}
	if preview.Total != 3 || preview.Sendable != 2 || preview.CreditBalance != 100 || !preview.HasSufficientCredits {
		t.Errorf("unexpected wire fields: %+v", preview)
	}
	if len(preview.BlockedMessages) != 1 || preview.BlockedMessages[0].Index != 2 || preview.BlockedMessages[0].To != "+447700900000" {
		t.Errorf("unexpected BlockedMessages: %+v", preview.BlockedMessages)
	}
	if gb := preview.ByCountry["GB"]; gb.Allowed || gb.Count != 1 || gb.Tier != "tier2" || gb.BlockedReason != "country_not_allowed" {
		t.Errorf("unexpected ByCountry[GB]: %+v", gb)
	}
	if preview.MessagingProfile == nil || !preview.MessagingProfile.CanSendDomestic || preview.MessagingProfile.CanSendInternational {
		t.Errorf("unexpected MessagingProfile: %+v", preview.MessagingProfile)
	}
	if preview.Compliance == nil || preview.Compliance.MessageType != "marketing" {
		t.Errorf("unexpected Compliance: %+v", preview.Compliance)
	}
	if preview.KeyType != "live" || !preview.HasWriteScope || len(preview.Warnings) != 1 {
		t.Errorf("unexpected key fields or warnings: %+v", preview)
	}
}

func TestBatchPreviewResponse_CanSend(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "opted-out recipients are skipped, not refused",
			body: `{"total":3,"sendable":2,"blocked":1,"creditsNeeded":4,"creditBalance":100,"hasSufficientCredits":true,"keyType":"live","hasWriteScope":true,"blockedMessages":[{"index":0,"to":"+15551230000","reason":"Contact has opted out (texted STOP)"}],"compliance":{"messageType":"marketing","optedOutBlocked":1}}`,
			want: true,
		},
		{
			name: "a test key needs no balance",
			body: `{"total":1,"sendable":1,"blocked":0,"creditsNeeded":2,"creditBalance":0,"hasSufficientCredits":false,"keyType":"test","hasWriteScope":true,"compliance":{"optedOutBlocked":0}}`,
			want: true,
		},
		{
			name: "a live key needs the balance",
			body: `{"total":1,"sendable":1,"blocked":0,"creditsNeeded":2,"creditBalance":0,"hasSufficientCredits":false,"keyType":"live","hasWriteScope":true,"compliance":{"optedOutBlocked":0}}`,
			want: false,
		},
		{
			name: "a key without sms:send cannot send",
			body: `{"total":1,"sendable":1,"blocked":0,"creditsNeeded":2,"creditBalance":10,"hasSufficientCredits":true,"keyType":"live","hasWriteScope":false,"compliance":{"optedOutBlocked":0}}`,
			want: false,
		},
		{
			name: "nothing sendable",
			body: `{"total":1,"sendable":0,"blocked":1,"creditsNeeded":0,"creditBalance":10,"hasSufficientCredits":true,"keyType":"live","hasWriteScope":true,"blockedMessages":[{"index":0,"to":"+15551230000","reason":"Contact has opted out (texted STOP)"}],"compliance":{"optedOutBlocked":1}}`,
			want: false,
		},
		{
			name: "a batch over 10,000 messages is refused",
			body: `{"total":10001,"sendable":10001,"blocked":0,"duplicates":0,"creditsNeeded":20002,"creditBalance":50000,"hasSufficientCredits":true,"pooled":false,"keyType":"live","keyScopes":["sms:send","sms:read"],"hasWriteScope":true,"blockedMessages":[],"compliance":{"messageType":"transactional","optedOutBlocked":0,"shaftBlocked":0,"quietHoursBlocked":0,"quietHoursRescheduled":0,"shaftBlockedMessages":[],"quietHoursBlockedMessages":[]},"warnings":["Batch size exceeds 10,000 limit - sending it will be rejected, split it into batches of 10,000 or fewer"]}`,
			want: false,
		},
		{
			name: "a test key's blocks are judged as a live send judges them",
			body: `{"total":2,"sendable":0,"blocked":2,"duplicates":0,"creditsNeeded":0,"creditBalance":0,"hasSufficientCredits":true,"pooled":false,"keyType":"test","keyScopes":["sms:send","sms:read"],"hasWriteScope":true,"messagingProfile":{"id":null,"canSendDomestic":false,"canSendInternational":false,"verificationStatus":null,"verificationType":null},"byCountry":{"US":{"count":2,"credits":0,"tier":"domestic","allowed":false,"blockedReason":"No messaging profile - complete verification first"}},"blockedMessages":[{"index":0,"to":"+15551230001","reason":"No messaging profile - complete verification first"},{"index":1,"to":"+15551230002","reason":"No messaging profile - complete verification first"}],"compliance":{"messageType":"marketing","optedOutBlocked":0,"shaftBlocked":0,"quietHoursBlocked":0,"quietHoursRescheduled":0,"shaftBlockedMessages":[],"quietHoursBlockedMessages":[]},"warnings":["Using TEST key - messages will be simulated in sandbox mode","No messaging profile - complete verification to send messages","Marketing messages are subject to quiet hours enforcement (8pm-8am recipient local time)"]}`,
			want: false,
		},
		{
			name: "a batch of 10,000 messages can be sent",
			body: `{"total":10000,"sendable":10000,"blocked":0,"duplicates":0,"creditsNeeded":20000,"creditBalance":50000,"hasSufficientCredits":true,"pooled":false,"keyType":"live","keyScopes":["sms:send","sms:read"],"hasWriteScope":true,"blockedMessages":[],"compliance":{"messageType":"transactional","optedOutBlocked":0,"shaftBlocked":0,"quietHoursBlocked":0,"quietHoursRescheduled":0,"shaftBlockedMessages":[],"quietHoursBlockedMessages":[]},"warnings":[]}`,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var preview BatchPreviewResponse
			if err := json.Unmarshal([]byte(tt.body), &preview); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if preview.CanSend != tt.want {
				t.Errorf("expected CanSend %v, got %v", tt.want, preview.CanSend)
			}
		})
	}
}

func TestBatchPreviewResponse_RoundTrip(t *testing.T) {
	var original BatchPreviewResponse
	if err := json.Unmarshal([]byte(wirePreviewCountryBlocked), &original); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded BatchPreviewResponse
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip changed the value:\n before %+v\n after  %+v", original, decoded)
	}
}
