package sendly

import (
	"context"
	"encoding/json"
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

// TemplatePreview represents a template rendered with sample values.
type TemplatePreview struct {
	TemplateID     string `json:"template_id"`
	OriginalText   string `json:"original_text"`
	RenderedText   string `json:"rendered_text"`
	CharacterCount int    `json:"character_count"`
	SegmentCount   int    `json:"segment_count"`

	// ID mirrors TemplateID.
	//
	// Deprecated: use TemplateID.
	ID string `json:"id"`
	// Name is always empty.
	//
	// Deprecated: a preview carries no template name. Read it from Get.
	Name string `json:"name"`
	// PreviewText mirrors RenderedText.
	//
	// Deprecated: use RenderedText.
	PreviewText string `json:"preview_text"`
	// Variables is always empty.
	//
	// Deprecated: a preview carries no variable definitions. Read them from
	// Get as Template.Variables.
	Variables []TemplateVariable `json:"variables"`
}

// UnmarshalJSON decodes a preview payload and mirrors it onto the deprecated
// fields.
func (p *TemplatePreview) UnmarshalJSON(data []byte) error {
	type templatePreviewAlias TemplatePreview
	var raw templatePreviewAlias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*p = TemplatePreview(raw)
	if p.ID == "" {
		p.ID = p.TemplateID
	}
	if p.PreviewText == "" {
		p.PreviewText = p.RenderedText
	}
	return nil
}

// List retrieves all templates.
func (s *TemplatesService) List(ctx context.Context) (*TemplateListResponse, error) {
	var resp TemplateListResponse
	err := s.client.request(ctx, "GET", "/templates", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Presets retrieves preset templates only.
func (s *TemplatesService) Presets(ctx context.Context) (*TemplateListResponse, error) {
	var resp TemplateListResponse
	err := s.client.request(ctx, "GET", "/templates/presets", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get retrieves a template by ID.
func (s *TemplatesService) Get(ctx context.Context, id string) (*Template, error) {
	var resp Template
	err := s.client.request(ctx, "GET", fmt.Sprintf("/templates/%s", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Create creates a new template.
func (s *TemplatesService) Create(ctx context.Context, req *CreateTemplateRequest) (*Template, error) {
	var resp Template
	err := s.client.request(ctx, "POST", "/templates", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update updates a template.
func (s *TemplatesService) Update(ctx context.Context, id string, req *UpdateTemplateRequest) (*Template, error) {
	var resp Template
	err := s.client.request(ctx, "PATCH", fmt.Sprintf("/templates/%s", id), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Publish publishes a draft template.
func (s *TemplatesService) Publish(ctx context.Context, id string) (*Template, error) {
	var resp Template
	err := s.client.request(ctx, "POST", fmt.Sprintf("/templates/%s/publish", id), nil, &resp)
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
	err := s.client.request(ctx, "POST", fmt.Sprintf("/templates/%s/preview", id), body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Delete deletes a template.
func (s *TemplatesService) Delete(ctx context.Context, id string) error {
	return s.client.request(ctx, "DELETE", fmt.Sprintf("/templates/%s", id), nil, nil)
}

// CloneTemplateRequest represents the parameters for cloning a template.
type CloneTemplateRequest struct {
	Name string `json:"name,omitempty"`
}

// Clone copies an existing template into a new draft.
//
// Not available yet: the versioned API serves no clone route, so this call
// fails with a *NotFoundError. To copy a template today, read it with Get and
// pass its Text to Create.
func (s *TemplatesService) Clone(ctx context.Context, id string, req *CloneTemplateRequest) (*Template, error) {
	body := map[string]interface{}{}
	if req != nil && req.Name != "" {
		body["name"] = req.Name
	}

	var resp Template
	err := s.client.request(ctx, "POST", fmt.Sprintf("/templates/%s/clone", id), body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Generate generates a template from a description using AI.
func (s *TemplatesService) Generate(ctx context.Context, req *GenerateTemplateRequest) (*GeneratedTemplate, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.Description == "" {
		return nil, &ValidationError{APIError: APIError{Message: "description is required"}}
	}

	var resp GeneratedTemplate
	err := s.client.request(ctx, "POST", "/templates/generate", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
