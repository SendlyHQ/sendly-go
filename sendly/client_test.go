package sendly

import (
	"context"
	"encoding/json"
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
