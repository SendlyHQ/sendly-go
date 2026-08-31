package sendly

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

// RequestOption configures a single API operation.
type RequestOption func(*requestConfig)

// requestConfig carries per-request idempotency behavior through the
// request pipeline.
type requestConfig struct {
	idempotencyKey     string
	autoIdempotencyKey bool
}

// newRequestConfig applies options over the default configuration.
func newRequestConfig(opts []RequestOption) requestConfig {
	cfg := requestConfig{autoIdempotencyKey: true}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithIdempotencyKey sets the idempotency key for this operation (1-255
// printable ASCII characters).
//
// The SDK already generates a key per logical request automatically, so
// the server can dedupe the SDK's own timeout retries. Supply your own
// key when you need idempotency across process restarts or your own
// retry loops — repeating a request with the same key within 24 hours
// returns the original response instead of executing again.
//
// Note: a response is cached under the key once the original attempt
// completes, including error responses — retrying a failed request with
// the same key returns the recorded failure; use a fresh key to
// re-execute.
func WithIdempotencyKey(key string) RequestOption {
	return func(cfg *requestConfig) {
		cfg.idempotencyKey = key
	}
}

// withoutAutoIdempotencyKey skips auto-generating an idempotency key for a
// POST. Used for the batch endpoint, where the server dedupes header-less
// retries by request content and an auto key would bypass that net.
// A caller-supplied key is always sent regardless.
func withoutAutoIdempotencyKey() RequestOption {
	return func(cfg *requestConfig) {
		cfg.autoIdempotencyKey = false
	}
}

// generateIdempotencyKey generates an idempotency key for a logical request.
// Reused across retry attempts so the server can recognize a retry of a
// timed-out POST that actually reached it.
func generateIdempotencyKey() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand does not fail on supported platforms; degrade to a
		// timestamp-based key rather than aborting the send.
		return fmt.Sprintf("sendly-go-retry-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("sendly-go-retry-%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// normalizeIdempotencyKey validates and normalizes a caller-supplied
// idempotency key. Empty and whitespace-only values are treated as absent
// (auto-generation still applies); invalid values fail fast instead of
// surfacing later as a retried network error.
func normalizeIdempotencyKey(key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", nil
	}
	if len(trimmed) > 255 {
		return "", &ValidationError{APIError: APIError{Message: "idempotency key must be 1-255 printable ASCII characters"}}
	}
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] < 0x20 || trimmed[i] > 0x7e {
			return "", &ValidationError{APIError: APIError{Message: "idempotency key must be 1-255 printable ASCII characters"}}
		}
	}
	return trimmed, nil
}

// isServerErrorResponse reports whether the error carries an actual 5xx
// response from the server, as opposed to a timeout or network failure where
// the outcome of the request is unknown. NetworkError carries no status code.
func isServerErrorResponse(err error) bool {
	if se, ok := err.(*SendlyError); ok {
		return se.StatusCode >= 500
	}
	return false
}
