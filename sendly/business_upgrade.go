package sendly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"reflect"
	"strconv"
)

// BusinessUpgradeService handles the toll-free entity-upgrade
// ("fork-with-new-number") flow: when a customer forms a new legal entity
// (e.g. an LLC), this resource lets them reserve a new toll-free number
// under the new entity, submit it for carrier review, and atomically swap
// to it on approval — without disrupting outbound SMS during the 1-2 week
// review window.
//
// See https://sendly.live/docs/business-upgrade for the full flow.
type BusinessUpgradeService struct {
	client *Client
}

// PreflightCandidate is the candidate entity-upgrade payload validated by
// Preflight before submission. Required fields mirror StartUpgradeParams;
// optional fields surface as warnings rather than blockers.
type PreflightCandidate struct {
	BusinessName          string `json:"businessName"`
	DoingBusinessAs       string `json:"doingBusinessAs,omitempty"`
	BRN                   string `json:"brn"`
	BRNType               string `json:"brnType"`
	BRNCountry            string `json:"brnCountry"`
	EntityType            string `json:"entityType"`
	Website               string `json:"website,omitempty"`
	Address1              string `json:"address1,omitempty"`
	Address2              string `json:"address2,omitempty"`
	City                  string `json:"city,omitempty"`
	State                 string `json:"state,omitempty"`
	Zip                   string `json:"zip,omitempty"`
	AddressCountry        string `json:"addressCountry,omitempty"`
	ContactFirstName      string `json:"contactFirstName,omitempty"`
	ContactLastName       string `json:"contactLastName,omitempty"`
	ContactEmail          string `json:"contactEmail,omitempty"`
	ContactPhone          string `json:"contactPhone,omitempty"`
	MonthlyVolume         string `json:"monthlyVolume,omitempty"`
	UseCase               string `json:"useCase,omitempty"`
	UseCaseSummary        string `json:"useCaseSummary,omitempty"`
	SampleMessages        string `json:"sampleMessages,omitempty"`
	OptInWorkflow         string `json:"optInWorkflow,omitempty"`
	PrivacyURL            string `json:"privacyUrl,omitempty"`
	TermsURL              string `json:"termsUrl,omitempty"`
	AdditionalInformation string `json:"additionalInformation,omitempty"`
	AgeGatedContent       *bool  `json:"ageGatedContent,omitempty"`
}

// PreflightIssue is one validation problem surfaced by Preflight.
// Severity is one of "blocker", "warning", or "info".
type PreflightIssue struct {
	Severity   string `json:"severity"`
	Field      string `json:"field"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// PreflightProposedFix is an auto-correction the server suggests for a
// problematic field. The caller decides whether to apply it.
type PreflightProposedFix struct {
	Field    string      `json:"field"`
	Current  interface{} `json:"current"`
	Proposed interface{} `json:"proposed"`
	Reason   string      `json:"reason"`
}

// PreflightReport is the verdict returned by Preflight. Verdict is one of
// "ready", "warnings", or "blocked"; Country is one of "CA", "US",
// "OTHER", or "UNKNOWN".
type PreflightReport struct {
	VerificationID string                 `json:"verificationId"`
	BusinessName   *string                `json:"businessName"`
	Country        string                 `json:"country"`
	Verdict        string                 `json:"verdict"`
	Issues         []PreflightIssue       `json:"issues"`
	ProposedFixes  []PreflightProposedFix `json:"proposedFixes"`
}

// StartUpgradeParams is the payload submitted to Start (and accepted in
// partial form by Resubmit).
type StartUpgradeParams struct {
	BusinessName          string `json:"businessName"`
	BRN                   string `json:"brn"`
	BRNType               string `json:"brnType"`
	BRNCountry            string `json:"brnCountry"`
	EntityType            string `json:"entityType"`
	DoingBusinessAs       string `json:"doingBusinessAs,omitempty"`
	Website               string `json:"website,omitempty"`
	Address1              string `json:"address1,omitempty"`
	Address2              string `json:"address2,omitempty"`
	City                  string `json:"city,omitempty"`
	State                 string `json:"state,omitempty"`
	Zip                   string `json:"zip,omitempty"`
	AddressCountry        string `json:"addressCountry,omitempty"`
	ContactFirstName      string `json:"contactFirstName,omitempty"`
	ContactLastName       string `json:"contactLastName,omitempty"`
	ContactEmail          string `json:"contactEmail,omitempty"`
	ContactPhone          string `json:"contactPhone,omitempty"`
	MonthlyVolume         string `json:"monthlyVolume,omitempty"`
	UseCase               string `json:"useCase,omitempty"`
	UseCaseSummary        string `json:"useCaseSummary,omitempty"`
	SampleMessages        string `json:"sampleMessages,omitempty"`
	OptInWorkflow         string `json:"optInWorkflow,omitempty"`
	PrivacyURL            string `json:"privacyUrl,omitempty"`
	TermsURL              string `json:"termsUrl,omitempty"`
	AdditionalInformation string `json:"additionalInformation,omitempty"`
	AgeGatedContent       *bool  `json:"ageGatedContent,omitempty"`
}

// EinDocument is the optional EIN/CP-575 PDF attached to Start or Resubmit.
// Filename defaults to "ein-doc.pdf", ContentType to "application/pdf".
type EinDocument struct {
	Data        []byte
	Filename    string
	ContentType string
}

// StartUpgradeResponse is returned by Start when the upgrade is accepted
// and the new toll-free number is reserved.
type StartUpgradeResponse struct {
	Success                  bool   `json:"success"`
	PendingVerificationID    string `json:"pendingVerificationId"`
	TelnyxVerificationID     string `json:"telnyxVerificationId"`
	TollFreeNumber           string `json:"tollFreeNumber"`
	TelnyxMessagingProfileID string `json:"telnyxMessagingProfileId"`
	EinDocStored             bool   `json:"einDocStored"`
	Message                  string `json:"message"`
}

// UpgradePending describes a workspace's pending entity-upgrade. Most fields
// are nullable because they're only populated once the carrier picks the
// submission up.
type UpgradePending struct {
	ID              string  `json:"id"`
	BusinessName    string  `json:"businessName"`
	Status          string  `json:"status"`
	EntityType      *string `json:"entityType"`
	BRNType         *string `json:"brnType"`
	BRNCountry      *string `json:"brnCountry"`
	TollFreeNumber  *string `json:"tollFreeNumber"`
	RejectionReason *string `json:"rejectionReason"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

// UpgradeStatusResponse wraps the pending upgrade (or nil if none in flight).
type UpgradeStatusResponse struct {
	Pending *UpgradePending `json:"pending"`
}

// CancelUpgradeResponse is returned by Cancel. Idempotent — calling Cancel
// when no upgrade is in flight returns Cancelled=false with a no-op message.
type CancelUpgradeResponse struct {
	Success                  bool   `json:"success"`
	Cancelled                bool   `json:"cancelled"`
	CancelledVerificationID  string `json:"cancelledVerificationId,omitempty"`
	Message                  string `json:"message"`
}

// ResubmitUpgradeResponse is returned by Resubmit.
type ResubmitUpgradeResponse struct {
	Success               bool   `json:"success"`
	PendingVerificationID string `json:"pendingVerificationId"`
	Message               string `json:"message"`
}

// DispositionResponse is returned by SetDisposition after an approved
// upgrade. Disposition is one of "moved" or "released".
type DispositionResponse struct {
	Success                  bool   `json:"success"`
	Disposition              string `json:"disposition"`
	SupersededVerificationID string `json:"supersededVerificationId"`
	Message                  string `json:"message"`
}

// BestPrefillFields is the most-recent non-empty value per messaging field
// across the caller's verified workspaces.
type BestPrefillFields struct {
	MonthlyVolume         string `json:"monthlyVolume,omitempty"`
	UseCase               string `json:"useCase,omitempty"`
	UseCaseSummary        string `json:"useCaseSummary,omitempty"`
	SampleMessages        string `json:"sampleMessages,omitempty"`
	OptInWorkflow         string `json:"optInWorkflow,omitempty"`
	OptInImageURLs        string `json:"optInImageUrls,omitempty"`
	OptInSource           string `json:"optInSource,omitempty"`
	PrivacyURL            string `json:"privacyUrl,omitempty"`
	TermsURL              string `json:"termsUrl,omitempty"`
	AdditionalInformation string `json:"additionalInformation,omitempty"`
	IsvReseller           string `json:"isvReseller,omitempty"`
	AgeGatedContent       *bool  `json:"ageGatedContent,omitempty"`
}

// BestPrefillResponse is returned by BestPrefill.
type BestPrefillResponse struct {
	Prefill              BestPrefillFields `json:"prefill"`
	SourceWorkspaceCount int               `json:"sourceWorkspaceCount"`
}

// SetDispositionRequest selects what happens to the old toll-free number
// after an approved upgrade. Disposition is one of "moved" or "released";
// TargetWorkspaceID is required when Disposition=="moved".
type SetDispositionRequest struct {
	Disposition       string `json:"disposition"`
	TargetWorkspaceID string `json:"targetWorkspaceId,omitempty"`
}

// Preflight validates a candidate entity-upgrade payload before submission.
// Returns issues + proposed auto-fixes. No writes — purely advisory.
func (s *BusinessUpgradeService) Preflight(ctx context.Context, candidate *PreflightCandidate) (*PreflightReport, error) {
	if candidate == nil {
		return nil, &ValidationError{APIError: APIError{Message: "candidate is required"}}
	}

	var resp PreflightReport
	if err := s.client.request(ctx, "POST", "/verification/preflight", candidate, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// BestPrefill returns a "best-of" prefill across all the caller's verified
// workspaces — most-recent non-empty values per messaging field. Use this
// to pre-populate the upgrade form for users whose current workspace has
// incomplete data.
func (s *BusinessUpgradeService) BestPrefill(ctx context.Context) (*BestPrefillResponse, error) {
	var resp BestPrefillResponse
	if err := s.client.request(ctx, "GET", "/verification/best-prefill", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Start kicks off an entity upgrade for the given workspace. Auto-provisions
// a new toll-free number + messaging profile and submits to the carrier for
// review. The current toll-free number continues sending throughout the
// 1-2 week carrier review; on approval, an atomic swap promotes the new
// number.
//
// Pass nil for einDoc to skip the optional EIN/CP-575 document upload.
func (s *BusinessUpgradeService) Start(ctx context.Context, workspaceID string, params *StartUpgradeParams, einDoc *EinDocument) (*StartUpgradeResponse, error) {
	if workspaceID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if params == nil {
		return nil, &ValidationError{APIError: APIError{Message: "params is required"}}
	}

	path := fmt.Sprintf("/workspaces/%s/upgrade", url.PathEscape(workspaceID))
	var resp StartUpgradeResponse
	if err := s.client.requestUpgradeMultipart(ctx, path, params, einDoc, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Status returns the pending entity upgrade for the given workspace, or
// a response with Pending=nil if no upgrade is in flight.
func (s *BusinessUpgradeService) Status(ctx context.Context, workspaceID string) (*UpgradeStatusResponse, error) {
	if workspaceID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp UpgradeStatusResponse
	if err := s.client.request(ctx, "GET", fmt.Sprintf("/workspaces/%s/upgrade/status", url.PathEscape(workspaceID)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Cancel cancels a pending entity upgrade for the given workspace. Releases
// the reserved toll-free number, deletes the new messaging profile, and
// removes the stored EIN document. Idempotent.
func (s *BusinessUpgradeService) Cancel(ctx context.Context, workspaceID string) (*CancelUpgradeResponse, error) {
	if workspaceID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp CancelUpgradeResponse
	if err := s.client.request(ctx, "POST", fmt.Sprintf("/workspaces/%s/upgrade/cancel", url.PathEscape(workspaceID)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Resubmit resubmits a rejected (or waiting-for-customer) entity upgrade
// with updated fields and optionally a new EIN document. All StartUpgradeParams
// fields are optional on resubmit — only the fields you want to change need
// be set; the server merges the rest from the existing record.
func (s *BusinessUpgradeService) Resubmit(ctx context.Context, workspaceID string, params *StartUpgradeParams, einDoc *EinDocument) (*ResubmitUpgradeResponse, error) {
	if workspaceID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if params == nil {
		params = &StartUpgradeParams{}
	}

	path := fmt.Sprintf("/workspaces/%s/upgrade/resubmit", url.PathEscape(workspaceID))
	var resp ResubmitUpgradeResponse
	if err := s.client.requestUpgradeMultipart(ctx, path, params, einDoc, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetDisposition decides what happens to the old toll-free number after a
// successful entity-upgrade approval:
//
//   - "moved": keep it active under another workspace owned by the same
//     user. TargetWorkspaceID must be set.
//   - "released": return the number to the carrier pool.
func (s *BusinessUpgradeService) SetDisposition(ctx context.Context, workspaceID string, req *SetDispositionRequest) (*DispositionResponse, error) {
	if workspaceID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "disposition request is required"}}
	}
	if req.Disposition == "" {
		return nil, &ValidationError{APIError: APIError{Message: "disposition is required"}}
	}

	body := map[string]string{"disposition": req.Disposition}
	if req.TargetWorkspaceID != "" {
		body["targetOrgId"] = req.TargetWorkspaceID
	}

	var resp DispositionResponse
	if err := s.client.request(ctx, "POST", fmt.Sprintf("/workspaces/%s/upgrade/disposition", url.PathEscape(workspaceID)), body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// requestUpgradeMultipart submits a StartUpgradeParams payload as multipart
// form-data, optionally including the einDoc file part. Centralised so Start
// and Resubmit share serialization rules.
func (c *Client) requestUpgradeMultipart(ctx context.Context, path string, params interface{}, einDoc *EinDocument, result interface{}) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return &NetworkError{Message: "rate limiter error", Err: err}
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writeUpgradeFields(writer, params); err != nil {
		return err
	}

	if einDoc != nil && len(einDoc.Data) > 0 {
		filename := einDoc.Filename
		if filename == "" {
			filename = "ein-doc.pdf"
		}
		contentType := einDoc.ContentType
		if contentType == "" {
			contentType = "application/pdf"
		}

		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="einDoc"; filename=%q`, filename))
		header.Set("Content-Type", contentType)
		part, err := writer.CreatePart(header)
		if err != nil {
			return &NetworkError{Message: "failed to create einDoc part", Err: err}
		}
		if _, err := part.Write(einDoc.Data); err != nil {
			return &NetworkError{Message: "failed to write einDoc data", Err: err}
		}
	}

	if err := writer.Close(); err != nil {
		return &NetworkError{Message: "failed to close multipart writer", Err: err}
	}

	fullURL := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, &buf)
	if err != nil {
		return &NetworkError{Message: "failed to create request", Err: err}
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sendly-go/"+Version)
	req.Header.Set("Idempotency-Key", generateIdempotencyKey())
	if c.OrganizationID != "" {
		req.Header.Set("X-Organization-Id", c.OrganizationID)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return &NetworkError{Message: "request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &NetworkError{Message: "failed to read response body", Err: err}
	}

	if resp.StatusCode >= 400 {
		return c.handleErrorResponse(resp, respBody)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return &NetworkError{Message: "failed to unmarshal response", Err: err}
		}
	}

	return nil
}

// writeUpgradeFields serialises a StartUpgradeParams (or pointer thereto)
// into multipart form fields keyed by JSON tag — matching the Node SDK's
// FormData shape exactly. Empty strings and nil pointers are skipped so
// the server can merge partial resubmits.
func writeUpgradeFields(writer *multipart.Writer, params interface{}) error {
	v := reflect.ValueOf(params)
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return &ValidationError{APIError: APIError{Message: "params must be a struct"}}
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("json")
		if tag == "-" || tag == "" {
			continue
		}
		name := tag
		if comma := indexComma(tag); comma >= 0 {
			name = tag[:comma]
		}
		if name == "" {
			continue
		}

		fv := v.Field(i)
		switch fv.Kind() {
		case reflect.String:
			if fv.Len() == 0 {
				continue
			}
			if err := writer.WriteField(name, fv.String()); err != nil {
				return &NetworkError{Message: "failed to write form field " + name, Err: err}
			}
		case reflect.Ptr:
			if fv.IsNil() {
				continue
			}
			elem := fv.Elem()
			switch elem.Kind() {
			case reflect.Bool:
				if err := writer.WriteField(name, strconv.FormatBool(elem.Bool())); err != nil {
					return &NetworkError{Message: "failed to write form field " + name, Err: err}
				}
			case reflect.String:
				if elem.Len() == 0 {
					continue
				}
				if err := writer.WriteField(name, elem.String()); err != nil {
					return &NetworkError{Message: "failed to write form field " + name, Err: err}
				}
			default:
				if err := writer.WriteField(name, fmt.Sprintf("%v", elem.Interface())); err != nil {
					return &NetworkError{Message: "failed to write form field " + name, Err: err}
				}
			}
		case reflect.Bool:
			if err := writer.WriteField(name, strconv.FormatBool(fv.Bool())); err != nil {
				return &NetworkError{Message: "failed to write form field " + name, Err: err}
			}
		default:
			if !fv.IsZero() {
				if err := writer.WriteField(name, fmt.Sprintf("%v", fv.Interface())); err != nil {
					return &NetworkError{Message: "failed to write form field " + name, Err: err}
				}
			}
		}
	}
	return nil
}

func indexComma(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return i
		}
	}
	return -1
}
