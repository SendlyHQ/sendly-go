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
