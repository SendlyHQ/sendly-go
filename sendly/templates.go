package sendly

import (
	"context"
	"fmt"
)

// TemplatesService provides template management operations.
type TemplatesService struct {
	client *Client
}

// TemplateVariable represents a variable in a template.
type TemplateVariable struct {
	Key      string `json:"key"`
	Type     string `json:"type"`
	Fallback string `json:"fallback,omitempty"`
}

// Template represents an SMS template.
type Template struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Text        string             `json:"text"`
	Variables   []TemplateVariable `json:"variables"`
	IsPreset    bool               `json:"is_preset"`
	PresetSlug  string             `json:"preset_slug,omitempty"`
	Status      string             `json:"status"`
	Version     int                `json:"version"`
	PublishedAt string             `json:"published_at,omitempty"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

// TemplateListResponse is the response from listing templates.
type TemplateListResponse struct {
	Templates []Template `json:"templates"`
}

// CreateTemplateRequest represents the parameters for creating a template.
type CreateTemplateRequest struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

// UpdateTemplateRequest represents the parameters for updating a template.
type UpdateTemplateRequest struct {
	Name string `json:"name,omitempty"`
	Text string `json:"text,omitempty"`
}

// TemplatePreview represents a template preview.
type TemplatePreview struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	OriginalText string             `json:"original_text"`
	PreviewText  string             `json:"preview_text"`
	Variables    []TemplateVariable `json:"variables"`
}

// List retrieves all templates.
func (s *TemplatesService) List(ctx context.Context) (*TemplateListResponse, error) {
	var resp TemplateListResponse
	err := s.client.doRequest(ctx, "GET", "/templates", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Presets retrieves preset templates only.
func (s *TemplatesService) Presets(ctx context.Context) (*TemplateListResponse, error) {
	var resp TemplateListResponse
	err := s.client.doRequest(ctx, "GET", "/templates/presets", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get retrieves a template by ID.
func (s *TemplatesService) Get(ctx context.Context, id string) (*Template, error) {
	var resp Template
	err := s.client.doRequest(ctx, "GET", fmt.Sprintf("/templates/%s", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Create creates a new template.
func (s *TemplatesService) Create(ctx context.Context, req *CreateTemplateRequest) (*Template, error) {
	var resp Template
	err := s.client.doRequest(ctx, "POST", "/templates", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update updates a template.
func (s *TemplatesService) Update(ctx context.Context, id string, req *UpdateTemplateRequest) (*Template, error) {
	var resp Template
	err := s.client.doRequest(ctx, "PATCH", fmt.Sprintf("/templates/%s", id), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Publish publishes a draft template.
func (s *TemplatesService) Publish(ctx context.Context, id string) (*Template, error) {
	var resp Template
	err := s.client.doRequest(ctx, "POST", fmt.Sprintf("/templates/%s/publish", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Preview previews a template with sample values.
func (s *TemplatesService) Preview(ctx context.Context, id string, variables map[string]string) (*TemplatePreview, error) {
	body := map[string]interface{}{}
	if variables != nil {
		body["variables"] = variables
	}

	var resp TemplatePreview
	err := s.client.doRequest(ctx, "POST", fmt.Sprintf("/templates/%s/preview", id), body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Delete deletes a template.
func (s *TemplatesService) Delete(ctx context.Context, id string) error {
	return s.client.doRequest(ctx, "DELETE", fmt.Sprintf("/templates/%s", id), nil, nil)
}
