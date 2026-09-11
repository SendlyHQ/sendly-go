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
	// Deprecated: the API has never emitted this and rejects it when you
	// subscribe. It will be removed in the next major version.
	WebhookEventMessageQueued WebhookEventType = "message.queued"
	// Deprecated: the API has never emitted this and rejects it when you
	// subscribe. It will be removed in the next major version.
	WebhookEventMessageUndelivered WebhookEventType = "message.undelivered"

	WebhookEventMessageSent                WebhookEventType = "message.sent"
	WebhookEventMessageDelivered           WebhookEventType = "message.delivered"
	WebhookEventMessageRead                WebhookEventType = "message.read"
	WebhookEventMessageFailed              WebhookEventType = "message.failed"
	WebhookEventMessageBounced             WebhookEventType = "message.bounced"
	WebhookEventMessageRetrying            WebhookEventType = "message.retrying"
	WebhookEventMessageReceived            WebhookEventType = "message.received"
	WebhookEventMessageOptOut              WebhookEventType = "message.opt_out"
	WebhookEventMessageOptIn               WebhookEventType = "message.opt_in"
	WebhookEventVerificationCreated        WebhookEventType = "verification.created"
	WebhookEventVerificationDelivered      WebhookEventType = "verification.delivered"
	WebhookEventVerificationVerified       WebhookEventType = "verification.verified"
	WebhookEventVerificationExpired        WebhookEventType = "verification.expired"
	WebhookEventVerificationFailed         WebhookEventType = "verification.failed"
	WebhookEventVerificationResent         WebhookEventType = "verification.resent"
	WebhookEventVerificationDeliveryFailed WebhookEventType = "verification.delivery_failed"
	WebhookEventConversationCreated        WebhookEventType = "conversation.created"
	WebhookEventConversationUpdated        WebhookEventType = "conversation.updated"
	WebhookEventDraftCreated               WebhookEventType = "draft.created"
	WebhookEventDraftApproved              WebhookEventType = "draft.approved"
	WebhookEventDraftRejected              WebhookEventType = "draft.rejected"
	WebhookEventContactAutoFlagged         WebhookEventType = "contact.auto_flagged"
	WebhookEventContactMarkedValid         WebhookEventType = "contact.marked_valid"
	WebhookEventContactsLookupCompleted    WebhookEventType = "contacts.lookup_completed"
	WebhookEventContactsBulkMarkedValid    WebhookEventType = "contacts.bulk_marked_valid"
	WebhookEventBrandVerified              WebhookEventType = "brand.verified"
	WebhookEventBrandFailed                WebhookEventType = "brand.failed"
	WebhookEventCampaignApproved           WebhookEventType = "campaign.approved"
	WebhookEventCampaignRejected           WebhookEventType = "campaign.rejected"
	WebhookEventCampaignSuspended          WebhookEventType = "campaign.suspended"
	WebhookEventAssignmentConfirmed        WebhookEventType = "assignment.confirmed"
	WebhookEventAssignmentFailed           WebhookEventType = "assignment.failed"
	WebhookEventRcsBrandVerified           WebhookEventType = "rcs_brand.verified"
	WebhookEventRcsBrandFailed             WebhookEventType = "rcs_brand.failed"
	WebhookEventRcsAgentTesting            WebhookEventType = "rcs_agent.testing"
	WebhookEventRcsAgentLive               WebhookEventType = "rcs_agent.live"
	WebhookEventRcsAgentRejected           WebhookEventType = "rcs_agent.rejected"
	WebhookEventRcsAgentActionRequired     WebhookEventType = "rcs_agent.action_required"
	WebhookEventPortCompleted              WebhookEventType = "port.completed"
	WebhookEventPortOutRequested           WebhookEventType = "port_out.requested"
	WebhookEventPortOutCompleted           WebhookEventType = "port_out.completed"
	WebhookEventPortOutRejected            WebhookEventType = "port_out.rejected"
	WebhookEventPortOutCancelled           WebhookEventType = "port_out.cancelled"
	WebhookEventNumberActivated            WebhookEventType = "number.activated"
	WebhookEventNumberFailed               WebhookEventType = "number.failed"
	WebhookEventNumberRequirementsRequired WebhookEventType = "number.requirements_required"
	WebhookEventNumberReleased             WebhookEventType = "number.released"
	WebhookEventWhatsappAccountConnected   WebhookEventType = "whatsapp_account.connected"
	WebhookEventWhatsappAccountFailed      WebhookEventType = "whatsapp_account.failed"
	WebhookEventWhatsappTemplateApproved   WebhookEventType = "whatsapp_template.approved"
	WebhookEventWhatsappTemplateRejected   WebhookEventType = "whatsapp_template.rejected"
	WebhookEventWhatsappTemplatePaused     WebhookEventType = "whatsapp_template.paused"
	WebhookEventCallStarted                WebhookEventType = "call.started"
	WebhookEventCallCompleted              WebhookEventType = "call.completed"
	WebhookEventCallRecordingReady         WebhookEventType = "call.recording.ready"

	signatureToleranceSeconds = 300
)

// ListHealthEventSource is the source of a list-health event. Frozen enum —
// new values will be added in minor SDK versions, never removed.
type ListHealthEventSource string

const (
	ListHealthSourceSendFailure   ListHealthEventSource = "send_failure"
	ListHealthSourceCarrierLookup ListHealthEventSource = "carrier_lookup"
	ListHealthSourceUserAction    ListHealthEventSource = "user_action"
	ListHealthSourceBulkMarkValid ListHealthEventSource = "bulk_mark_valid"
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
	ID             string                 `json:"id"`
	Status         WebhookMessageStatus   `json:"status"`
	To             string                 `json:"to"`
	From           string                 `json:"from"`
	Direction      string                 `json:"direction,omitempty"`
	OrganizationID *string                `json:"organization_id,omitempty"`
	Text           string                 `json:"text,omitempty"`
	Error          string                 `json:"error,omitempty"`
	ErrorCode      string                 `json:"error_code,omitempty"`
	DeliveredAt    interface{}            `json:"delivered_at,omitempty"`
	FailedAt       interface{}            `json:"failed_at,omitempty"`
	CreatedAt      interface{}            `json:"created_at,omitempty"`
	Segments       int                    `json:"segments"`
	CreditsUsed    int                    `json:"credits_used"`
	MessageFormat  string                 `json:"message_format,omitempty"`
	MediaUrls      []string               `json:"media_urls,omitempty"`
	RetryCount     int                    `json:"retry_count,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	BatchID        *string                `json:"batch_id,omitempty"`
}

type WebhookVerificationData struct {
	ID             string                 `json:"id"`
	OrganizationID *string                `json:"organization_id,omitempty"`
	Phone          string                 `json:"phone"`
	Status         string                 `json:"status"`
	DeliveryStatus string                 `json:"delivery_status"`
	Attempts       int                    `json:"attempts"`
	MaxAttempts    int                    `json:"max_attempts"`
	ExpiresAt      interface{}            `json:"expires_at,omitempty"`
	VerifiedAt     interface{}            `json:"verified_at,omitempty"`
	CreatedAt      interface{}            `json:"created_at,omitempty"`
	AppName        string                 `json:"app_name,omitempty"`
	TemplateID     string                 `json:"template_id,omitempty"`
	ProfileID      string                 `json:"profile_id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
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
	ID   string           `json:"id"`
	Type WebhookEventType `json:"type"`
	// Data is the event's data.object decoded as a message. It is only
	// meaningful for message.* events. Lifecycle events — rcs_*, whatsapp_*,
	// call.*, brand.*, campaign.*, assignment.*, number.* and port* — carry a
	// completely different object, and decoding those into this struct leaves
	// it zeroed. Use RawObject or DecodeObject for them.
	Data       WebhookMessageData `json:"-"`
	Created    interface{}        `json:"created"`
	APIVersion string             `json:"api_version"`
	Livemode   bool               `json:"livemode"`
	// RawObject is the event's data.object exactly as it arrived. Every event
	// type has one, so this is always the complete payload even when Data is not
	// the right shape for it.
	RawObject json.RawMessage `json:"-"`
}

// DecodeObject unmarshals the event's data.object into v.
//
// Use it for lifecycle events, whose payload is not message-shaped:
//
//	var agent struct {
//		AgentID string `json:"agent_id"`
//		Name    string `json:"name"`
//		Stage   string `json:"stage"`
//	}
//	if err := event.DecodeObject(&agent); err != nil { /* ... */ }
func (e *WebhookEvent) DecodeObject(v interface{}) error {
	if len(e.RawObject) == 0 {
		return errors.New("sendly: event carries no data.object")
	}
	return json.Unmarshal(e.RawObject, v)
}

// isMessageEvent reports whether an event's data.object is message-shaped.
//
// Only message.* events are, and not even all of those: message.opt_in and
// message.opt_out share the prefix but carry an opt-out record
// ({phone_number, keyword, from_number, timestamp}), so decoding them as a
// message would invent to/from/segments values the server never sent. The
// Python, Ruby and PHP SDKs draw the same line.
func isMessageEvent(t WebhookEventType) bool {
	s := string(t)
	if s == string(WebhookEventMessageOptIn) || s == string(WebhookEventMessageOptOut) {
		return false
	}
	return strings.HasPrefix(s, "message.")
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
	var rawObject json.RawMessage
	var wrapper rawDataWrapper
	if err := json.Unmarshal(raw.Data, &wrapper); err == nil && wrapper.Object != nil {
		rawObject = wrapper.Object
		// Only decode the message view for events that actually carry one, and
		// never fail the whole parse if it does not fit. A lifecycle payload
		// that happens to reuse a message field name at a different type — a
		// nested `status` object, a numeric `id` — used to make the entire
		// event unreadable, RawObject included, which defeated the point of
		// adding RawObject at all.
		if isMessageEvent(raw.Type) {
			// Strict for a real message: a message event whose object does not
			// decode is a genuine problem and should surface, exactly as it did
			// before. Only lifecycle events are exempt, and they are exempt by
			// never being decoded at all rather than by ignoring the error.
			if err := json.Unmarshal(wrapper.Object, &msgData); err != nil {
				return nil, fmt.Errorf("failed to parse webhook data.object: %w", err)
			}
		}
	} else {
		rawObject = raw.Data
		if isMessageEvent(raw.Type) {
			if err := json.Unmarshal(raw.Data, &msgData); err != nil {
				return nil, fmt.Errorf("failed to parse webhook data: %w", err)
			}
			// Inside the message gate. Outside it, this filled a NON-message
			// event's ID from `message_id` — which on contact.auto_flagged is a
			// different row entirely, reintroducing the wrong-record bug this
			// release exists to fix.
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
	}

	created := raw.Created
	if created == nil {
		created = raw.CreatedAt
	}

	return &WebhookEvent{
		ID:         raw.ID,
		Type:       raw.Type,
		Data:       msgData,
		RawObject:  rawObject,
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
