package sendly

import (
	"context"
	"strconv"
)

// AccountService provides methods for accessing account information.
type AccountService struct {
	client *Client
}

// accountAPIResponse is the API response with snake_case fields.
type accountAPIResponse struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name,omitempty"`
	CreatedAt string  `json:"created_at"`
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

// apiKeyAPIResponse is the API response with snake_case fields.
type apiKeyAPIResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Prefix      string   `json:"prefix"`
	LastFour    string   `json:"last_four"`
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"created_at"`
	LastUsedAt  *string  `json:"last_used_at,omitempty"`
	ExpiresAt   *string  `json:"expires_at,omitempty"`
	IsRevoked   bool     `json:"is_revoked"`
}

// Get retrieves account information.
func (s *AccountService) Get(ctx context.Context) (*Account, error) {
	var apiResp accountAPIResponse
	if err := s.client.request(ctx, "GET", "/account", nil, &apiResp); err != nil {
		return nil, err
	}

	return &Account{
		ID:        apiResp.ID,
		Email:     apiResp.Email,
		Name:      apiResp.Name,
		CreatedAt: apiResp.CreatedAt,
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

	var apiResp []transactionAPIResponse
	if err := s.client.request(ctx, "GET", path, nil, &apiResp); err != nil {
		return nil, err
	}

	transactions := make([]CreditTransaction, len(apiResp))
	for i, api := range apiResp {
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
	var apiResp []apiKeyAPIResponse
	if err := s.client.request(ctx, "GET", "/keys", nil, &apiResp); err != nil {
		return nil, err
	}

	keys := make([]APIKey, len(apiResp))
	for i, api := range apiResp {
		keys[i] = APIKey{
			ID:          api.ID,
			Name:        api.Name,
			Type:        api.Type,
			Prefix:      api.Prefix,
			LastFour:    api.LastFour,
			Permissions: api.Permissions,
			CreatedAt:   api.CreatedAt,
			LastUsedAt:  api.LastUsedAt,
			ExpiresAt:   api.ExpiresAt,
			IsRevoked:   api.IsRevoked,
		}
	}
	return keys, nil
}

// GetAPIKey retrieves a specific API key by ID.
func (s *AccountService) GetAPIKey(ctx context.Context, keyID string) (*APIKey, error) {
	var apiResp apiKeyAPIResponse
	if err := s.client.request(ctx, "GET", "/keys/"+keyID, nil, &apiResp); err != nil {
		return nil, err
	}

	return &APIKey{
		ID:          apiResp.ID,
		Name:        apiResp.Name,
		Type:        apiResp.Type,
		Prefix:      apiResp.Prefix,
		LastFour:    apiResp.LastFour,
		Permissions: apiResp.Permissions,
		CreatedAt:   apiResp.CreatedAt,
		LastUsedAt:  apiResp.LastUsedAt,
		ExpiresAt:   apiResp.ExpiresAt,
		IsRevoked:   apiResp.IsRevoked,
	}, nil
}

// APIKeyUsage contains usage statistics for an API key.
type APIKeyUsage struct {
	KeyID             string `json:"keyId"`
	MessagesSent      int    `json:"messagesSent"`
	MessagesDelivered int    `json:"messagesDelivered"`
	MessagesFailed    int    `json:"messagesFailed"`
	CreditsUsed       int    `json:"creditsUsed"`
	PeriodStart       string `json:"periodStart"`
	PeriodEnd         string `json:"periodEnd"`
}

// GetAPIKeyUsage retrieves usage statistics for an API key.
func (s *AccountService) GetAPIKeyUsage(ctx context.Context, keyID string) (*APIKeyUsage, error) {
	var usage APIKeyUsage
	if err := s.client.request(ctx, "GET", "/keys/"+keyID+"/usage", nil, &usage); err != nil {
		return nil, err
	}
	return &usage, nil
}

// CreateAPIKeyRequest is the request to create a new API key.
type CreateAPIKeyRequest struct {
	Name      string  `json:"name"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
}

// CreateAPIKeyResponse is the response from creating an API key.
type CreateAPIKeyResponse struct {
	APIKey APIKey `json:"apiKey"`
	Key    string `json:"key"` // Full key value - only shown once!
}

// CreateAPIKey creates a new API key.
func (s *AccountService) CreateAPIKey(ctx context.Context, name string) (*CreateAPIKeyResponse, error) {
	return s.CreateAPIKeyWithOptions(ctx, CreateAPIKeyRequest{Name: name})
}

// CreateAPIKeyWithOptions creates a new API key with full options.
func (s *AccountService) CreateAPIKeyWithOptions(ctx context.Context, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	if req.Name == "" {
		return nil, &ValidationError{APIError: APIError{Message: "API key name is required"}}
	}

	var resp CreateAPIKeyResponse
	if err := s.client.request(ctx, "POST", "/account/keys", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RevokeAPIKey revokes an API key.
func (s *AccountService) RevokeAPIKey(ctx context.Context, keyID string) error {
	if keyID == "" {
		return &ValidationError{APIError: APIError{Message: "API key ID is required"}}
	}

	return s.client.request(ctx, "DELETE", "/account/keys/"+keyID, nil, nil)
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
