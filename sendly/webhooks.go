// Package sendly provides the official Go SDK for the Sendly SMS API.
package sendly

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// WebhookEventType represents the type of webhook event
type WebhookEventType string

const (
	WebhookEventMessageQueued      WebhookEventType = "message.queued"
	WebhookEventMessageSent        WebhookEventType = "message.sent"
	WebhookEventMessageDelivered   WebhookEventType = "message.delivered"
	WebhookEventMessageFailed      WebhookEventType = "message.failed"
	WebhookEventMessageBounced     WebhookEventType = "message.bounced"
	WebhookEventMessageRetrying    WebhookEventType = "message.retrying"
	WebhookEventMessageReceived    WebhookEventType = "message.received"
	WebhookEventMessageOptOut      WebhookEventType = "message.opt_out"
	WebhookEventMessageOptIn       WebhookEventType = "message.opt_in"
	WebhookEventMessageUndelivered WebhookEventType = "message.undelivered"

	signatureToleranceSeconds = 300
)

// WebhookMessageStatus represents the status of a message in webhook events
type WebhookMessageStatus string

const (
	WebhookStatusQueued      WebhookMessageStatus = "queued"
	WebhookStatusSent        WebhookMessageStatus = "sent"
	WebhookStatusDelivered   WebhookMessageStatus = "delivered"
	WebhookStatusFailed      WebhookMessageStatus = "failed"
	WebhookStatusBounced     WebhookMessageStatus = "bounced"
	WebhookStatusReceived    WebhookMessageStatus = "received"
	WebhookStatusUndelivered WebhookMessageStatus = "undelivered"
)

// WebhookMessageData contains the data payload for message webhook events
type WebhookMessageData struct {
	ID             string               `json:"id"`
	Status         WebhookMessageStatus `json:"status"`
	To             string               `json:"to"`
	From           string               `json:"from"`
	Direction      string               `json:"direction,omitempty"`
	OrganizationID *string              `json:"organization_id,omitempty"`
	Text           string               `json:"text,omitempty"`
	Error          string               `json:"error,omitempty"`
	ErrorCode      string               `json:"error_code,omitempty"`
	DeliveredAt    interface{}          `json:"delivered_at,omitempty"`
	FailedAt       interface{}          `json:"failed_at,omitempty"`
	CreatedAt      interface{}          `json:"created_at,omitempty"`
	Segments       int                  `json:"segments"`
	CreditsUsed    int                  `json:"credits_used"`
	MessageFormat  string               `json:"message_format,omitempty"`
	MediaUrls      []string             `json:"media_urls,omitempty"`
}

// MessageID returns the message ID (backwards-compatible alias for ID)
func (d *WebhookMessageData) MessageID() string {
	return d.ID
}

// rawWebhookEvent is used for flexible JSON unmarshaling
type rawWebhookEvent struct {
	ID         string           `json:"id"`
	Type       WebhookEventType `json:"type"`
	Data       json.RawMessage  `json:"data"`
	Created    interface{}      `json:"created"`
	CreatedAt  interface{}      `json:"created_at"`
	APIVersion string           `json:"api_version"`
	Livemode   bool             `json:"livemode"`
}

// rawDataWrapper handles the data.object nesting
type rawDataWrapper struct {
	Object json.RawMessage `json:"object"`
}

// WebhookEvent represents a webhook event from Sendly
type WebhookEvent struct {
	ID         string             `json:"id"`
	Type       WebhookEventType   `json:"type"`
	Data       WebhookMessageData `json:"-"`
	Created    interface{}        `json:"created"`
	APIVersion string             `json:"api_version"`
	Livemode   bool               `json:"livemode"`
}

// ErrInvalidSignature is returned when webhook signature verification fails
var ErrInvalidSignature = errors.New("invalid webhook signature")

// Webhooks provides utilities for verifying and parsing Sendly webhook events
type Webhooks struct{}

// VerifySignature verifies the webhook signature from Sendly.
// Pass an empty string for timestamp to skip timestamp verification (backwards compat).
func (w Webhooks) VerifySignature(payload, signature, secret, timestamp string) bool {
	if payload == "" || signature == "" || secret == "" {
		return false
	}

	signedPayload := payload
	if timestamp != "" {
		signedPayload = timestamp + "." + payload
		ts, err := strconv.ParseFloat(timestamp, 64)
		if err == nil {
			if math.Abs(float64(time.Now().Unix())-ts) > signatureToleranceSeconds {
				return false
			}
		}
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if len(signature) != len(expected) {
		return false
	}

	return hmac.Equal([]byte(signature), []byte(expected))
}

// ParseEvent parses and validates a webhook event.
// Pass an empty string for timestamp to skip timestamp verification.
func (w Webhooks) ParseEvent(payload, signature, secret, timestamp string) (*WebhookEvent, error) {
	if !w.VerifySignature(payload, signature, secret, timestamp) {
		return nil, ErrInvalidSignature
	}

	var raw rawWebhookEvent
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	if raw.ID == "" || raw.Type == "" {
		return nil, errors.New("invalid event structure")
	}

	var msgData WebhookMessageData
	var wrapper rawDataWrapper
	if err := json.Unmarshal(raw.Data, &wrapper); err == nil && wrapper.Object != nil {
		if err := json.Unmarshal(wrapper.Object, &msgData); err != nil {
			return nil, fmt.Errorf("failed to parse webhook data.object: %w", err)
		}
	} else {
		if err := json.Unmarshal(raw.Data, &msgData); err != nil {
			return nil, fmt.Errorf("failed to parse webhook data: %w", err)
		}
		if msgData.ID == "" {
			var legacy struct {
				MessageID string `json:"message_id"`
			}
			json.Unmarshal(raw.Data, &legacy)
			if legacy.MessageID != "" {
				msgData.ID = legacy.MessageID
			}
		}
	}

	created := raw.Created
	if created == nil {
		created = raw.CreatedAt
	}

	return &WebhookEvent{
		ID:         raw.ID,
		Type:       raw.Type,
		Data:       msgData,
		Created:    created,
		APIVersion: raw.APIVersion,
		Livemode:   raw.Livemode,
	}, nil
}

// GenerateSignature generates a webhook signature for testing purposes.
// Pass an empty string for timestamp to skip timestamp in signature.
func (w Webhooks) GenerateSignature(payload, secret, timestamp string) string {
	signedPayload := payload
	if timestamp != "" {
		signedPayload = timestamp + "." + payload
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Helper function to check signature with constant-time comparison
func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	a = strings.TrimPrefix(a, "sha256=")
	b = strings.TrimPrefix(b, "sha256=")

	return hmac.Equal([]byte(a), []byte(b))
}
