package sendly

import (
	"context"
	"net/url"
	"strconv"
)

// RCSRegistrationService reads the workspace's RCS registration at a glance:
// the current brand, the newest agent, its test devices, and where the
// registration stands.
type RCSRegistrationService struct {
	client *Client
}

// RCSDossierService prefills a brand draft from what Sendly already knows
// about the business (its local-number registration or toll-free
// verification), so a brand can be drafted without retyping it.
type RCSDossierService struct {
	client *Client
}

// RCSBrandsService drafts and edits the business identity behind an RCS
// agent. Registration is US-only for now: the brand address must carry
// CountryCode "US".
type RCSBrandsService struct {
	client *Client
}

// RcsCustomerStage is where a registration stands, from the customer's
// point of view. It is derived from the brand and agent together and is
// serialized as customerStage on both, and as stage on the registration,
// agent and list responses.
type RcsCustomerStage string

const (
	// RcsCustomerStageDraft means the registration has not been submitted.
	RcsCustomerStageDraft RcsCustomerStage = "draft"
	// RcsCustomerStageInReview means Sendly is reviewing the submission.
	RcsCustomerStageInReview RcsCustomerStage = "in_review"
	// RcsCustomerStageChangesRequested means Sendly asked for edits; see ReviewNote.
	RcsCustomerStageChangesRequested RcsCustomerStage = "changes_requested"
	// RcsCustomerStageRejected means the registration was declined; see RejectionReason.
	RcsCustomerStageRejected RcsCustomerStage = "rejected"
	// RcsCustomerStageBrandVerification means the brand is with the carrier network for verification.
	RcsCustomerStageBrandVerification RcsCustomerStage = "brand_verification"
	// RcsCustomerStageAgentReview means the agent is with the carrier network for review.
	RcsCustomerStageAgentReview RcsCustomerStage = "agent_review"
	// RcsCustomerStageTesting means the agent can message invited test devices.
	RcsCustomerStageTesting RcsCustomerStage = "testing"
	// RcsCustomerStageLaunchReview means Sendly is reviewing the launch request.
	RcsCustomerStageLaunchReview RcsCustomerStage = "launch_review"
	// RcsCustomerStageLaunching means the launch is with the carrier network.
	RcsCustomerStageLaunching RcsCustomerStage = "launching"
	// RcsCustomerStageLaunchRejected means the carrier network declined the launch; see RejectionReason.
	RcsCustomerStageLaunchRejected RcsCustomerStage = "launch_rejected"
	// RcsCustomerStageLive means the agent can message anyone.
	RcsCustomerStageLive RcsCustomerStage = "live"
	// RcsCustomerStageSuspended means the agent was suspended.
	RcsCustomerStageSuspended RcsCustomerStage = "suspended"
	// RcsCustomerStageFailed means the registration failed.
	RcsCustomerStageFailed RcsCustomerStage = "failed"
)

// RcsReviewStatus is the review state of a brand or agent, serialized as
// reviewStatus.
type RcsReviewStatus string

const (
	// RcsReviewStatusDraft means the record is editable and not yet submitted.
	RcsReviewStatusDraft RcsReviewStatus = "draft"
	// RcsReviewStatusAwaitingReview means the record is locked while Sendly reviews it.
	RcsReviewStatusAwaitingReview RcsReviewStatus = "awaiting_review"
	// RcsReviewStatusChangesRequested means the record is editable again; see ReviewNote.
	RcsReviewStatusChangesRequested RcsReviewStatus = "changes_requested"
	// RcsReviewStatusApprovedForCarrier means Sendly approved the record for the carrier network.
	RcsReviewStatusApprovedForCarrier RcsReviewStatus = "approved_for_carrier"
	// RcsReviewStatusRejected means the record was declined.
	RcsReviewStatusRejected RcsReviewStatus = "rejected"
	// RcsReviewStatusLaunchRequested means the agent is locked while Sendly reviews the launch request.
	RcsReviewStatusLaunchRequested RcsReviewStatus = "launch_requested"
	// RcsReviewStatusLaunchSubmitted means the launch is with the carrier network.
	RcsReviewStatusLaunchSubmitted RcsReviewStatus = "launch_submitted"
	// RcsReviewStatusLaunchRejected means the carrier network declined the launch; the campaign and testing sections are editable again.
	RcsReviewStatusLaunchRejected RcsReviewStatus = "launch_rejected"
	// RcsReviewStatusFailed means the record failed.
	RcsReviewStatusFailed RcsReviewStatus = "failed"
)

// Error codes returned by the RCS registration endpoints, readable from
// the Code field of the typed error (for example
// err.(*NotFoundError).Code).
const (
	// RcsErrorCodeNotEnabled (404, *NotFoundError): RCS registration isn't
	// enabled for the account yet. Every registration endpoint answers this
	// while the channel is off.
	RcsErrorCodeNotEnabled = "rcs_not_enabled"
	// RcsErrorCodeNotFound (404, *NotFoundError): no brand or agent with that
	// ID in this workspace.
	RcsErrorCodeNotFound = "rcs_not_found"
	// RcsErrorCodeFieldLocked (409, *SendlyError): the record is locked while
	// it is being reviewed, or the section was already submitted.
	RcsErrorCodeFieldLocked = "rcs_field_locked"
	// RcsErrorCodeUSOnly (422, *ValidationError): the brand address must be in the US.
	RcsErrorCodeUSOnly = "rcs_us_only"
	// RcsErrorCodeInvalidContent (422, *ValidationError): one or more fields
	// are invalid or incomplete; see the Errors field for each path.
	RcsErrorCodeInvalidContent = "rcs_invalid_content"
	// RcsErrorCodeBrandNotVerified (409, *SendlyError): the brand failed
	// carrier verification, so the agent cannot be submitted.
	RcsErrorCodeBrandNotVerified = "rcs_brand_not_verified"
	// RcsErrorCodeLaunchNotReady (409, *SendlyError): the agent must finish
	// testing on an invited device before launch can be requested.
	RcsErrorCodeLaunchNotReady = "rcs_launch_not_ready"
	// RcsErrorCodeInternal (500, *SendlyError): something went wrong on Sendly's side.
	RcsErrorCodeInternal = "rcs_internal_error"
)

// RcsBrandAddress is the brand's registered business address. CountryCode
// must be "US".
type RcsBrandAddress struct {
	Line1       string `json:"line1,omitempty"`
	Line2       string `json:"line2,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	PostalCode  string `json:"postalCode,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

// RcsBrandContact is the person the carrier network can reach about the
// brand. PhoneNumber is E.164.
type RcsBrandContact struct {
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	Title       string `json:"title,omitempty"`
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
}

// RcsBrandInput is the brand draft accepted by RCSBrandsService.Create and,
// in partial form, by RCSBrandsService.Update: only the fields you set are
// written, so on an update a zero value leaves the field untouched.
//
// LegalEntityType is one of "LIMITED_LIABILITY_COMPANY",
// "SOLE_PROPRIETORSHIP", "PARTNERSHIP", "CORPORATION" or "S_CORPORATION";
// OrganizationType is one of "PRIVATE_PROFIT", "PUBLIC_PROFIT",
// "NON_PROFIT", "GOVERNMENT" or "UNKNOWN". WebsiteURL must be https, EIN
// is 9 digits ("123456789" or "12-3456789"), StockSymbol is
// "EXCHANGE:TICKER". Address.CountryCode must be "US". Required-field
// checks run when the agent is submitted, not when the draft is saved.
type RcsBrandInput struct {
	DisplayName      string           `json:"displayName,omitempty"`
	LegalName        string           `json:"legalName,omitempty"`
	LegalEntityType  string           `json:"legalEntityType,omitempty"`
	OrganizationType string           `json:"organizationType,omitempty"`
	WebsiteURL       string           `json:"websiteUrl,omitempty"`
	EIN              string           `json:"ein,omitempty"`
	StockSymbol      string           `json:"stockSymbol,omitempty"`
	Address          *RcsBrandAddress `json:"address,omitempty"`
	Contact          *RcsBrandContact `json:"contact,omitempty"`
}

// RcsBrand is a brand as Sendly stores it. ReviewNote carries Sendly's
// feedback when changes are requested; RejectionReason carries the carrier
// network's reason when verification fails. Timestamps are ISO 8601 and
// nil until the step happens.
type RcsBrand struct {
	ID                   string           `json:"id"`
	ReviewStatus         RcsReviewStatus  `json:"reviewStatus"`
	CustomerStage        RcsCustomerStage `json:"customerStage"`
	DisplayName          string           `json:"displayName"`
	LegalName            string           `json:"legalName"`
	LegalEntityType      string           `json:"legalEntityType"`
	OrganizationType     string           `json:"organizationType"`
	StockSymbol          *string          `json:"stockSymbol,omitempty"`
	WebsiteURL           string           `json:"websiteUrl"`
	EIN                  string           `json:"ein"`
	Address              RcsBrandAddress  `json:"address"`
	Contact              RcsBrandContact  `json:"contact"`
	ReviewNote           *string          `json:"reviewNote,omitempty"`
	RejectionReason      *string          `json:"rejectionReason,omitempty"`
	SubmittedForReviewAt *string          `json:"submittedForReviewAt,omitempty"`
	SentToCarrierAt      *string          `json:"sentToCarrierAt,omitempty"`
	VerifiedAt           *string          `json:"verifiedAt,omitempty"`
	CreatedAt            string           `json:"createdAt"`
	UpdatedAt            string           `json:"updatedAt"`
}

// RcsBrandResponse wraps a single brand.
type RcsBrandResponse struct {
	Brand RcsBrand `json:"brand"`
}

// RcsDossierResponse is a brand draft prefilled from what Sendly already
// knows. Source is "tendlc" (from the workspace's local-number brand),
// "verification" (from its toll-free verification) or "none" (Brand is
// empty). USEligible is false when something on file names a non-US
// country. Fill in the gaps and pass Brand to RCSBrandsService.Create.
type RcsDossierResponse struct {
	Brand      RcsBrandInput `json:"brand"`
	USEligible bool          `json:"usEligible"`
	Source     string        `json:"source"`
}

// RcsAgentPhoneContact is a phone number shown on the agent's info sheet.
// Number is E.164.
type RcsAgentPhoneContact struct {
	Number string `json:"number,omitempty"`
	Label  string `json:"label,omitempty"`
}

// RcsAgentWebsiteContact is a website shown on the agent's info sheet.
// URL must be https.
type RcsAgentWebsiteContact struct {
	URL   string `json:"url,omitempty"`
	Label string `json:"label,omitempty"`
}

// RcsAgentEmailContact is an email address shown on the agent's info sheet.
type RcsAgentEmailContact struct {
	Address string `json:"address,omitempty"`
	Label   string `json:"label,omitempty"`
}

// RcsAgentBasics is what recipients see: the agent's name, look and
// contact details. UseCase is one of "MULTI_USE", "PROMOTIONAL",
// "TRANSACTIONAL" or "OTP". LogoURL and HeroURL must be public https URLs
// (assets cannot be uploaded over the API; upload them in the dashboard or
// host them yourself). BrandColor is "#RGB" or "#RRGGBB";
// PrivacyPolicyURL and TermsAndConditionsURL must be https. HostingRegion
// is set by Sendly and ignored on input.
type RcsAgentBasics struct {
	DisplayName           string                  `json:"displayName,omitempty"`
	UseCase               string                  `json:"useCase,omitempty"`
	HostingRegion         string                  `json:"hostingRegion,omitempty"`
	Description           string                  `json:"description,omitempty"`
	LogoURL               string                  `json:"logoUrl,omitempty"`
	HeroURL               string                  `json:"heroUrl,omitempty"`
	BrandColor            string                  `json:"brandColor,omitempty"`
	PrivacyPolicyURL      string                  `json:"privacyPolicyUrl,omitempty"`
	TermsAndConditionsURL string                  `json:"termsAndConditionsUrl,omitempty"`
	PhoneNumber           *RcsAgentPhoneContact   `json:"phoneNumber,omitempty"`
	Website               *RcsAgentWebsiteContact `json:"website,omitempty"`
	Email                 *RcsAgentEmailContact   `json:"email,omitempty"`
}

// RcsInteraction describes one kind of conversation the agent will have.
// InteractionType is one of "TRANSACTIONAL_UPDATES", "CUSTOMER_SUPPORT",
// "LOYALTY_OR_REWARD", "MARKETING_OR_PROMOTIONAL", "ACCOUNT_ALERTS",
// "TWO_WAY_CONVERSATION" or "OTHER".
type RcsInteraction struct {
	InteractionType string `json:"interactionType,omitempty"`
	Description     string `json:"description,omitempty"`
}

// RcsOptInMethod describes one way recipients consent to messages.
// MethodType is one of "SMS", "WEBSITE", "MOBILE_APP", "QR_CODE",
// "SALE_POINT" or "OTHER".
type RcsOptInMethod struct {
	MethodType  string `json:"methodType,omitempty"`
	Description string `json:"description,omitempty"`
}

// RcsConsentSettings describes how recipients opt in and out.
// CallToActionMediaURL must be a public https URL.
type RcsConsentSettings struct {
	OptInMethods         []RcsOptInMethod `json:"optInMethods,omitempty"`
	CallToAction         string           `json:"callToAction,omitempty"`
	CallToActionURL      string           `json:"callToActionUrl,omitempty"`
	CallToActionMediaURL string           `json:"callToActionMediaUrl,omitempty"`
	DoubleOptIn          *bool            `json:"doubleOptIn,omitempty"`
	DoubleOptInMessage   string           `json:"doubleOptInMessage,omitempty"`
	OptInMessage         string           `json:"optInMessage,omitempty"`
	HelpResponse         string           `json:"helpResponse,omitempty"`
	OptOutResponse       string           `json:"optOutResponse,omitempty"`
}

// RcsCampaign is the launch dossier reviewed before the agent goes live:
// what the business does, what the agent will send, and how recipients
// consent. Launch needs AgentOverview, at least one interaction, at least
// three message examples and consent settings.
type RcsCampaign struct {
	CompanyOverview       string              `json:"companyOverview,omitempty"`
	AgentOverview         string              `json:"agentOverview,omitempty"`
	AdditionalInformation string              `json:"additionalInformation,omitempty"`
	Interactions          []RcsInteraction    `json:"interactions,omitempty"`
	MessageExamples       []string            `json:"messageExamples,omitempty"`
	ConsentSettings       *RcsConsentSettings `json:"consentSettings,omitempty"`
}

// RcsTesting tells the reviewer how to try the agent: a test URL, the ID
// of a message sent to an invited device, and any notes.
type RcsTesting struct {
	TestURL               string `json:"testUrl,omitempty"`
	MessageID             string `json:"messageId,omitempty"`
	AdditionalInformation string `json:"additionalInformation,omitempty"`
}

// RcsTestDevice is a phone number invited to message the agent before it
// is live. InviteStatus is the carrier network's invite state (for example
// "PENDING") and nil until the invite is sent.
type RcsTestDevice struct {
	ID           string  `json:"id"`
	PhoneNumber  string  `json:"phoneNumber"`
	Label        *string `json:"label,omitempty"`
	InviteStatus *string `json:"inviteStatus,omitempty"`
	CreatedAt    string  `json:"createdAt"`
}

// RcsTestDeviceInput is a device to invite. PhoneNumber is E.164 (a
// formatted 10-digit US number is accepted and normalized).
type RcsTestDeviceInput struct {
	PhoneNumber string `json:"phoneNumber"`
	Label       string `json:"label,omitempty"`
}

// RcsTestDeviceListResponse wraps an agent's test devices.
type RcsTestDeviceListResponse struct {
	Devices []RcsTestDevice `json:"devices"`
}

// RcsAgentDetail is an agent as Sendly stores it, with its brand, basics,
// campaign, testing notes and test devices. Status is the send status
// ("draft", "submitted", "testing", "approved" or "suspended");
// ReviewStatus and CustomerStage track the registration. ReviewNote
// carries Sendly's feedback when changes are requested; RejectionReason
// carries the carrier network's reason when a review fails. Timestamps
// are ISO 8601 and nil until the step happens.
type RcsAgentDetail struct {
	ID                   string           `json:"id"`
	BrandID              *string          `json:"brandId,omitempty"`
	Status               string           `json:"status"`
	ReviewStatus         RcsReviewStatus  `json:"reviewStatus"`
	CustomerStage        RcsCustomerStage `json:"customerStage"`
	DisplayName          string           `json:"displayName"`
	UseCase              *string          `json:"useCase,omitempty"`
	HostingRegion        *string          `json:"hostingRegion,omitempty"`
	Basics               RcsAgentBasics   `json:"basics"`
	Campaign             *RcsCampaign     `json:"campaign,omitempty"`
	Testing              *RcsTesting      `json:"testing,omitempty"`
	ReviewNote           *string          `json:"reviewNote,omitempty"`
	RejectionReason      *string          `json:"rejectionReason,omitempty"`
	TestDevices          []RcsTestDevice  `json:"testDevices"`
	SubmittedForReviewAt *string          `json:"submittedForReviewAt,omitempty"`
	BasicsSubmittedAt    *string          `json:"basicsSubmittedAt,omitempty"`
	LaunchSubmittedAt    *string          `json:"launchSubmittedAt,omitempty"`
	LiveAt               *string          `json:"liveAt,omitempty"`
	CreatedAt            string           `json:"createdAt"`
	UpdatedAt            string           `json:"updatedAt"`
}

// RcsAgentResponse wraps a single agent. Devices is filled by Get and
// Stage by Get, Submit and RequestLaunch; Agent.TestDevices and
// Agent.CustomerStage always carry the same values.
type RcsAgentResponse struct {
	Agent   RcsAgentDetail   `json:"agent"`
	Devices []RcsTestDevice  `json:"devices,omitempty"`
	Stage   RcsCustomerStage `json:"stage,omitempty"`
}

// RcsRegistrationResponse is the workspace's registration at a glance.
// Brand is the newest agent's brand (else the newest brand), Agent the
// newest agent, Devices that agent's test devices; all are nil or empty
// until drafted. Stage is derived from the pair ("draft" when nothing
// exists). USEligible is false when something on file names a non-US
// country.
type RcsRegistrationResponse struct {
	Brand      *RcsBrand        `json:"brand"`
	Agent      *RcsAgentDetail  `json:"agent"`
	Devices    []RcsTestDevice  `json:"devices"`
	Stage      RcsCustomerStage `json:"stage"`
	USEligible bool             `json:"usEligible"`
}

// CreateRcsAgentRequest drafts an agent under a brand. BrandID is required.
// DisplayName and UseCase are shorthands that override Basics.DisplayName
// and Basics.UseCase. Campaign and Testing can be drafted now or added
// later with Update.
type CreateRcsAgentRequest struct {
	BrandID     string          `json:"brandId"`
	DisplayName string          `json:"displayName,omitempty"`
	UseCase     string          `json:"useCase,omitempty"`
	Basics      *RcsAgentBasics `json:"basics,omitempty"`
	Campaign    *RcsCampaign    `json:"campaign,omitempty"`
	Testing     *RcsTesting     `json:"testing,omitempty"`
}

// UpdateRcsAgentRequest patches an agent draft. Only the groups you set are
// written: DisplayName, UseCase and Basics are merged into the basics;
// Campaign and Testing are merged section-wise. Nil leaves a section
// untouched.
type UpdateRcsAgentRequest struct {
	DisplayName string          `json:"displayName,omitempty"`
	UseCase     string          `json:"useCase,omitempty"`
	Basics      *RcsAgentBasics `json:"basics,omitempty"`
	Campaign    *RcsCampaign    `json:"campaign,omitempty"`
	Testing     *RcsTesting     `json:"testing,omitempty"`
}

// RcsLaunchRequest is the optional body for RCSAgentsService.RequestLaunch.
// When set, TestURL and TestingAdditionalInformation are stored into the
// agent's testing section before the request is filed.
type RcsLaunchRequest struct {
	TestURL                      string `json:"testUrl,omitempty"`
	TestingAdditionalInformation string `json:"testingAdditionalInformation,omitempty"`
}

// Get returns the workspace's RCS registration at a glance: the brand,
// the newest agent, its test devices and the current stage. Poll it to
// follow a submission through review. Requires the rcs:read scope.
func (s *RCSRegistrationService) Get(ctx context.Context) (*RcsRegistrationResponse, error) {
	var resp RcsRegistrationResponse
	if err := s.client.request(ctx, "GET", "/rcs/registration", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get returns a brand draft prefilled from the workspace's local-number
// brand or toll-free verification, whichever Sendly has. Requires the
// rcs:read scope.
func (s *RCSDossierService) Get(ctx context.Context) (*RcsDossierResponse, error) {
	var resp RcsDossierResponse
	if err := s.client.request(ctx, "GET", "/rcs/dossier", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Create drafts a brand. The draft starts in review status "draft" and can
// be edited with Update until it is submitted. The address must be in the
// US (422 rcs_us_only otherwise). Requires the rcs:write scope. An
// Idempotency-Key is generated automatically; pass WithIdempotencyKey to
// supply your own.
func (s *RCSBrandsService) Create(ctx context.Context, req *RcsBrandInput, opts ...RequestOption) (*RcsBrandResponse, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	var resp RcsBrandResponse
	if err := s.client.request(ctx, "POST", "/rcs/brands", req, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update patches a brand draft; only the fields you set are written.
// Brands are locked while under review (409 rcs_field_locked) and once
// the carrier network has verified them. Requires the rcs:write scope.
// Pass WithIdempotencyKey to make the patch replay-safe.
func (s *RCSBrandsService) Update(ctx context.Context, id string, req *RcsBrandInput, opts ...RequestOption) (*RcsBrandResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "brand ID is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	var resp RcsBrandResponse
	if err := s.client.request(ctx, "PATCH", "/rcs/brands/"+url.PathEscape(id), req, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Create drafts an agent under a brand. BrandID is required. Logo, hero
// and call-to-action media must be public https URLs (422
// rcs_invalid_content otherwise); assets cannot be uploaded over the API.
// Requires the rcs:write scope. An Idempotency-Key is generated
// automatically; pass WithIdempotencyKey to supply your own.
func (s *RCSAgentsService) Create(ctx context.Context, req *CreateRcsAgentRequest, opts ...RequestOption) (*RcsAgentResponse, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.BrandID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "brandId is required"}}
	}

	var resp RcsAgentResponse
	if err := s.client.request(ctx, "POST", "/rcs/agents", req, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get fetches one agent with its test devices and current stage. Poll it
// to follow the agent through review, testing and launch. Requires the
// rcs:read scope.
func (s *RCSAgentsService) Get(ctx context.Context, id string) (*RcsAgentResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "agent ID is required"}}
	}

	var resp RcsAgentResponse
	if err := s.client.request(ctx, "GET", "/rcs/agents/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update patches an agent draft; only the groups you set are written.
// Agents are locked while under review (409 rcs_field_locked); the basics
// lock once submitted, and the campaign and testing sections lock once a
// launch is filed unless it was rejected. Media URLs must be public https.
// Requires the rcs:write scope. Pass WithIdempotencyKey to make the patch
// replay-safe.
func (s *RCSAgentsService) Update(ctx context.Context, id string, req *UpdateRcsAgentRequest, opts ...RequestOption) (*RcsAgentResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "agent ID is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	var resp RcsAgentResponse
	if err := s.client.request(ctx, "PATCH", "/rcs/agents/"+url.PathEscape(id), req, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetTestDevices replaces the agent's test devices (up to 20). The list is
// authoritative: numbers missing from it are removed and new ones are
// invited, so pass an empty list to remove them all. Requires the
// rcs:write scope. Pass WithIdempotencyKey to make the call replay-safe.
func (s *RCSAgentsService) SetTestDevices(ctx context.Context, id string, devices []RcsTestDeviceInput, opts ...RequestOption) (*RcsTestDeviceListResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "agent ID is required"}}
	}
	if len(devices) > 20 {
		return nil, &ValidationError{APIError: APIError{Message: "You can invite up to 20 test devices"}}
	}
	for i, device := range devices {
		if device.PhoneNumber == "" {
			return nil, &ValidationError{APIError: APIError{Message: "devices[" + strconv.Itoa(i) + "].phoneNumber is required"}}
		}
	}
	if devices == nil {
		devices = []RcsTestDeviceInput{}
	}

	body := map[string]interface{}{"devices": devices}
	path := "/rcs/agents/" + url.PathEscape(id) + "/test-devices"
	var resp RcsTestDeviceListResponse
	if err := s.client.request(ctx, "PUT", path, body, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Submit sends the brand and agent basics to Sendly for review. Both move
// to "awaiting_review" and lock until the review ends; incomplete fields
// are reported as 422 rcs_invalid_content with a path per field
// ("brand.ein", "agent.logoUrl"). Poll RCSRegistrationService.Get or Get
// for the outcome. Requires the rcs:write scope. An Idempotency-Key is
// generated automatically; pass WithIdempotencyKey to supply your own.
func (s *RCSAgentsService) Submit(ctx context.Context, id string, opts ...RequestOption) (*RcsAgentResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "agent ID is required"}}
	}

	path := "/rcs/agents/" + url.PathEscape(id) + "/submit"
	var resp RcsAgentResponse
	if err := s.client.request(ctx, "POST", path, struct{}{}, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RequestLaunch asks Sendly to take a tested agent live. The agent must be
// in testing with a complete campaign (409 rcs_launch_not_ready or 422
// rcs_invalid_content otherwise); it moves to "launch_requested" and
// locks until the review ends. req is optional and may be nil. Requires
// the rcs:write scope. An Idempotency-Key is generated automatically;
// pass WithIdempotencyKey to supply your own.
func (s *RCSAgentsService) RequestLaunch(ctx context.Context, id string, req *RcsLaunchRequest, opts ...RequestOption) (*RcsAgentResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "agent ID is required"}}
	}
	if req == nil {
		req = &RcsLaunchRequest{}
	}

	path := "/rcs/agents/" + url.PathEscape(id) + "/request-launch"
	var resp RcsAgentResponse
	if err := s.client.request(ctx, "POST", path, req, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}
