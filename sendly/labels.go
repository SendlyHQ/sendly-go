package sendly

import (
	"context"
	"fmt"
	"net/url"
)

// LabelsService handles label-related API operations.
type LabelsService struct {
	client *Client
}

// Create creates a new label.
func (s *LabelsService) Create(ctx context.Context, req *CreateLabelRequest) (*Label, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.Name == "" {
		return nil, &ValidationError{APIError: APIError{Message: "name is required"}}
	}

	var resp Label
	err := s.client.request(ctx, "POST", "/labels", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// List retrieves all labels.
func (s *LabelsService) List(ctx context.Context) (*LabelListResponse, error) {
	var resp LabelListResponse
	err := s.client.request(ctx, "GET", "/labels", nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Delete deletes a label by ID.
func (s *LabelsService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return &ValidationError{APIError: APIError{Message: "label ID is required"}}
	}

	path := fmt.Sprintf("/labels/%s", url.PathEscape(id))
	return s.client.request(ctx, "DELETE", path, nil, nil)
}
