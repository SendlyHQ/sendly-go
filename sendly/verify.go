package sendly

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// VerifyService provides OTP verification operations.
type VerifyService struct {
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
