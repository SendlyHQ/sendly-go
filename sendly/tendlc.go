package sendly

import (
	"context"
	"net/url"
)

// TenDlcService registers your business for carrier review so you can text
// from local (10-digit) US numbers. The flow is brand -> campaign -> assign
// number; writes require a live API key.
type TenDlcService struct {
	client *Client
}

// TenDlcBrand is a business identity registered for carrier review. Status
// is "pending" (awaiting review), "verified" (campaigns can be created), or
// "failed" (see FailureReasons).
type TenDlcBrand struct {
	ID         string  `json:"id"`
	LegalName  string  `json:"legalName"`
	DBA        *string `json:"dba,omitempty"`
	EntityType string  `json:"entityType"`
	EIN        *string `json:"ein,omitempty"`
	Vertical   *string `json:"vertical,omitempty"`
	Website    *string `json:"website,omitempty"`
	Status     string  `json:"status"`
	// IdentityStatus is identity-verification detail from the carrier review, when available.
	IdentityStatus *string  `json:"identityStatus,omitempty"`
	FailureReasons []string `json:"failureReasons,omitempty"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}

// TenDlcBrandListResponse wraps the workspace's brands.
type TenDlcBrandListResponse struct {
	Data []TenDlcBrand `json:"data"`
}

// TenDlcBrandResponse wraps a single brand.
type TenDlcBrandResponse struct {
	Data TenDlcBrand `json:"data"`
}

// CreateTenDlcBrandRequest registers a business identity for carrier review.
// LegalName is required; EntityType defaults to "PRIVATE_PROFIT" and Country
// to "US". VerificationID prefills business details from an existing Sendly
// verification.
type CreateTenDlcBrandRequest struct {
	LegalName      string `json:"legalName"`
	DBA            string `json:"dba,omitempty"`
	EIN            string `json:"ein,omitempty"`
	EntityType     string `json:"entityType,omitempty"`
	Vertical       string `json:"vertical,omitempty"`
	Website        string `json:"website,omitempty"`
	Email          string `json:"email,omitempty"`
	Phone          string `json:"phone,omitempty"`
	MobilePhone    string `json:"mobilePhone,omitempty"`
	Street         string `json:"street,omitempty"`
	City           string `json:"city,omitempty"`
	State          string `json:"state,omitempty"`
	PostalCode     string `json:"postalCode,omitempty"`
	Country        string `json:"country,omitempty"`
	VerificationID string `json:"verificationId,omitempty"`
}

// TenDlcThroughput is the messaging throughput granted by the carrier
// network. Tier is "High volume", "Standard", or "Low volume".
type TenDlcThroughput struct {
	Tier          string `json:"tier"`
	CarriersReady int    `json:"carriersReady"`
}

// TenDlcQualifyResult is a use-case qualification pre-check. Reason is set
// only when Qualified is false; Throughput when the carrier network reports it.
type TenDlcQualifyResult struct {
	UseCase    string            `json:"useCase"`
	Qualified  bool              `json:"qualified"`
	Reason     *string           `json:"reason,omitempty"`
	Throughput *TenDlcThroughput `json:"throughput,omitempty"`
}

// TenDlcQualifyResponse wraps a use-case qualification pre-check.
type TenDlcQualifyResponse struct {
	Data TenDlcQualifyResult `json:"data"`
}

// TenDlcCampaign is a messaging campaign registered for carrier review.
// Status is "pending" (awaiting review), "active" (numbers can be assigned),
// "failed" (see FailureReasons), "suspended", or "expired".
type TenDlcCampaign struct {
	ID             string            `json:"id"`
	BrandID        string            `json:"brandId"`
	UseCase        string            `json:"useCase"`
	SubUseCases    []string          `json:"subUseCases"`
	Description    *string           `json:"description,omitempty"`
	Status         string            `json:"status"`
	SampleMessages []string          `json:"sampleMessages"`
	Throughput     *TenDlcThroughput `json:"throughput,omitempty"`
	FailureReasons []string          `json:"failureReasons,omitempty"`
	CreatedAt      string            `json:"createdAt"`
	UpdatedAt      string            `json:"updatedAt"`
}

// TenDlcCampaignListResponse wraps the workspace's campaigns.
type TenDlcCampaignListResponse struct {
	Data []TenDlcCampaign `json:"data"`
}

// TenDlcCampaignResponse wraps a single campaign.
type TenDlcCampaignResponse struct {
	Data TenDlcCampaign `json:"data"`
}

// CreateTenDlcCampaignRequest creates a campaign under a verified brand.
// BrandID, UseCase, Description, MessageFlow (how recipients opt in), and
// SampleMessages (first 5 used) are required. EmbeddedLink defaults to true,
// EmbeddedPhone to false.
type CreateTenDlcCampaignRequest struct {
	BrandID        string   `json:"brandId"`
	UseCase        string   `json:"useCase"`
	Description    string   `json:"description"`
	MessageFlow    string   `json:"messageFlow"`
	SampleMessages []string `json:"sampleMessages"`
	SubUseCases    []string `json:"subUseCases,omitempty"`
	OptInKeywords  string   `json:"optInKeywords,omitempty"`
	OptOutKeywords string   `json:"optOutKeywords,omitempty"`
	HelpKeywords   string   `json:"helpKeywords,omitempty"`
	OptInMessage   string   `json:"optInMessage,omitempty"`
	OptOutMessage  string   `json:"optOutMessage,omitempty"`
	HelpMessage    string   `json:"helpMessage,omitempty"`
	EmbeddedLink   *bool    `json:"embeddedLink,omitempty"`
	EmbeddedPhone  *bool    `json:"embeddedPhone,omitempty"`
}

// TenDlcAssignment is a phone number assigned to a campaign. The number can
// send once Status is "Active"; other values are "Under review" and
// "Action needed". AssignedAt is nil while the assignment is in progress.
type TenDlcAssignment struct {
	ID          string  `json:"id"`
	CampaignID  string  `json:"campaignId"`
	PhoneNumber string  `json:"phoneNumber"`
	Status      string  `json:"status"`
	AssignedAt  *string `json:"assignedAt,omitempty"`
}

// TenDlcAssignmentResponse wraps a single assignment.
type TenDlcAssignmentResponse struct {
	Data TenDlcAssignment `json:"data"`
}

// TenDlcAssignmentListResponse wraps the workspace's assignments.
type TenDlcAssignmentListResponse struct {
	Data []TenDlcAssignment `json:"data"`
}

// ListBrands returns the workspace's brands and their carrier-review status.
func (s *TenDlcService) ListBrands(ctx context.Context) (*TenDlcBrandListResponse, error) {
	var resp TenDlcBrandListResponse
	if err := s.client.request(ctx, "GET", "/tendlc/brands", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateBrand registers a brand for carrier review — step 1 of enabling
// local-number texting. The brand starts "pending"; poll GetBrand until it
// becomes "verified" before creating a campaign. Requires a live API key.
func (s *TenDlcService) CreateBrand(ctx context.Context, req *CreateTenDlcBrandRequest) (*TenDlcBrandResponse, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.LegalName == "" {
		return nil, &ValidationError{APIError: APIError{Message: "legalName is required"}}
	}

	var resp TenDlcBrandResponse
	if err := s.client.request(ctx, "POST", "/tendlc/brands", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetBrand fetches one brand. Fetching also refreshes the carrier-review
// status, so polling this method shows progress ("pending" -> "verified"/"failed").
func (s *TenDlcService) GetBrand(ctx context.Context, id string) (*TenDlcBrandResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "brand ID is required"}}
	}

	var resp TenDlcBrandResponse
	if err := s.client.request(ctx, "GET", "/tendlc/brands/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Qualify pre-checks whether a use case (e.g. "MIXED", "MARKETING",
// "ACCOUNT_NOTIFICATION", "2FA") qualifies for a brand on the carrier
// network before creating a campaign.
func (s *TenDlcService) Qualify(ctx context.Context, brandID, useCase string) (*TenDlcQualifyResponse, error) {
	if brandID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "brand ID is required"}}
	}
	if useCase == "" {
		return nil, &ValidationError{APIError: APIError{Message: "use case is required"}}
	}

	path := "/tendlc/brands/" + url.PathEscape(brandID) + "/qualify/" + url.PathEscape(useCase)
	var resp TenDlcQualifyResponse
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListCampaigns returns the workspace's campaigns and their carrier-review status.
func (s *TenDlcService) ListCampaigns(ctx context.Context) (*TenDlcCampaignListResponse, error) {
	var resp TenDlcCampaignListResponse
	if err := s.client.request(ctx, "GET", "/tendlc/campaigns", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateCampaign creates a messaging campaign under a verified brand and
// submits it for carrier review. The campaign starts "pending"; poll
// GetCampaign until it becomes "active" before assigning numbers. Requires
// a live API key.
func (s *TenDlcService) CreateCampaign(ctx context.Context, req *CreateTenDlcCampaignRequest) (*TenDlcCampaignResponse, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.BrandID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "brandId is required"}}
	}
	if req.UseCase == "" {
		return nil, &ValidationError{APIError: APIError{Message: "useCase is required"}}
	}
	if req.Description == "" {
		return nil, &ValidationError{APIError: APIError{Message: "description is required"}}
	}
	if req.MessageFlow == "" {
		return nil, &ValidationError{APIError: APIError{Message: "messageFlow is required"}}
	}
	if len(req.SampleMessages) == 0 {
		return nil, &ValidationError{APIError: APIError{Message: "sampleMessages is required"}}
	}

	var resp TenDlcCampaignResponse
	if err := s.client.request(ctx, "POST", "/tendlc/campaigns", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetCampaign fetches one campaign. Fetching also refreshes the
// carrier-review status, so polling this method shows progress
// ("pending" -> "active"), including throughput once carriers approve.
func (s *TenDlcService) GetCampaign(ctx context.Context, id string) (*TenDlcCampaignResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "campaign ID is required"}}
	}

	var resp TenDlcCampaignResponse
	if err := s.client.request(ctx, "GET", "/tendlc/campaigns/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AssignNumber assigns a phone number the workspace owns to an active
// (carrier-approved) campaign, making the number sendable once the
// assignment is "Active". Idempotent: re-assigning the same number to the
// same campaign returns the existing assignment. Requires a live API key.
func (s *TenDlcService) AssignNumber(ctx context.Context, campaignID, phoneNumber string) (*TenDlcAssignmentResponse, error) {
	if campaignID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "campaign ID is required"}}
	}
	if phoneNumber == "" {
		return nil, &ValidationError{APIError: APIError{Message: "phoneNumber is required"}}
	}

	body := map[string]string{"phoneNumber": phoneNumber}
	path := "/tendlc/campaigns/" + url.PathEscape(campaignID) + "/assign"
	var resp TenDlcAssignmentResponse
	if err := s.client.request(ctx, "POST", path, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListAssignments returns the workspace's number-to-campaign assignments.
func (s *TenDlcService) ListAssignments(ctx context.Context) (*TenDlcAssignmentListResponse, error) {
	var resp TenDlcAssignmentListResponse
	if err := s.client.request(ctx, "GET", "/tendlc/assignments", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
