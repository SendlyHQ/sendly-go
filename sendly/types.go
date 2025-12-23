package sendly

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
	// TelnyxMessageID is the Telnyx message ID for tracking.
	TelnyxMessageID *string `json:"telnyxMessageId,omitempty"`
	// Warning contains a warning message (e.g., when 'from' is ignored).
	Warning *string `json:"warning,omitempty"`
	// SenderNote contains a note about sender behavior.
	SenderNote *string `json:"senderNote,omitempty"`
	// CreatedAt is when the message was created.
	CreatedAt string `json:"createdAt,omitempty"`
	// DeliveredAt is when the message was delivered (if applicable).
	DeliveredAt *string `json:"deliveredAt,omitempty"`
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
	// MessageStatusFailed means the message failed to deliver.
	MessageStatusFailed MessageStatus = "failed"
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

// SendMessageRequest is the request to send a message.
type SendMessageRequest struct {
	// To is the recipient phone number in E.164 format (required).
	To string `json:"to"`
	// Text is the message content (required).
	Text string `json:"text"`
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

// APIError represents an error from the API.
type APIError struct {
	// Code is the error code.
	Code string `json:"code"`
	// Message is the error message.
	Message string `json:"message"`
	// Details contains additional error details.
	Details map[string]interface{} `json:"details,omitempty"`
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
}

// SendBatchRequest is the request to send batch messages.
type SendBatchRequest struct {
	// Messages is the list of messages to send (required).
	Messages []BatchMessageItem `json:"messages"`
	// From is the sender ID or phone number (optional, applies to all).
	From string `json:"from,omitempty"`
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
	// To is the recipient phone number.
	To string `json:"to"`
	// MessageID is the message ID if successful.
	MessageID *string `json:"messageId,omitempty"`
	// Status is the message status.
	Status string `json:"status"`
	// Error is the error message if failed.
	Error *string `json:"error,omitempty"`
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
	// Name is the display name.
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
	// LastFour is the last 4 characters of the key.
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
