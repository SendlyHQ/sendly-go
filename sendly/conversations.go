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

// AddLabels adds labels to a conversation.
func (s *ConversationsService) AddLabels(ctx context.Context, id string, labelIds []string) (*Conversation, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "conversation ID is required"}}
	}
	if len(labelIds) == 0 {
		return nil, &ValidationError{APIError: APIError{Message: "at least one label ID is required"}}
	}

	path := "/conversations/" + url.PathEscape(id) + "/labels"

	body := &AddLabelsRequest{LabelIds: labelIds}

	var resp Conversation
	err := s.client.request(ctx, "POST", path, body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// RemoveLabel removes a label from a conversation.
func (s *ConversationsService) RemoveLabel(ctx context.Context, id string, labelId string) (*Conversation, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "conversation ID is required"}}
	}
	if labelId == "" {
		return nil, &ValidationError{APIError: APIError{Message: "label ID is required"}}
	}

	path := "/conversations/" + url.PathEscape(id) + "/labels/" + url.PathEscape(labelId)

	var resp Conversation
	err := s.client.request(ctx, "DELETE", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetContext retrieves the conversation context for AI/LLM consumption.
func (s *ConversationsService) GetContext(ctx context.Context, id string, req *GetConversationContextRequest) (*ConversationContextResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "conversation ID is required"}}
	}

	params := make(map[string]string)

	if req != nil {
		if req.MaxMessages > 0 {
			params["max_messages"] = strconv.Itoa(req.MaxMessages)
		}
	}

	path := "/conversations/" + url.PathEscape(id) + "/context" + buildQueryString(params)

	var resp ConversationContextResponse
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// SuggestReplies generates AI-suggested replies for a conversation.
func (s *ConversationsService) SuggestReplies(ctx context.Context, id string) (*SuggestRepliesResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "conversation ID is required"}}
	}

	path := "/conversations/" + url.PathEscape(id) + "/suggest-replies"

	var resp SuggestRepliesResponse
	err := s.client.request(ctx, "POST", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
