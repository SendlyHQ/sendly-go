package sendly

import (
	"context"
	"net/url"
	"strconv"
)

// DraftsService handles draft-related API operations.
type DraftsService struct {
	client *Client
}

// Create creates a new message draft.
func (s *DraftsService) Create(ctx context.Context, req *CreateDraftRequest) (*MessageDraft, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.ConversationId == "" {
		return nil, &ValidationError{APIError: APIError{Message: "conversation ID is required"}}
	}
	if req.Text == "" {
		return nil, &ValidationError{APIError: APIError{Message: "text is required"}}
	}

	var resp MessageDraft
	err := s.client.request(ctx, "POST", "/drafts", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// List retrieves a list of drafts.
func (s *DraftsService) List(ctx context.Context, req *ListDraftsRequest) (*DraftListResponse, error) {
	params := make(map[string]string)

	if req != nil {
		if req.ConversationId != "" {
			params["conversation_id"] = req.ConversationId
		}
		if req.Status != "" {
			params["status"] = string(req.Status)
		}
		if req.Limit > 0 {
			params["limit"] = strconv.Itoa(req.Limit)
		}
		if req.Offset > 0 {
			params["offset"] = strconv.Itoa(req.Offset)
		}
	}

	path := "/drafts" + buildQueryString(params)

	var resp DraftListResponse
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Get retrieves a single draft by ID.
func (s *DraftsService) Get(ctx context.Context, id string) (*MessageDraft, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "draft ID is required"}}
	}

	path := "/drafts/" + url.PathEscape(id)

	var resp MessageDraft
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Update updates a draft.
func (s *DraftsService) Update(ctx context.Context, id string, req *UpdateDraftRequest) (*MessageDraft, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "draft ID is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	path := "/drafts/" + url.PathEscape(id)

	var resp MessageDraft
	err := s.client.request(ctx, "PATCH", path, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Approve approves a draft for sending.
func (s *DraftsService) Approve(ctx context.Context, id string) (*MessageDraft, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "draft ID is required"}}
	}

	path := "/drafts/" + url.PathEscape(id) + "/approve"

	var resp MessageDraft
	err := s.client.request(ctx, "POST", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Reject rejects a draft with an optional reason.
func (s *DraftsService) Reject(ctx context.Context, id string, reason string) (*MessageDraft, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "draft ID is required"}}
	}

	path := "/drafts/" + url.PathEscape(id) + "/reject"

	var body interface{}
	if reason != "" {
		body = &RejectDraftRequest{Reason: reason}
	}

	var resp MessageDraft
	err := s.client.request(ctx, "POST", path, body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
