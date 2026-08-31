package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

var autoKeyPattern = regexp.MustCompile(`^sendly-go-retry-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// keyOfHeader returns the Idempotency-Key header of a recorded request.
func keyOfHeader(r *http.Request) string {
	return r.Header.Get("Idempotency-Key")
}

// hijackAndClose drops the connection mid-request so the client sees a
// network error without ever receiving an HTTP response.
func hijackAndClose(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	hj, ok := w.(http.Hijacker)
	if !ok {
		t.Fatal("response writer does not support hijacking")
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		t.Fatalf("failed to hijack connection: %v", err)
	}
	conn.Close()
}

// noKeepAliveClient returns an HTTP client that opens a fresh connection per
// attempt, so a dropped connection is never transparently retried by the
// transport and each SDK attempt reaches the test server exactly once.
func noKeepAliveClient() *http.Client {
	return &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
}

func TestIdempotency_AutoKeyOnPost(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123", Status: MessageStatusQueued})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.Send(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if !autoKeyPattern.MatchString(keys[0]) {
		t.Errorf("expected auto-generated key matching %s, got '%s'", autoKeyPattern, keys[0])
	}
	if len(keys[0]) > 255 {
		t.Errorf("expected key length <= 255, got %d", len(keys[0]))
	}
}

func TestIdempotency_NoKeyOnGet(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ListMessagesResponse{})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.List(ctx, &ListMessagesRequest{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if keys[0] != "" {
		t.Errorf("expected no Idempotency-Key on GET, got '%s'", keys[0])
	}
}

func TestIdempotency_NoKeyOnDelete(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "schd_x", "status": "cancelled"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.CancelScheduled(ctx, "3f8b7c1a-9d4e-4f6a-8b2c-1e5d7a9c3b6f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if keys[0] != "" {
		t.Errorf("expected no Idempotency-Key on DELETE, got '%s'", keys[0])
	}
}

func TestIdempotency_NoAutoKeyOnBatch(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BatchMessageResponse{BatchID: "batch_x", Queued: 1})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.SendBatch(ctx, &SendBatchRequest{
		Messages: []BatchMessageItem{{To: "+1234567890", Text: "Hi!"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if keys[0] != "" {
		t.Errorf("expected no auto Idempotency-Key on batch send, got '%s'", keys[0])
	}
}

func TestIdempotency_AutoKeyOnMediaUpload(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MediaFile{ID: "med_x"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Media.Upload(ctx, "x.jpg", strings.NewReader("fake-image-bytes"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if !strings.HasPrefix(keys[0], "sendly-go-retry-") {
		t.Errorf("expected auto-generated key on media upload, got '%s'", keys[0])
	}
}

func TestIdempotency_DistinctKeysPerLogicalRequest(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	if _, err := client.Messages.Send(ctx, &SendMessageRequest{To: "+1234567890", Text: "First"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := client.Messages.Send(ctx, &SendMessageRequest{To: "+1234567890", Text: "Second"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(keys))
	}
	if keys[0] == "" || keys[1] == "" {
		t.Fatal("expected a key on both requests")
	}
	if keys[0] == keys[1] {
		t.Errorf("expected distinct keys per logical request, got '%s' twice", keys[0])
	}
}

func TestIdempotency_KeyReusedAcrossTimeoutRetry(t *testing.T) {
	attempts := 0
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		keys = append(keys, keyOfHeader(r))
		if attempts == 1 {
			time.Sleep(500 * time.Millisecond)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(1), WithTimeout(100*time.Millisecond))
	ctx := context.Background()

	_, err := client.Messages.Send(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if keys[0] != keys[1] {
		t.Errorf("expected key to be reused across a timeout retry, got '%s' then '%s'", keys[0], keys[1])
	}
}

func TestIdempotency_KeyReusedAcrossNetworkErrorRetry(t *testing.T) {
	attempts := 0
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		keys = append(keys, keyOfHeader(r))
		if attempts == 1 {
			hijackAndClose(t, w)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(1), WithHTTPClient(noKeepAliveClient()))
	ctx := context.Background()

	_, err := client.Messages.Send(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if keys[0] != keys[1] {
		t.Errorf("expected key to be reused across a network-error retry, got '%s' then '%s'", keys[0], keys[1])
	}
}

func TestIdempotency_KeyRotatedAfter5xx(t *testing.T) {
	attempts := 0
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		keys = append(keys, keyOfHeader(r))
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIError{Code: "SERVER_ERROR", Message: "Internal server error"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(1))
	ctx := context.Background()

	_, err := client.Messages.Send(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if keys[0] == "" || keys[1] == "" {
		t.Fatal("expected a key on both attempts")
	}
	if keys[0] == keys[1] {
		t.Errorf("expected key to be rotated after a 5xx response, got '%s' twice", keys[0])
	}
}

func TestIdempotency_RotatedKeyKeptAcrossSubsequentNetworkError(t *testing.T) {
	attempts := 0
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		keys = append(keys, keyOfHeader(r))
		switch attempts {
		case 1:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIError{Code: "SERVER_ERROR", Message: "Internal server error"})
		case 2:
			hijackAndClose(t, w)
		default:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(Message{ID: "msg_123"})
		}
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(2), WithHTTPClient(noKeepAliveClient()))
	ctx := context.Background()

	_, err := client.Messages.Send(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if keys[1] == keys[0] {
		t.Errorf("expected key to be rotated after the 5xx, got '%s' twice", keys[0])
	}
	if keys[2] != keys[1] {
		t.Errorf("expected rotated key to be kept across the network error, got '%s' then '%s'", keys[1], keys[2])
	}
}

func TestIdempotency_KeyKeptAcrossNon5xxRetry(t *testing.T) {
	attempts := 0
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		keys = append(keys, keyOfHeader(r))
		if attempts == 1 {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(APIError{Code: "CONFLICT", Message: "Resource busy"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(1))
	ctx := context.Background()

	_, err := client.Messages.Send(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if keys[0] != keys[1] {
		t.Errorf("expected key to be kept across a non-5xx retry, got '%s' then '%s'", keys[0], keys[1])
	}
}

func TestIdempotency_CallerKeySentVerbatim(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.SendWithOptions(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"}, WithIdempotencyKey("order-4821-shipped"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if keys[0] != "order-4821-shipped" {
		t.Errorf("expected caller key 'order-4821-shipped', got '%s'", keys[0])
	}
}

func TestIdempotency_CallerKeyNeverRotatedAcross5xxRetry(t *testing.T) {
	attempts := 0
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		keys = append(keys, keyOfHeader(r))
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIError{Code: "SERVER_ERROR", Message: "Internal server error"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(1))
	ctx := context.Background()

	_, err := client.Messages.SendWithOptions(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"}, WithIdempotencyKey("order-4821-shipped"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if keys[0] != "order-4821-shipped" || keys[1] != "order-4821-shipped" {
		t.Errorf("expected caller key on both attempts, got '%s' then '%s'", keys[0], keys[1])
	}
}

func TestIdempotency_CallerKeyReusedAcrossNetworkErrorRetry(t *testing.T) {
	attempts := 0
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		keys = append(keys, keyOfHeader(r))
		if attempts == 1 {
			hijackAndClose(t, w)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(1), WithHTTPClient(noKeepAliveClient()))
	ctx := context.Background()

	_, err := client.Messages.SendWithOptions(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"}, WithIdempotencyKey("signup-otp-user-99"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if keys[0] != "signup-otp-user-99" || keys[1] != "signup-otp-user-99" {
		t.Errorf("expected caller key on both attempts, got '%s' then '%s'", keys[0], keys[1])
	}
}

func TestIdempotency_CallerKeyOnSendBatch(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BatchMessageResponse{BatchID: "batch_x", Queued: 1})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.SendBatchWithOptions(ctx, &SendBatchRequest{
		Messages: []BatchMessageItem{{To: "+1234567890", Text: "Hi!"}},
	}, WithIdempotencyKey("campaign-77-wave-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if keys[0] != "campaign-77-wave-1" {
		t.Errorf("expected caller key 'campaign-77-wave-1', got '%s'", keys[0])
	}
}

func TestIdempotency_CallerKeyOnSchedule(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "schd_x", "status": "scheduled"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.ScheduleWithOptions(ctx, &ScheduleMessageRequest{
		To:          "+1234567890",
		Text:        "Reminder!",
		ScheduledAt: "2030-01-01T10:00:00Z",
	}, WithIdempotencyKey("reminder-visit-31"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if keys[0] != "reminder-visit-31" {
		t.Errorf("expected caller key 'reminder-visit-31', got '%s'", keys[0])
	}
}

func TestIdempotency_CallerKeyOnSendGroup(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "msg_x", "group_message_id": "grp_x"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.SendGroupWithOptions(ctx, &SendGroupMessageRequest{
		To:   []string{"+14155551234", "+14155555678"},
		Text: "Team sync at noon",
	}, WithIdempotencyKey("standup-ping-0823"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if keys[0] != "standup-ping-0823" {
		t.Errorf("expected caller key 'standup-ping-0823', got '%s'", keys[0])
	}
}

func TestIdempotency_CallerKeyOnSendWhatsApp(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "msg_wa", "channel": "whatsapp"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.SendWhatsAppWithOptions(ctx, &SendWhatsAppMessageRequest{
		To:   "+15551234567",
		From: "+15559876543",
		Text: "Hello!",
	}, WithIdempotencyKey("wa-hello-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if keys[0] != "wa-hello-1" {
		t.Errorf("expected caller key 'wa-hello-1', got '%s'", keys[0])
	}
}

func TestIdempotency_CallerKeyOnSendRcs(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "msg_rcs", "channel": "rcs"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.SendRcsWithOptions(ctx, &SendRcsMessageRequest{
		To:   "+15551234567",
		Text: "Hello!",
	}, WithIdempotencyKey("rcs-hello-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if keys[0] != "rcs-hello-1" {
		t.Errorf("expected caller key 'rcs-hello-1', got '%s'", keys[0])
	}
}

func TestIdempotency_EmptyCallerKeyFallsBackToAuto(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.SendWithOptions(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"}, WithIdempotencyKey(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if !strings.HasPrefix(keys[0], "sendly-go-retry-") {
		t.Errorf("expected auto-generated key for empty caller key, got '%s'", keys[0])
	}
}

func TestIdempotency_WhitespaceCallerKeyFallsBackToAuto(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, keyOfHeader(r))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.Messages.SendWithOptions(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"}, WithIdempotencyKey("   "))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 request, got %d", len(keys))
	}
	if !strings.HasPrefix(keys[0], "sendly-go-retry-") {
		t.Errorf("expected auto-generated key for whitespace-only caller key, got '%s'", keys[0])
	}
}

func TestIdempotency_InvalidCallerKeyRejectedImmediately(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Message{ID: "msg_123"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	tests := []struct {
		name string
		key  string
	}{
		{
			name: "non-ASCII key",
			key:  "Заказ-42",
		},
		{
			name: "key longer than 255 characters",
			key:  strings.Repeat("k", 256),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Messages.SendWithOptions(ctx, &SendMessageRequest{To: "+1234567890", Text: "Hello!"}, WithIdempotencyKey(tt.key))

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}

	if attempts != 0 {
		t.Errorf("expected no network calls for invalid keys, got %d", attempts)
	}
}
