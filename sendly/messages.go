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

// Send sends an SMS message.
func (s *MessagesService) Send(ctx context.Context, req *SendMessageRequest) (*Message, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.To == "" {
		return nil, &ValidationError{APIError: APIError{Message: "to is required"}}
	}
	if req.Text == "" {
		return nil, &ValidationError{APIError: APIError{Message: "text is required"}}
	}

	var resp Message
	err := s.client.request(ctx, "POST", "/messages", req, &resp)
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
	err := s.client.request(ctx, "POST", "/messages/schedule", req, &resp)
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

	var resp BatchMessageResponse
	err := s.client.request(ctx, "POST", "/messages/batch", req, &resp)
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
