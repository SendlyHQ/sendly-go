// Package sendly provides the official Go SDK for the Sendly SMS API.
package sendly

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// WebhookEventType represents the type of webhook event
type WebhookEventType string

const (
	WebhookEventMessageQueued      WebhookEventType = "message.queued"
	WebhookEventMessageSent        WebhookEventType = "message.sent"
	WebhookEventMessageDelivered   WebhookEventType = "message.delivered"
	WebhookEventMessageFailed      WebhookEventType = "message.failed"
	WebhookEventMessageUndelivered WebhookEventType = "message.undelivered"
)

// WebhookMessageStatus represents the status of a message in webhook events
type WebhookMessageStatus string

const (
	WebhookStatusQueued      WebhookMessageStatus = "queued"
	WebhookStatusSent        WebhookMessageStatus = "sent"
	WebhookStatusDelivered   WebhookMessageStatus = "delivered"
	WebhookStatusFailed      WebhookMessageStatus = "failed"
	WebhookStatusBounced     WebhookMessageStatus = "bounced"
	WebhookStatusUndelivered WebhookMessageStatus = "undelivered"
)

// WebhookMessageData contains the data payload for message webhook events
type WebhookMessageData struct {
	MessageID   string               `json:"message_id"`
	Status      WebhookMessageStatus `json:"status"`
	To          string               `json:"to"`
	From        string               `json:"from"`
	Error       string               `json:"error,omitempty"`
	ErrorCode   string               `json:"error_code,omitempty"`
	DeliveredAt string               `json:"delivered_at,omitempty"`
	FailedAt    string               `json:"failed_at,omitempty"`
	Segments    int                  `json:"segments"`
	CreditsUsed int                  `json:"credits_used"`
}

// WebhookEvent represents a webhook event from Sendly
type WebhookEvent struct {
	ID         string             `json:"id"`
	Type       WebhookEventType   `json:"type"`
	Data       WebhookMessageData `json:"data"`
	CreatedAt  string             `json:"created_at"`
	APIVersion string             `json:"api_version"`
}

// ErrInvalidSignature is returned when webhook signature verification fails
var ErrInvalidSignature = errors.New("invalid webhook signature")

// Webhooks provides utilities for verifying and parsing Sendly webhook events
type Webhooks struct{}

// VerifySignature verifies the webhook signature from Sendly
//
// Parameters:
//   - payload: Raw request body as string
//   - signature: X-Sendly-Signature header value
//   - secret: Your webhook secret from dashboard
//
// # Returns true if signature is valid, false otherwise
//
// Example:
//
//	isValid := sendly.Webhooks{}.VerifySignature(rawBody, signature, secret)
func (w Webhooks) VerifySignature(payload, signature, secret string) bool {
	if payload == "" || signature == "" || secret == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Timing-safe comparison
	if len(signature) != len(expected) {
		return false
	}

	return hmac.Equal([]byte(signature), []byte(expected))
}

// ParseEvent parses and validates a webhook event
//
// Parameters:
//   - payload: Raw request body as string
//   - signature: X-Sendly-Signature header value
//   - secret: Your webhook secret from dashboard
//
// # Returns the parsed WebhookEvent or an error if signature is invalid
//
// Example:
//
//	event, err := sendly.Webhooks{}.ParseEvent(rawBody, signature, secret)
//	if err != nil {
//	    log.Fatal("Invalid webhook signature")
//	}
//	fmt.Printf("Event type: %s\n", event.Type)
func (w Webhooks) ParseEvent(payload, signature, secret string) (*WebhookEvent, error) {
	if !w.VerifySignature(payload, signature, secret) {
		return nil, ErrInvalidSignature
	}

	var event WebhookEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Basic validation
	if event.ID == "" || event.Type == "" || event.CreatedAt == "" {
		return nil, errors.New("invalid event structure")
	}

	return &event, nil
}

// GenerateSignature generates a webhook signature for testing purposes
//
// Parameters:
//   - payload: The payload to sign
//   - secret: The secret to use for signing
//
// Returns the signature in the format "sha256=..."
//
// Example:
//
//	signature := sendly.Webhooks{}.GenerateSignature(testPayload, "test_secret")
func (w Webhooks) GenerateSignature(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Helper function to check signature with constant-time comparison
func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	// Remove sha256= prefix if present
	a = strings.TrimPrefix(a, "sha256=")
	b = strings.TrimPrefix(b, "sha256=")

	return hmac.Equal([]byte(a), []byte(b))
}
