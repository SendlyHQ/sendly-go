package sendly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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
	// CampaignStatusSent is never returned: a campaign that has been sent is
	// CampaignStatusCompleted. List treats a "sent" filter as "completed".
	//
	// Deprecated: use CampaignStatusCompleted.
	CampaignStatusSent CampaignStatus = "sent"
	// CampaignStatusPaused is never returned: campaigns cannot be paused.
	//
	// Deprecated: no campaign has this status.
	CampaignStatusPaused    CampaignStatus = "paused"
	CampaignStatusCancelled CampaignStatus = "cancelled"
	CampaignStatusFailed    CampaignStatus = "failed"
	// CampaignStatusCompleted is the status of a campaign that has been sent.
	CampaignStatusCompleted CampaignStatus = "completed"
)

type Campaign struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Text string `json:"text"`
	// TemplateID is always nil: a campaign keeps the template's text, not a
	// reference to the template.
	//
	// Deprecated: the API does not return it.
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
	// StartedAt is when the send began.
	StartedAt   *string `json:"started_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	// BatchID is the batch the campaign's messages went out in, once sent.
	// Read the per-message results with Messages.GetBatch.
	BatchID string `json:"batchId,omitempty"`
	// FromSender is the sender the campaign sends from, when one was set.
	FromSender *string `json:"fromSender,omitempty"`
	// TargetType is who the campaign targets, usually "contact_list".
	TargetType string `json:"targetType,omitempty"`
}

// UnmarshalJSON decodes a campaign as the API returns it, with camelCase
// keys (messageText, totalRecipients, sentCount, sentAt and so on), and
// still reads the snake_case keys the struct is tagged with.
func (c *Campaign) UnmarshalJSON(data []byte) error {
	type campaignAlias Campaign
	var raw campaignAlias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var camel struct {
		MessageText      *string  `json:"messageText"`
		TargetListID     *string  `json:"targetListId"`
		TotalRecipients  *int     `json:"totalRecipients"`
		SentCount        *int     `json:"sentCount"`
		DeliveredCount   *int     `json:"deliveredCount"`
		FailedCount      *int     `json:"failedCount"`
		EstimatedCredits *float64 `json:"estimatedCredits"`
		CreditsUsed      *float64 `json:"creditsUsed"`
		ScheduledAt      *string  `json:"scheduledAt"`
		SentAt           *string  `json:"sentAt"`
		CompletedAt      *string  `json:"completedAt"`
		CreatedAt        *string  `json:"createdAt"`
		UpdatedAt        *string  `json:"updatedAt"`
	}
	if err := json.Unmarshal(data, &camel); err != nil {
		return err
	}

	*c = Campaign(raw)
	if camel.MessageText != nil {
		c.Text = *camel.MessageText
	}
	if camel.TargetListID != nil && *camel.TargetListID != "" {
		c.ContactListIDs = []string{*camel.TargetListID}
	}
	if camel.TotalRecipients != nil {
		c.RecipientCount = *camel.TotalRecipients
	}
	if camel.SentCount != nil {
		c.SentCount = *camel.SentCount
	}
	if camel.DeliveredCount != nil {
		c.DeliveredCount = *camel.DeliveredCount
	}
	if camel.FailedCount != nil {
		c.FailedCount = *camel.FailedCount
	}
	if camel.EstimatedCredits != nil {
		c.EstimatedCredits = camel.EstimatedCredits
	}
	if camel.CreditsUsed != nil {
		c.CreditsUsed = camel.CreditsUsed
	}
	if camel.ScheduledAt != nil {
		c.ScheduledAt = camel.ScheduledAt
	}
	if camel.SentAt != nil {
		c.StartedAt = camel.SentAt
	}
	if camel.CompletedAt != nil {
		c.CompletedAt = camel.CompletedAt
	}
	if camel.CreatedAt != nil {
		c.CreatedAt = *camel.CreatedAt
	}
	if camel.UpdatedAt != nil {
		c.UpdatedAt = *camel.UpdatedAt
	}
	return nil
}

type CampaignListResponse struct {
	Campaigns []Campaign `json:"campaigns"`
	Total     int        `json:"total"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

type CampaignPreview struct {
	RecipientCount   int     `json:"recipient_count"`
	EstimatedCredits float64 `json:"estimated_credits"`
	// EstimatedCost is always 0.
	//
	// Deprecated: the API does not return a cost; use EstimatedCredits.
	EstimatedCost    float64                      `json:"estimated_cost"`
	BlockedCount     *int                         `json:"blocked_count,omitempty"`
	SendableCount    *int                         `json:"sendable_count,omitempty"`
	ByCountry        map[string]CountryAccessInfo `json:"by_country,omitempty"`
	Warnings         []string                     `json:"warnings,omitempty"`
	MessagingProfile *MessagingProfileAccess      `json:"messaging_profile,omitempty"`
	// CurrentBalance is the credit balance the send would draw on.
	CurrentBalance int `json:"currentBalance"`
	// HasEnoughCredits reports whether CurrentBalance covers
	// EstimatedCredits. It is always true with a test key.
	HasEnoughCredits bool `json:"hasEnoughCredits"`
	// OptedOutCount is the number of contacts left out because they opted out.
	OptedOutCount int `json:"optedOutCount"`
	// InvalidCount is the number of contacts whose phone number is missing
	// or too short.
	InvalidCount int `json:"invalidCount"`
	// InvalidNumberCount is the number of contacts left out because they are
	// flagged as unable to receive SMS.
	InvalidNumberCount int `json:"invalidNumberCount"`
	// LandlineCount is how many of InvalidNumberCount are landlines.
	LandlineCount int `json:"landlineCount"`
	// SampleRecipients is up to five of the recipients.
	SampleRecipients []CampaignSampleRecipient `json:"sampleRecipients,omitempty"`
}

// CampaignSampleRecipient is one of the recipients a campaign preview shows.
type CampaignSampleRecipient struct {
	Phone string `json:"phone"`
	Name  string `json:"name,omitempty"`
}

// UnmarshalJSON decodes a preview as the API returns it, with camelCase keys
// (totalRecipients or recipientCount, estimatedCredits, blockedCount,
// sendableCount, byCountry, messagingProfile), and still reads the
// snake_case keys the struct is tagged with.
func (p *CampaignPreview) UnmarshalJSON(data []byte) error {
	type campaignPreviewAlias CampaignPreview
	var raw campaignPreviewAlias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var camel struct {
		RecipientCount   *int                         `json:"recipientCount"`
		TotalRecipients  *int                         `json:"totalRecipients"`
		EstimatedCredits *float64                     `json:"estimatedCredits"`
		BlockedCount     *int                         `json:"blockedCount"`
		SendableCount    *int                         `json:"sendableCount"`
		ByCountry        map[string]CountryAccessInfo `json:"byCountry"`
		MessagingProfile *MessagingProfileAccess      `json:"messagingProfile"`
	}
	if err := json.Unmarshal(data, &camel); err != nil {
		return err
	}

	*p = CampaignPreview(raw)
	if camel.RecipientCount != nil {
		p.RecipientCount = *camel.RecipientCount
	} else if camel.TotalRecipients != nil {
		p.RecipientCount = *camel.TotalRecipients
	}
	if camel.EstimatedCredits != nil {
		p.EstimatedCredits = *camel.EstimatedCredits
	}
	if camel.BlockedCount != nil {
		p.BlockedCount = camel.BlockedCount
	}
	if camel.SendableCount != nil {
		p.SendableCount = camel.SendableCount
	}
	if camel.ByCountry != nil {
		p.ByCountry = camel.ByCountry
	}
	if camel.MessagingProfile != nil {
		p.MessagingProfile = camel.MessagingProfile
	}
	return nil
}

type CountryAccessInfo struct {
	Count         int    `json:"count"`
	Credits       int    `json:"credits"`
	Allowed       bool   `json:"allowed"`
	BlockedReason string `json:"blocked_reason,omitempty"`
}

// UnmarshalJSON reads blockedReason as the API sends it, and still reads
// blocked_reason.
func (c *CountryAccessInfo) UnmarshalJSON(data []byte) error {
	type countryAccessInfoAlias CountryAccessInfo
	var raw struct {
		countryAccessInfoAlias
		CamelBlockedReason *string `json:"blockedReason"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*c = CountryAccessInfo(raw.countryAccessInfoAlias)
	if raw.CamelBlockedReason != nil {
		c.BlockedReason = *raw.CamelBlockedReason
	}
	return nil
}

type MessagingProfileAccess struct {
	CanSendDomestic      bool    `json:"can_send_domestic"`
	CanSendInternational bool    `json:"can_send_international"`
	VerificationType     *string `json:"verification_type"`
	VerificationStatus   *string `json:"verification_status"`
}

// UnmarshalJSON reads the camelCase keys the API sends (canSendDomestic,
// canSendInternational, verificationType, verificationStatus), and still
// reads the snake_case keys the struct is tagged with.
func (m *MessagingProfileAccess) UnmarshalJSON(data []byte) error {
	type messagingProfileAccessAlias MessagingProfileAccess
	var raw struct {
		messagingProfileAccessAlias
		CamelCanSendDomestic      *bool   `json:"canSendDomestic"`
		CamelCanSendInternational *bool   `json:"canSendInternational"`
		CamelVerificationType     *string `json:"verificationType"`
		CamelVerificationStatus   *string `json:"verificationStatus"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*m = MessagingProfileAccess(raw.messagingProfileAccessAlias)
	if raw.CamelCanSendDomestic != nil {
		m.CanSendDomestic = *raw.CamelCanSendDomestic
	}
	if raw.CamelCanSendInternational != nil {
		m.CanSendInternational = *raw.CamelCanSendInternational
	}
	if raw.CamelVerificationType != nil {
		m.VerificationType = raw.CamelVerificationType
	}
	if raw.CamelVerificationStatus != nil {
		m.VerificationStatus = raw.CamelVerificationStatus
	}
	return nil
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
	ScheduledAt string `json:"scheduledAt"`
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
	err := s.client.request(ctx, "GET", fmt.Sprintf("/campaigns/%s", url.PathEscape(id)), nil, &resp)
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
	err := s.client.request(ctx, "PATCH", fmt.Sprintf("/campaigns/%s", url.PathEscape(id)), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CampaignsService) Delete(ctx context.Context, id string) error {
	return s.client.request(ctx, "DELETE", fmt.Sprintf("/campaigns/%s", url.PathEscape(id)), nil, nil)
}

func (s *CampaignsService) Preview(ctx context.Context, id string) (*CampaignPreview, error) {
	var resp CampaignPreview
	err := s.client.request(ctx, "GET", fmt.Sprintf("/campaigns/%s/preview", url.PathEscape(id)), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type campaignSendResult struct {
	BatchID     string  `json:"batchId"`
	Status      string  `json:"status"`
	Total       int     `json:"total"`
	Sent        int     `json:"sent"`
	Failed      int     `json:"failed"`
	CreditsUsed float64 `json:"creditsUsed"`
}

// Send sends a campaign now. It returns the campaign as it stands after the
// send, with BatchID naming the batch its messages went out in, which costs a
// second request to read it back. Status is the status the send reported for
// that batch: "completed", "partial_failure" when some messages failed,
// "failed" when every message failed, or "processing" while the messages are
// still going out. The campaign itself is recorded as completed after any
// accepted send, and that is the status Get returns. If the read fails, the
// send has still happened, and the returned Campaign carries only what the
// send reported: ID, Status, BatchID, RecipientCount, SentCount, FailedCount
// and CreditsUsed.
func (s *CampaignsService) Send(ctx context.Context, id string) (*Campaign, error) {
	var result campaignSendResult
	err := s.client.request(ctx, "POST", fmt.Sprintf("/campaigns/%s/send", url.PathEscape(id)), map[string]interface{}{}, &result)
	if err != nil {
		return nil, err
	}

	if campaign, err := s.Get(ctx, id); err == nil {
		if result.Status != "" {
			campaign.Status = result.Status
		}
		return campaign, nil
	}

	creditsUsed := result.CreditsUsed
	return &Campaign{
		ID:             id,
		Status:         result.Status,
		BatchID:        result.BatchID,
		RecipientCount: result.Total,
		SentCount:      result.Sent,
		FailedCount:    result.Failed,
		CreditsUsed:    &creditsUsed,
	}, nil
}

func (s *CampaignsService) Schedule(ctx context.Context, id string, req *ScheduleCampaignRequest) (*Campaign, error) {
	var resp Campaign
	err := s.client.request(ctx, "POST", fmt.Sprintf("/campaigns/%s/schedule", url.PathEscape(id)), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CampaignsService) Cancel(ctx context.Context, id string) (*Campaign, error) {
	var resp Campaign
	err := s.client.request(ctx, "POST", fmt.Sprintf("/campaigns/%s/cancel", url.PathEscape(id)), map[string]interface{}{}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CampaignsService) Clone(ctx context.Context, id string) (*Campaign, error) {
	var resp Campaign
	err := s.client.request(ctx, "POST", fmt.Sprintf("/campaigns/%s/clone", url.PathEscape(id)), map[string]interface{}{}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
