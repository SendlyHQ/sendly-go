package sendly

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		opts     []ClientOption
		validate func(*testing.T, *Client)
	}{
		{
			name:   "default configuration",
			apiKey: "test-api-key",
			opts:   nil,
			validate: func(t *testing.T, c *Client) {
				if c.APIKey != "test-api-key" {
					t.Errorf("expected APIKey to be 'test-api-key', got '%s'", c.APIKey)
				}
				if c.BaseURL != DefaultBaseURL {
					t.Errorf("expected BaseURL to be '%s', got '%s'", DefaultBaseURL, c.BaseURL)
				}
				if c.Timeout != DefaultTimeout {
					t.Errorf("expected Timeout to be %v, got %v", DefaultTimeout, c.Timeout)
				}
				if c.MaxRetries != 3 {
					t.Errorf("expected MaxRetries to be 3, got %d", c.MaxRetries)
				}
				if c.Messages == nil {
					t.Error("expected Messages service to be initialized")
				}
			},
		},
		{
			name:   "with custom base URL",
			apiKey: "test-api-key",
			opts:   []ClientOption{WithBaseURL("https://custom.example.com")},
			validate: func(t *testing.T, c *Client) {
				if c.BaseURL != "https://custom.example.com" {
					t.Errorf("expected BaseURL to be 'https://custom.example.com', got '%s'", c.BaseURL)
				}
			},
		},
		{
			name:   "with custom timeout",
			apiKey: "test-api-key",
			opts:   []ClientOption{WithTimeout(60 * time.Second)},
			validate: func(t *testing.T, c *Client) {
				if c.Timeout != 60*time.Second {
					t.Errorf("expected Timeout to be 60s, got %v", c.Timeout)
				}
				if c.HTTPClient.Timeout != 60*time.Second {
					t.Errorf("expected HTTPClient.Timeout to be 60s, got %v", c.HTTPClient.Timeout)
				}
			},
		},
		{
			name:   "with custom max retries",
			apiKey: "test-api-key",
			opts:   []ClientOption{WithMaxRetries(5)},
			validate: func(t *testing.T, c *Client) {
				if c.MaxRetries != 5 {
					t.Errorf("expected MaxRetries to be 5, got %d", c.MaxRetries)
				}
			},
		},
		{
			name:   "with debug enabled",
			apiKey: "test-api-key",
			opts:   []ClientOption{WithDebug(true)},
			validate: func(t *testing.T, c *Client) {
				if !c.Debug {
					t.Error("expected Debug to be true")
				}
			},
		},
		{
			name:   "with custom HTTP client",
			apiKey: "test-api-key",
			opts: []ClientOption{WithHTTPClient(&http.Client{
				Timeout: 90 * time.Second,
			})},
			validate: func(t *testing.T, c *Client) {
				if c.HTTPClient.Timeout != 90*time.Second {
					t.Errorf("expected HTTPClient.Timeout to be 90s, got %v", c.HTTPClient.Timeout)
				}
			},
		},
		{
			name:   "with multiple options",
			apiKey: "test-api-key",
			opts: []ClientOption{
				WithBaseURL("https://custom.example.com"),
				WithTimeout(45 * time.Second),
				WithMaxRetries(10),
				WithDebug(true),
			},
			validate: func(t *testing.T, c *Client) {
				if c.BaseURL != "https://custom.example.com" {
					t.Errorf("expected BaseURL to be 'https://custom.example.com', got '%s'", c.BaseURL)
				}
				if c.Timeout != 45*time.Second {
					t.Errorf("expected Timeout to be 45s, got %v", c.Timeout)
				}
				if c.MaxRetries != 10 {
					t.Errorf("expected MaxRetries to be 10, got %d", c.MaxRetries)
				}
				if !c.Debug {
					t.Error("expected Debug to be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.apiKey, tt.opts...)
			tt.validate(t, client)
		})
	}
}

func TestClientRequest_Headers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-api-key" {
			t.Errorf("expected Authorization header to be 'Bearer test-api-key', got '%s'", auth)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type header to be 'application/json', got '%s'", ct)
		}
		if accept := r.Header.Get("Accept"); accept != "application/json" {
			t.Errorf("expected Accept header to be 'application/json', got '%s'", accept)
		}
		if ua := r.Header.Get("User-Agent"); ua != "sendly-go/"+Version {
			t.Errorf("expected User-Agent header to be 'sendly-go/%s', got '%s'", Version, ua)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	var result map[string]string
	err := client.request(ctx, "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientRequest_Retries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIError{
				Code:    "SERVER_ERROR",
				Message: "Internal server error",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(3))
	ctx := context.Background()

	var result map[string]string
	err := client.request(ctx, "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClientRequest_NoRetryOnAuthError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(APIError{
			Code:    "UNAUTHORIZED",
			Message: "Invalid API key",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(3))
	ctx := context.Background()

	var result map[string]string
	err := client.request(ctx, "GET", "/test", nil, &result)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry on auth error), got %d", attempts)
	}
}

func TestClientRequest_NoRetryOnValidationError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{
			Code:    "VALIDATION_ERROR",
			Message: "Invalid phone number",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(3))
	ctx := context.Background()

	var result map[string]string
	err := client.request(ctx, "GET", "/test", nil, &result)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry on validation error), got %d", attempts)
	}
}

func TestClientRequest_RateLimitWithRetryAfter(t *testing.T) {
	attempts := 0
	start := time.Now()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(APIError{
				Code:    "RATE_LIMIT_EXCEEDED",
				Message: "Too many requests",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(3))
	ctx := context.Background()

	var result map[string]string
	err := client.request(ctx, "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < time.Second {
		t.Errorf("expected to wait at least 1 second for Retry-After, waited %v", elapsed)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestClientRequest_DecodeFailureIsNotRetried(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"unexpected":true}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(3))

	_, err := client.Webhooks.List(context.Background())

	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("expected *NetworkError, got %T: %v", err, err)
	}
	if netErr.Message != "failed to unmarshal response" {
		t.Errorf("expected message 'failed to unmarshal response', got %q", netErr.Message)
	}
	var typeErr *json.UnmarshalTypeError
	if !errors.As(err, &typeErr) {
		t.Errorf("expected the JSON decode error to stay reachable, got %v", netErr.Err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 request, got %d", attempts)
	}
}

func TestClientRequest_DecodeFailureOnPostIsSentOnce(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"unexpected":true}]`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(3))

	_, err := client.Messages.Send(context.Background(), &SendMessageRequest{To: "+15551234567", Text: "Hello"})
	if !IsNetworkError(err) {
		t.Fatalf("expected *NetworkError, got %T: %v", err, err)
	}
	if attempts != 1 {
		t.Errorf("expected the POST to be sent once, got %d requests", attempts)
	}
}

func TestBuildQueryString(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]string
		expected string
	}{
		{
			name:     "empty params",
			params:   map[string]string{},
			expected: "",
		},
		{
			name: "single param",
			params: map[string]string{
				"limit": "10",
			},
			expected: "?limit=10",
		},
		{
			name: "multiple params",
			params: map[string]string{
				"limit":  "10",
				"offset": "20",
			},
			expected: "?limit=10&offset=20",
		},
		{
			name: "empty values ignored",
			params: map[string]string{
				"limit":  "10",
				"status": "",
			},
			expected: "?limit=10",
		},
		{
			name: "all empty values",
			params: map[string]string{
				"status": "",
				"to":     "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildQueryString(tt.params)

			// For multiple params, we need to check both possible orders
			// since map iteration is non-deterministic
			if tt.name == "multiple params" {
				alt := "?offset=20&limit=10"
				if result != tt.expected && result != alt {
					t.Errorf("expected '%s' or '%s', got '%s'", tt.expected, alt, result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}

func TestClientRequest_RetriesConcurrentKeyCheckRefusal(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"too_many_concurrent_verifications","message":"Too many API key checks are already running for this account from this address. Try again in 1 second.","retryAfter":1}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"balance":10,"reservedBalance":0,"availableBalance":10,"billingMode":"prepaid"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	start := time.Now()
	credits, err := client.Account.GetCredits(context.Background())
	if err != nil {
		t.Fatalf("expected the refusal to be retried, got %v", err)
	}
	if credits.Balance != 10 {
		t.Errorf("expected Balance 10, got %d", credits.Balance)
	}
	if attempts != 2 {
		t.Errorf("expected 2 requests, got %d", attempts)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("expected to wait for Retry-After, waited %v", elapsed)
	}
}

func TestClientRequest_ReturnsTheFailedKeyLockoutAtOnce(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "240")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"too_many_failed_key_attempts","message":"Too many failed API key attempts. Try again in 240 seconds.","retryAfter":240}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	_, err := client.Account.GetCredits(ctx)
	rateErr, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if rateErr.Code != "too_many_failed_key_attempts" {
		t.Errorf("expected code too_many_failed_key_attempts, got %q", rateErr.Code)
	}
	if rateErr.RetryAfter != 240 {
		t.Errorf("expected RetryAfter 240, got %d", rateErr.RetryAfter)
	}
	if attempts != 1 {
		t.Errorf("expected 1 request, got %d", attempts)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("expected no wait, waited %v", elapsed)
	}
}

func TestClientRequest_ReturnsOtherFinal429sAtOnce(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"max_attempts_exceeded","message":"Maximum verification attempts exceeded"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	_, err := client.Account.GetCredits(ctx)
	if rateErr, ok := err.(*RateLimitError); !ok || rateErr.Code != "max_attempts_exceeded" {
		t.Fatalf("expected *RateLimitError max_attempts_exceeded, got %T: %v", err, err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 request, got %d", attempts)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("expected no wait, waited %v", elapsed)
	}
}

func TestClientRequest_StillRetriesAnOrdinaryRateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Rate limit exceeded. Limit: 60 requests per minute.","retryAfter":1}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"balance":10,"reservedBalance":0,"availableBalance":10,"billingMode":"prepaid"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	credits, err := client.Account.GetCredits(context.Background())
	if err != nil {
		t.Fatalf("expected the rate limit to be retried, got %v", err)
	}
	if credits.Balance != 10 || attempts != 2 {
		t.Errorf("expected Balance 10 after 2 requests, got %d after %d", credits.Balance, attempts)
	}
}

func TestClientRequest_ReturnsALongRateLimitAtOnce(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Too many OTPs sent to this phone number. Max 5 per 10 minutes.","retryAfter":600}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	_, err := client.Account.GetCredits(ctx)
	rateErr, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if rateErr.RetryAfter != 600 {
		t.Errorf("expected RetryAfter 600 from the body, got %d", rateErr.RetryAfter)
	}
	if attempts != 1 {
		t.Errorf("expected 1 request, got %d", attempts)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("expected no wait, waited %v", elapsed)
	}
}

func TestClientRequest_RetriesA429WithoutACode(t *testing.T) {
	for name, body := range map[string]string{
		"json": `{"message":"Too many requests"}`,
		"text": `Too Many Requests`,
	} {
		t.Run(name, func(t *testing.T) {
			attempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				if attempts == 1 {
					w.Header().Set("Retry-After", "1")
					w.WriteHeader(http.StatusTooManyRequests)
					w.Write([]byte(body))
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"balance":10,"reservedBalance":0,"availableBalance":10,"billingMode":"prepaid"}`))
			}))
			defer server.Close()

			client := NewClient("test-api-key", WithBaseURL(server.URL))

			credits, err := client.Account.GetCredits(context.Background())
			if err != nil {
				t.Fatalf("expected the 429 to be retried, got %v", err)
			}
			if credits.Balance != 10 || attempts != 2 {
				t.Errorf("expected Balance 10 after 2 requests, got %d after %d", credits.Balance, attempts)
			}
		})
	}
}

func TestRateLimitError_NamesTheCauseWhenItIsNotTheRequestLimit(t *testing.T) {
	lockout := &RateLimitError{
		APIError:   APIError{Code: "too_many_failed_key_attempts", Message: "Too many failed API key attempts. Try again in 240 seconds."},
		RetryAfter: 240,
	}
	want := "sendly: Too many failed API key attempts. Try again in 240 seconds. (code: too_many_failed_key_attempts, retry after 240 seconds)"
	if got := lockout.Error(); got != want {
		t.Errorf("lockout Error() = %q, want %q", got, want)
	}

	ordinary := &RateLimitError{
		APIError:   APIError{Code: "rate_limit_exceeded", Message: "Rate limit exceeded. Limit: 60 requests per minute."},
		RetryAfter: 30,
	}
	if got := ordinary.Error(); got != "sendly: rate limit exceeded, retry after 30 seconds" {
		t.Errorf("ordinary Error() = %q, want the unchanged format", got)
	}
}

func TestRateLimitError_ALongWaitKeepsTheReason(t *testing.T) {
	otp := &RateLimitError{
		APIError:   APIError{Code: "rate_limit_exceeded", Message: "Too many OTPs sent to this phone number. Max 5 per 10 minutes."},
		RetryAfter: 600,
	}
	want := "sendly: Too many OTPs sent to this phone number. Max 5 per 10 minutes. (code: rate_limit_exceeded, retry after 600 seconds)"
	if got := otp.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestRateLimitError_LeavesOutAZeroWait(t *testing.T) {
	final := &RateLimitError{APIError: APIError{Code: "max_attempts_exceeded", Message: "Maximum verification attempts exceeded"}}
	want := "sendly: Maximum verification attempts exceeded (code: max_attempts_exceeded)"
	if got := final.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestClientRequest_WaitsRetryAfterWithoutAddingBackoff(t *testing.T) {
	var stamps []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stamps = append(stamps, time.Now())
		if len(stamps) == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Rate limit exceeded. Limit: 60 requests per minute.","retryAfter":2}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"balance":10,"reservedBalance":0,"availableBalance":10,"billingMode":"prepaid"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	if _, err := client.Account.GetCredits(context.Background()); err != nil {
		t.Fatalf("expected the rate limit to be retried, got %v", err)
	}
	if len(stamps) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(stamps))
	}
	if gap := stamps[1].Sub(stamps[0]); gap < 2*time.Second || gap >= 2900*time.Millisecond {
		t.Errorf("expected a wait of about 2 s, got %v", gap)
	}
}

func TestClientRequest_DoesNotWaitAfterTheLastAttempt(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Rate limit exceeded. Limit: 60 requests per minute.","retryAfter":2}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))

	start := time.Now()
	_, err := client.Account.GetCredits(context.Background())
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 request, got %d", attempts)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("expected no wait after the last attempt, waited %v", elapsed)
	}
}

func TestClientRequest_WaitsOutThePerMinuteProvisioningLimit(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		if len(keys) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"provision_rate_limit","message":"Max 120 provisions per minute.","retryAfter":1}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"workspace":{"id":"ws_1","name":"Acme"}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	start := time.Now()
	if _, err := client.Enterprise.Provision(context.Background(), &ProvisionWorkspaceRequest{Name: "Acme"}); err != nil {
		t.Fatalf("expected the provisioning limit to be waited out, got %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(keys))
	}
	if keys[0] == "" || keys[0] != keys[1] {
		t.Errorf("expected the retry to keep the idempotency key, got %q then %q", keys[0], keys[1])
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("expected to wait for retryAfter, waited %v", elapsed)
	}
}

func TestClientRequest_ReturnsTheHourlyProvisioningLimitAtOnce(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"provision_rate_limit","message":"Max 1000 provisions per hour.","retryAfter":3100}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.Enterprise.Provision(ctx, &ProvisionWorkspaceRequest{Name: "Acme"})
	rateErr, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if rateErr.Code != "provision_rate_limit" || rateErr.RetryAfter != 3100 {
		t.Errorf("expected provision_rate_limit with RetryAfter 3100, got %q %d", rateErr.Code, rateErr.RetryAfter)
	}
	if attempts != 1 {
		t.Errorf("expected 1 request, got %d", attempts)
	}
}
