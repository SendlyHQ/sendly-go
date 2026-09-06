package sendly

import "encoding/json"

// Message represents an SMS message.
type Message struct {
	// ID is the unique message identifier.
	ID string `json:"id"`
	// To is the recipient phone number in E.164 format.
	To string `json:"to"`
	// From is the sender ID or phone number.
	From string `json:"from,omitempty"`
	// Text is the message content.
	Text string `json:"text"`
	// Status is the delivery status.
	Status MessageStatus `json:"status"`
	// Direction is the message direction (outbound or inbound).
	Direction string `json:"direction,omitempty"`
	// Error contains error message if delivery failed.
	Error *string `json:"error,omitempty"`
	// Segments is the number of SMS segments.
	Segments int `json:"segments,omitempty"`
	// CreditsUsed is the number of credits consumed.
	CreditsUsed int `json:"creditsUsed,omitempty"`
	// IsSandbox indicates if the message was sent in sandbox mode.
	IsSandbox bool `json:"isSandbox,omitempty"`
	// SenderType indicates how the message was sent (number_pool, alphanumeric, sandbox).
	SenderType string `json:"senderType,omitempty"`
	// TelnyxMessageID is the carrier message ID for tracking.
	TelnyxMessageID *string `json:"telnyxMessageId,omitempty"`
	// Warning contains a warning message (e.g., when 'from' is ignored).
	Warning *string `json:"warning,omitempty"`
	// SenderNote contains a note about sender behavior.
	SenderNote *string `json:"senderNote,omitempty"`
	// CreatedAt is when the message was created.
	CreatedAt string `json:"createdAt,omitempty"`
	// DeliveredAt is when the message was delivered (if applicable).
	DeliveredAt *string `json:"deliveredAt,omitempty"`
	// ErrorCode is the error code if delivery failed.
	ErrorCode *string `json:"errorCode,omitempty"`
	// RetryCount is the number of delivery retry attempts.
	RetryCount int `json:"retryCount,omitempty"`
	// Metadata contains custom user-provided metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// AiMetadata contains AI classification metadata for inbound messages.
	AiMetadata *AiMetadata `json:"aiMetadata,omitempty"`
}

// AiMetadata contains AI classification results for an inbound message.
type AiMetadata struct {
	// Intent is the classified intent of the message.
	Intent string `json:"intent"`
	// IntentConfidence is the confidence score for the intent (0-1).
	IntentConfidence float64 `json:"intentConfidence"`
	// Sentiment is the classified sentiment of the message.
	Sentiment string `json:"sentiment"`
	// SentimentConfidence is the confidence score for the sentiment (0-1).
	SentimentConfidence float64 `json:"sentimentConfidence"`
	// ClassifiedAt is the ISO 8601 timestamp of when classification occurred.
	ClassifiedAt string `json:"classifiedAt"`
	// Model is the AI model used for classification.
	Model string `json:"model"`
}

// MessageStatus represents the status of a message.
type MessageStatus string

const (
	// MessageStatusQueued means the message is queued for delivery.
	MessageStatusQueued MessageStatus = "queued"
	// MessageStatusSent means the message was sent to the carrier.
	MessageStatusSent MessageStatus = "sent"
	// MessageStatusDelivered means the message was delivered.
	MessageStatusDelivered MessageStatus = "delivered"
	// MessageStatusRead means the recipient read the message. Read receipts
	// exist on RCS and WhatsApp only — SMS never reports one.
	MessageStatusRead MessageStatus = "read"
	// MessageStatusFailed means the message failed to deliver.
	MessageStatusFailed MessageStatus = "failed"
	// MessageStatusBounced means the message bounced (carrier rejected).
	MessageStatusBounced MessageStatus = "bounced"
	// MessageStatusRetrying means the message is being retried after a transient failure.
	MessageStatusRetrying MessageStatus = "retrying"
)

// SenderType indicates how a message was sent.
type SenderType string

const (
	// SenderTypeNumberPool means sent from toll-free number pool.
	SenderTypeNumberPool SenderType = "number_pool"
	// SenderTypeAlphanumeric means sent with alphanumeric sender ID.
	SenderTypeAlphanumeric SenderType = "alphanumeric"
	// SenderTypeSandbox means sent in sandbox/test mode.
	SenderTypeSandbox SenderType = "sandbox"
)

// MessageType represents the type of message for compliance.
type MessageType string

const (
	// MessageTypeMarketing is for promotional content (subject to quiet hours).
	MessageTypeMarketing MessageType = "marketing"
	// MessageTypeTransactional is for OTPs/confirmations (bypasses quiet hours).
	MessageTypeTransactional MessageType = "transactional"
)

// SendMessageRequest is the request to send a message.
type SendMessageRequest struct {
	// To is the recipient phone number in E.164 format (required).
	To string `json:"to"`
	// Text is the message content (required).
	Text string `json:"text"`
	// MessageType is the message type for compliance: "marketing" (default) or "transactional".
	MessageType MessageType `json:"messageType,omitempty"`
	// From is the sender phone number or alphanumeric sender ID (optional; defaults to the account's number).
	From string `json:"from,omitempty"`
	// MediaUrls is a list of media URLs to attach (converts message to MMS).
	MediaUrls []string `json:"mediaUrls,omitempty"`
	// Metadata is custom JSON metadata to attach to the message (max 4KB).
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// WhatsAppTemplateButtonVariables supplies variable values for one
// dynamic-URL button on an approved WhatsApp template.
type WhatsAppTemplateButtonVariables struct {
	// Index is the zero-based index of the button on the approved template.
	Index int `json:"index"`
	// Variables are values for the button's URL placeholders, keyed by placeholder number: {"1": "4821"}.
	Variables map[string]string `json:"variables"`
}

// WhatsAppTemplateSendParams is the approved WhatsApp template to send, with
// its variable values.
type WhatsAppTemplateSendParams struct {
	// Name is the template name as approved (e.g. "order_shipped").
	Name string `json:"name"`
	// Language is the template language code (e.g. "en_US") — must match the approved template's language exactly.
	Language string `json:"language"`
	// Variables are body variable values keyed by placeholder number: {"1": "Acme Inc", "2": "#4821"}.
	Variables map[string]string `json:"variables,omitempty"`
	// Buttons are variable values for dynamic-URL buttons.
	Buttons []WhatsAppTemplateButtonVariables `json:"buttons,omitempty"`
}

// SendWhatsAppMessageRequest is the request to send a WhatsApp message.
// Provide exactly one of:
//   - Text — free-form text; only deliverable inside an open 24-hour
//     customer-service window (the recipient messaged you in the last 24h)
//   - MediaUrls — a single media attachment (optional Text becomes its
//     caption); also window-bound
//   - Template — an approved template; works regardless of the window
//
// WhatsApp sends require a live API key and a From number that has been
// connected to WhatsApp (see WhatsAppSignupService).
type SendWhatsAppMessageRequest struct {
	// Channel selects the WhatsApp channel; SendWhatsApp sets it to "whatsapp" automatically.
	Channel string `json:"channel"`
	// To is the destination phone number in E.164 format (required).
	To string `json:"to"`
	// From is the sending number in E.164 format (required) — must be one of
	// your numbers with an active WhatsApp connection.
	From string `json:"from"`
	// Text is the free-form message text (max 4096 bytes), or the caption
	// when MediaUrls is provided (max 1024 bytes). Requires an open 24-hour
	// window — outside it the API rejects with whatsapp_window_closed; send
	// a Template instead.
	Text string `json:"text,omitempty"`
	// MediaUrls is the media attachment URL. WhatsApp accepts exactly one
	// per message. Must be a publicly accessible HTTPS URL.
	MediaUrls []string `json:"mediaUrls,omitempty"`
	// Template is the approved template to send. Works regardless of the 24-hour window.
	Template *WhatsAppTemplateSendParams `json:"template,omitempty"`
	// Metadata is custom JSON metadata to attach to the message (max 4KB).
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// WhatsAppSentTemplate is the template a WhatsApp message was sent with.
type WhatsAppSentTemplate struct {
	// Name is the template name.
	Name string `json:"name"`
	// Language is the template language code.
	Language string `json:"language"`
	// Category is the billed category: "marketing", "utility", or
	// "authentication" (Meta reviews and may reclassify templates; the
	// category on the send response is what was billed).
	Category string `json:"category"`
}

// WhatsAppMessageDetails contains the WhatsApp-specific details on a sent message.
type WhatsAppMessageDetails struct {
	// Kind is what was sent: "text", "media", or "template".
	Kind string `json:"kind"`
	// Template is the template that was sent (template sends only).
	Template *WhatsAppSentTemplate `json:"template,omitempty"`
	// MessageID is the WhatsApp message id — nil until the first delivery
	// report lands; populated on the message record afterwards.
	MessageID *string `json:"messageId,omitempty"`
}

// WhatsAppMessage is a sent WhatsApp message.
type WhatsAppMessage struct {
	// ID is the unique message identifier.
	ID string `json:"id"`
	// Channel is always "whatsapp".
	Channel string `json:"channel"`
	// MessageFormat is always "whatsapp".
	MessageFormat string `json:"message_format"`
	// To is the destination phone number.
	To string `json:"to"`
	// From is the sending number.
	From string `json:"from"`
	// Text is the body text for free-form text sends; nil for template and media sends.
	Text *string `json:"text,omitempty"`
	// Status is the delivery status.
	Status MessageStatus `json:"status"`
	// Segments is always 1 — WhatsApp has no segment concept.
	Segments int `json:"segments"`
	// CreditsUsed is the credits charged for this message (priced by
	// destination country and category).
	CreditsUsed int `json:"creditsUsed"`
	// WhatsApp contains the WhatsApp-specific details.
	WhatsApp WhatsAppMessageDetails `json:"whatsapp"`
	// CreatedAt is when the message was created.
	CreatedAt string `json:"createdAt"`
	// Metadata contains custom JSON metadata attached to the message.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RcsSuggestedReply is a tappable reply chip on an RCS message. Tapping it
// sends Text back, carrying PostbackData.
type RcsSuggestedReply struct {
	// Text is the chip label — what the recipient sees and sends back.
	Text string `json:"text"`
	// PostbackData is the machine-readable payload returned when the chip is tapped.
	PostbackData string `json:"postbackData"`
}

// RcsSuggestedAction is a tappable chip on an RCS message that opens a URL.
type RcsSuggestedAction struct {
	// Text is the chip label.
	Text string `json:"text"`
	// PostbackData is the machine-readable payload returned when the chip is tapped.
	PostbackData string `json:"postbackData"`
	// URL is the link opened when the chip is tapped.
	URL string `json:"url"`
}

// RcsSuggestion is a tappable chip on an RCS message. Set exactly one of
// Reply or Action.
type RcsSuggestion struct {
	// Reply sends the chip text back as a reply when tapped.
	Reply *RcsSuggestedReply `json:"reply,omitempty"`
	// Action opens a URL when tapped.
	Action *RcsSuggestedAction `json:"action,omitempty"`
}

// RcsCard is a standalone RCS rich card: a title and description with an
// optional image and tappable suggestions.
type RcsCard struct {
	// Title is the card title (required).
	Title string `json:"title"`
	// Description is the card body text (required).
	Description string `json:"description"`
	// MediaURL is a public JPEG, PNG, or GIF image URL shown on the card.
	MediaURL string `json:"mediaUrl,omitempty"`
	// Orientation is the card layout: "vertical" (default) or "horizontal".
	Orientation string `json:"orientation,omitempty"`
	// Suggestions are tappable chips on the card.
	Suggestions []RcsSuggestion `json:"suggestions,omitempty"`
}

// SendRcsMessageRequest is the request to send an RCS message.
// Provide exactly one of:
//   - Text — rich text; optional tappable Suggestions ride along
//   - Card — a standalone rich card (title and description, with an
//     optional image and card-level suggestions)
//
// When the recipient's device or network doesn't support RCS, text sends
// fall back to plain SMS (billed as SMS) unless FallbackToSms is false;
// suggestions have no SMS form and are dropped on the fallback. Card sends
// have no SMS form and never fall back.
//
// RCS sends require a live API key and a sendable RCS agent on the
// workspace (see RCSService).
type SendRcsMessageRequest struct {
	// Channel selects the RCS channel; SendRcs sets it to "rcs" automatically.
	Channel string `json:"channel"`
	// To is the destination phone number in E.164 format (required).
	To string `json:"to"`
	// AgentID is the RCS agent to send as; optional when the workspace has
	// exactly one agent.
	AgentID string `json:"agentId,omitempty"`
	// Text is the message text. Exactly one of Text or Card is required.
	Text string `json:"text,omitempty"`
	// Suggestions are tappable chips riding on a text message (not valid
	// with Card — put card buttons in Card.Suggestions instead).
	Suggestions []RcsSuggestion `json:"suggestions,omitempty"`
	// Card is the rich card to send. Exactly one of Text or Card is required.
	Card *RcsCard `json:"card,omitempty"`
	// FallbackToSms controls the SMS fallback for text sends when the
	// recipient isn't RCS-capable; defaults to true when omitted.
	FallbackToSms *bool `json:"fallbackToSms,omitempty"`
	// Metadata is custom JSON metadata to attach to the message (max 4KB).
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RcsMessageDetails contains the RCS-specific details on a sent message.
type RcsMessageDetails struct {
	// Kind is what was sent over RCS: "text" or "card". Present on messages
	// delivered over RCS; empty on SMS fallbacks.
	Kind string `json:"kind,omitempty"`
	// AgentID is the RCS agent the send was routed through.
	AgentID string `json:"agentId"`
	// AgentName is the agent name recipients see. Present on messages
	// delivered over RCS.
	AgentName string `json:"agentName,omitempty"`
	// RequestedChannel is "rcs" on SMS fallbacks — the channel that was
	// requested before the recipient's RCS support was probed.
	RequestedChannel string `json:"requestedChannel,omitempty"`
	// SuggestionsDropped is true when the message fell back to SMS and its
	// suggestions were dropped (suggestions have no SMS form).
	SuggestionsDropped bool `json:"suggestionsDropped,omitempty"`
}

// RcsMessage is a sent RCS message — or its SMS fallback. Channel reports
// the leg that actually delivered: check FellBackTo (or Channel) to see
// whether the message went out over RCS or fell back to SMS.
type RcsMessage struct {
	// ID is the unique message identifier.
	ID string `json:"id"`
	// Channel is "rcs" when delivered over RCS, or "sms" when the message
	// fell back to SMS for a recipient without RCS support.
	Channel string `json:"channel"`
	// FellBackTo is "sms" when the message fell back to SMS; empty on
	// messages delivered over RCS.
	FellBackTo string `json:"fellBackTo,omitempty"`
	// MessageFormat matches Channel: "rcs" or "sms".
	MessageFormat string `json:"message_format"`
	// To is the destination phone number.
	To string `json:"to"`
	// From is the RCS agent name, or the SMS sender on a fallback.
	From string `json:"from"`
	// Text is the message text for text sends; nil for card sends.
	Text *string `json:"text,omitempty"`
	// Status is the delivery status.
	Status MessageStatus `json:"status"`
	// Segments is 1 on RCS deliveries; the SMS segment count on a fallback.
	Segments int `json:"segments"`
	// CreditsUsed is the credits charged — RCS pricing on an RCS delivery,
	// SMS pricing on a fallback.
	CreditsUsed int `json:"creditsUsed"`
	// RCS contains the RCS-specific details.
	RCS RcsMessageDetails `json:"rcs"`
	// CreatedAt is when the message was created.
	CreatedAt string `json:"createdAt"`
	// Metadata contains custom JSON metadata attached to the message.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SendGroupMessageRequest is the request to send a group MMS to multiple
// recipients (US/Canada only). Group messaging is an A2P 10DLC capability:
// the sending number must be an MMS-enabled, 10DLC-registered number you own.
type SendGroupMessageRequest struct {
	// To is the list of 2-8 recipient phone numbers in E.164 format (US/CA only, required).
	To []string `json:"to"`
	// Text is the message content (required unless MediaUrls is provided).
	Text string `json:"text,omitempty"`
	// From is the sending number in E.164 (optional; omit to use the workspace default).
	From string `json:"from,omitempty"`
	// MediaUrls is a list of HTTPS media URLs to attach (required unless Text is provided).
	MediaUrls []string `json:"mediaUrls,omitempty"`
	// MessageType is the message type for compliance: "transactional" (default) or "marketing".
	MessageType MessageType `json:"messageType,omitempty"`
}

// GroupMessageResponse is the response from sending a group MMS.
type GroupMessageResponse struct {
	// ID is the message identifier (matches the id in delivery webhooks).
	ID string `json:"id"`
	// Status is the delivery status ("sent" on a live send, "delivered" when simulated).
	Status MessageStatus `json:"status"`
	// To is the recipients the group message was sent to.
	To []string `json:"to"`
	// GroupMessageID identifies the group conversation (present on live sends).
	GroupMessageID string `json:"group_message_id,omitempty"`
	// Simulated is true when the send was simulated (test key, or before domestic verification).
	Simulated bool `json:"simulated,omitempty"`
	// Message is a human-readable note, present on simulated sends.
	Message string `json:"message,omitempty"`
}

// EnhanceMessageRequest is the request to AI-enhance a draft message. Provide
// Text, MessageType, or both — at least one is required.
type EnhanceMessageRequest struct {
	// Text is the draft message text to rewrite (optional if MessageType is provided).
	Text string `json:"text,omitempty"`
	// MessageType steers the rewrite, e.g. "marketing" or "transactional" (optional if Text is provided).
	MessageType MessageType `json:"messageType,omitempty"`
}

// EnhanceMessageResponse is the result of an AI message enhancement.
type EnhanceMessageResponse struct {
	// Enhanced is the rewritten message, capped at 160 characters (one SMS segment).
	// When AI enhancement is unavailable, this falls back to the original text.
	Enhanced string `json:"enhanced"`
	// Explanation is a short description of what changed (empty on the fallback path).
	Explanation string `json:"explanation"`
	// Model is the AI model that produced the enhancement, when available.
	Model string `json:"model,omitempty"`
}

// MediaFile represents an uploaded media file.
type MediaFile struct {
	// ID is the unique media file identifier.
	ID string `json:"id"`
	// URL is the public URL of the uploaded file.
	URL string `json:"url"`
	// ContentType is the MIME type of the file.
	ContentType string `json:"contentType"`
	// SizeBytes is the file size in bytes.
	SizeBytes int64 `json:"sizeBytes"`
}

// SendMessageResponse is the response from sending a message.
// The API returns the message directly at the top level.
type SendMessageResponse Message

// ListMessagesRequest is the request to list messages.
type ListMessagesRequest struct {
	// Limit is the maximum number of messages to return (default: 20, max: 100).
	Limit int
	// Offset is the number of messages to skip.
	Offset int
	// Status filters by message status.
	Status MessageStatus
	// To filters by recipient phone number.
	To string
}

// ListMessagesResponse is the response from listing messages.
type ListMessagesResponse struct {
	// Data contains the list of messages.
	Data []Message `json:"data"`
	// Count is the total number of messages matching the query.
	Count int `json:"count"`
}

// APIFieldError points at one invalid field in a request. Path is the
// JSON path of the field, such as "brand.ein" or "devices.0.phoneNumber".
type APIFieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// APIError represents an error from the API.
type APIError struct {
	// Code is the error code.
	Code string `json:"code"`
	// Message is the error message.
	Message string `json:"message"`
	// Details contains additional error details.
	Details map[string]interface{} `json:"details,omitempty"`
	// Errors lists the invalid fields on a validation failure, when the API
	// reports them.
	Errors []APIFieldError `json:"errors,omitempty"`
}

// UnmarshalJSON reads the error code from either the "code" or the
// "error" key, since the API sends the latter, and keeps a per-field
// "errors" list when the API reports one.
func (e *APIError) UnmarshalJSON(data []byte) error {
	var raw struct {
		Code    string                 `json:"code"`
		Error   json.RawMessage        `json:"error"`
		Message string                 `json:"message"`
		Details map[string]interface{} `json:"details"`
		Errors  json.RawMessage        `json:"errors"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.Code = raw.Code
	e.Message = raw.Message
	e.Details = raw.Details
	e.Errors = nil
	if e.Code == "" && len(raw.Error) > 0 {
		var code string
		if json.Unmarshal(raw.Error, &code) == nil {
			e.Code = code
		}
	}
	if len(raw.Errors) > 0 {
		var fields []APIFieldError
		if json.Unmarshal(raw.Errors, &fields) == nil {
			e.Errors = fields
		}
	}
	return nil
}

// ScheduledMessageStatus represents the status of a scheduled message.
type ScheduledMessageStatus string

const (
	// ScheduledMessageStatusScheduled means the message is scheduled.
	ScheduledMessageStatusScheduled ScheduledMessageStatus = "scheduled"
	// ScheduledMessageStatusSent means the scheduled message was sent.
	ScheduledMessageStatusSent ScheduledMessageStatus = "sent"
	// ScheduledMessageStatusCancelled means the scheduled message was cancelled.
	ScheduledMessageStatusCancelled ScheduledMessageStatus = "cancelled"
	// ScheduledMessageStatusFailed means the scheduled message failed.
	ScheduledMessageStatusFailed ScheduledMessageStatus = "failed"
)

// ScheduledMessage represents a scheduled SMS message.
type ScheduledMessage struct {
	// ID is the unique scheduled message identifier.
	ID string `json:"id"`
	// To is the recipient phone number in E.164 format.
	To string `json:"to"`
	// From is the sender ID or phone number.
	From string `json:"from,omitempty"`
	// Text is the message content.
	Text string `json:"text"`
	// ScheduledAt is when the message is scheduled to be sent (ISO 8601).
	ScheduledAt string `json:"scheduledAt"`
	// Status is the scheduled message status.
	Status ScheduledMessageStatus `json:"status"`
	// CreditsReserved is the number of credits reserved for this message.
	CreditsReserved int `json:"creditsReserved,omitempty"`
	// CreatedAt is when the scheduled message was created.
	CreatedAt string `json:"createdAt,omitempty"`
	// SentAt is when the message was actually sent.
	SentAt *string `json:"sentAt,omitempty"`
	// CancelledAt is when the message was cancelled.
	CancelledAt *string `json:"cancelledAt,omitempty"`
	// MessageID is the ID of the sent message (after sending).
	MessageID *string `json:"messageId,omitempty"`
}

// ScheduleMessageRequest is the request to schedule a message.
type ScheduleMessageRequest struct {
	// To is the recipient phone number in E.164 format (required).
	To string `json:"to"`
	// Text is the message content (required).
	Text string `json:"text"`
	// ScheduledAt is when to send the message in ISO 8601 format (required).
	ScheduledAt string `json:"scheduledAt"`
	// From is the sender ID or phone number (optional).
	From string `json:"from,omitempty"`
	// MessageType is the message type for compliance: "marketing" (default) or "transactional".
	MessageType MessageType `json:"messageType,omitempty"`
	// Metadata is custom JSON metadata to attach to the message (max 4KB).
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ListScheduledMessagesRequest is the request to list scheduled messages.
type ListScheduledMessagesRequest struct {
	// Limit is the maximum number of messages to return (default: 20, max: 100).
	Limit int
	// Offset is the number of messages to skip.
	Offset int
	// Status filters by scheduled message status.
	Status ScheduledMessageStatus
}

// ListScheduledMessagesResponse is the response from listing scheduled messages.
type ListScheduledMessagesResponse struct {
	// Data contains the list of scheduled messages.
	Data []ScheduledMessage `json:"data"`
	// Count is the total number of scheduled messages.
	Count int `json:"count"`
}

// CancelScheduledMessageResponse is the response from cancelling a scheduled message.
type CancelScheduledMessageResponse struct {
	// ID is the scheduled message ID.
	ID string `json:"id"`
	// Status is the new status (cancelled).
	Status ScheduledMessageStatus `json:"status"`
	// CreditsRefunded is the number of credits refunded.
	CreditsRefunded int `json:"creditsRefunded"`
}

// BatchMessageItem represents a single message in a batch request.
type BatchMessageItem struct {
	// To is the recipient phone number in E.164 format (required).
	To string `json:"to"`
	// Text is the message content (required).
	Text string `json:"text"`
	// Metadata is custom JSON metadata for this message (max 4KB, merged with batch metadata).
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SendBatchRequest is the request to send batch messages.
type SendBatchRequest struct {
	// Messages is the list of messages to send (required).
	Messages []BatchMessageItem `json:"messages"`
	// From is the sender ID or phone number (optional, applies to all).
	From string `json:"from,omitempty"`
	// MessageType is the message type for compliance: "marketing" (default) or "transactional".
	MessageType MessageType `json:"messageType,omitempty"`
	// Metadata is custom JSON metadata to attach to all messages in the batch (max 4KB).
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BatchStatus represents the status of a batch.
type BatchStatus string

const (
	// BatchStatusProcessing means the batch is being processed.
	BatchStatusProcessing BatchStatus = "processing"
	// BatchStatusCompleted means the batch has been completed.
	BatchStatusCompleted BatchStatus = "completed"
	// BatchStatusPartialFailure means some messages in the batch failed.
	BatchStatusPartialFailure BatchStatus = "partial_failure"
	// BatchStatusFailed means the batch failed.
	BatchStatusFailed BatchStatus = "failed"
)

// BatchMessageResult represents the result of a single message in a batch.
type BatchMessageResult struct {
	// ID is the message ID.
	ID string `json:"id"`
	// To is the recipient phone number.
	To string `json:"to"`
	// Status is the message status.
	Status string `json:"status"`
	// Error is the error message if failed.
	Error *string `json:"error,omitempty"`
	// CreatedAt is when the message was created.
	CreatedAt *string `json:"createdAt,omitempty"`
	// DeliveredAt is when the message was delivered.
	DeliveredAt *string `json:"deliveredAt,omitempty"`
}

// BatchMessageResponse represents the response from sending batch messages.
type BatchMessageResponse struct {
	// BatchID is the unique batch identifier.
	BatchID string `json:"batchId"`
	// Status is the batch status.
	Status BatchStatus `json:"status"`
	// Total is the total number of messages in the batch.
	Total int `json:"total"`
	// Queued is the number of messages queued.
	Queued int `json:"queued"`
	// Sent is the number of messages sent.
	Sent int `json:"sent"`
	// Failed is the number of messages that failed.
	Failed int `json:"failed"`
	// CreditsUsed is the total credits used.
	CreditsUsed int `json:"creditsUsed"`
	// Messages contains the results for each message.
	Messages []BatchMessageResult `json:"messages,omitempty"`
	// CreatedAt is when the batch was created.
	CreatedAt string `json:"createdAt,omitempty"`
	// CompletedAt is when the batch completed.
	CompletedAt *string `json:"completedAt,omitempty"`
}

// ListBatchesRequest is the request to list batches.
type ListBatchesRequest struct {
	// Limit is the maximum number of batches to return (default: 20, max: 100).
	Limit int
	// Offset is the number of batches to skip.
	Offset int
	// Status filters by batch status.
	Status BatchStatus
}

// ListBatchesResponse is the response from listing batches.
type ListBatchesResponse struct {
	// Data contains the list of batches.
	Data []BatchMessageResponse `json:"data"`
	// Count is the total number of batches.
	Count int `json:"count"`
}

// BatchPreviewItem represents a single message in a batch preview.
type BatchPreviewItem struct {
	// To is the recipient phone number.
	To string `json:"to"`
	// Text is the message content.
	Text string `json:"text"`
	// Segments is the number of SMS segments.
	Segments int `json:"segments"`
	// Credits is the credits needed for this message.
	Credits int `json:"credits"`
	// CanSend indicates if this message can be sent.
	CanSend bool `json:"canSend"`
	// BlockReason is the reason if message is blocked.
	BlockReason *string `json:"blockReason,omitempty"`
	// Country is the destination country code.
	Country *string `json:"country,omitempty"`
	// PricingTier is the pricing tier for this message.
	PricingTier *string `json:"pricingTier,omitempty"`
}

// BatchPreviewResponse is the response from previewing a batch.
type BatchPreviewResponse struct {
	// CanSend indicates if the entire batch can be sent.
	CanSend bool `json:"canSend"`
	// TotalMessages is the total number of messages.
	TotalMessages int `json:"totalMessages"`
	// WillSend is the number of messages that will be sent.
	WillSend int `json:"willSend"`
	// Blocked is the number of messages that are blocked.
	Blocked int `json:"blocked"`
	// CreditsNeeded is the total credits needed.
	CreditsNeeded int `json:"creditsNeeded"`
	// CurrentBalance is the current credit balance.
	CurrentBalance int `json:"currentBalance"`
	// HasEnoughCredits indicates if there are enough credits.
	HasEnoughCredits bool `json:"hasEnoughCredits"`
	// Messages contains the preview for each message.
	Messages []BatchPreviewItem `json:"messages"`
	// BlockReasons is a count of block reasons.
	BlockReasons map[string]int `json:"blockReasons,omitempty"`
}

// ============================================================================
// Webhooks
// ============================================================================

// WebhookMode represents the webhook event mode filter.
type WebhookMode string

const (
	// WebhookModeAll receives both test and live events.
	WebhookModeAll WebhookMode = "all"
	// WebhookModeTest receives only sandbox/test events.
	WebhookModeTest WebhookMode = "test"
	// WebhookModeLive receives only production events (requires verification).
	WebhookModeLive WebhookMode = "live"
)

// CircuitState represents the circuit breaker state.
type CircuitState string

const (
	CircuitStateClosed   CircuitState = "closed"
	CircuitStateOpen     CircuitState = "open"
	CircuitStateHalfOpen CircuitState = "half_open"
)

// DeliveryStatus represents the webhook delivery status.
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusCancelled DeliveryStatus = "cancelled"
)

// Webhook represents a configured webhook endpoint.
type Webhook struct {
	// ID is the unique webhook identifier (whk_xxx).
	ID string `json:"id"`
	// URL is the HTTPS endpoint URL.
	URL string `json:"url"`
	// Events is the list of subscribed event types.
	Events []string `json:"events"`
	// Description is an optional description.
	Description *string `json:"description,omitempty"`
	// Mode is the event mode filter (all, test, live).
	Mode WebhookMode `json:"mode"`
	// IsActive indicates whether the webhook is active.
	IsActive bool `json:"isActive"`
	// FailureCount is the number of consecutive failures.
	FailureCount int `json:"failureCount"`
	// LastFailureAt is when the last failure occurred.
	LastFailureAt *string `json:"lastFailureAt,omitempty"`
	// CircuitState is the circuit breaker state.
	CircuitState CircuitState `json:"circuitState"`
	// CircuitOpenedAt is when the circuit was opened.
	CircuitOpenedAt *string `json:"circuitOpenedAt,omitempty"`
	// APIVersion is the API version for payloads.
	APIVersion string `json:"apiVersion"`
	// Metadata is custom metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// CreatedAt is when the webhook was created.
	CreatedAt string `json:"createdAt"`
	// UpdatedAt is when the webhook was last updated.
	UpdatedAt string `json:"updatedAt"`
	// TotalDeliveries is the total number of delivery attempts.
	TotalDeliveries int `json:"totalDeliveries"`
	// SuccessfulDeliveries is the number of successful deliveries.
	SuccessfulDeliveries int `json:"successfulDeliveries"`
	// SuccessRate is the success rate (0-100).
	SuccessRate float64 `json:"successRate"`
	// LastDeliveryAt is when the last successful delivery occurred.
	LastDeliveryAt *string `json:"lastDeliveryAt,omitempty"`
}

// WebhookCreatedResponse is returned when creating a webhook.
type WebhookCreatedResponse struct {
	Webhook
	// Secret is the webhook signing secret (only shown once!).
	Secret string `json:"secret"`
}

// CreateWebhookRequest is the request to create a webhook.
type CreateWebhookRequest struct {
	// URL is the HTTPS endpoint URL (required).
	URL string `json:"url"`
	// Events is the list of event types to subscribe to (required).
	Events []string `json:"events"`
	// Description is an optional description.
	Description string `json:"description,omitempty"`
	// Mode is the event mode filter (all, test, live). Live requires verification.
	Mode WebhookMode `json:"mode,omitempty"`
	// Metadata is optional custom metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateWebhookRequest is the request to update a webhook.
type UpdateWebhookRequest struct {
	// URL is the new URL.
	URL *string `json:"url,omitempty"`
	// Events is the new event subscriptions.
	Events []string `json:"events,omitempty"`
	// Description is the new description.
	Description *string `json:"description,omitempty"`
	// IsActive enables/disables the webhook.
	IsActive *bool `json:"is_active,omitempty"`
	// Mode is the event mode filter (all, test, live).
	Mode *WebhookMode `json:"mode,omitempty"`
	// Metadata is the new custom metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// WebhookDelivery represents a webhook delivery attempt.
type WebhookDelivery struct {
	// ID is the unique delivery identifier (del_xxx).
	ID string `json:"id"`
	// WebhookID is the webhook ID.
	WebhookID string `json:"webhookId"`
	// EventID is the event ID for idempotency.
	EventID string `json:"eventId"`
	// EventType is the event type.
	EventType string `json:"eventType"`
	// AttemptNumber is the attempt number (1-6).
	AttemptNumber int `json:"attemptNumber"`
	// MaxAttempts is the maximum number of attempts.
	MaxAttempts int `json:"maxAttempts"`
	// Status is the delivery status.
	Status DeliveryStatus `json:"status"`
	// ResponseStatusCode is the HTTP response status code.
	ResponseStatusCode *int `json:"responseStatusCode,omitempty"`
	// ResponseTimeMs is the response time in milliseconds.
	ResponseTimeMs *int `json:"responseTimeMs,omitempty"`
	// ErrorMessage is the error message if failed.
	ErrorMessage *string `json:"errorMessage,omitempty"`
	// ErrorCode is the error code if failed.
	ErrorCode *string `json:"errorCode,omitempty"`
	// NextRetryAt is when the next retry will occur.
	NextRetryAt *string `json:"nextRetryAt,omitempty"`
	// CreatedAt is when the delivery was created.
	CreatedAt string `json:"createdAt"`
	// DeliveredAt is when the delivery succeeded.
	DeliveredAt *string `json:"deliveredAt,omitempty"`
}

// WebhookTestResult is the result of testing a webhook.
type WebhookTestResult struct {
	// Success indicates whether the test was successful.
	Success bool `json:"success"`
	// StatusCode is the HTTP response status code.
	StatusCode *int `json:"statusCode,omitempty"`
	// ResponseTimeMs is the response time in milliseconds.
	ResponseTimeMs *int `json:"responseTimeMs,omitempty"`
	// Error is the error message if failed.
	Error *string `json:"error,omitempty"`
}

// WebhookSecretRotation is the response from rotating a webhook secret.
type WebhookSecretRotation struct {
	// Webhook is the updated webhook.
	Webhook Webhook `json:"webhook"`
	// NewSecret is the new signing secret.
	NewSecret string `json:"newSecret"`
	// OldSecretExpiresAt is when the old secret expires.
	OldSecretExpiresAt string `json:"oldSecretExpiresAt"`
	// Message is information about the grace period.
	Message string `json:"message"`
}

// ============================================================================
// Account & Credits
// ============================================================================

// Account represents account information.
type Account struct {
	// ID is the user ID.
	ID string `json:"id"`
	// Email is the email address.
	Email string `json:"email"`
	// Name is the display name. Always nil: the account payload carries no
	// display name.
	//
	// Deprecated: use Email to identify the account.
	Name *string `json:"name,omitempty"`
	// CreatedAt is when the account was created.
	CreatedAt string `json:"createdAt"`
}

// Credits represents credit balance information.
type Credits struct {
	// Balance is the available credit balance.
	Balance int `json:"balance"`
	// ReservedBalance is credits reserved for scheduled messages.
	ReservedBalance int `json:"reservedBalance"`
	// AvailableBalance is the total usable credits.
	AvailableBalance int `json:"availableBalance"`
}

// TransactionType represents a credit transaction type.
type TransactionType string

const (
	TransactionTypePurchase   TransactionType = "purchase"
	TransactionTypeUsage      TransactionType = "usage"
	TransactionTypeRefund     TransactionType = "refund"
	TransactionTypeAdjustment TransactionType = "adjustment"
	TransactionTypeBonus      TransactionType = "bonus"
)

// CreditTransaction represents a credit transaction record.
type CreditTransaction struct {
	// ID is the transaction ID.
	ID string `json:"id"`
	// Type is the transaction type.
	Type TransactionType `json:"type"`
	// Amount is the amount (positive for in, negative for out).
	Amount int `json:"amount"`
	// BalanceAfter is the balance after the transaction.
	BalanceAfter int `json:"balanceAfter"`
	// Description is the transaction description.
	Description string `json:"description"`
	// MessageID is the related message ID (for usage transactions).
	MessageID *string `json:"messageId,omitempty"`
	// CreatedAt is when the transaction occurred.
	CreatedAt string `json:"createdAt"`
}

// APIKey represents an API key.
type APIKey struct {
	// ID is the key ID.
	ID string `json:"id"`
	// Name is the key name/label.
	Name string `json:"name"`
	// Type is the key type (test or live).
	Type string `json:"type"`
	// Prefix is the key prefix for identification.
	Prefix string `json:"prefix"`
	// LastFour is the last 4 characters of the key. Always empty: the key
	// payload carries only the leading prefix, never the tail.
	//
	// Deprecated: use Prefix to identify a key.
	LastFour string `json:"lastFour"`
	// Permissions is the list of permissions granted.
	Permissions []string `json:"permissions"`
	// CreatedAt is when the key was created.
	CreatedAt string `json:"createdAt"`
	// LastUsedAt is when the key was last used.
	LastUsedAt *string `json:"lastUsedAt,omitempty"`
	// ExpiresAt is when the key expires.
	ExpiresAt *string `json:"expiresAt,omitempty"`
	// IsRevoked indicates whether the key is revoked.
	IsRevoked bool `json:"isRevoked"`
}

// ============================================================================
// Enterprise
// ============================================================================

type EnterpriseWorkspaceSummary struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Slug               string  `json:"slug"`
	VerificationStatus *string `json:"verificationStatus,omitempty"`
	VerificationType   *string `json:"verificationType,omitempty"`
	TollFreeNumber     *string `json:"tollFreeNumber,omitempty"`
	CreditBalance      int     `json:"creditBalance"`
}

type EnterpriseAccount struct {
	ID             string                       `json:"id"`
	MaxWorkspaces  int                          `json:"maxWorkspaces"`
	WorkspaceCount int                          `json:"workspaceCount"`
	Workspaces     []EnterpriseWorkspaceSummary `json:"workspaces"`
	Metadata       map[string]interface{}       `json:"metadata,omitempty"`
}

type EnterpriseWorkspace struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"createdAt"`
}

type WorkspaceVerificationInfo struct {
	Status         string  `json:"status"`
	Type           string  `json:"type"`
	TollFreeNumber *string `json:"tollFreeNumber,omitempty"`
	BusinessName   string  `json:"businessName,omitempty"`
}

type WorkspaceDetail struct {
	ID           string                     `json:"id"`
	Name         string                     `json:"name"`
	Slug         string                     `json:"slug"`
	CreatedAt    string                     `json:"createdAt"`
	Verification *WorkspaceVerificationInfo `json:"verification,omitempty"`
	Credits      int                        `json:"credits"`
	KeyCount     int                        `json:"keyCount"`
}

type SubmitVerificationAddress struct {
	Street   string `json:"street"`
	Address2 string `json:"address2,omitempty"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Country  string `json:"country,omitempty"`
}

type SubmitVerificationContact struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

type SubmitVerificationRequest struct {
	BusinessName          string                    `json:"businessName"`
	DoingBusinessAs       string                    `json:"doingBusinessAs,omitempty"`
	Website               string                    `json:"website"`
	Address               SubmitVerificationAddress `json:"address"`
	Contact               SubmitVerificationContact `json:"contact"`
	BRN                   string                    `json:"brn,omitempty"`
	BRNType               string                    `json:"brnType,omitempty"`
	BRNCountry            string                    `json:"brnCountry,omitempty"`
	UseCase               string                    `json:"useCase"`
	UseCaseSummary        string                    `json:"useCaseSummary"`
	SampleMessages        string                    `json:"sampleMessages"`
	OptInWorkflow         string                    `json:"optInWorkflow"`
	OptInImageUrls        string                    `json:"optInImageUrls,omitempty"`
	MonthlyVolume         string                    `json:"monthlyVolume,omitempty"`
	AdditionalInformation string                    `json:"additionalInformation,omitempty"`
	AgeGatedContent       bool                      `json:"ageGatedContent,omitempty"`
	ISVReseller           bool                      `json:"isvReseller,omitempty"`
	PrivacyURL            string                    `json:"privacyUrl,omitempty"`
	TermsURL              string                    `json:"termsUrl,omitempty"`
	EntityType            string                    `json:"entityType,omitempty"`
}

// VerificationAddressInput is the address payload for VerificationSubmitInput.
// All fields are pointers so unset values are omitted from the JSON body,
// allowing partial-update resubmits to leave existing values untouched.
type VerificationAddressInput struct {
	Street   *string `json:"street,omitempty"`
	Address1 *string `json:"address1,omitempty"`
	Address2 *string `json:"address2,omitempty"`
	City     *string `json:"city,omitempty"`
	State    *string `json:"state,omitempty"`
	Zip      *string `json:"zip,omitempty"`
	Country  *string `json:"country,omitempty"`
}

// VerificationContactInput is the contact payload for VerificationSubmitInput.
type VerificationContactInput struct {
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
}

// VerificationSubmitInput is the payload for
// WorkspacesService.SubmitVerification and ResubmitVerification. All fields
// are pointers so unset values are omitted from the JSON body — for partial
// resubmits on an existing workspace, the server merges with the existing
// record. For initial submission the server requires at minimum:
// BusinessName, Website, Address, Contact, UseCase, UseCaseSummary,
// SampleMessages, OptInWorkflow.
//
// For sole proprietors, leave BRN, BRNType, BRNCountry as nil — the server
// strips them before forwarding to the carrier.
type VerificationSubmitInput struct {
	BusinessName          *string                   `json:"businessName,omitempty"`
	DoingBusinessAs       *string                   `json:"doingBusinessAs,omitempty"`
	Website               *string                   `json:"website,omitempty"`
	Address               *VerificationAddressInput `json:"address,omitempty"`
	Contact               *VerificationContactInput `json:"contact,omitempty"`
	BRN                   *string                   `json:"brn,omitempty"`
	BRNType               *string                   `json:"brnType,omitempty"`
	BRNCountry            *string                   `json:"brnCountry,omitempty"`
	EntityType            *string                   `json:"entityType,omitempty"`
	UseCase               *string                   `json:"useCase,omitempty"`
	UseCaseSummary        *string                   `json:"useCaseSummary,omitempty"`
	SampleMessages        *string                   `json:"sampleMessages,omitempty"`
	OptInWorkflow         *string                   `json:"optInWorkflow,omitempty"`
	OptInImageUrls        *string                   `json:"optInImageUrls,omitempty"`
	MonthlyVolume         *string                   `json:"monthlyVolume,omitempty"`
	AdditionalInformation *string                   `json:"additionalInformation,omitempty"`
	AgeGatedContent       *bool                     `json:"ageGatedContent,omitempty"`
	ISVReseller           *string                   `json:"isvReseller,omitempty"`
	PrivacyURL            *string                   `json:"privacyUrl,omitempty"`
	TermsURL              *string                   `json:"termsUrl,omitempty"`
}

type VerificationResponse struct {
	VerificationID  string  `json:"verificationId"`
	Status          string  `json:"status"`
	TollFreeNumber  *string `json:"tollFreeNumber,omitempty"`
	BusinessName    string  `json:"businessName"`
	TelnyxProfileID string  `json:"telnyxProfileId,omitempty"`
}

type InheritVerificationResponse struct {
	VerificationID string  `json:"verificationId"`
	Status         string  `json:"status"`
	Type           string  `json:"type"`
	TollFreeNumber *string `json:"tollFreeNumber,omitempty"`
	InheritedFrom  string  `json:"inheritedFrom"`
}

type VerificationStatusResponse struct {
	VerificationID string  `json:"verificationId,omitempty"`
	Status         string  `json:"status"`
	Type           string  `json:"type,omitempty"`
	TollFreeNumber *string `json:"tollFreeNumber,omitempty"`
	BusinessName   string  `json:"businessName,omitempty"`
	SubmittedAt    string  `json:"submittedAt,omitempty"`
}

type TransferCreditsResult struct {
	Success       bool `json:"success"`
	Amount        int  `json:"amount"`
	SourceBalance int  `json:"sourceBalance"`
	TargetBalance int  `json:"targetBalance"`
}

type WorkspaceCredits struct {
	Balance         int `json:"balance"`
	LifetimeCredits int `json:"lifetimeCredits"`
}

type PoolCredits struct {
	Balance         int `json:"balance"`
	LifetimeCredits int `json:"lifetimeCredits"`
	ReservedBalance int `json:"reservedBalance"`
}

type WorkspaceKey struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	KeyPrefix  string   `json:"keyPrefix"`
	Type       string   `json:"type"`
	Scopes     []string `json:"scopes"`
	LastUsedAt *string  `json:"lastUsedAt,omitempty"`
	CreatedAt  string   `json:"createdAt"`
}

type WorkspaceKeyCreated struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Key       string   `json:"key"`
	KeyPrefix string   `json:"keyPrefix"`
	Type      string   `json:"type"`
	Scopes    []string `json:"scopes"`
	CreatedAt string   `json:"createdAt"`
}

type ProvisionVerificationData struct {
	BusinessName          string                    `json:"businessName"`
	DoingBusinessAs       string                    `json:"doingBusinessAs,omitempty"`
	Website               string                    `json:"website"`
	Address               SubmitVerificationAddress `json:"address"`
	Contact               SubmitVerificationContact `json:"contact"`
	BRN                   string                    `json:"brn,omitempty"`
	BRNType               string                    `json:"brnType,omitempty"`
	BRNCountry            string                    `json:"brnCountry,omitempty"`
	UseCase               string                    `json:"useCase"`
	UseCaseSummary        string                    `json:"useCaseSummary"`
	SampleMessages        string                    `json:"sampleMessages"`
	OptInWorkflow         string                    `json:"optInWorkflow"`
	OptInImageUrls        string                    `json:"optInImageUrls,omitempty"`
	MonthlyVolume         string                    `json:"monthlyVolume,omitempty"`
	AdditionalInformation string                    `json:"additionalInformation,omitempty"`
	AgeGatedContent       bool                      `json:"ageGatedContent,omitempty"`
	ISVReseller           bool                      `json:"isvReseller,omitempty"`
	PrivacyURL            string                    `json:"privacyUrl,omitempty"`
	TermsURL              string                    `json:"termsUrl,omitempty"`
	EntityType            string                    `json:"entityType,omitempty"`
}

type ProvisionWorkspaceRequest struct {
	Name                    string                     `json:"name"`
	SourceWorkspaceID       string                     `json:"sourceWorkspaceId,omitempty"`
	InheritWithNewNumber    bool                       `json:"inheritWithNewNumber,omitempty"`
	Verification            *ProvisionVerificationData `json:"verification,omitempty"`
	CreditAmount            int                        `json:"creditAmount,omitempty"`
	CreditSourceWorkspaceID string                     `json:"creditSourceWorkspaceId,omitempty"`
	KeyName                 string                     `json:"keyName,omitempty"`
	KeyType                 string                     `json:"keyType,omitempty"`
	WebhookURL              string                     `json:"webhookUrl,omitempty"`
	GenerateOptInPage       *bool                      `json:"generateOptInPage,omitempty"`
}

type ProvisionVerificationResult struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	Type           string  `json:"type,omitempty"`
	TollFreeNumber *string `json:"tollFreeNumber,omitempty"`
	Inherited      bool    `json:"inherited"`
}

type ProvisionCreditsResult struct {
	Transferred int    `json:"transferred,omitempty"`
	Balance     int    `json:"balance,omitempty"`
	Error       string `json:"error,omitempty"`
}

type ProvisionKeyResult struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"`
	KeyPrefix string `json:"keyPrefix"`
	Type      string `json:"type"`
}

type ProvisionWebhookResult struct {
	URL   string `json:"url,omitempty"`
	Error string `json:"error,omitempty"`
}

type ProvisionOptInPageResult struct {
	URL    string `json:"url"`
	Slug   string `json:"slug"`
	PageID string `json:"pageId"`
}

type ProvisionLegalPagesResult struct {
	PrivacyURL string `json:"privacyUrl,omitempty"`
	TermsURL   string `json:"termsUrl,omitempty"`
}

type ProvisionWorkspaceResponse struct {
	Workspace    EnterpriseWorkspace          `json:"workspace"`
	Verification *ProvisionVerificationResult `json:"verification,omitempty"`
	Credits      *ProvisionCreditsResult      `json:"credits,omitempty"`
	Key          *ProvisionKeyResult          `json:"key,omitempty"`
	Webhook      *ProvisionWebhookResult      `json:"webhook,omitempty"`
	OptInPage    *ProvisionOptInPageResult    `json:"optInPage,omitempty"`
	LegalPages   *ProvisionLegalPagesResult   `json:"legalPages,omitempty"`
	APIBaseURL   string                       `json:"apiBaseUrl,omitempty"`
	DashboardURL string                       `json:"dashboardUrl,omitempty"`
}

type EnterpriseWebhook struct {
	URL string `json:"url"`
}

type EnterpriseWebhookTestResult struct {
	Success    bool   `json:"success"`
	StatusCode *int   `json:"statusCode,omitempty"`
	StatusText string `json:"statusText,omitempty"`
	Error      string `json:"error,omitempty"`
}

type EnterpriseWebhookRotateSecretResponse struct {
	Success   bool   `json:"success"`
	Secret    string `json:"secret"`
	RotatedAt string `json:"rotated_at"`
	Message   string `json:"message"`
}

type AnalyticsOverview struct {
	TotalMessages     int `json:"totalMessages"`
	DeliveredMessages int `json:"deliveredMessages"`
	FailedMessages    int `json:"failedMessages"`
	DeliveryRate      int `json:"deliveryRate"`
	TotalCreditsUsed  int `json:"totalCreditsUsed"`
	ActiveWorkspaces  int `json:"activeWorkspaces"`
}

type AnalyticsMessagesOptions struct {
	Period      string
	WorkspaceID string
}

type AnalyticsMessageDay struct {
	Date      string `json:"date"`
	Sent      int    `json:"sent"`
	Delivered int    `json:"delivered"`
	Failed    int    `json:"failed"`
}

type AnalyticsMessagesResponse struct {
	Period string                `json:"period"`
	Data   []AnalyticsMessageDay `json:"data"`
}

type DeliveryAnalytics struct {
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	Sent        int    `json:"sent"`
	Delivered   int    `json:"delivered"`
	Failed      int    `json:"failed"`
	Rate        int    `json:"rate"`
}

type AnalyticsCreditsOptions struct {
	Period string
}

type AnalyticsCreditDay struct {
	Date        string `json:"date"`
	Used        int    `json:"used"`
	Transferred int    `json:"transferred"`
	Purchased   int    `json:"purchased"`
}

type AnalyticsCreditsResponse struct {
	Period string               `json:"period"`
	Data   []AnalyticsCreditDay `json:"data"`
}

type OptInPage struct {
	ID             string  `json:"id"`
	Slug           string  `json:"slug"`
	URL            string  `json:"url"`
	BusinessName   string  `json:"businessName"`
	UseCase        *string `json:"useCase,omitempty"`
	IsActive       bool    `json:"isActive"`
	ViewCount      int     `json:"viewCount"`
	LogoURL        *string `json:"logoUrl,omitempty"`
	HeaderColor    *string `json:"headerColor,omitempty"`
	ButtonColor    *string `json:"buttonColor,omitempty"`
	CustomHeadline *string `json:"customHeadline,omitempty"`
	CreatedAt      string  `json:"createdAt"`
}

type CreateOptInPageRequest struct {
	BusinessName   string `json:"businessName"`
	UseCase        string `json:"useCase,omitempty"`
	UseCaseSummary string `json:"useCaseSummary,omitempty"`
	SampleMessages string `json:"sampleMessages,omitempty"`
}

type CreateOptInPageResponse struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	URL          string `json:"url"`
	BusinessName string `json:"businessName"`
}

type UpdateOptInPageRequest struct {
	LogoURL        *string  `json:"logoUrl,omitempty"`
	HeaderColor    *string  `json:"headerColor,omitempty"`
	ButtonColor    *string  `json:"buttonColor,omitempty"`
	CustomHeadline *string  `json:"customHeadline,omitempty"`
	CustomBenefits []string `json:"customBenefits,omitempty"`
}

type WorkspaceWebhook struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	IsActive  bool     `json:"isActive"`
	CreatedAt string   `json:"createdAt"`
}

type SetWorkspaceWebhookRequest struct {
	URL         string   `json:"url"`
	Events      []string `json:"events,omitempty"`
	Description string   `json:"description,omitempty"`
}

type SetWorkspaceWebhookResponse struct {
	ID      string   `json:"id"`
	URL     string   `json:"url"`
	Events  []string `json:"events"`
	Secret  string   `json:"secret,omitempty"`
	Created bool     `json:"created,omitempty"`
	Updated bool     `json:"updated,omitempty"`
}

type SuspendWorkspaceRequest struct {
	Reason string `json:"reason,omitempty"`
}

type SuspendWorkspaceResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	SuspendedAt string `json:"suspendedAt"`
}

type ResumeWorkspaceResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type AutoTopUpSettings struct {
	Enabled           bool    `json:"enabled"`
	Threshold         int     `json:"threshold"`
	Amount            int     `json:"amount"`
	SourceWorkspaceID *string `json:"sourceWorkspaceId"`
}

type UpdateAutoTopUpRequest struct {
	Enabled           bool    `json:"enabled"`
	Threshold         int     `json:"threshold"`
	Amount            int     `json:"amount"`
	SourceWorkspaceID *string `json:"sourceWorkspaceId,omitempty"`
}

type WorkspaceBillingItem struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	CreditsUsed           int    `json:"creditsUsed"`
	CreditsPurchased      int    `json:"creditsPurchased"`
	CreditsTransferredIn  int    `json:"creditsTransferredIn"`
	CreditsTransferredOut int    `json:"creditsTransferredOut"`
	MessagesSent          int    `json:"messagesSent"`
	MessagesDelivered     int    `json:"messagesDelivered"`
	WorkspaceFee          int    `json:"workspaceFee"`
	AllocatedPlatformFee  int    `json:"allocatedPlatformFee"`
	TotalCost             int    `json:"totalCost"`
}

type BillingBreakdownSummary struct {
	PlatformFee        int `json:"platformFee"`
	TotalWorkspaceFees int `json:"totalWorkspaceFees"`
	TotalCreditsUsed   int `json:"totalCreditsUsed"`
	TotalCost          int `json:"totalCost"`
}

type BillingBreakdown struct {
	Period     string                  `json:"period"`
	Summary    BillingBreakdownSummary `json:"summary"`
	Workspaces []WorkspaceBillingItem  `json:"workspaces"`
}

type BillingBreakdownOptions struct {
	Period string
	Page   int
	Limit  int
}

type BulkProvisionWorkspace struct {
	Name                    string `json:"name"`
	SourceWorkspaceID       string `json:"sourceWorkspaceId,omitempty"`
	CreditAmount            int    `json:"creditAmount,omitempty"`
	CreditSourceWorkspaceID string `json:"creditSourceWorkspaceId,omitempty"`
}

type BulkProvisionRequest struct {
	Workspaces []BulkProvisionWorkspace `json:"workspaces"`
}

type BulkProvisionResultItem struct {
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	WorkspaceID *string `json:"workspaceId,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Warning     *string `json:"warning,omitempty"`
	Error       *string `json:"error,omitempty"`
}

type BulkProvisionSummary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

type BulkProvisionResult struct {
	Results []BulkProvisionResultItem `json:"results"`
	Summary BulkProvisionSummary      `json:"summary"`
}

type DnsRecord struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DnsInstructions struct {
	CNAME DnsRecord `json:"cname"`
	TXT   DnsRecord `json:"txt"`
}

type SetCustomDomainRequest struct {
	Domain string `json:"domain"`
}

type SetCustomDomainResponse struct {
	Domain          string          `json:"domain"`
	Verified        bool            `json:"verified"`
	DnsInstructions DnsInstructions `json:"dnsInstructions"`
}

type SendInvitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type Invitation struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expiresAt"`
}

type QuotaSettings struct {
	MonthlyMessageQuota *int    `json:"monthlyMessageQuota"`
	MessagesThisMonth   int     `json:"messagesThisMonth"`
	QuotaResetAt        *string `json:"quotaResetAt"`
}

type UpdateQuotaRequest struct {
	MonthlyMessageQuota *int `json:"monthlyMessageQuota"`
}

type GenerateBusinessPageRequest struct {
	BusinessName    string `json:"businessName"`
	UseCase         string `json:"useCase,omitempty"`
	UseCaseSummary  string `json:"useCaseSummary,omitempty"`
	ContactEmail    string `json:"contactEmail,omitempty"`
	ContactPhone    string `json:"contactPhone,omitempty"`
	BusinessAddress string `json:"businessAddress,omitempty"`
	SocialURL       string `json:"socialUrl,omitempty"`
}

type GenerateBusinessPageResponse struct {
	Slug   string `json:"slug"`
	URL    string `json:"url"`
	PageID string `json:"pageId"`
}

type VerificationDocumentUploadResponse struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}

// ============================================================================
// Conversations
// ============================================================================

// ConversationStatus represents the status of a conversation.
type ConversationStatus string

const (
	// ConversationStatusActive means the conversation is active.
	ConversationStatusActive ConversationStatus = "active"
	// ConversationStatusClosed means the conversation is closed.
	ConversationStatusClosed ConversationStatus = "closed"
)

// Conversation represents an SMS conversation thread.
type Conversation struct {
	// ID is the unique conversation identifier.
	ID string `json:"id"`
	// PhoneNumber is the phone number of the contact.
	PhoneNumber string `json:"phoneNumber"`
	// Status is the conversation status.
	Status ConversationStatus `json:"status"`
	// UnreadCount is the number of unread messages.
	UnreadCount int `json:"unreadCount"`
	// MessageCount is the total number of messages.
	MessageCount int `json:"messageCount"`
	// LastMessageText is the text of the last message.
	LastMessageText *string `json:"lastMessageText"`
	// LastMessageAt is when the last message was sent/received.
	LastMessageAt *string `json:"lastMessageAt"`
	// LastMessageDirection is the direction of the last message.
	LastMessageDirection *string `json:"lastMessageDirection"`
	// Metadata contains custom metadata.
	Metadata map[string]interface{} `json:"metadata"`
	// Tags contains conversation tags.
	Tags []string `json:"tags"`
	// ContactID is the associated contact ID.
	ContactID *string `json:"contactId"`
	// CreatedAt is when the conversation was created.
	CreatedAt string `json:"createdAt"`
	// UpdatedAt is when the conversation was last updated.
	UpdatedAt string `json:"updatedAt"`
}

// ConversationPagination contains pagination info for conversation lists.
type ConversationPagination struct {
	// Total is the total number of conversations.
	Total int `json:"total"`
	// Limit is the maximum number of conversations returned.
	Limit int `json:"limit"`
	// Offset is the number of conversations skipped.
	Offset int `json:"offset"`
	// HasMore indicates if more conversations are available.
	HasMore bool `json:"hasMore"`
}

// ConversationListResponse is the response from listing conversations.
type ConversationListResponse struct {
	// Data contains the list of conversations.
	Data []Conversation `json:"data"`
	// Pagination contains pagination info.
	Pagination ConversationPagination `json:"pagination"`
}

// ConversationMessages contains messages within a conversation.
type ConversationMessages struct {
	// Data contains the list of messages.
	Data []Message `json:"data"`
	// Pagination contains pagination info.
	Pagination ConversationPagination `json:"pagination"`
}

// ConversationWithMessages is a conversation with its messages.
type ConversationWithMessages struct {
	Conversation
	// Messages contains the conversation messages (when include_messages=true).
	Messages *ConversationMessages `json:"messages,omitempty"`
}

// ListConversationsRequest is the request to list conversations.
type ListConversationsRequest struct {
	// Limit is the maximum number of conversations to return.
	Limit int
	// Offset is the number of conversations to skip.
	Offset int
	// Status filters by conversation status.
	Status ConversationStatus
}

// GetConversationRequest is the request to get a single conversation.
type GetConversationRequest struct {
	// IncludeMessages includes messages in the response.
	IncludeMessages bool
	// MessageLimit is the maximum number of messages to return.
	MessageLimit int
	// MessageOffset is the number of messages to skip.
	MessageOffset int
}

// UpdateConversationRequest is the request to update a conversation.
type UpdateConversationRequest struct {
	// Metadata is the new metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// Tags is the new tags.
	Tags []string `json:"tags,omitempty"`
}

// ReplyToConversationRequest is the request to reply to a conversation.
type ReplyToConversationRequest struct {
	// Text is the message content (required).
	Text string `json:"text"`
	// MessageType is the message type for compliance.
	MessageType MessageType `json:"messageType,omitempty"`
	// Metadata is custom JSON metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// MediaUrls is a list of media URLs to attach.
	MediaUrls []string `json:"mediaUrls,omitempty"`
}

// AddLabelsRequest is the request to add labels to a conversation.
type AddLabelsRequest struct {
	// LabelIds is the list of label IDs to add.
	LabelIds []string `json:"labelIds"`
}

// ============================================================================
// Labels
// ============================================================================

// Label represents a conversation label.
type Label struct {
	// ID is the unique label identifier.
	ID string `json:"id"`
	// Name is the label name.
	Name string `json:"name"`
	// Color is the label color.
	Color string `json:"color"`
	// Description is the label description.
	Description *string `json:"description,omitempty"`
	// CreatedAt is when the label was created.
	CreatedAt string `json:"createdAt"`
}

// LabelListResponse is the response from listing labels.
type LabelListResponse struct {
	// Data contains the list of labels.
	Data []Label `json:"data"`
}

// CreateLabelRequest is the request to create a label.
type CreateLabelRequest struct {
	// Name is the label name (required).
	Name string `json:"name"`
	// Color is the label color.
	Color string `json:"color,omitempty"`
	// Description is the label description.
	Description string `json:"description,omitempty"`
}

// ============================================================================
// Drafts
// ============================================================================

// DraftStatus represents the status of a message draft.
type DraftStatus string

const (
	// DraftStatusPending means the draft is awaiting review.
	DraftStatusPending DraftStatus = "pending"
	// DraftStatusApproved means the draft has been approved.
	DraftStatusApproved DraftStatus = "approved"
	// DraftStatusRejected means the draft has been rejected.
	DraftStatusRejected DraftStatus = "rejected"
	// DraftStatusSent means the draft has been sent.
	DraftStatusSent DraftStatus = "sent"
	// DraftStatusFailed means the draft failed to send.
	DraftStatusFailed DraftStatus = "failed"
)

// MessageDraft represents a message draft.
type MessageDraft struct {
	// ID is the unique draft identifier.
	ID string `json:"id"`
	// ConversationId is the associated conversation ID.
	ConversationId string `json:"conversationId"`
	// Text is the draft message content.
	Text string `json:"text"`
	// MediaUrls is a list of media URLs.
	MediaUrls []string `json:"mediaUrls,omitempty"`
	// Metadata contains custom metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// Status is the draft status.
	Status DraftStatus `json:"status"`
	// Source indicates the origin of the draft.
	Source *string `json:"source,omitempty"`
	// CreatedBy is the user who created the draft.
	CreatedBy *string `json:"createdBy,omitempty"`
	// ReviewedBy is the user who reviewed the draft.
	ReviewedBy *string `json:"reviewedBy,omitempty"`
	// ReviewedAt is when the draft was reviewed.
	ReviewedAt *string `json:"reviewedAt,omitempty"`
	// RejectionReason is the reason for rejection.
	RejectionReason *string `json:"rejectionReason,omitempty"`
	// MessageId is the ID of the sent message (if sent).
	MessageId *string `json:"messageId,omitempty"`
	// CreatedAt is when the draft was created.
	CreatedAt string `json:"createdAt"`
	// UpdatedAt is when the draft was last updated.
	UpdatedAt string `json:"updatedAt"`
}

// DraftPagination contains pagination info for draft lists.
type DraftPagination struct {
	// Total is the total number of drafts.
	Total int `json:"total"`
}

// DraftListResponse is the response from listing drafts.
type DraftListResponse struct {
	// Data contains the list of drafts.
	Data []MessageDraft `json:"data"`
	// Pagination contains pagination info.
	Pagination DraftPagination `json:"pagination"`
}

// CreateDraftRequest is the request to create a draft.
type CreateDraftRequest struct {
	// ConversationId is the associated conversation ID (required).
	ConversationId string `json:"conversationId"`
	// Text is the draft message content (required).
	Text string `json:"text"`
	// MediaUrls is a list of media URLs.
	MediaUrls []string `json:"mediaUrls,omitempty"`
	// Metadata contains custom metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// Source indicates the origin of the draft.
	Source string `json:"source,omitempty"`
}

// UpdateDraftRequest is the request to update a draft.
type UpdateDraftRequest struct {
	// Text is the updated draft message content.
	Text string `json:"text,omitempty"`
	// MediaUrls is the updated list of media URLs.
	MediaUrls []string `json:"mediaUrls,omitempty"`
	// Metadata is the updated custom metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RejectDraftRequest is the request to reject a draft.
type RejectDraftRequest struct {
	// Reason is the rejection reason.
	Reason string `json:"reason,omitempty"`
}

// ListDraftsRequest is the request to list drafts.
type ListDraftsRequest struct {
	// ConversationId filters by conversation ID.
	ConversationId string
	// Status filters by draft status.
	Status DraftStatus
	// Limit is the maximum number of drafts to return.
	Limit int
	// Offset is the number of drafts to skip.
	Offset int
}

// ============================================================================
// Conversation Context
// ============================================================================

// ConversationContextInfo contains summary info about a conversation.
type ConversationContextInfo struct {
	ID           string `json:"id"`
	PhoneNumber  string `json:"phoneNumber"`
	Status       string `json:"status"`
	MessageCount int    `json:"messageCount"`
	UnreadCount  int    `json:"unreadCount"`
}

// ConversationContextBusiness contains business info for context.
type ConversationContextBusiness struct {
	Name    string `json:"name"`
	UseCase string `json:"useCase,omitempty"`
}

// ConversationContextResponse is the response from getting conversation context.
type ConversationContextResponse struct {
	Context       string                       `json:"context"`
	Conversation  ConversationContextInfo       `json:"conversation"`
	TokenEstimate int                           `json:"tokenEstimate"`
	Business      *ConversationContextBusiness  `json:"business,omitempty"`
}

// GetConversationContextRequest is the request to get conversation context.
type GetConversationContextRequest struct {
	MaxMessages int
}

// SuggestedReply represents a single AI-generated reply suggestion.
type SuggestedReply struct {
	// Text is the suggested reply text.
	Text string `json:"text"`
	// Tone is the tone of the suggestion (professional, friendly, or concise).
	Tone string `json:"tone"`
}

// SuggestRepliesResponse is the response from generating suggested replies.
type SuggestRepliesResponse struct {
	// Suggestions contains the AI-generated reply suggestions.
	Suggestions []SuggestedReply `json:"suggestions"`
	// BasedOnMessageId is the ID of the inbound message the suggestions were based on.
	BasedOnMessageId string `json:"basedOnMessageId,omitempty"`
	// Model is the AI model used to generate the suggestions.
	Model string `json:"model,omitempty"`
}

// ============================================================================
// Rules
// ============================================================================

// Rule represents an auto-labeling rule.
type Rule struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Conditions map[string]interface{} `json:"conditions"`
	Actions    map[string]interface{} `json:"actions"`
	Priority   *int                   `json:"priority,omitempty"`
	CreatedAt  string                 `json:"createdAt"`
	UpdatedAt  string                 `json:"updatedAt"`
}

// RuleListResponse is the response from listing rules.
type RuleListResponse struct {
	Data []Rule `json:"data"`
}

// CreateRuleRequest is the request to create a rule.
type CreateRuleRequest struct {
	Name       string                 `json:"name"`
	Conditions map[string]interface{} `json:"conditions"`
	Actions    map[string]interface{} `json:"actions"`
	Priority   *int                   `json:"priority,omitempty"`
}

// UpdateRuleRequest is the request to update a rule.
type UpdateRuleRequest struct {
	Name       string                 `json:"name,omitempty"`
	Conditions map[string]interface{} `json:"conditions,omitempty"`
	Actions    map[string]interface{} `json:"actions,omitempty"`
	Priority   *int                   `json:"priority,omitempty"`
}

// ============================================================================
// Generated Template
// ============================================================================

// GenerateTemplateRequest is the request to generate a template.
type GenerateTemplateRequest struct {
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
}

// GeneratedTemplate is the response from generating a template.
type GeneratedTemplate struct {
	Name      string   `json:"name"`
	Text      string   `json:"text"`
	Variables []string `json:"variables"`
	Category  string   `json:"category"`
}

// ============================================================================
// Branded Short Links (URL shortening)
// ============================================================================

// ShortLink is a newly minted branded short link.
type ShortLink struct {
	// Code is the short code (the segment after the domain, e.g. "Ab3xY7").
	Code string `json:"code"`
	// ShortURL is the full branded short URL to share (e.g. "https://sendly.live/l/Ab3xY7").
	ShortURL string `json:"shortUrl"`
	// DestinationURL is the destination the short link redirects to.
	DestinationURL string `json:"destinationUrl"`
}

// ListShortLinksRequest is the request to list short links.
type ListShortLinksRequest struct {
	// Limit is the maximum number of links to return (default 50, max 200).
	Limit int
	// Offset is the number of links to skip for pagination (default 0).
	Offset int
}

// ShortLinkListItem is a short link with click analytics, as returned by List.
type ShortLinkListItem struct {
	// Code is the short code (the segment after the domain).
	Code string `json:"code"`
	// ShortURL is the full branded short URL.
	ShortURL string `json:"shortUrl"`
	// DestinationURL is the destination the short link redirects to.
	DestinationURL string `json:"destinationUrl"`
	// BrandSlug is the workspace brand slug segment, or nil when unbranded.
	BrandSlug *string `json:"brandSlug,omitempty"`
	// ClickCount is the total human clicks recorded (link-preview bots are excluded).
	ClickCount int `json:"clickCount"`
	// Disabled indicates whether the link is disabled (the redirect then returns 404).
	Disabled bool `json:"disabled"`
	// LastCountry is the ISO 3166-1 alpha-2 country of the most recent click, or nil.
	LastCountry *string `json:"lastCountry,omitempty"`
	// LastClickedAt is when the link was last clicked (ISO 8601), or nil.
	LastClickedAt *string `json:"lastClickedAt,omitempty"`
	// CreatedAt is when the link was created (ISO 8601).
	CreatedAt string `json:"createdAt"`
	// Spark is a 14-day daily click histogram, oldest first (today last).
	Spark []int `json:"spark,omitempty"`
}

// ShortLinkListResponse is the response from listing short links.
type ShortLinkListResponse struct {
	// Links is the workspace's short links, newest first.
	Links []ShortLinkListItem `json:"links"`
	// Total is the total number of short links in the workspace.
	Total int `json:"total"`
}

// ShortLinkDisabledResponse is the response from enabling or disabling a short link.
type ShortLinkDisabledResponse struct {
	// Code is the short code that was updated.
	Code string `json:"code"`
	// Disabled is the new disabled state.
	Disabled bool `json:"disabled"`
}
