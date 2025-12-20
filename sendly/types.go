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
	// Error contains error message if delivery failed.
	Error *string `json:"error,omitempty"`
	// Segments is the number of SMS segments.
	Segments int `json:"segments,omitempty"`
	// CreditsUsed is the number of credits consumed.
	CreditsUsed int `json:"creditsUsed,omitempty"`
	// IsSandbox indicates if the message was sent in sandbox mode.
	IsSandbox bool `json:"isSandbox,omitempty"`
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
	// MessageStatusSending means the message is being sent.
	MessageStatusSending MessageStatus = "sending"
	// MessageStatusSent means the message was sent to the carrier.
	MessageStatusSent MessageStatus = "sent"
	// MessageStatusDelivered means the message was delivered.
	MessageStatusDelivered MessageStatus = "delivered"
	// MessageStatusFailed means the message failed to deliver.
	MessageStatusFailed MessageStatus = "failed"
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
