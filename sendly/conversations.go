package sendly

import (
	"context"
	"net/url"
	"strconv"
)

// ConversationsService handles conversation-related API operations.
type ConversationsService struct {
	client *Client
}

// List retrieves a list of conversations.
func (s *ConversationsService) List(ctx context.Context, req *ListConversationsRequest) (*ConversationListResponse, error) {
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

	path := "/conversations" + buildQueryString(params)

	var resp ConversationListResponse
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Get retrieves a single conversation by ID.
func (s *ConversationsService) Get(ctx context.Context, id string, req *GetConversationRequest) (*ConversationWithMessages, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "conversation ID is required"}}
	}

	params := make(map[string]string)

	if req != nil {
		if req.IncludeMessages {
			params["include_messages"] = "true"
		}
		if req.MessageLimit > 0 {
			params["message_limit"] = strconv.Itoa(req.MessageLimit)
		}
		if req.MessageOffset > 0 {
			params["message_offset"] = strconv.Itoa(req.MessageOffset)
		}
	}

	path := "/conversations/" + url.PathEscape(id) + buildQueryString(params)

	var resp ConversationWithMessages
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Reply sends a message in a conversation.
func (s *ConversationsService) Reply(ctx context.Context, id string, req *ReplyToConversationRequest) (*Message, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "conversation ID is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.Text == "" {
		return nil, &ValidationError{APIError: APIError{Message: "text is required"}}
	}

	path := "/conversations/" + url.PathEscape(id) + "/messages"

	var resp Message
	err := s.client.request(ctx, "POST", path, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Update updates a conversation's metadata or tags.
func (s *ConversationsService) Update(ctx context.Context, id string, req *UpdateConversationRequest) (*Conversation, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "conversation ID is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	path := "/conversations/" + url.PathEscape(id)

	var resp Conversation
	err := s.client.request(ctx, "PATCH", path, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Close closes a conversation.
func (s *ConversationsService) Close(ctx context.Context, id string) (*Conversation, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "conversation ID is required"}}
	}

	path := "/conversations/" + url.PathEscape(id) + "/close"

	var resp Conversation
	err := s.client.request(ctx, "POST", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Reopen reopens a closed conversation.
func (s *ConversationsService) Reopen(ctx context.Context, id string) (*Conversation, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "conversation ID is required"}}
	}

	path := "/conversations/" + url.PathEscape(id) + "/reopen"

	var resp Conversation
	err := s.client.request(ctx, "POST", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// MarkRead marks a conversation as read.
func (s *ConversationsService) MarkRead(ctx context.Context, id string) (*Conversation, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "conversation ID is required"}}
	}

	path := "/conversations/" + url.PathEscape(id) + "/mark-read"

	var resp Conversation
	err := s.client.request(ctx, "POST", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
