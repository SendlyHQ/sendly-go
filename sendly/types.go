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
	// ErrorCode is the error code if delivery failed.
	ErrorCode *string `json:"errorCode,omitempty"`
	// RetryCount is the number of delivery retry attempts.
	RetryCount int `json:"retryCount,omitempty"`
	// Metadata contains custom user-provided metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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
	// MediaUrls is a list of media URLs to attach (converts message to MMS).
	MediaUrls []string `json:"mediaUrls,omitempty"`
	// Metadata is custom JSON metadata to attach to the message (max 4KB).
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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
