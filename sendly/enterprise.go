package sendly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
)

type EnterpriseService struct {
	client     *Client
	Workspaces *WorkspacesService
	Webhooks   *EnterpriseWebhooksService
	Analytics  *AnalyticsService
	Settings   *SettingsService
	Billing    *BillingService
	Credits    *CreditsService
}

type CreditsService struct {
	client *Client
}

type WorkspacesService struct {
	client *Client
}

type EnterpriseWebhooksService struct {
	client *Client
}

type AnalyticsService struct {
	client *Client
}

type SettingsService struct {
	client *Client
}

type BillingService struct {
	client *Client
}

func (s *EnterpriseService) GetAccount(ctx context.Context) (*EnterpriseAccount, error) {
	var resp EnterpriseAccount
	if err := s.client.request(ctx, "GET", "/enterprise/account", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *EnterpriseService) Provision(ctx context.Context, opts *ProvisionWorkspaceRequest) (*ProvisionWorkspaceResponse, error) {
	if opts == nil || opts.Name == "" {
		return nil, &ValidationError{APIError: APIError{Message: "name is required"}}
	}

	var resp ProvisionWorkspaceResponse
	if err := s.client.request(ctx, "POST", "/enterprise/workspaces/provision", opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *EnterpriseService) GenerateBusinessPage(ctx context.Context, opts *GenerateBusinessPageRequest) (*GenerateBusinessPageResponse, error) {
	if opts == nil || opts.BusinessName == "" {
		return nil, &ValidationError{APIError: APIError{Message: "businessName is required"}}
	}

	var resp GenerateBusinessPageResponse
	if err := s.client.request(ctx, "POST", "/enterprise/business-page/generate", opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *EnterpriseService) UploadVerificationDocument(ctx context.Context, filename string, file io.Reader, workspaceID, verificationID string) (*VerificationDocumentUploadResponse, error) {
	if file == nil {
		return nil, &ValidationError{APIError: APIError{Message: "file is required"}}
	}
	if filename == "" {
		return nil, &ValidationError{APIError: APIError{Message: "filename is required"}}
	}

	if err := s.client.rateLimiter.Wait(ctx); err != nil {
		return nil, &NetworkError{Message: "rate limiter error", Err: err}
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return nil, &NetworkError{Message: "failed to create form file", Err: err}
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, &NetworkError{Message: "failed to write file data", Err: err}
	}

	if workspaceID != "" {
		if err := writer.WriteField("workspaceId", workspaceID); err != nil {
			return nil, &NetworkError{Message: "failed to write workspaceId field", Err: err}
		}
	}
	if verificationID != "" {
		if err := writer.WriteField("verificationId", verificationID); err != nil {
			return nil, &NetworkError{Message: "failed to write verificationId field", Err: err}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, &NetworkError{Message: "failed to close multipart writer", Err: err}
	}

	fullURL := s.client.BaseURL + "/enterprise/verification-document/upload"

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, &buf)
	if err != nil {
		return nil, &NetworkError{Message: "failed to create request", Err: err}
	}

	req.Header.Set("Authorization", "Bearer "+s.client.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sendly-go/"+Version)
	req.Header.Set("Idempotency-Key", generateIdempotencyKey())
	if s.client.OrganizationID != "" {
		req.Header.Set("X-Organization-Id", s.client.OrganizationID)
	}

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Message: "request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &NetworkError{Message: "failed to read response body", Err: err}
	}

	if resp.StatusCode >= 400 {
		return nil, s.client.handleErrorResponse(resp, respBody)
	}

	var result VerificationDocumentUploadResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, &NetworkError{Message: "failed to unmarshal response", Err: err}
	}

	return &result, nil
}

func (s *WorkspacesService) Create(ctx context.Context, name, description string) (*EnterpriseWorkspace, error) {
	if name == "" {
		return nil, &ValidationError{APIError: APIError{Message: "name is required"}}
	}

	body := map[string]string{"name": name}
	if description != "" {
		body["description"] = description
	}

	var resp EnterpriseWorkspace
	if err := s.client.request(ctx, "POST", "/enterprise/workspaces", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) List(ctx context.Context) (*EnterpriseAccount, error) {
	var resp EnterpriseAccount
	if err := s.client.request(ctx, "GET", "/enterprise/workspaces", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) Get(ctx context.Context, id string) (*WorkspaceDetail, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp WorkspaceDetail
	if err := s.client.request(ctx, "GET", fmt.Sprintf("/enterprise/workspaces/%s", url.PathEscape(id)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	return s.client.request(ctx, "DELETE", fmt.Sprintf("/enterprise/workspaces/%s", url.PathEscape(id)), nil, nil)
}

// SubmitVerification submits (or resubmits) a verification for an enterprise
// workspace.
//
// Partial-update friendly (May 2026): for resubmits on an existing workspace
// you only need to send the fields you want to change — everything else is
// preserved from the existing record. Hosted page URLs (/biz/, /opt-in/,
// /legal/) generated during provision are auto-preserved.
//
// For initial submission the server requires at minimum: BusinessName,
// Website, Address, Contact, UseCase, UseCaseSummary, SampleMessages,
// OptInWorkflow. For sole proprietors leave BRN, BRNType, BRNCountry nil —
// the server strips them before forwarding to the carrier.
func (s *WorkspacesService) SubmitVerification(ctx context.Context, id string, data *VerificationSubmitInput) (*VerificationResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if data == nil {
		return nil, &ValidationError{APIError: APIError{Message: "verification data is required"}}
	}

	var resp VerificationResponse
	if err := s.client.request(ctx, "POST", fmt.Sprintf("/enterprise/workspaces/%s/verification/submit", url.PathEscape(id)), data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ResubmitVerification is a convenience alias for SubmitVerification that
// reads more naturally when you only want to change a few fields after a
// rejection. All unset (nil) fields are omitted from the request body, so
// the server merges the partial update with the existing verification
// record.
func (s *WorkspacesService) ResubmitVerification(ctx context.Context, workspaceID string, partial *VerificationSubmitInput) (*VerificationResponse, error) {
	return s.SubmitVerification(ctx, workspaceID, partial)
}

func (s *WorkspacesService) InheritVerification(ctx context.Context, id, sourceID string) (*InheritVerificationResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if sourceID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "source workspace ID is required"}}
	}

	body := map[string]string{"sourceWorkspaceId": sourceID}
	var resp InheritVerificationResponse
	if err := s.client.request(ctx, "POST", fmt.Sprintf("/enterprise/workspaces/%s/verification/inherit", url.PathEscape(id)), body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) GetVerification(ctx context.Context, id string) (*VerificationStatusResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp VerificationStatusResponse
	if err := s.client.request(ctx, "GET", fmt.Sprintf("/enterprise/workspaces/%s/verification", url.PathEscape(id)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) TransferCredits(ctx context.Context, id, sourceID string, amount int) (*TransferCreditsResult, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if sourceID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "source workspace ID is required"}}
	}
	if amount <= 0 {
		return nil, &ValidationError{APIError: APIError{Message: "amount must be a positive integer"}}
	}

	body := map[string]interface{}{
		"sourceWorkspaceId": sourceID,
		"amount":            amount,
	}
	var resp TransferCreditsResult
	if err := s.client.request(ctx, "POST", fmt.Sprintf("/enterprise/workspaces/%s/transfer-credits", url.PathEscape(id)), body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) GetCredits(ctx context.Context, id string) (*WorkspaceCredits, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp WorkspaceCredits
	if err := s.client.request(ctx, "GET", fmt.Sprintf("/enterprise/workspaces/%s/credits", url.PathEscape(id)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) CreateKey(ctx context.Context, id, name, keyType string) (*WorkspaceKeyCreated, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if name == "" {
		return nil, &ValidationError{APIError: APIError{Message: "key name is required"}}
	}

	body := map[string]string{"name": name}
	if keyType != "" {
		body["type"] = keyType
	}

	var resp WorkspaceKeyCreated
	if err := s.client.request(ctx, "POST", fmt.Sprintf("/enterprise/workspaces/%s/keys", url.PathEscape(id)), body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) ListKeys(ctx context.Context, id string) ([]WorkspaceKey, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp []WorkspaceKey
	if err := s.client.request(ctx, "GET", fmt.Sprintf("/enterprise/workspaces/%s/keys", url.PathEscape(id)), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkspacesService) RevokeKey(ctx context.Context, workspaceID, keyID string) error {
	if workspaceID == "" {
		return &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if keyID == "" {
		return &ValidationError{APIError: APIError{Message: "key ID is required"}}
	}

	return s.client.request(ctx, "DELETE", fmt.Sprintf("/enterprise/workspaces/%s/keys/%s", url.PathEscape(workspaceID), url.PathEscape(keyID)), nil, nil)
}

func (s *WorkspacesService) ListOptInPages(ctx context.Context, id string) ([]OptInPage, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp []OptInPage
	if err := s.client.request(ctx, "GET", fmt.Sprintf("/enterprise/workspaces/%s/opt-in-pages", url.PathEscape(id)), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkspacesService) CreateOptInPage(ctx context.Context, id string, opts *CreateOptInPageRequest) (*CreateOptInPageResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if opts == nil || opts.BusinessName == "" {
		return nil, &ValidationError{APIError: APIError{Message: "businessName is required"}}
	}

	var resp CreateOptInPageResponse
	if err := s.client.request(ctx, "POST", fmt.Sprintf("/enterprise/workspaces/%s/opt-in-pages", url.PathEscape(id)), opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) UpdateOptInPage(ctx context.Context, id, pageID string, opts *UpdateOptInPageRequest) (*OptInPage, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if pageID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "page ID is required"}}
	}

	var resp OptInPage
	if err := s.client.request(ctx, "PATCH", fmt.Sprintf("/enterprise/workspaces/%s/opt-in-pages/%s", url.PathEscape(id), url.PathEscape(pageID)), opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) DeleteOptInPage(ctx context.Context, id, pageID string) error {
	if id == "" {
		return &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if pageID == "" {
		return &ValidationError{APIError: APIError{Message: "page ID is required"}}
	}

	return s.client.request(ctx, "DELETE", fmt.Sprintf("/enterprise/workspaces/%s/opt-in-pages/%s", url.PathEscape(id), url.PathEscape(pageID)), nil, nil)
}

func (s *WorkspacesService) SetWebhook(ctx context.Context, id string, opts *SetWorkspaceWebhookRequest) (*SetWorkspaceWebhookResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if opts == nil || opts.URL == "" {
		return nil, &ValidationError{APIError: APIError{Message: "webhook URL is required"}}
	}

	var resp SetWorkspaceWebhookResponse
	if err := s.client.request(ctx, "PUT", fmt.Sprintf("/enterprise/workspaces/%s/webhooks", url.PathEscape(id)), opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) ListWebhooks(ctx context.Context, id string) ([]WorkspaceWebhook, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp []WorkspaceWebhook
	if err := s.client.request(ctx, "GET", fmt.Sprintf("/enterprise/workspaces/%s/webhooks", url.PathEscape(id)), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkspacesService) DeleteWebhooks(ctx context.Context, id string, webhookID string) error {
	if id == "" {
		return &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	path := fmt.Sprintf("/enterprise/workspaces/%s/webhooks", url.PathEscape(id))
	if webhookID != "" {
		params := make(map[string]string)
		params["webhookId"] = webhookID
		path += buildQueryString(params)
	}

	return s.client.request(ctx, "DELETE", path, nil, nil)
}

func (s *WorkspacesService) TestWebhook(ctx context.Context, id string) (*EnterpriseWebhookTestResult, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp EnterpriseWebhookTestResult
	if err := s.client.request(ctx, "POST", fmt.Sprintf("/enterprise/workspaces/%s/webhooks/test", url.PathEscape(id)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) Suspend(ctx context.Context, id string, reason string) (*SuspendWorkspaceResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	body := &SuspendWorkspaceRequest{}
	if reason != "" {
		body.Reason = reason
	}

	var resp SuspendWorkspaceResponse
	if err := s.client.request(ctx, "POST", fmt.Sprintf("/enterprise/workspaces/%s/suspend", url.PathEscape(id)), body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) Resume(ctx context.Context, id string) (*ResumeWorkspaceResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp ResumeWorkspaceResponse
	if err := s.client.request(ctx, "POST", fmt.Sprintf("/enterprise/workspaces/%s/resume", url.PathEscape(id)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) ProvisionBulk(ctx context.Context, workspaces []BulkProvisionWorkspace) (*BulkProvisionResult, error) {
	if len(workspaces) == 0 {
		return nil, &ValidationError{APIError: APIError{Message: "workspaces array is required"}}
	}
	if len(workspaces) > 50 {
		return nil, &ValidationError{APIError: APIError{Message: "maximum 50 workspaces per bulk provision"}}
	}

	body := &BulkProvisionRequest{Workspaces: workspaces}
	var resp BulkProvisionResult
	if err := s.client.request(ctx, "POST", "/enterprise/workspaces/provision/bulk", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) SetCustomDomain(ctx context.Context, id, pageID, domain string) (*SetCustomDomainResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if pageID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "page ID is required"}}
	}
	if domain == "" {
		return nil, &ValidationError{APIError: APIError{Message: "domain is required"}}
	}

	body := &SetCustomDomainRequest{Domain: domain}
	var resp SetCustomDomainResponse
	if err := s.client.request(ctx, "PUT", fmt.Sprintf("/enterprise/workspaces/%s/pages/%s/domain", url.PathEscape(id), url.PathEscape(pageID)), body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) SendInvitation(ctx context.Context, id string, opts *SendInvitationRequest) (*Invitation, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if opts == nil || opts.Email == "" {
		return nil, &ValidationError{APIError: APIError{Message: "email is required"}}
	}
	if opts.Role == "" {
		return nil, &ValidationError{APIError: APIError{Message: "role is required"}}
	}

	var resp Invitation
	if err := s.client.request(ctx, "POST", fmt.Sprintf("/enterprise/workspaces/%s/invitations", url.PathEscape(id)), opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) ListInvitations(ctx context.Context, id string) ([]Invitation, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp []Invitation
	if err := s.client.request(ctx, "GET", fmt.Sprintf("/enterprise/workspaces/%s/invitations", url.PathEscape(id)), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *WorkspacesService) CancelInvitation(ctx context.Context, id, inviteID string) error {
	if id == "" {
		return &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if inviteID == "" {
		return &ValidationError{APIError: APIError{Message: "invitation ID is required"}}
	}

	return s.client.request(ctx, "DELETE", fmt.Sprintf("/enterprise/workspaces/%s/invitations/%s", url.PathEscape(id), url.PathEscape(inviteID)), nil, nil)
}

func (s *WorkspacesService) GetQuota(ctx context.Context, id string) (*QuotaSettings, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}

	var resp QuotaSettings
	if err := s.client.request(ctx, "GET", fmt.Sprintf("/enterprise/workspaces/%s/quota", url.PathEscape(id)), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *WorkspacesService) SetQuota(ctx context.Context, id string, opts *UpdateQuotaRequest) (*QuotaSettings, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "workspace ID is required"}}
	}
	if opts == nil {
		return nil, &ValidationError{APIError: APIError{Message: "quota settings are required"}}
	}

	var resp QuotaSettings
	if err := s.client.request(ctx, "PUT", fmt.Sprintf("/enterprise/workspaces/%s/quota", url.PathEscape(id)), opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *EnterpriseWebhooksService) Set(ctx context.Context, webhookURL string) (*EnterpriseWebhook, error) {
	if webhookURL == "" {
		return nil, &ValidationError{APIError: APIError{Message: "url is required"}}
	}

	body := map[string]string{"url": webhookURL}
	var resp EnterpriseWebhook
	if err := s.client.request(ctx, "POST", "/enterprise/webhooks", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *EnterpriseWebhooksService) Get(ctx context.Context) (*EnterpriseWebhook, error) {
	var resp EnterpriseWebhook
	if err := s.client.request(ctx, "GET", "/enterprise/webhooks", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *EnterpriseWebhooksService) Delete(ctx context.Context) error {
	return s.client.request(ctx, "DELETE", "/enterprise/webhooks", nil, nil)
}

func (s *EnterpriseWebhooksService) Test(ctx context.Context) (*EnterpriseWebhookTestResult, error) {
	var resp EnterpriseWebhookTestResult
	if err := s.client.request(ctx, "POST", "/enterprise/webhooks/test", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *EnterpriseWebhooksService) RotateSecret(ctx context.Context) (*EnterpriseWebhookRotateSecretResponse, error) {
	var resp EnterpriseWebhookRotateSecretResponse
	if err := s.client.request(ctx, "POST", "/enterprise/webhooks/rotate-secret", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *AnalyticsService) Overview(ctx context.Context) (*AnalyticsOverview, error) {
	var resp AnalyticsOverview
	if err := s.client.request(ctx, "GET", "/enterprise/analytics/overview", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *AnalyticsService) Messages(ctx context.Context, opts *AnalyticsMessagesOptions) (*AnalyticsMessagesResponse, error) {
	path := "/enterprise/analytics/messages"
	if opts != nil {
		params := make(map[string]string)
		if opts.Period != "" {
			params["period"] = opts.Period
		}
		if opts.WorkspaceID != "" {
			params["workspaceId"] = opts.WorkspaceID
		}
		path += buildQueryString(params)
	}

	var resp AnalyticsMessagesResponse
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *AnalyticsService) Delivery(ctx context.Context) ([]DeliveryAnalytics, error) {
	var resp []DeliveryAnalytics
	if err := s.client.request(ctx, "GET", "/enterprise/analytics/delivery", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *AnalyticsService) Credits(ctx context.Context, opts *AnalyticsCreditsOptions) (*AnalyticsCreditsResponse, error) {
	path := "/enterprise/analytics/credits"
	if opts != nil {
		params := make(map[string]string)
		if opts.Period != "" {
			params["period"] = opts.Period
		}
		path += buildQueryString(params)
	}

	var resp AnalyticsCreditsResponse
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *SettingsService) GetAutoTopUp(ctx context.Context) (*AutoTopUpSettings, error) {
	var resp AutoTopUpSettings
	if err := s.client.request(ctx, "GET", "/enterprise/settings/auto-top-up", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *SettingsService) UpdateAutoTopUp(ctx context.Context, opts *UpdateAutoTopUpRequest) (*AutoTopUpSettings, error) {
	if opts == nil {
		return nil, &ValidationError{APIError: APIError{Message: "auto top-up settings are required"}}
	}

	var resp AutoTopUpSettings
	if err := s.client.request(ctx, "PUT", "/enterprise/settings/auto-top-up", opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CreditsService) Get(ctx context.Context) (*PoolCredits, error) {
	var resp PoolCredits
	if err := s.client.request(ctx, "GET", "/enterprise/credits", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *CreditsService) Deposit(ctx context.Context, amount int, description string) (*PoolCredits, error) {
	if amount <= 0 {
		return nil, &ValidationError{APIError: APIError{Message: "amount must be a positive integer"}}
	}

	body := map[string]interface{}{
		"amount": amount,
	}
	if description != "" {
		body["description"] = description
	}

	var resp PoolCredits
	if err := s.client.request(ctx, "POST", "/enterprise/credits/deposit", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *BillingService) GetBreakdown(ctx context.Context, opts *BillingBreakdownOptions) (*BillingBreakdown, error) {
	path := "/enterprise/billing/workspace-breakdown"
	if opts != nil {
		params := make(map[string]string)
		if opts.Period != "" {
			params["period"] = opts.Period
		}
		if opts.Page > 0 {
			params["page"] = strconv.Itoa(opts.Page)
		}
		if opts.Limit > 0 {
			params["limit"] = strconv.Itoa(opts.Limit)
		}
		path += buildQueryString(params)
	}

	var resp BillingBreakdown
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
