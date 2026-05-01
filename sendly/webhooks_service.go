package sendly

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// WebhooksService provides methods for managing webhook endpoints.
type WebhooksService struct {
	client *Client
}

// webhookAPIResponse is the API response with snake_case fields.
type webhookAPIResponse struct {
	ID                   string                 `json:"id"`
	URL                  string                 `json:"url"`
	Events               []string               `json:"events"`
	Description          *string                `json:"description,omitempty"`
	Mode                 string                 `json:"mode"`
	IsActive             bool                   `json:"is_active"`
	FailureCount         int                    `json:"failure_count"`
	LastFailureAt        *string                `json:"last_failure_at,omitempty"`
	CircuitState         string                 `json:"circuit_state"`
	CircuitOpenedAt      *string                `json:"circuit_opened_at,omitempty"`
	APIVersion           string                 `json:"api_version"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt            string                 `json:"created_at"`
	UpdatedAt            string                 `json:"updated_at"`
	TotalDeliveries      int                    `json:"total_deliveries"`
	SuccessfulDeliveries int                    `json:"successful_deliveries"`
	SuccessRate          float64                `json:"success_rate"`
	LastDeliveryAt       *string                `json:"last_delivery_at,omitempty"`
	Secret               string                 `json:"secret,omitempty"`
}

// webhookDeliveryAPIResponse is the API response for webhook delivery.
type webhookDeliveryAPIResponse struct {
	ID                 string  `json:"id"`
	WebhookID          string  `json:"webhook_id"`
	EventID            string  `json:"event_id"`
	EventType          string  `json:"event_type"`
	AttemptNumber      int     `json:"attempt_number"`
	MaxAttempts        int     `json:"max_attempts"`
	Status             string  `json:"status"`
	ResponseStatusCode *int    `json:"response_status_code,omitempty"`
	ResponseTimeMs     *int    `json:"response_time_ms,omitempty"`
	ErrorMessage       *string `json:"error_message,omitempty"`
	ErrorCode          *string `json:"error_code,omitempty"`
	NextRetryAt        *string `json:"next_retry_at,omitempty"`
	CreatedAt          string  `json:"created_at"`
	DeliveredAt        *string `json:"delivered_at,omitempty"`
}

// transformWebhook converts API response to SDK type.
func transformWebhook(api webhookAPIResponse) Webhook {
	mode := WebhookMode(api.Mode)
	if mode == "" {
		mode = WebhookModeAll
	}
	return Webhook{
		ID:                   api.ID,
		URL:                  api.URL,
		Events:               api.Events,
		Description:          api.Description,
		Mode:                 mode,
		IsActive:             api.IsActive,
		FailureCount:         api.FailureCount,
		LastFailureAt:        api.LastFailureAt,
		CircuitState:         CircuitState(api.CircuitState),
		CircuitOpenedAt:      api.CircuitOpenedAt,
		APIVersion:           api.APIVersion,
		Metadata:             api.Metadata,
		CreatedAt:            api.CreatedAt,
		UpdatedAt:            api.UpdatedAt,
		TotalDeliveries:      api.TotalDeliveries,
		SuccessfulDeliveries: api.SuccessfulDeliveries,
		SuccessRate:          api.SuccessRate,
		LastDeliveryAt:       api.LastDeliveryAt,
	}
}

// transformDelivery converts API response to SDK type.
func transformDelivery(api webhookDeliveryAPIResponse) WebhookDelivery {
	return WebhookDelivery{
		ID:                 api.ID,
		WebhookID:          api.WebhookID,
		EventID:            api.EventID,
		EventType:          api.EventType,
		AttemptNumber:      api.AttemptNumber,
		MaxAttempts:        api.MaxAttempts,
		Status:             DeliveryStatus(api.Status),
		ResponseStatusCode: api.ResponseStatusCode,
		ResponseTimeMs:     api.ResponseTimeMs,
		ErrorMessage:       api.ErrorMessage,
		ErrorCode:          api.ErrorCode,
		NextRetryAt:        api.NextRetryAt,
		CreatedAt:          api.CreatedAt,
		DeliveredAt:        api.DeliveredAt,
	}
}

// Create creates a new webhook endpoint.
func (s *WebhooksService) Create(ctx context.Context, req CreateWebhookRequest) (*WebhookCreatedResponse, error) {
	if req.URL == "" || !strings.HasPrefix(req.URL, "https://") {
		return nil, errors.New("webhook URL must be HTTPS")
	}
	if len(req.Events) == 0 {
		return nil, errors.New("at least one event type is required")
	}

	var apiResp webhookAPIResponse
	if err := s.client.request(ctx, "POST", "/webhooks", req, &apiResp); err != nil {
		return nil, err
	}

	webhook := transformWebhook(apiResp)
	return &WebhookCreatedResponse{
		Webhook: webhook,
		Secret:  apiResp.Secret,
	}, nil
}

// List returns all webhooks for the account.
func (s *WebhooksService) List(ctx context.Context) ([]Webhook, error) {
	var apiResp []webhookAPIResponse
	if err := s.client.request(ctx, "GET", "/webhooks", nil, &apiResp); err != nil {
		return nil, err
	}

	webhooks := make([]Webhook, len(apiResp))
	for i, api := range apiResp {
		webhooks[i] = transformWebhook(api)
	}
	return webhooks, nil
}

// Get retrieves a specific webhook by ID.
func (s *WebhooksService) Get(ctx context.Context, webhookID string) (*Webhook, error) {
	if webhookID == "" || !strings.HasPrefix(webhookID, "whk_") {
		return nil, errors.New("invalid webhook ID format")
	}

	var apiResp webhookAPIResponse
	if err := s.client.request(ctx, "GET", "/webhooks/"+webhookID, nil, &apiResp); err != nil {
		return nil, err
	}

	webhook := transformWebhook(apiResp)
	return &webhook, nil
}

// Update updates a webhook configuration.
func (s *WebhooksService) Update(ctx context.Context, webhookID string, req UpdateWebhookRequest) (*Webhook, error) {
	if webhookID == "" || !strings.HasPrefix(webhookID, "whk_") {
		return nil, errors.New("invalid webhook ID format")
	}

	if req.URL != nil && !strings.HasPrefix(*req.URL, "https://") {
		return nil, errors.New("webhook URL must be HTTPS")
	}

	var apiResp webhookAPIResponse
	if err := s.client.request(ctx, "PATCH", "/webhooks/"+webhookID, req, &apiResp); err != nil {
		return nil, err
	}

	webhook := transformWebhook(apiResp)
	return &webhook, nil
}

// Delete removes a webhook.
func (s *WebhooksService) Delete(ctx context.Context, webhookID string) error {
	if webhookID == "" || !strings.HasPrefix(webhookID, "whk_") {
		return errors.New("invalid webhook ID format")
	}

	return s.client.request(ctx, "DELETE", "/webhooks/"+webhookID, nil, nil)
}

// Test sends a test event to a webhook endpoint.
func (s *WebhooksService) Test(ctx context.Context, webhookID string) (*WebhookTestResult, error) {
	if webhookID == "" || !strings.HasPrefix(webhookID, "whk_") {
		return nil, errors.New("invalid webhook ID format")
	}

	var result WebhookTestResult
	if err := s.client.request(ctx, "POST", "/webhooks/"+webhookID+"/test", nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ResetCircuit resets the circuit breaker for a webhook.
func (s *WebhooksService) ResetCircuit(ctx context.Context, webhookID string) (map[string]interface{}, error) {
	if webhookID == "" || !strings.HasPrefix(webhookID, "whk_") {
		return nil, errors.New("invalid webhook ID format")
	}

	var result map[string]interface{}
	if err := s.client.request(ctx, "POST", "/webhooks/"+webhookID+"/reset-circuit", nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// RedeliverOptions configures Webhooks.Redeliver. Pointer fields are
// optional — leave nil to use the server defaults.
type RedeliverOptions struct {
	// Since is the earliest delivery created_at to consider, ISO-8601.
	// Default: now − 24h.
	Since *string `json:"since,omitempty"`
	// Until is the latest delivery created_at to consider, ISO-8601.
	// Default: now.
	Until *string `json:"until,omitempty"`
	// EventTypes filters by event type. Default: all.
	EventTypes []string `json:"event_types,omitempty"`
	// Statuses filters by current delivery status.
	// Default: ["failed", "cancelled"].
	Statuses []string `json:"statuses,omitempty"`
	// Limit caps the number of deliveries requeued (default 1000, max 10000).
	Limit *int `json:"limit,omitempty"`
}

// RedeliverResult is returned by Webhooks.Redeliver.
type RedeliverResult struct {
	Message     string   `json:"message"`
	Requeued    int      `json:"requeued"`
	Skipped     int      `json:"skipped"`
	Truncated   bool     `json:"truncated"`
	WindowSize  int      `json:"window_size"`
	DeliveryIDs []string `json:"delivery_ids"`
	Since       string   `json:"since"`
	Until       string   `json:"until"`
	Limit       int      `json:"limit"`
}

// Redeliver replays failed or cancelled webhook deliveries from the audit
// log. Each replay creates a new delivery row preserving the original
// event_id so customers can dedupe.
//
// Rejects with HTTP 409 if the circuit is currently open — call
// ResetCircuit first.
func (s *WebhooksService) Redeliver(ctx context.Context, webhookID string, opts *RedeliverOptions) (*RedeliverResult, error) {
	if webhookID == "" || !strings.HasPrefix(webhookID, "whk_") {
		return nil, errors.New("invalid webhook ID format")
	}

	var body interface{}
	if opts != nil {
		body = opts
	}

	var result RedeliverResult
	if err := s.client.request(ctx, "POST", "/webhooks/"+webhookID+"/redeliver", body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// BackfillOptions configures Webhooks.Backfill.
type BackfillOptions struct {
	Since      *string  `json:"since,omitempty"`
	Until      *string  `json:"until,omitempty"`
	EventTypes []string `json:"event_types,omitempty"`
	Limit      *int     `json:"limit,omitempty"`
}

// BackfillResult is returned by Webhooks.Backfill.
type BackfillResult struct {
	Message           string         `json:"message"`
	Synthesized       int            `json:"synthesized"`
	ByType            map[string]int `json:"by_type"`
	Truncated         bool           `json:"truncated"`
	CandidatesScanned int            `json:"candidates_scanned"`
	DeliveryIDs       []string       `json:"delivery_ids"`
	Since             string         `json:"since"`
	Until             string         `json:"until"`
	Limit             int            `json:"limit"`
}

// Backfill synthesizes webhook deliveries from the underlying message log
// for events that have no audit row. Use this when a circuit-breaker
// outage left events with no delivery record (the case Redeliver cannot
// recover). Synthesized events have fresh IDs — clients should dedupe by
// event.data.object.id (the message ID).
//
// Rejects with HTTP 409 if the circuit is currently open — call
// ResetCircuit first.
func (s *WebhooksService) Backfill(ctx context.Context, webhookID string, opts *BackfillOptions) (*BackfillResult, error) {
	if webhookID == "" || !strings.HasPrefix(webhookID, "whk_") {
		return nil, errors.New("invalid webhook ID format")
	}

	var body interface{}
	if opts != nil {
		body = opts
	}

	var result BackfillResult
	if err := s.client.request(ctx, "POST", "/webhooks/"+webhookID+"/backfill", body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// RotateSecret rotates the webhook signing secret.
func (s *WebhooksService) RotateSecret(ctx context.Context, webhookID string) (*WebhookSecretRotation, error) {
	if webhookID == "" || !strings.HasPrefix(webhookID, "whk_") {
		return nil, errors.New("invalid webhook ID format")
	}

	// Raw response with snake_case
	var rawResp struct {
		Webhook            webhookAPIResponse `json:"webhook"`
		NewSecret          string             `json:"new_secret"`
		OldSecretExpiresAt string             `json:"old_secret_expires_at"`
		Message            string             `json:"message"`
	}

	if err := s.client.request(ctx, "POST", "/webhooks/"+webhookID+"/rotate-secret", nil, &rawResp); err != nil {
		return nil, err
	}

	return &WebhookSecretRotation{
		Webhook:            transformWebhook(rawResp.Webhook),
		NewSecret:          rawResp.NewSecret,
		OldSecretExpiresAt: rawResp.OldSecretExpiresAt,
		Message:            rawResp.Message,
	}, nil
}

// GetDeliveries retrieves delivery history for a webhook.
func (s *WebhooksService) GetDeliveries(ctx context.Context, webhookID string) ([]WebhookDelivery, error) {
	if webhookID == "" || !strings.HasPrefix(webhookID, "whk_") {
		return nil, errors.New("invalid webhook ID format")
	}

	var apiResp []webhookDeliveryAPIResponse
	if err := s.client.request(ctx, "GET", "/webhooks/"+webhookID+"/deliveries", nil, &apiResp); err != nil {
		return nil, err
	}

	deliveries := make([]WebhookDelivery, len(apiResp))
	for i, api := range apiResp {
		deliveries[i] = transformDelivery(api)
	}
	return deliveries, nil
}

// RetryDelivery retries a failed delivery.
func (s *WebhooksService) RetryDelivery(ctx context.Context, webhookID, deliveryID string) error {
	if webhookID == "" || !strings.HasPrefix(webhookID, "whk_") {
		return errors.New("invalid webhook ID format")
	}
	if deliveryID == "" || !strings.HasPrefix(deliveryID, "del_") {
		return errors.New("invalid delivery ID format")
	}

	path := fmt.Sprintf("/webhooks/%s/deliveries/%s/retry", webhookID, deliveryID)
	return s.client.request(ctx, "POST", path, nil, nil)
}

// ListEventTypes returns available event types.
func (s *WebhooksService) ListEventTypes(ctx context.Context) ([]string, error) {
	var resp struct {
		Events []struct {
			Type string `json:"type"`
		} `json:"events"`
	}

	if err := s.client.request(ctx, "GET", "/webhooks/event-types", nil, &resp); err != nil {
		return nil, err
	}

	eventTypes := make([]string, len(resp.Events))
	for i, e := range resp.Events {
		eventTypes[i] = e.Type
	}
	return eventTypes, nil
}
