package sendly

import (
	"context"
	"net/url"
)

// WhatsAppService connects numbers to WhatsApp and manages the channel:
// start a signup, list senders, manage Meta-reviewed message templates, and
// check 24-hour customer-service windows. Send WhatsApp messages with
// MessagesService.SendWhatsApp.
//
// Connecting a number is a one-time $19 setup (no monthly fee) and always
// ends with a human step: Signup.Create returns a connect URL a person must
// open in a browser and complete with a Facebook login to link their
// WhatsApp Business Account.
type WhatsAppService struct {
	client *Client
	// Signup connects numbers to WhatsApp.
	Signup *WhatsAppSignupService
	// Senders lists the numbers connected (or connecting) to WhatsApp.
	Senders *WhatsAppSendersService
	// Templates manages Meta-reviewed message templates.
	Templates *WhatsAppTemplatesService
}

// WhatsAppSignupService connects numbers to WhatsApp.
type WhatsAppSignupService struct {
	client *Client
}

// WhatsAppSendersService lists the numbers connected (or connecting) to WhatsApp.
type WhatsAppSendersService struct {
	client *Client
}

// WhatsAppTemplatesService manages Meta-reviewed WhatsApp message templates.
type WhatsAppTemplatesService struct {
	client *Client
}

// WhatsAppSignupSession is the response from starting a WhatsApp signup.
// Hand ConnectURL to a human — they open it in a browser and log in with
// Facebook to link their WhatsApp Business Account. Poll
// WhatsAppSignupService.Get with ID until the status is "active".
type WhatsAppSignupSession struct {
	// ID is the unique signup identifier.
	ID string `json:"id"`
	// ConnectURL is the hosted connect page URL. A person must open this in a browser.
	ConnectURL string `json:"connectUrl"`
	// Status is "initiated", "registering", "active", "failed", or "expired".
	Status string `json:"status"`
}

// WhatsAppSignup is the status of a WhatsApp signup.
type WhatsAppSignup struct {
	// ID is the unique signup identifier.
	ID string `json:"id"`
	// Status is "initiated", "registering", "active", "failed", or "expired".
	Status string `json:"status"`
	// PhoneNumber is the number being connected, in E.164 format.
	PhoneNumber string `json:"phoneNumber"`
	// BusinessAccountID is the customer's WhatsApp Business Account id once
	// linked; nil before the human completes the connect step.
	BusinessAccountID *string `json:"businessAccountId,omitempty"`
	// FailureReasons is why the signup failed, when Status is "failed".
	FailureReasons []string `json:"failureReasons,omitempty"`
	// UpdatedAt is the ISO 8601 timestamp of the last status change.
	UpdatedAt string `json:"updatedAt"`
}

// WhatsAppSender is a number connected (or connecting) to WhatsApp.
type WhatsAppSender struct {
	// PhoneNumber is the sender, in E.164 format.
	PhoneNumber string `json:"phoneNumber"`
	// DisplayName is the name recipients see — chosen during the connect
	// flow and reviewed by Meta; nil until set.
	DisplayName *string `json:"displayName,omitempty"`
	// Status is "pending", "active", or "suspended".
	Status string `json:"status"`
	// QualityRating is the Meta quality rating (e.g. "GREEN"), nil before the first rating.
	QualityRating *string `json:"qualityRating,omitempty"`
	// CreatedAt is the ISO 8601 timestamp when the sender was connected.
	CreatedAt string `json:"createdAt"`
}

// WhatsAppSenderListResponse wraps the workspace's WhatsApp senders.
type WhatsAppSenderListResponse struct {
	Senders []WhatsAppSender `json:"senders"`
}

// WhatsAppTemplateButton is a button on a WhatsApp template. Type is "url"
// (URL is required and may contain a {{1}} placeholder — supply Example
// values for review), "quick_reply" (e.g. a "Stop promotions" opt-out,
// recommended on marketing templates), or "otp" (copy-code button, required
// on AUTHENTICATION templates).
type WhatsAppTemplateButton struct {
	Type string `json:"type"`
	Text string `json:"text"`
	// URL is the link target (url buttons only); may contain a {{1}} placeholder.
	URL string `json:"url,omitempty"`
	// Example values for a url placeholder, for Meta review.
	Example []string `json:"example,omitempty"`
}

// CreateWhatsAppTemplateRequest creates a template and submits it to Meta
// for review. Sender (the WhatsApp-connected sending number in E.164),
// Name (lowercase letters, digits, and underscores, e.g. "order_shipped"),
// Language (e.g. "en_US"), Category, and Body are required. Use {{1}},
// {{2}}, … in Body for variables; every placeholder needs an example value
// in Examples.
type CreateWhatsAppTemplateRequest struct {
	Sender string `json:"sender"`
	Name   string `json:"name"`
	// Language is the template language code (e.g. "en_US").
	Language string `json:"language"`
	// Category is "AUTHENTICATION", "UTILITY", or "MARKETING" — drives Meta
	// review rules and pricing.
	Category string                   `json:"category"`
	Body     string                   `json:"body"`
	Footer   string                   `json:"footer,omitempty"`
	Header   string                   `json:"header,omitempty"`
	Buttons  []WhatsAppTemplateButton `json:"buttons,omitempty"`
	// Examples are example values for body placeholders, keyed by
	// placeholder number: {"1": "Acme Inc", "2": "#4821"}. Required when the
	// body has variables.
	Examples map[string]string `json:"examples,omitempty"`
}

// UpdateWhatsAppTemplateRequest edits a template. Supply only the fields to
// change; omitted fields keep their current value.
type UpdateWhatsAppTemplateRequest struct {
	Body     string                   `json:"body,omitempty"`
	Footer   string                   `json:"footer,omitempty"`
	Header   string                   `json:"header,omitempty"`
	Buttons  []WhatsAppTemplateButton `json:"buttons,omitempty"`
	Examples map[string]string        `json:"examples,omitempty"`
}

// WhatsAppTemplate is a WhatsApp message template.
type WhatsAppTemplate struct {
	// ID is the unique template identifier.
	ID string `json:"id"`
	// Name is the template name.
	Name string `json:"name"`
	// Language is the template language code.
	Language string `json:"language"`
	// Category is "AUTHENTICATION", "UTILITY", or "MARKETING" (Meta may
	// reclassify; this value drives pricing).
	Category string `json:"category"`
	// Status is "PENDING" (Meta review usually takes 24-48h), "APPROVED",
	// "REJECTED" (edit with Update to resubmit), "PAUSED", or "DISABLED".
	Status string `json:"status"`
	// QualityRating is the Meta quality rating (e.g. "GREEN"), nil before the first rating.
	QualityRating *string `json:"qualityRating,omitempty"`
	// RejectionReason is why Meta rejected the template, when Status is "REJECTED".
	RejectionReason *string `json:"rejectionReason,omitempty"`
	// CreatedAt is the ISO 8601 timestamp when the template was created.
	CreatedAt string `json:"createdAt"`
	// UpdatedAt is the ISO 8601 timestamp when the template was last updated.
	UpdatedAt string `json:"updatedAt"`
	// Warnings are non-blocking submission warnings (e.g. a marketing
	// template without an opt-out button). Present on create responses when
	// applicable.
	Warnings []string `json:"warnings,omitempty"`
}

// WhatsAppTemplateListResponse wraps the workspace's WhatsApp templates.
type WhatsAppTemplateListResponse struct {
	Templates []WhatsAppTemplate `json:"templates"`
}

// WhatsAppTemplateDeletedResponse confirms a template deletion.
type WhatsAppTemplateDeletedResponse struct {
	// ID is the deleted template's id.
	ID string `json:"id"`
	// Deleted is always true.
	Deleted bool `json:"deleted"`
}

// WhatsAppWindow reports whether a 24-hour customer-service window is open.
type WhatsAppWindow struct {
	// Open is true when a 24-hour customer-service window is currently open.
	Open bool `json:"open"`
	// ExpiresAt is when the window closes (ISO 8601), nil when no window is open.
	ExpiresAt *string `json:"expiresAt,omitempty"`
}

// Create starts connecting a number to WhatsApp. The number must be an
// active number in your workspace, in E.164 format.
//
// Charges a one-time $19 setup fee (no monthly fee) and returns a connect
// URL. Completing the connection requires a human: hand the URL to your
// user — they open it in a browser and log in with Facebook to link their
// WhatsApp Business Account. Then poll Get until the status is "active".
// Calling again for a number with an in-flight signup returns the existing
// signup (same ConnectURL) without charging again. Requires a live API key.
func (s *WhatsAppSignupService) Create(ctx context.Context, phoneNumber string) (*WhatsAppSignupSession, error) {
	if phoneNumber == "" {
		return nil, &ValidationError{APIError: APIError{Message: "phoneNumber is required"}}
	}

	body := map[string]string{"phoneNumber": phoneNumber}
	var resp WhatsAppSignupSession
	if err := s.client.request(ctx, "POST", "/whatsapp/signup", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get fetches the status of a WhatsApp signup. Poll this after Create until
// the status is "active" ("failed" sets FailureReasons).
func (s *WhatsAppSignupService) Get(ctx context.Context, id string) (*WhatsAppSignup, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "signup ID is required"}}
	}

	var resp WhatsAppSignup
	if err := s.client.request(ctx, "GET", "/whatsapp/signup/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List returns the numbers connected (or connecting) to WhatsApp on the
// workspace, newest first, with connection status and quality rating. An
// empty list means no number is connected yet — start one with
// WhatsAppSignupService.Create.
func (s *WhatsAppSendersService) List(ctx context.Context) (*WhatsAppSenderListResponse, error) {
	var resp WhatsAppSenderListResponse
	if err := s.client.request(ctx, "GET", "/whatsapp/senders", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List returns the workspace's WhatsApp templates with review status and
// quality rating.
func (s *WhatsAppTemplatesService) List(ctx context.Context) (*WhatsAppTemplateListResponse, error) {
	var resp WhatsAppTemplateListResponse
	if err := s.client.request(ctx, "GET", "/whatsapp/templates", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Create creates a template and submits it to Meta for review. Review
// usually takes 24-48h; the template is usable once its status is
// "APPROVED". Requires a live API key.
func (s *WhatsAppTemplatesService) Create(ctx context.Context, req *CreateWhatsAppTemplateRequest) (*WhatsAppTemplate, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.Sender == "" {
		return nil, &ValidationError{APIError: APIError{Message: "sender is required"}}
	}
	if req.Name == "" {
		return nil, &ValidationError{APIError: APIError{Message: "name is required"}}
	}
	if req.Language == "" {
		return nil, &ValidationError{APIError: APIError{Message: "language is required"}}
	}
	if req.Category == "" {
		return nil, &ValidationError{APIError: APIError{Message: "category is required"}}
	}
	if req.Body == "" {
		return nil, &ValidationError{APIError: APIError{Message: "body is required"}}
	}

	var resp WhatsAppTemplate
	if err := s.client.request(ctx, "POST", "/whatsapp/templates", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update edits an APPROVED or REJECTED template and resubmits it for
// review. This is the recovery path for rejections: template names are
// locked for ~30 days after deletion, so editing a rejected template
// (rather than deleting and re-creating it) is the way to fix it. The
// updated template goes back to "PENDING" review. Requires a live API key.
func (s *WhatsAppTemplatesService) Update(ctx context.Context, id string, req *UpdateWhatsAppTemplateRequest) (*WhatsAppTemplate, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "template ID is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	var resp WhatsAppTemplate
	if err := s.client.request(ctx, "PATCH", "/whatsapp/templates/"+url.PathEscape(id), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Delete deletes a template. Meta locks a deleted template's name for ~30
// days — re-creating it fails until the lock lifts. To fix a rejected
// template, prefer Update. Requires a live API key.
func (s *WhatsAppTemplatesService) Delete(ctx context.Context, id string) (*WhatsAppTemplateDeletedResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "template ID is required"}}
	}

	var resp WhatsAppTemplateDeletedResponse
	if err := s.client.request(ctx, "DELETE", "/whatsapp/templates/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Window checks whether a 24-hour customer-service window is open between
// one of your WhatsApp senders (from) and a recipient (to), both in E.164
// format. Free-form text and media only deliver while a window is open (it
// opens when the recipient messages you and lasts 24h from their last
// inbound message). Outside a window, send an approved template.
func (s *WhatsAppService) Window(ctx context.Context, from, to string) (*WhatsAppWindow, error) {
	if from == "" {
		return nil, &ValidationError{APIError: APIError{Message: "from is required"}}
	}
	if to == "" {
		return nil, &ValidationError{APIError: APIError{Message: "to is required"}}
	}

	path := "/whatsapp/window" + buildQueryString(map[string]string{"from": from, "to": to})
	var resp WhatsAppWindow
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
