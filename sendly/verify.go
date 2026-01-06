package sendly

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// VerifyService provides OTP verification operations.
type VerifyService struct {
	client   *Client
	Sessions *SessionsService
}

// SessionsService provides hosted verification flow operations.
type SessionsService struct {
	client *Client
}

// SendVerificationRequest represents the parameters for sending a verification.
type SendVerificationRequest struct {
	To          string `json:"to"`
	TemplateID  string `json:"template_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	AppName     string `json:"app_name,omitempty"`
	TimeoutSecs int    `json:"timeout_secs,omitempty"`
	CodeLength  int    `json:"code_length,omitempty"`
}

// SendVerificationResponse represents the response from sending a verification.
type SendVerificationResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Phone       string `json:"phone"`
	ExpiresAt   string `json:"expires_at"`
	Sandbox     bool   `json:"sandbox"`
	SandboxCode string `json:"sandbox_code,omitempty"`
	Message     string `json:"message,omitempty"`
}

// CheckVerificationRequest represents the parameters for checking a verification.
type CheckVerificationRequest struct {
	Code string `json:"code"`
}

// CheckVerificationResponse represents the response from checking a verification.
type CheckVerificationResponse struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	Phone             string `json:"phone"`
	VerifiedAt        string `json:"verified_at,omitempty"`
	RemainingAttempts int    `json:"remaining_attempts,omitempty"`
}

// Verification represents a verification record.
type Verification struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	Phone          string `json:"phone"`
	DeliveryStatus string `json:"delivery_status"`
	Attempts       int    `json:"attempts"`
	MaxAttempts    int    `json:"max_attempts"`
	ExpiresAt      string `json:"expires_at"`
	VerifiedAt     string `json:"verified_at,omitempty"`
	CreatedAt      string `json:"created_at"`
	Sandbox        bool   `json:"sandbox"`
	AppName        string `json:"app_name,omitempty"`
	TemplateID     string `json:"template_id,omitempty"`
	ProfileID      string `json:"profile_id,omitempty"`
}

// VerificationListOptions are options for listing verifications.
type VerificationListOptions struct {
	Limit  int
	Status string
}

// VerificationListResponse is the response from listing verifications.
type VerificationListResponse struct {
	Verifications []Verification `json:"verifications"`
	Pagination    struct {
		Limit   int  `json:"limit"`
		HasMore bool `json:"has_more"`
	} `json:"pagination"`
}

// CreateSessionRequest represents the parameters for creating a verification session.
type CreateSessionRequest struct {
	SuccessURL string                 `json:"success_url"`
	CancelURL  string                 `json:"cancel_url,omitempty"`
	BrandName  string                 `json:"brand_name,omitempty"`
	BrandColor string                 `json:"brand_color,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// VerifySession represents a hosted verification session.
type VerifySession struct {
	ID             string                 `json:"id"`
	URL            string                 `json:"url"`
	Status         string                 `json:"status"`
	SuccessURL     string                 `json:"success_url"`
	CancelURL      string                 `json:"cancel_url,omitempty"`
	BrandName      string                 `json:"brand_name,omitempty"`
	BrandColor     string                 `json:"brand_color,omitempty"`
	Phone          string                 `json:"phone,omitempty"`
	VerificationID string                 `json:"verification_id,omitempty"`
	Token          string                 `json:"token,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	ExpiresAt      string                 `json:"expires_at"`
	CreatedAt      string                 `json:"created_at"`
}

// ValidateSessionRequest represents the parameters for validating a session token.
type ValidateSessionRequest struct {
	Token string `json:"token"`
}

// ValidateSessionResponse represents the response from validating a session token.
type ValidateSessionResponse struct {
	Valid      bool                   `json:"valid"`
	SessionID  string                 `json:"session_id,omitempty"`
	Phone      string                 `json:"phone,omitempty"`
	VerifiedAt string                 `json:"verified_at,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Create creates a hosted verification session.
func (s *SessionsService) Create(ctx context.Context, req *CreateSessionRequest) (*VerifySession, error) {
	var resp VerifySession
	err := s.client.doRequest(ctx, "POST", "/verify/sessions", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Validate validates a session token after user completes verification.
func (s *SessionsService) Validate(ctx context.Context, req *ValidateSessionRequest) (*ValidateSessionResponse, error) {
	var resp ValidateSessionResponse
	err := s.client.doRequest(ctx, "POST", "/verify/sessions/validate", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Send sends an OTP verification code.
func (s *VerifyService) Send(ctx context.Context, req *SendVerificationRequest) (*SendVerificationResponse, error) {
	var resp SendVerificationResponse
	err := s.client.doRequest(ctx, "POST", "/verify", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Resend resends an OTP verification code.
func (s *VerifyService) Resend(ctx context.Context, id string) (*SendVerificationResponse, error) {
	var resp SendVerificationResponse
	err := s.client.doRequest(ctx, "POST", fmt.Sprintf("/verify/%s/resend", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Check verifies an OTP code.
func (s *VerifyService) Check(ctx context.Context, id string, req *CheckVerificationRequest) (*CheckVerificationResponse, error) {
	var resp CheckVerificationResponse
	err := s.client.doRequest(ctx, "POST", fmt.Sprintf("/verify/%s/check", id), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get retrieves a verification by ID.
func (s *VerifyService) Get(ctx context.Context, id string) (*Verification, error) {
	var resp Verification
	err := s.client.doRequest(ctx, "GET", fmt.Sprintf("/verify/%s", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// List retrieves recent verifications.
func (s *VerifyService) List(ctx context.Context, opts *VerificationListOptions) (*VerificationListResponse, error) {
	path := "/verify"
	if opts != nil {
		params := url.Values{}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
	}

	var resp VerificationListResponse
	err := s.client.doRequest(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
