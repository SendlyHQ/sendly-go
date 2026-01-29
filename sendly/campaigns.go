package sendly

import (
	"context"
	"fmt"
	"strconv"
)

type CampaignsService struct {
	client *Client
}

type CampaignStatus string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusScheduled CampaignStatus = "scheduled"
	CampaignStatusSending   CampaignStatus = "sending"
	CampaignStatusSent      CampaignStatus = "sent"
	CampaignStatusPaused    CampaignStatus = "paused"
	CampaignStatusCancelled CampaignStatus = "cancelled"
	CampaignStatusFailed    CampaignStatus = "failed"
)

type Campaign struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Text             string   `json:"text"`
	TemplateID       *string  `json:"template_id,omitempty"`
	ContactListIDs   []string `json:"contact_list_ids"`
	Status           string   `json:"status"`
	RecipientCount   int      `json:"recipient_count"`
	SentCount        int      `json:"sent_count"`
	DeliveredCount   int      `json:"delivered_count"`
	FailedCount      int      `json:"failed_count"`
	EstimatedCredits *float64 `json:"estimated_credits,omitempty"`
	CreditsUsed      *float64 `json:"credits_used,omitempty"`
	ScheduledAt      *string  `json:"scheduled_at,omitempty"`
	Timezone         *string  `json:"timezone,omitempty"`
	StartedAt        *string  `json:"started_at,omitempty"`
	CompletedAt      *string  `json:"completed_at,omitempty"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type CampaignListResponse struct {
	Campaigns []Campaign `json:"campaigns"`
	Total     int        `json:"total"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

type CampaignPreview struct {
	RecipientCount   int                          `json:"recipient_count"`
	EstimatedCredits float64                      `json:"estimated_credits"`
	EstimatedCost    float64                      `json:"estimated_cost"`
	BlockedCount     *int                         `json:"blocked_count,omitempty"`
	SendableCount    *int                         `json:"sendable_count,omitempty"`
	ByCountry        map[string]CountryAccessInfo `json:"by_country,omitempty"`
	Warnings         []string                     `json:"warnings,omitempty"`
	MessagingProfile *MessagingProfileAccess      `json:"messaging_profile,omitempty"`
}

type CountryAccessInfo struct {
	Count         int    `json:"count"`
	Credits       int    `json:"credits"`
	Allowed       bool   `json:"allowed"`
	BlockedReason string `json:"blocked_reason,omitempty"`
}

type MessagingProfileAccess struct {
	CanSendDomestic      bool    `json:"can_send_domestic"`
	CanSendInternational bool    `json:"can_send_international"`
	VerificationType     *string `json:"verification_type"`
	VerificationStatus   *string `json:"verification_status"`
}

type CreateCampaignRequest struct {
	Name           string   `json:"name"`
	Text           string   `json:"text"`
	ContactListIDs []string `json:"contact_list_ids"`
	TemplateID     string   `json:"template_id,omitempty"`
}

type UpdateCampaignRequest struct {
	Name           string   `json:"name,omitempty"`
	Text           string   `json:"text,omitempty"`
	ContactListIDs []string `json:"contact_list_ids,omitempty"`
	TemplateID     string   `json:"template_id,omitempty"`
}

type ListCampaignsRequest struct {
	Limit  int
	Offset int
	Status CampaignStatus
}

type ScheduleCampaignRequest struct {
	ScheduledAt string `json:"scheduled_at"`
	Timezone    string `json:"timezone,omitempty"`
}

func (s *CampaignsService) List(ctx context.Context, req *ListCampaignsRequest) (*CampaignListResponse, error) {
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

	path := "/campaigns" + buildQueryString(params)
	var resp CampaignListResponse
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CampaignsService) Get(ctx context.Context, id string) (*Campaign, error) {
	var resp Campaign
	err := s.client.request(ctx, "GET", fmt.Sprintf("/campaigns/%s", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CampaignsService) Create(ctx context.Context, req *CreateCampaignRequest) (*Campaign, error) {
	var resp Campaign
	err := s.client.request(ctx, "POST", "/campaigns", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CampaignsService) Update(ctx context.Context, id string, req *UpdateCampaignRequest) (*Campaign, error) {
	var resp Campaign
	err := s.client.request(ctx, "PATCH", fmt.Sprintf("/campaigns/%s", id), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CampaignsService) Delete(ctx context.Context, id string) error {
	return s.client.request(ctx, "DELETE", fmt.Sprintf("/campaigns/%s", id), nil, nil)
}

func (s *CampaignsService) Preview(ctx context.Context, id string) (*CampaignPreview, error) {
	var resp CampaignPreview
	err := s.client.request(ctx, "GET", fmt.Sprintf("/campaigns/%s/preview", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CampaignsService) Send(ctx context.Context, id string) (*Campaign, error) {
	var resp Campaign
	err := s.client.request(ctx, "POST", fmt.Sprintf("/campaigns/%s/send", id), map[string]interface{}{}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CampaignsService) Schedule(ctx context.Context, id string, req *ScheduleCampaignRequest) (*Campaign, error) {
	var resp Campaign
	err := s.client.request(ctx, "POST", fmt.Sprintf("/campaigns/%s/schedule", id), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CampaignsService) Cancel(ctx context.Context, id string) (*Campaign, error) {
	var resp Campaign
	err := s.client.request(ctx, "POST", fmt.Sprintf("/campaigns/%s/cancel", id), map[string]interface{}{}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CampaignsService) Clone(ctx context.Context, id string) (*Campaign, error) {
	var resp Campaign
	err := s.client.request(ctx, "POST", fmt.Sprintf("/campaigns/%s/clone", id), map[string]interface{}{}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
