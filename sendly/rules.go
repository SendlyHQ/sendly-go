package sendly

import (
	"context"
	"net/url"
)

// RulesService handles rule-related API operations.
type RulesService struct {
	client *Client
}

// List retrieves all rules.
func (s *RulesService) List(ctx context.Context) (*RuleListResponse, error) {
	var resp RuleListResponse
	err := s.client.request(ctx, "GET", "/rules", nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Create creates a new rule.
func (s *RulesService) Create(ctx context.Context, req *CreateRuleRequest) (*Rule, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.Name == "" {
		return nil, &ValidationError{APIError: APIError{Message: "name is required"}}
	}

	var resp Rule
	err := s.client.request(ctx, "POST", "/rules", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Update updates a rule.
func (s *RulesService) Update(ctx context.Context, id string, req *UpdateRuleRequest) (*Rule, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "rule ID is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	path := "/rules/" + url.PathEscape(id)

	var resp Rule
	err := s.client.request(ctx, "PATCH", path, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Delete deletes a rule by ID.
func (s *RulesService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return &ValidationError{APIError: APIError{Message: "rule ID is required"}}
	}

	path := "/rules/" + url.PathEscape(id)
	return s.client.request(ctx, "DELETE", path, nil, nil)
}
