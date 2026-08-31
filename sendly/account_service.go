package sendly

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
)

// AccountService provides methods for accessing account information.
type AccountService struct {
	client *Client
}

// accountAPIResponse is the API response envelope. The account fields arrive
// nested under "user" with camelCase keys.
type accountAPIResponse struct {
	User struct {
		ID        string  `json:"id"`
		Email     string  `json:"email"`
		Name      *string `json:"name,omitempty"`
		CreatedAt string  `json:"createdAt"`
	} `json:"user"`
}

// creditsAPIResponse is the API response with snake_case fields.
type creditsAPIResponse struct {
	Balance          int `json:"balance"`
	ReservedBalance  int `json:"reserved_balance"`
	AvailableBalance int `json:"available_balance"`
}

// transactionAPIResponse is the API response with snake_case fields.
type transactionAPIResponse struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Amount       int     `json:"amount"`
	BalanceAfter int     `json:"balance_after"`
	Description  string  `json:"description"`
	MessageID    *string `json:"message_id,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

// apiKeyAPIResponse is the API response for a single API key.
type apiKeyAPIResponse struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Prefix     string   `json:"prefix"`
	Scopes     []string `json:"scopes"`
	IsActive   bool     `json:"isActive"`
	CreatedAt  string   `json:"createdAt"`
	LastUsedAt *string  `json:"lastUsedAt,omitempty"`
	ExpiresAt  *string  `json:"expiresAt,omitempty"`
	RevokedAt  *string  `json:"revokedAt,omitempty"`
}

func (a apiKeyAPIResponse) toAPIKey() APIKey {
	return APIKey{
		ID:          a.ID,
		Name:        a.Name,
		Type:        a.Type,
		Prefix:      a.Prefix,
		Permissions: a.Scopes,
		CreatedAt:   a.CreatedAt,
		LastUsedAt:  a.LastUsedAt,
		ExpiresAt:   a.ExpiresAt,
		IsRevoked:   !a.IsActive,
	}
}

// Get retrieves account information.
func (s *AccountService) Get(ctx context.Context) (*Account, error) {
	var apiResp accountAPIResponse
	if err := s.client.request(ctx, "GET", "/account", nil, &apiResp); err != nil {
		return nil, err
	}
	if apiResp.User.ID == "" {
		return nil, &SendlyError{APIError: APIError{
			Code:    "invalid_response",
			Message: "account response did not include user details",
		}}
	}

	return &Account{
		ID:        apiResp.User.ID,
		Email:     apiResp.User.Email,
		Name:      apiResp.User.Name,
		CreatedAt: apiResp.User.CreatedAt,
	}, nil
}

// GetCredits retrieves credit balance information.
func (s *AccountService) GetCredits(ctx context.Context) (*Credits, error) {
	var apiResp creditsAPIResponse
	if err := s.client.request(ctx, "GET", "/credits", nil, &apiResp); err != nil {
		return nil, err
	}

	return &Credits{
		Balance:          apiResp.Balance,
		ReservedBalance:  apiResp.ReservedBalance,
		AvailableBalance: apiResp.AvailableBalance,
	}, nil
}

// ListCreditTransactionsOptions are options for listing credit transactions.
type ListCreditTransactionsOptions struct {
	Limit  int
	Offset int
}

// GetCreditTransactions retrieves credit transaction history.
func (s *AccountService) GetCreditTransactions(ctx context.Context, opts *ListCreditTransactionsOptions) ([]CreditTransaction, error) {
	path := "/credits/transactions"
	if opts != nil {
		params := make(map[string]string)
		if opts.Limit > 0 {
			params["limit"] = strconv.Itoa(opts.Limit)
		}
		if opts.Offset > 0 {
			params["offset"] = strconv.Itoa(opts.Offset)
		}
		path += buildQueryString(params)
	}

	var apiResp struct {
		Transactions []transactionAPIResponse `json:"transactions"`
	}
	if err := s.client.request(ctx, "GET", path, nil, &apiResp); err != nil {
		return nil, err
	}

	transactions := make([]CreditTransaction, len(apiResp.Transactions))
	for i, api := range apiResp.Transactions {
		transactions[i] = CreditTransaction{
			ID:           api.ID,
			Type:         TransactionType(api.Type),
			Amount:       api.Amount,
			BalanceAfter: api.BalanceAfter,
			Description:  api.Description,
			MessageID:    api.MessageID,
			CreatedAt:    api.CreatedAt,
		}
	}
	return transactions, nil
}

type TransferCreditsRequest struct {
	TargetOrganizationID string `json:"targetOrganizationId"`
	Amount               int    `json:"amount"`
}

type TransferCreditsResponse struct {
	Success       bool `json:"success"`
	Amount        int  `json:"amount"`
	SourceBalance int  `json:"sourceBalance"`
	TargetBalance int  `json:"targetBalance"`
}

func (s *AccountService) TransferCredits(ctx context.Context, req TransferCreditsRequest) (*TransferCreditsResponse, error) {
	if req.TargetOrganizationID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "target organization ID is required"}}
	}
	if req.Amount <= 0 {
		return nil, &ValidationError{APIError: APIError{Message: "amount must be a positive integer"}}
	}

	var resp TransferCreditsResponse
	if err := s.client.request(ctx, "POST", "/credits/transfer", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListAPIKeys retrieves all API keys for the account.
func (s *AccountService) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	var apiResp struct {
		Keys []apiKeyAPIResponse `json:"keys"`
	}
	if err := s.client.request(ctx, "GET", "/account/keys", nil, &apiResp); err != nil {
		return nil, err
	}

	keys := make([]APIKey, len(apiResp.Keys))
	for i, api := range apiResp.Keys {
		keys[i] = api.toAPIKey()
	}
	return keys, nil
}

// GetAPIKey retrieves a specific API key by ID.
func (s *AccountService) GetAPIKey(ctx context.Context, keyID string) (*APIKey, error) {
	if keyID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "API key ID is required"}}
	}

	var apiResp apiKeyAPIResponse
	if err := s.client.request(ctx, "GET", "/account/keys/"+keyID, nil, &apiResp); err != nil {
		return nil, err
	}

	key := apiResp.toAPIKey()
	return &key, nil
}

// APIKeyUsageSummary is the aggregate usage for an API key.
type APIKeyUsageSummary struct {
	TotalRequests int     `json:"totalRequests"`
	TotalCredits  int     `json:"totalCredits"`
	LastUsed      *string `json:"lastUsed,omitempty"`
}

// APIKeyUsageRequest is one recent request made with an API key.
type APIKeyUsageRequest struct {
	Endpoint    string `json:"endpoint"`
	Method      string `json:"method"`
	StatusCode  int    `json:"statusCode"`
	CreditsUsed int    `json:"creditsUsed"`
	CreatedAt   string `json:"createdAt"`
}

// APIKeyUsageEndpoint is a per-endpoint call count for an API key.
type APIKeyUsageEndpoint struct {
	Endpoint string `json:"endpoint"`
	Count    int    `json:"count"`
}

// APIKeyUsage contains usage statistics for an API key.
type APIKeyUsage struct {
	KeyID             string                `json:"keyId"`
	KeyName           string                `json:"keyName"`
	Summary           APIKeyUsageSummary    `json:"summary"`
	RecentRequests    []APIKeyUsageRequest  `json:"recentRequests"`
	EndpointBreakdown []APIKeyUsageEndpoint `json:"endpointBreakdown"`

	// MessagesSent is always 0.
	//
	// Deprecated: usage is reported per API request, not per message. Use
	// Summary.TotalRequests for call volume, or Messages.List to count
	// messages.
	MessagesSent int `json:"messagesSent"`
	// MessagesDelivered is always 0.
	//
	// Deprecated: usage is reported per API request, not per message. Use
	// Messages.List and count messages with status "delivered".
	MessagesDelivered int `json:"messagesDelivered"`
	// MessagesFailed is always 0.
	//
	// Deprecated: usage is reported per API request, not per message. Use
	// Messages.List and count messages with status "failed".
	MessagesFailed int `json:"messagesFailed"`
	// CreditsUsed mirrors Summary.TotalCredits.
	//
	// Deprecated: use Summary.TotalCredits.
	CreditsUsed int `json:"creditsUsed"`
	// PeriodStart is always empty.
	//
	// Deprecated: usage covers the most recent requests rather than a
	// billing period. Use RecentRequests[].CreatedAt for the window covered.
	PeriodStart string `json:"periodStart"`
	// PeriodEnd is always empty.
	//
	// Deprecated: usage covers the most recent requests rather than a
	// billing period. Use Summary.LastUsed for the latest activity.
	PeriodEnd string `json:"periodEnd"`
}

// UnmarshalJSON decodes a usage payload and mirrors it onto the deprecated
// aggregate fields.
func (u *APIKeyUsage) UnmarshalJSON(data []byte) error {
	type apiKeyUsageAlias APIKeyUsage
	var raw apiKeyUsageAlias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*u = APIKeyUsage(raw)
	if u.CreditsUsed == 0 {
		u.CreditsUsed = u.Summary.TotalCredits
	}
	return nil
}

// GetAPIKeyUsage retrieves usage statistics for an API key.
func (s *AccountService) GetAPIKeyUsage(ctx context.Context, keyID string) (*APIKeyUsage, error) {
	if keyID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "API key ID is required"}}
	}

	var usage APIKeyUsage
	if err := s.client.request(ctx, "GET", "/account/keys/"+keyID+"/usage", nil, &usage); err != nil {
		return nil, err
	}
	return &usage, nil
}

// CreateAPIKeyRequest is the request to create a new API key. Type is
// required by the API and must be "test" or "live"; leave it empty to create
// a test key. Scopes defaults to the standard set when omitted.
type CreateAPIKeyRequest struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Scopes    []string `json:"scopes,omitempty"`
	ExpiresAt *string  `json:"expiresAt,omitempty"`
}

// CreateAPIKeyResponse is the response from creating an API key.
type CreateAPIKeyResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"` // Full key value - only shown once!
	KeyPrefix string `json:"keyPrefix"`
	Type      string `json:"type"`
	CreatedAt string `json:"createdAt"`

	// APIKey mirrors the flat fields above. Permissions and LastFour are
	// not returned when a key is created and stay empty.
	//
	// Deprecated: use ID, Name, Type, KeyPrefix and CreatedAt directly.
	APIKey APIKey `json:"apiKey"`
}

// UnmarshalJSON decodes a created key and mirrors the flat fields onto the
// deprecated nested APIKey.
func (r *CreateAPIKeyResponse) UnmarshalJSON(data []byte) error {
	type createAPIKeyResponseAlias CreateAPIKeyResponse
	var raw createAPIKeyResponseAlias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*r = CreateAPIKeyResponse(raw)
	if reflect.DeepEqual(r.APIKey, APIKey{}) {
		r.APIKey = APIKey{
			ID:        r.ID,
			Name:      r.Name,
			Type:      r.Type,
			Prefix:    r.KeyPrefix,
			CreatedAt: r.CreatedAt,
		}
	}
	return nil
}

// CreateAPIKey creates a new test API key. Use CreateAPIKeyWithOptions with
// Type "live" to create a live key.
func (s *AccountService) CreateAPIKey(ctx context.Context, name string) (*CreateAPIKeyResponse, error) {
	return s.CreateAPIKeyWithOptions(ctx, CreateAPIKeyRequest{Name: name})
}

// CreateAPIKeyWithOptions creates a new API key with full options.
func (s *AccountService) CreateAPIKeyWithOptions(ctx context.Context, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	if req.Name == "" {
		return nil, &ValidationError{APIError: APIError{Message: "API key name is required"}}
	}
	if req.Type == "" {
		req.Type = "test"
	}
	if req.Type != "test" && req.Type != "live" {
		return nil, &ValidationError{APIError: APIError{Message: "API key type must be 'test' or 'live'"}}
	}

	var resp CreateAPIKeyResponse
	if err := s.client.request(ctx, "POST", "/account/keys", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RevokeAPIKey revokes an API key. The key currently authenticating the
// client cannot revoke itself.
func (s *AccountService) RevokeAPIKey(ctx context.Context, keyID string) error {
	if keyID == "" {
		return &ValidationError{APIError: APIError{Message: "API key ID is required"}}
	}

	return s.client.request(ctx, "PATCH", "/account/keys/"+keyID+"/revoke", nil, nil)
}

// RotateAPIKeyRequest is the body for RotateAPIKey. GracePeriodHours keeps the
// old key valid for the given window (24-168 hours inclusive) before it
// expires, so running code keeps working during the cutover. Leave it 0 to use
// the server default of 24 hours.
type RotateAPIKeyRequest struct {
	GracePeriodHours int `json:"gracePeriodHours,omitempty"`
}

// RotatedAPIKey is the freshly issued key returned by RotateAPIKey. It carries
// every APIKey field plus the one-time raw secret (Key) and a Warning to store
// it now — the raw secret is shown only once.
type RotatedAPIKey struct {
	APIKey
	// Key is the raw new secret ("sk_…"). Shown only once — store it now.
	Key string `json:"key"`
	// Warning is a human-readable caution about the one-time secret.
	Warning string `json:"warning"`
}

// RotateAPIKeyResponse is the result of rotating an API key.
type RotateAPIKeyResponse struct {
	// NewKey is the newly issued key, including its one-time raw Key and Warning.
	NewKey RotatedAPIKey `json:"newKey"`
	// OldKey is the predecessor key, now counting down its grace period.
	OldKey APIKey `json:"oldKey"`
	// Message is a human-readable summary (e.g. when the old key expires).
	Message string `json:"message"`
}

// RotateAPIKey issues a new value for an existing key and keeps the old one
// valid for a grace period (24-168 hours, default 24) so running code keeps
// working during the cutover. The returned NewKey.Key is the raw secret and is
// shown only once — store it immediately. Pass nil req (or a zero
// GracePeriodHours) to use the default grace period.
func (s *AccountService) RotateAPIKey(ctx context.Context, keyID string, req *RotateAPIKeyRequest) (*RotateAPIKeyResponse, error) {
	if keyID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "API key ID is required"}}
	}
	if req != nil && req.GracePeriodHours != 0 && (req.GracePeriodHours < 24 || req.GracePeriodHours > 168) {
		return nil, &ValidationError{APIError: APIError{Message: "gracePeriodHours must be between 24 and 168"}}
	}

	body := &RotateAPIKeyRequest{}
	if req != nil {
		body = req
	}

	var resp RotateAPIKeyResponse
	if err := s.client.request(ctx, "POST", "/account/keys/"+keyID+"/rotate", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
