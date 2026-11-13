package sendly

import "context"

// ShortCodesService lists the workspace's short codes and manages its
// short code application.
//
// A short code is a 5 or 6 digit US sender. Getting one is an application,
// not a purchase: you fill it in, Sendly reviews it, the carrier forms are
// signed, Sendly files it, and each carrier certifies the code separately.
// The application lives at Application.
//
// Reads need the short_codes:read scope; Application.Update and
// Application.Submit need short_codes:write. While short codes are not
// enabled for the account every method returns a *NotFoundError with Code
// "short_codes_not_enabled".
type ShortCodesService struct {
	client *Client
	// Application is the workspace's short code application.
	Application *ShortCodeApplicationService
}

// ShortCodeApplicationService reads, saves, checks and submits the
// workspace's short code application.
type ShortCodeApplicationService struct {
	client *Client
}

// ShortCodeStatus is where a short code is in provisioning. Only Active can
// send. The type is a plain string, so a status added later decodes as its
// own value.
type ShortCodeStatus string

const (
	ShortCodeStatusRequested    ShortCodeStatus = "requested"
	ShortCodeStatusProvisioning ShortCodeStatus = "provisioning"
	// ShortCodeStatusParked is a code that is assigned but not cleared to
	// send.
	ShortCodeStatusParked    ShortCodeStatus = "parked"
	ShortCodeStatusActive    ShortCodeStatus = "active"
	ShortCodeStatusSuspended ShortCodeStatus = "suspended"
	ShortCodeStatusCancelled ShortCodeStatus = "cancelled"
)

// ShortCodeReviewStatus is where Sendly's review of a short code application
// stands. The type is a plain string, so a status added later decodes as its
// own value.
type ShortCodeReviewStatus string

const (
	ShortCodeReviewStatusDraft             ShortCodeReviewStatus = "draft"
	ShortCodeReviewStatusAwaitingReview    ShortCodeReviewStatus = "awaiting_review"
	ShortCodeReviewStatusChangesRequested  ShortCodeReviewStatus = "changes_requested"
	ShortCodeReviewStatusApprovedForFiling ShortCodeReviewStatus = "approved_for_filing"
	ShortCodeReviewStatusFiled             ShortCodeReviewStatus = "filed"
	ShortCodeReviewStatusRejected          ShortCodeReviewStatus = "rejected"
)

// ShortCodeLeasePastDue is the oldest unpaid lease charge.
type ShortCodeLeasePastDue struct {
	ChargeID      string  `json:"chargeId"`
	AmountCents   int     `json:"amountCents"`
	Status        string  `json:"status"`
	PeriodStart   *string `json:"periodStart"`
	PeriodEnd     *string `json:"periodEnd"`
	FirstFailedAt *string `json:"firstFailedAt"`
	// PauseAt is when sending from the code pauses if the month is still
	// unpaid.
	PauseAt     *string `json:"pauseAt"`
	NoticeStage *string `json:"noticeStage"`
}

// ShortCodeLease is the monthly lease of a short code. It starts when the
// code goes live.
type ShortCodeLease struct {
	MonthlyCents      int `json:"monthlyCents"`
	MinimumTermMonths int `json:"minimumTermMonths"`
	// StartsAt is "go_live" on the application, and empty on a listed code.
	StartsAt string `json:"startsAt,omitempty"`
	// State is "not_started", "running", "past_due", "paused", "ending" or
	// "ended". More values may be added.
	State string `json:"state"`
	// BillingEnabled is false while Sendly is not billing leases yet.
	BillingEnabled    bool    `json:"billingEnabled"`
	StartedAt         *string `json:"startedAt"`
	NextChargeAt      *string `json:"nextChargeAt"`
	PaidThrough       *string `json:"paidThrough"`
	TermEndsAt        *string `json:"termEndsAt"`
	EndsAt            *string `json:"endsAt"`
	CancelRequestedAt *string `json:"cancelRequestedAt"`
	// PastDue is nil when nothing is unpaid.
	PastDue *ShortCodeLeasePastDue `json:"pastDue"`
	// DueMonths are months used but not yet charged, costing DueCents.
	DueMonths int `json:"dueMonths"`
	DueCents  int `json:"dueCents"`
}

// ShortCodeCharge is one short code charge: the setup fee or a lease month.
type ShortCodeCharge struct {
	ID string `json:"id"`
	// Kind is "setup" or "lease".
	Kind        string `json:"kind"`
	AmountCents int    `json:"amountCents"`
	// Status is such as "paid", "processing", "failed", "waived",
	// "refunded", or "not_charged" for a lease month that was never billed.
	Status        string  `json:"status"`
	PaidAt        *string `json:"paidAt"`
	RefundedCents int     `json:"refundedCents"`
	RefundedAt    *string `json:"refundedAt"`
	PeriodStart   *string `json:"periodStart"`
	PeriodEnd     *string `json:"periodEnd"`
	CreatedAt     *string `json:"createdAt"`
}

// ShortCodeSetupFee is where the one-time setup fee of an application
// stands.
type ShortCodeSetupFee struct {
	AmountCents int `json:"amountCents"`
	// Status is "unpaid", "processing", "requires_action", "failed",
	// "no_payment_method", "paid", "waived", "refund_pending" or "refunded".
	// More values may be added.
	Status string `json:"status"`
	// ChargedAt is always "submit".
	ChargedAt     string  `json:"chargedAt"`
	PaidAt        *string `json:"paidAt"`
	RefundedCents int     `json:"refundedCents"`
	RefundedAt    *string `json:"refundedAt"`
}

// ShortCodeTerms is the version of the short code terms, and which version
// the workspace accepted at submit.
type ShortCodeTerms struct {
	Version         string  `json:"version"`
	AcceptedVersion *string `json:"acceptedVersion"`
	AcceptedAt      *string `json:"acceptedAt"`
}

// ShortCodeBilling is where the money stands on a short code application.
type ShortCodeBilling struct {
	SetupFee ShortCodeSetupFee `json:"setupFee"`
	Lease    ShortCodeLease    `json:"lease"`
	Terms    ShortCodeTerms    `json:"terms"`
	// RefundPolicy is always "full_refund_before_filing".
	RefundPolicy string `json:"refundPolicy"`
	// Exempt is "mock" or "enterprise" when the workspace isn't charged by
	// card, and nil otherwise.
	Exempt  *string           `json:"exempt"`
	Charges []ShortCodeCharge `json:"charges"`
}

// ShortCodeListSetupFee is a listed code's setup fee.
type ShortCodeListSetupFee struct {
	Status      string `json:"status"`
	AmountCents int    `json:"amountCents"`
}

// ShortCodeListBilling is where a listed code's setup fee and lease stand.
type ShortCodeListBilling struct {
	SetupFee ShortCodeListSetupFee `json:"setupFee"`
	Lease    ShortCodeLease        `json:"lease"`
}

// ShortCode is one of the workspace's short code applications, with its
// code once Sendly assigns one. ShortCode is nil until then; drafts,
// rejected and cancelled applications are listed too.
type ShortCode struct {
	ID           string                `json:"id"`
	ShortCode    *string               `json:"shortCode"`
	CountryCode  string                `json:"countryCode"`
	Status       ShortCodeStatus       `json:"status"`
	ReviewStatus ShortCodeReviewStatus `json:"reviewStatus"`
	UseCase      string                `json:"useCase"`
	CreatedAt    string                `json:"createdAt"`
	Billing      ShortCodeListBilling  `json:"billing"`
}

// ShortCodeList is the response from ShortCodesService.List.
type ShortCodeList struct {
	ShortCodes []ShortCode `json:"shortCodes"`
}

// ShortCodeIssue is a compliance problem with a short code application, by
// field.
type ShortCodeIssue struct {
	// Path is the field the issue is about, such as "helpResponse" or
	// "documents.loa".
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ShortCodeDocument is a carrier form the application needs, and where it
// stands.
type ShortCodeDocument struct {
	// Kind is such as "order_brief", "brand_registration",
	// "content_provider_registration", "migration_letter" or "loa".
	Kind              string  `json:"kind"`
	TemplateReady     bool    `json:"templateReady"`
	SignedAt          *string `json:"signedAt"`
	SignedByName      *string `json:"signedByName"`
	AwaitingSignature bool    `json:"awaitingSignature"`
}

// ShortCodeApplication is the short code application a workspace is filling
// in, or has filed. With no application yet it is the empty shape: ID is
// nil, the choices show their defaults, and every answer is nil.
type ShortCodeApplication struct {
	ID           *string               `json:"id"`
	ShortCode    *string               `json:"shortCode"`
	CountryCode  string                `json:"countryCode"`
	Status       ShortCodeStatus       `json:"status"`
	ReviewStatus ShortCodeReviewStatus `json:"reviewStatus"`
	ReviewNote   *string               `json:"reviewNote"`
	// OrderType is "new" or "migration".
	OrderType string `json:"orderType"`
	// CodeType is "random" or "vanity".
	CodeType                    string   `json:"codeType"`
	ContentProviderSameAsBrand  bool     `json:"contentProviderSameAsBrand"`
	SampleMessages              []string `json:"sampleMessages"`
	UseCase                     *string  `json:"useCase"`
	OptInFlow                   *string  `json:"optInFlow"`
	OptInConfirmation           *string  `json:"optInConfirmation"`
	HelpResponse                *string  `json:"helpResponse"`
	StopConfirmation            *string  `json:"stopConfirmation"`
	MessageFrequency            *string  `json:"messageFrequency"`
	CampaignKeyword             *string  `json:"campaignKeyword"`
	ExpectedMonthlyVolume       *string  `json:"expectedMonthlyVolume"`
	ExpectedDailyVolume         *string  `json:"expectedDailyVolume"`
	PrivacyPolicyURL            *string  `json:"privacyPolicyUrl"`
	TermsURL                    *string  `json:"termsUrl"`
	RequestedDigits             *string  `json:"requestedDigits"`
	LosingProvider              *string  `json:"losingProvider"`
	BrandContactName            *string  `json:"brandContactName"`
	BrandContactEmail           *string  `json:"brandContactEmail"`
	BrandContactPhone           *string  `json:"brandContactPhone"`
	ContentProviderLegalName    *string  `json:"contentProviderLegalName"`
	ContentProviderEin          *string  `json:"contentProviderEin"`
	ContentProviderContactName  *string  `json:"contentProviderContactName"`
	ContentProviderContactEmail *string  `json:"contentProviderContactEmail"`
	ContentProviderContactPhone *string  `json:"contentProviderContactPhone"`
	// BrandRegistrationStatus and ContentProviderRegistrationStatus are
	// where the Short Code Registry registrations stand: "not_started",
	// "submitted", "awaiting_email_verification", "vetting", "registered"
	// or "rejected". More values may be added.
	BrandRegistrationStatus           string  `json:"brandRegistrationStatus"`
	ContentProviderRegistrationStatus string  `json:"contentProviderRegistrationStatus"`
	SubmittedAt                       *string `json:"submittedAt"`
	// ReviewedAt is when Sendly finished reviewing the application.
	ReviewedAt  *string `json:"reviewedAt"`
	FiledAt     *string `json:"filedAt"`
	ActivatedAt *string `json:"activatedAt"`
	// RegistryRevettingDueAt is when the registry registration is next due
	// to be re-vetted.
	RegistryRevettingDueAt *string `json:"registryRevettingDueAt"`
	CreatedAt              *string `json:"createdAt"`
	UpdatedAt              *string `json:"updatedAt"`
}

// ShortCodePrefillAddress is the business address known to Sendly. Only the
// parts that are known are set.
type ShortCodePrefillAddress struct {
	Line1       string `json:"line1,omitempty"`
	Line2       string `json:"line2,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	PostalCode  string `json:"postalCode,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

// ShortCodePrefillContact is the business contact known to Sendly. Only the
// parts that are known are set.
type ShortCodePrefillContact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// ShortCodePrefill holds suggestions from what the workspace has already
// told Sendly: its 10DLC brand or its toll-free verification. They are never
// saved on the application unless sent back with a save. Fields with no
// value are empty.
type ShortCodePrefill struct {
	// Source is "tendlc", "verification" or "none".
	Source     string                   `json:"source"`
	LegalName  string                   `json:"legalName,omitempty"`
	Dba        string                   `json:"dba,omitempty"`
	Ein        string                   `json:"ein,omitempty"`
	WebsiteURL string                   `json:"websiteUrl,omitempty"`
	Address    *ShortCodePrefillAddress `json:"address,omitempty"`
	Contact    *ShortCodePrefillContact `json:"contact,omitempty"`
	// PrivacyPolicyURL and TermsURL come from the toll-free verification.
	PrivacyPolicyURL string `json:"privacyPolicyUrl,omitempty"`
	TermsURL         string `json:"termsUrl,omitempty"`
	// UsesHostedLegalPages is whether the toll-free verification uses
	// Sendly's hosted legal pages. Nil when unknown.
	UsesHostedLegalPages *bool `json:"usesHostedLegalPages,omitempty"`
}

// ShortCodeGeneratedDefaults is wording Sendly suggested for the
// application. Submit refuses it with short_code_ai_draft_unedited until a
// person has edited it.
type ShortCodeGeneratedDefaults struct {
	UseCase   string `json:"useCase,omitempty"`
	OptInFlow string `json:"optInFlow,omitempty"`
}

// ShortCodeQuote is the price: the published one for the code type, or once
// submitted the one fixed at submit.
type ShortCodeQuote struct {
	CodeType string `json:"codeType"`
	// MonthlyUsd is the monthly lease, in US dollars.
	MonthlyUsd int `json:"monthlyUsd"`
	// SetupUsd is the one-time setup fee, in US dollars.
	SetupUsd int    `json:"setupUsd"`
	Currency string `json:"currency"`
	// AutoBilled is always false, kept for older clients; Billing says what
	// is charged.
	AutoBilled        bool `json:"autoBilled"`
	SetupCents        int  `json:"setupCents"`
	MonthlyCents      int  `json:"monthlyCents"`
	MinimumTermMonths int  `json:"minimumTermMonths"`
	// SetupChargedAt is always "submit".
	SetupChargedAt string `json:"setupChargedAt"`
	// LeaseStartsAt is always "go_live".
	LeaseStartsAt string `json:"leaseStartsAt"`
}

// ShortCodeVerification is whether the workspace's business is verified,
// which submitting needs.
type ShortCodeVerification struct {
	Verified bool `json:"verified"`
	// State is "verified", "in_review", "needs_attention" or "not_started".
	State string `json:"state"`
	// Href is the dashboard page that shows the verification.
	Href string `json:"href"`
}

// ShortCodeCarrierApproval is one carrier's certification of the code.
type ShortCodeCarrierApproval struct {
	// Carrier is "ATT", "TMOBILE", "VERIZON" or "US_CELLULAR".
	Carrier string `json:"carrier"`
	Label   string `json:"label"`
	// Status is "not_submitted", "submitted", "in_review", "approved" or
	// "rejected".
	Status    string  `json:"status"`
	UpdatedAt *string `json:"updatedAt"`
}

// ShortCodeCarriers is where each carrier's certification stands.
type ShortCodeCarriers struct {
	Approved int `json:"approved"`
	Total    int `json:"total"`
	// Overall is "not_submitted", "in_progress", "approved" or "rejected".
	Overall   string                     `json:"overall"`
	ByCarrier []ShortCodeCarrierApproval `json:"byCarrier"`
}

// ShortCodePayment is the setup fee's status after a submit, and whether
// that call charged it.
type ShortCodePayment struct {
	Status  string `json:"status"`
	Charged bool   `json:"charged"`
}

// ShortCodeApplicationView is the application plus everything needed to act
// on it.
type ShortCodeApplicationView struct {
	Enabled     bool                 `json:"enabled"`
	Application ShortCodeApplication `json:"application"`
	Prefill     ShortCodePrefill     `json:"prefill"`
	Editable    bool                 `json:"editable"`
	// LockMessage is why the application cannot be edited, nil while it can.
	LockMessage *string `json:"lockMessage"`
	// CanStartNewApplication is true when the application is finished
	// (rejected or cancelled), so the next save starts a new one.
	CanStartNewApplication bool `json:"canStartNewApplication"`
	// GeneratedDefaults is nil when Sendly suggested nothing.
	GeneratedDefaults *ShortCodeGeneratedDefaults `json:"generatedDefaults"`
	RequiredDocuments []string                    `json:"requiredDocuments"`
	MissingDocuments  []string                    `json:"missingDocuments"`
	Documents         []ShortCodeDocument         `json:"documents"`
	Quote             ShortCodeQuote              `json:"quote"`
	Billing           ShortCodeBilling            `json:"billing"`
	Verification      ShortCodeVerification       `json:"verification"`
	Carriers          ShortCodeCarriers           `json:"carriers"`
	// Submitted is true on a submit's answer.
	Submitted bool `json:"submitted,omitempty"`
	// AlreadySubmitted is true on a submit's answer when the application was
	// already with Sendly, so nothing was charged.
	AlreadySubmitted bool `json:"alreadySubmitted,omitempty"`
	// Notified is whether Sendly's review team was told, on a submit's
	// answer.
	Notified bool `json:"notified,omitempty"`
	// Payment is the setup fee after a submit that collected it; nil
	// otherwise.
	Payment *ShortCodePayment `json:"payment,omitempty"`
	// IgnoredFields names fields a save sent that Sendly or the carriers
	// own, which were not saved.
	IgnoredFields []string `json:"ignoredFields,omitempty"`
}

// ShortCodePreflight is what a dry run of the application says, without
// changing anything.
type ShortCodePreflight struct {
	OK                bool                `json:"ok"`
	Issues            []ShortCodeIssue    `json:"issues"`
	RequiredDocuments []string            `json:"requiredDocuments"`
	MissingDocuments  []string            `json:"missingDocuments"`
	Documents         []ShortCodeDocument `json:"documents"`
	IgnoredFields     []string            `json:"ignoredFields,omitempty"`
}

// ShortCodeApplicationInput holds answers to save on the application. A nil
// field is left out and keeps its saved value; a pointer to an empty string
// clears it. Strings are trimmed by the API.
type ShortCodeApplicationInput struct {
	UseCase               *string `json:"useCase,omitempty"`
	OptInFlow             *string `json:"optInFlow,omitempty"`
	OptInConfirmation     *string `json:"optInConfirmation,omitempty"`
	HelpResponse          *string `json:"helpResponse,omitempty"`
	StopConfirmation      *string `json:"stopConfirmation,omitempty"`
	MessageFrequency      *string `json:"messageFrequency,omitempty"`
	CampaignKeyword       *string `json:"campaignKeyword,omitempty"`
	ExpectedMonthlyVolume *string `json:"expectedMonthlyVolume,omitempty"`
	ExpectedDailyVolume   *string `json:"expectedDailyVolume,omitempty"`
	PrivacyPolicyURL      *string `json:"privacyPolicyUrl,omitempty"`
	TermsURL              *string `json:"termsUrl,omitempty"`
	// CountryCode is only "US".
	CountryCode string `json:"countryCode,omitempty"`
	// OrderType is "new" or "migration"; any other value answers 422.
	OrderType string `json:"orderType,omitempty"`
	// CodeType is "random" or "vanity"; any other value answers 422.
	CodeType                    string  `json:"codeType,omitempty"`
	RequestedDigits             *string `json:"requestedDigits,omitempty"`
	LosingProvider              *string `json:"losingProvider,omitempty"`
	BrandContactName            *string `json:"brandContactName,omitempty"`
	BrandContactEmail           *string `json:"brandContactEmail,omitempty"`
	BrandContactPhone           *string `json:"brandContactPhone,omitempty"`
	ContentProviderSameAsBrand  *bool   `json:"contentProviderSameAsBrand,omitempty"`
	ContentProviderLegalName    *string `json:"contentProviderLegalName,omitempty"`
	ContentProviderEin          *string `json:"contentProviderEin,omitempty"`
	ContentProviderContactName  *string `json:"contentProviderContactName,omitempty"`
	ContentProviderContactEmail *string `json:"contentProviderContactEmail,omitempty"`
	ContentProviderContactPhone *string `json:"contentProviderContactPhone,omitempty"`
	// SampleMessages are real examples of the messages. Left out when nil
	// or empty.
	SampleMessages []string `json:"sampleMessages,omitempty"`
}

// ShortCodeSubmitInput is the body of a submit: final answers, saved before
// the check, plus acceptance of the price.
type ShortCodeSubmitInput struct {
	ShortCodeApplicationInput
	// AcceptTerms accepts the one-time setup fee, charged to the workspace's
	// card at submit, the monthly lease from go-live and the minimum term.
	// Submit is refused with short_code_invalid_application until it is
	// true.
	AcceptTerms bool `json:"acceptTerms,omitempty"`
}

// List lists every short code application of the workspace, newest first,
// with its code once Sendly assigns one. Drafts (with ShortCode nil),
// rejected and cancelled applications are listed too, so filter on Status
// for the codes you can send from. Each comes with Billing: where its setup
// fee and lease stand.
func (s *ShortCodesService) List(ctx context.Context) (*ShortCodeList, error) {
	var resp ShortCodeList
	if err := s.client.request(ctx, "GET", "/short_codes", nil, &resp); err != nil {
		return nil, err
	}
	if resp.ShortCodes == nil {
		resp.ShortCodes = []ShortCode{}
	}
	return &resp, nil
}

// Get fetches the workspace's short code application: the open one, or with
// none open the newest finished one, or the empty shape when there is none.
// With it come the suggestions from the workspace's business details, the
// carrier forms, the quote, Billing, the business verification and each
// carrier's certification.
func (s *ShortCodeApplicationService) Get(ctx context.Context) (*ShortCodeApplicationView, error) {
	var resp ShortCodeApplicationView
	if err := s.client.request(ctx, "GET", "/short_codes/application", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update saves answers on the application, creating it on the first save.
// After a rejected or cancelled application the save starts a new one. It
// never changes the status or the review status, and never submits. Fields
// Sendly or the carriers own are not saved and are named back in
// IgnoredFields.
//
// Once the application is with Sendly or the carriers it answers 409
// short_code_locked; another save that changed it first answers 409
// short_code_conflict. An unknown OrderType or CodeType answers 422
// short_code_invalid_application.
//
// No Idempotency-Key is generated for this PUT; pass WithIdempotencyKey to
// send one, so a repeat of the same save returns the first answer.
func (s *ShortCodeApplicationService) Update(ctx context.Context, input *ShortCodeApplicationInput, opts ...RequestOption) (*ShortCodeApplicationView, error) {
	if input == nil {
		input = &ShortCodeApplicationInput{}
	}
	var resp ShortCodeApplicationView
	if err := s.client.request(ctx, "PUT", "/short_codes/application", input, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Check checks the saved application, with the input's unsaved answers laid
// over it, against the rules a submit enforces, and reports every issue at
// once. It saves nothing; input may be nil.
func (s *ShortCodeApplicationService) Check(ctx context.Context, input *ShortCodeApplicationInput) (*ShortCodePreflight, error) {
	if input == nil {
		input = &ShortCodeApplicationInput{}
	}
	var resp ShortCodePreflight
	if err := s.client.request(ctx, "POST", "/short_codes/application/preflight", input, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Submit saves the input's answers, then charges the one-time setup fee to
// the workspace's card and submits the application to Sendly for review.
// Set AcceptTerms to accept the fee, the monthly lease from go-live and the
// minimum term. The fee is refunded in full if Sendly rejects the
// application before filing it. Submitting again while it is in review
// answers with AlreadySubmitted and never charges twice.
//
// Errors: 403 verification_required until the workspace's business is
// verified (the answers are saved anyway); 422
// short_code_invalid_application, with every issue, or
// short_code_ai_draft_unedited when the only issues are wording Sendly
// suggested that nobody edited; 402 payment_method_required, payment_failed
// or payment_requires_authentication when the fee can't be taken (the
// application stays a draft); 409 short_code_locked or short_code_conflict.
//
// An Idempotency-Key is sent automatically; pass WithIdempotencyKey to
// supply your own, so a repeat of the same submit returns the first answer.
func (s *ShortCodeApplicationService) Submit(ctx context.Context, input *ShortCodeSubmitInput, opts ...RequestOption) (*ShortCodeApplicationView, error) {
	if input == nil {
		input = &ShortCodeSubmitInput{}
	}
	var resp ShortCodeApplicationView
	if err := s.client.request(ctx, "POST", "/short_codes/application/submit", input, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}
