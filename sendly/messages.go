package sendly

import (
	"context"
	"net/url"
	"strconv"
)

// MessagesService handles message-related API operations.
type MessagesService struct {
	client *Client
}

// Send sends an SMS or MMS message.
func (s *MessagesService) Send(ctx context.Context, req *SendMessageRequest) (*Message, error) {
	return s.SendWithOptions(ctx, req)
}

// SendWithOptions sends an SMS or MMS message with per-request options.
func (s *MessagesService) SendWithOptions(ctx context.Context, req *SendMessageRequest, opts ...RequestOption) (*Message, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.To == "" {
		return nil, &ValidationError{APIError: APIError{Message: "to is required"}}
	}
	if req.Text == "" && len(req.MediaUrls) == 0 {
		return nil, &ValidationError{APIError: APIError{Message: "either text or media_urls is required"}}
	}

	var resp Message
	err := s.client.request(ctx, "POST", "/messages", req, &resp, opts...)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// SendWhatsApp sends a WhatsApp message: free-form text, media with an
// optional caption, or an approved template.
//
// Requires a live API key and a From number with an active WhatsApp
// connection (see client.WhatsApp.Signup). Free-form Text and media only
// deliver inside an open 24-hour customer-service window — outside it, send
// an approved Template instead (check with client.WhatsApp.Window).
func (s *MessagesService) SendWhatsApp(ctx context.Context, req *SendWhatsAppMessageRequest) (*WhatsAppMessage, error) {
	return s.SendWhatsAppWithOptions(ctx, req)
}

// SendWhatsAppWithOptions sends a WhatsApp message with per-request
// options. See SendWhatsApp for the channel requirements.
func (s *MessagesService) SendWhatsAppWithOptions(ctx context.Context, req *SendWhatsAppMessageRequest, opts ...RequestOption) (*WhatsAppMessage, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.To == "" {
		return nil, &ValidationError{APIError: APIError{Message: "to is required"}}
	}
	if req.From == "" {
		return nil, &ValidationError{APIError: APIError{Message: "from is required"}}
	}
	if req.Text == "" && len(req.MediaUrls) == 0 && req.Template == nil {
		return nil, &ValidationError{APIError: APIError{Message: "either text, media_urls, or template is required"}}
	}

	body := *req
	body.Channel = "whatsapp"

	var resp WhatsAppMessage
	err := s.client.request(ctx, "POST", "/messages", &body, &resp, opts...)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// SendRcs sends an RCS message: rich text with optional tappable
// suggestions, or a standalone rich card.
//
// Requires a live API key and a sendable RCS agent on the workspace (see
// client.RCS). When the recipient's device or network doesn't support RCS,
// text sends fall back to plain SMS (billed as SMS) — check FellBackTo (or
// Channel) on the response to see which leg delivered. Card sends have no
// SMS form and never fall back.
func (s *MessagesService) SendRcs(ctx context.Context, req *SendRcsMessageRequest) (*RcsMessage, error) {
	return s.SendRcsWithOptions(ctx, req)
}

// SendRcsWithOptions sends an RCS message with per-request options. See
// SendRcs for the fallback behaviour.
func (s *MessagesService) SendRcsWithOptions(ctx context.Context, req *SendRcsMessageRequest, opts ...RequestOption) (*RcsMessage, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.To == "" {
		return nil, &ValidationError{APIError: APIError{Message: "to is required"}}
	}
	if (req.Text == "") == (req.Card == nil) {
		return nil, &ValidationError{APIError: APIError{Message: "exactly one of text or card is required"}}
	}
	if req.Card != nil && len(req.Suggestions) > 0 {
		return nil, &ValidationError{APIError: APIError{Message: "suggestions ride on text messages — put card buttons in card.suggestions"}}
	}

	body := *req
	body.Channel = "rcs"

	var resp RcsMessage
	err := s.client.request(ctx, "POST", "/messages", &body, &resp, opts...)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// SendGroup sends a group MMS to 2-8 US/Canada recipients.
//
// Everyone in the group shares one thread and replies fan out to all
// participants. Group messaging is an A2P 10DLC capability: the sending number
// must be an MMS-enabled, 10DLC-registered number you own. Omit From to use the
// workspace's default sender. Requires the group_mms feature (and enable_mms
// when attaching media).
func (s *MessagesService) SendGroup(ctx context.Context, req *SendGroupMessageRequest) (*GroupMessageResponse, error) {
	return s.SendGroupWithOptions(ctx, req)
}

// SendGroupWithOptions sends a group MMS with per-request options. See
// SendGroup for the sender requirements.
func (s *MessagesService) SendGroupWithOptions(ctx context.Context, req *SendGroupMessageRequest, opts ...RequestOption) (*GroupMessageResponse, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if len(req.To) < 2 {
		return nil, &ValidationError{APIError: APIError{Message: "group messaging requires at least 2 recipients in 'to'"}}
	}
	if len(req.To) > 8 {
		return nil, &ValidationError{APIError: APIError{Message: "group messaging supports at most 8 recipients"}}
	}
	if req.Text == "" && len(req.MediaUrls) == 0 {
		return nil, &ValidationError{APIError: APIError{Message: "either text or media_urls is required"}}
	}

	var resp GroupMessageResponse
	err := s.client.request(ctx, "POST", "/messages/group", req, &resp, opts...)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Enhance rewrites a draft message with AI for clarity, compliance, and
// send-readiness. Provide Text, MessageType, or both — at least one is
// required. When AI enhancement is unavailable for the account, the response
// falls back to the original text with an empty explanation. Requires the
// ai_classification feature.
func (s *MessagesService) Enhance(ctx context.Context, req *EnhanceMessageRequest) (*EnhanceMessageResponse, error) {
	if req == nil || (req.Text == "" && req.MessageType == "") {
		return nil, &ValidationError{APIError: APIError{Message: "either text or messageType is required"}}
	}

	var resp EnhanceMessageResponse
	err := s.client.request(ctx, "POST", "/ai/enhance", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// List retrieves a list of messages.
func (s *MessagesService) List(ctx context.Context, req *ListMessagesRequest) (*ListMessagesResponse, error) {
	params := make(map[string]string)

	if req != nil {
		if req.Limit > 0 {
			params["limit"] = strconv.Itoa(req.Limit)
		}
		if req.Offset > 0 {
			params["offset"] = strconv.Itoa(req.Offset)
		}
		if req.Status != "" {
			params["status"] = string(req.Status)
		}
		if req.To != "" {
			params["to"] = req.To
		}
	}

	path := "/messages" + buildQueryString(params)

	var resp ListMessagesResponse
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Get retrieves a single message by ID.
func (s *MessagesService) Get(ctx context.Context, id string) (*Message, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "message ID is required"}}
	}

	// URL encode the ID to prevent path injection
	path := "/messages/" + url.PathEscape(id)

	var resp Message
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Schedule schedules an SMS message for future delivery.
func (s *MessagesService) Schedule(ctx context.Context, req *ScheduleMessageRequest) (*ScheduledMessage, error) {
	return s.ScheduleWithOptions(ctx, req)
}

// ScheduleWithOptions schedules an SMS message for future delivery with
// per-request options.
func (s *MessagesService) ScheduleWithOptions(ctx context.Context, req *ScheduleMessageRequest, opts ...RequestOption) (*ScheduledMessage, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.To == "" {
		return nil, &ValidationError{APIError: APIError{Message: "to is required"}}
	}
	if req.Text == "" {
		return nil, &ValidationError{APIError: APIError{Message: "text is required"}}
	}
	if req.ScheduledAt == "" {
		return nil, &ValidationError{APIError: APIError{Message: "scheduledAt is required"}}
	}

	var resp ScheduledMessage
	err := s.client.request(ctx, "POST", "/messages/schedule", req, &resp, opts...)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// ListScheduled retrieves a list of scheduled messages.
func (s *MessagesService) ListScheduled(ctx context.Context, req *ListScheduledMessagesRequest) (*ListScheduledMessagesResponse, error) {
	params := make(map[string]string)

	if req != nil {
		if req.Limit > 0 {
			params["limit"] = strconv.Itoa(req.Limit)
		}
		if req.Offset > 0 {
			params["offset"] = strconv.Itoa(req.Offset)
		}
		if req.Status != "" {
			params["status"] = string(req.Status)
		}
	}

	path := "/messages/scheduled" + buildQueryString(params)

	var resp ListScheduledMessagesResponse
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetScheduled retrieves a single scheduled message by ID.
func (s *MessagesService) GetScheduled(ctx context.Context, id string) (*ScheduledMessage, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "scheduled message ID is required"}}
	}

	path := "/messages/scheduled/" + url.PathEscape(id)

	var resp ScheduledMessage
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// CancelScheduled cancels a scheduled message.
func (s *MessagesService) CancelScheduled(ctx context.Context, id string) (*CancelScheduledMessageResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "scheduled message ID is required"}}
	}

	path := "/messages/scheduled/" + url.PathEscape(id)

	var resp CancelScheduledMessageResponse
	err := s.client.request(ctx, "DELETE", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// SendBatch sends multiple SMS messages in a batch.
func (s *MessagesService) SendBatch(ctx context.Context, req *SendBatchRequest) (*BatchMessageResponse, error) {
	return s.SendBatchWithOptions(ctx, req)
}

// SendBatchWithOptions sends multiple SMS messages in a batch with
// per-request options.
func (s *MessagesService) SendBatchWithOptions(ctx context.Context, req *SendBatchRequest, opts ...RequestOption) (*BatchMessageResponse, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if len(req.Messages) == 0 {
		return nil, &ValidationError{APIError: APIError{Message: "messages are required"}}
	}

	// Validate each message
	for i, msg := range req.Messages {
		if msg.To == "" {
			return nil, &ValidationError{APIError: APIError{Message: "to is required for message at index " + strconv.Itoa(i)}}
		}
		if msg.Text == "" {
			return nil, &ValidationError{APIError: APIError{Message: "text is required for message at index " + strconv.Itoa(i)}}
		}
	}

	// The batch endpoint dedupes header-less retries server-side by hashing
	// the request content; an auto-generated key would bypass that net for
	// identical cross-process re-runs, so only caller-supplied keys are sent.
	var resp BatchMessageResponse
	err := s.client.request(ctx, "POST", "/messages/batch", req, &resp, append(opts, withoutAutoIdempotencyKey())...)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetBatch retrieves the status of a batch by ID.
func (s *MessagesService) GetBatch(ctx context.Context, batchID string) (*BatchMessageResponse, error) {
	if batchID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "batch ID is required"}}
	}

	path := "/messages/batch/" + url.PathEscape(batchID)

	var resp BatchMessageResponse
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// ListBatches retrieves a list of batches.
func (s *MessagesService) ListBatches(ctx context.Context, req *ListBatchesRequest) (*ListBatchesResponse, error) {
	params := make(map[string]string)

	if req != nil {
		if req.Limit > 0 {
			params["limit"] = strconv.Itoa(req.Limit)
		}
		if req.Offset > 0 {
			params["offset"] = strconv.Itoa(req.Offset)
		}
		if req.Status != "" {
			params["status"] = string(req.Status)
		}
	}

	path := "/messages/batches" + buildQueryString(params)

	var resp ListBatchesResponse
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// PreviewBatch previews a batch without sending (dry run).
func (s *MessagesService) PreviewBatch(ctx context.Context, req *SendBatchRequest) (*BatchPreviewResponse, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if len(req.Messages) == 0 {
		return nil, &ValidationError{APIError: APIError{Message: "messages are required"}}
	}

	// Validate each message
	for i, msg := range req.Messages {
		if msg.To == "" {
			return nil, &ValidationError{APIError: APIError{Message: "to is required for message at index " + strconv.Itoa(i)}}
		}
		if msg.Text == "" {
			return nil, &ValidationError{APIError: APIError{Message: "text is required for message at index " + strconv.Itoa(i)}}
		}
	}

	var resp BatchPreviewResponse
	err := s.client.request(ctx, "POST", "/messages/batch/preview", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
