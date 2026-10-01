package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestAccountGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account" {
			t.Errorf("expected path '/account', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"user": {"id": "usr_123", "email": "dev@example.com", "createdAt": "2026-01-05T10:00:00.000Z"},
			"organization": {"id": "org_1", "name": "Acme", "isPersonal": false},
			"credits": {"balance": "120", "reservedBalance": "0"},
			"verification": null,
			"apiKey": {"id": "key_1", "name": "CLI", "type": "live", "scopes": ["sms:send"]},
			"limits": {"messagesPerMinute": 60, "messagesPerDay": 10000}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	account, err := client.Account.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.ID != "usr_123" {
		t.Errorf("expected ID 'usr_123', got '%s'", account.ID)
	}
	if account.Email != "dev@example.com" {
		t.Errorf("expected Email 'dev@example.com', got '%s'", account.Email)
	}
	if account.CreatedAt != "2026-01-05T10:00:00.000Z" {
		t.Errorf("expected CreatedAt '2026-01-05T10:00:00.000Z', got '%s'", account.CreatedAt)
	}
}

func TestAccountGet_MissingUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"credits": {"balance": "0"}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	account, err := client.Account.Get(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if account != nil {
		t.Errorf("expected nil account, got %+v", account)
	}
}

func TestAccountGetAPIKeyUsage_DeprecatedFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"keyId": "key_1",
			"keyName": "CLI",
			"summary": {"totalRequests": 12, "totalCredits": 34, "lastUsed": "2026-02-01T00:00:00.000Z"},
			"recentRequests": [{"endpoint": "/messages", "method": "POST", "statusCode": 200, "creditsUsed": 2, "createdAt": "2026-02-01T00:00:00.000Z"}],
			"endpointBreakdown": [{"endpoint": "POST /messages", "count": 12}]
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	usage, err := client.Account.GetAPIKeyUsage(context.Background(), "key_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.Summary.TotalRequests != 12 {
		t.Errorf("expected Summary.TotalRequests 12, got %d", usage.Summary.TotalRequests)
	}
	if len(usage.RecentRequests) != 1 || usage.RecentRequests[0].Endpoint != "/messages" {
		t.Errorf("expected recent requests to decode, got %+v", usage.RecentRequests)
	}
	if len(usage.EndpointBreakdown) != 1 || usage.EndpointBreakdown[0].Count != 12 {
		t.Errorf("expected endpoint breakdown to decode, got %+v", usage.EndpointBreakdown)
	}
	if usage.CreditsUsed != 34 {
		t.Errorf("expected deprecated CreditsUsed to mirror Summary.TotalCredits (34), got %d", usage.CreditsUsed)
	}
	if usage.MessagesSent != 0 || usage.MessagesDelivered != 0 || usage.MessagesFailed != 0 {
		t.Errorf("expected deprecated message counters to stay zero, got %d/%d/%d", usage.MessagesSent, usage.MessagesDelivered, usage.MessagesFailed)
	}
	if usage.PeriodStart != "" || usage.PeriodEnd != "" {
		t.Errorf("expected deprecated period fields to stay empty, got '%s'/'%s'", usage.PeriodStart, usage.PeriodEnd)
	}
}

func TestAccountCreateAPIKey_DeprecatedAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "key_2",
			"name": "Deploy bot",
			"key": "sk_test_abc123",
			"keyPrefix": "sk_test_abc",
			"type": "test",
			"createdAt": "2026-02-02T00:00:00.000Z"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.Account.CreateAPIKey(context.Background(), "Deploy bot")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "key_2" || resp.Key != "sk_test_abc123" || resp.KeyPrefix != "sk_test_abc" {
		t.Errorf("expected flat fields to decode, got %+v", resp)
	}
	if resp.APIKey.ID != "key_2" {
		t.Errorf("expected deprecated APIKey.ID 'key_2', got '%s'", resp.APIKey.ID)
	}
	if resp.APIKey.Name != "Deploy bot" {
		t.Errorf("expected deprecated APIKey.Name 'Deploy bot', got '%s'", resp.APIKey.Name)
	}
	if resp.APIKey.Prefix != "sk_test_abc" {
		t.Errorf("expected deprecated APIKey.Prefix 'sk_test_abc', got '%s'", resp.APIKey.Prefix)
	}
	if resp.APIKey.Type != "test" {
		t.Errorf("expected deprecated APIKey.Type 'test', got '%s'", resp.APIKey.Type)
	}
	if resp.APIKey.CreatedAt != "2026-02-02T00:00:00.000Z" {
		t.Errorf("expected deprecated APIKey.CreatedAt to mirror CreatedAt, got '%s'", resp.APIKey.CreatedAt)
	}
}

func TestAccountCreateAPIKey_LegacyNestedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"key": "sk_test_abc123",
			"apiKey": {
				"id": "key_3",
				"name": "Legacy bot",
				"type": "live",
				"prefix": "sk_live_abc",
				"permissions": ["sms:send"],
				"createdAt": "2026-02-03T00:00:00.000Z"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.Account.CreateAPIKey(context.Background(), "Legacy bot")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.APIKey.ID != "key_3" {
		t.Errorf("expected nested APIKey.ID 'key_3', got '%s'", resp.APIKey.ID)
	}
	if resp.APIKey.Name != "Legacy bot" {
		t.Errorf("expected nested APIKey.Name 'Legacy bot', got '%s'", resp.APIKey.Name)
	}
	if resp.APIKey.Prefix != "sk_live_abc" {
		t.Errorf("expected nested APIKey.Prefix 'sk_live_abc', got '%s'", resp.APIKey.Prefix)
	}
	if len(resp.APIKey.Permissions) != 1 || resp.APIKey.Permissions[0] != "sms:send" {
		t.Errorf("expected nested APIKey.Permissions to decode, got %+v", resp.APIKey.Permissions)
	}
}

func TestAccountCreateAPIKey_FlatAndNestedPayload(t *testing.T) {
	var sent map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "key_5",
			"name": "Temp bot",
			"key": "sk_test_v1_raw",
			"keyPrefix": "sk_test_v1_r",
			"type": "test",
			"createdAt": "2026-09-24T10:00:00.000Z",
			"expiresAt": "2026-10-24T10:00:00.000Z",
			"apiKey": {
				"id": "key_5",
				"name": "Temp bot",
				"type": "test",
				"prefix": "sk_test_v1_r...",
				"scopes": ["sms:send", "sms:read"],
				"permissions": ["sms:send", "sms:read"],
				"isActive": true,
				"isRevoked": false,
				"createdAt": "2026-09-24T10:00:00.000Z",
				"lastUsedAt": null,
				"expiresAt": "2026-10-24T10:00:00.000Z"
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	expiresAt := "2026-10-24T10:00:00.000Z"
	resp, err := client.Account.CreateAPIKeyWithOptions(context.Background(), CreateAPIKeyRequest{
		Name:      "Temp bot",
		ExpiresAt: &expiresAt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sent["expiresAt"] != expiresAt || sent["type"] != "test" {
		t.Errorf("expected request to carry expiresAt and type, got %+v", sent)
	}
	if resp.ID != "key_5" || resp.Key != "sk_test_v1_raw" || resp.KeyPrefix != "sk_test_v1_r" {
		t.Errorf("expected flat fields to decode, got %+v", resp)
	}
	if resp.APIKey.Prefix != "sk_test_v1_r..." {
		t.Errorf("expected nested APIKey to win over the mirror, got prefix '%s'", resp.APIKey.Prefix)
	}
	if len(resp.APIKey.Permissions) != 2 || resp.APIKey.Permissions[0] != "sms:send" {
		t.Errorf("expected nested APIKey.Permissions to decode, got %+v", resp.APIKey.Permissions)
	}
	if resp.APIKey.ExpiresAt == nil || *resp.APIKey.ExpiresAt != expiresAt {
		t.Errorf("expected nested APIKey.ExpiresAt '%s', got %v", expiresAt, resp.APIKey.ExpiresAt)
	}
	if resp.APIKey.IsRevoked {
		t.Errorf("expected nested APIKey.IsRevoked false")
	}
}

func TestCreateAPIKeyResponse_RoundTrip(t *testing.T) {
	lastUsed := "2026-02-04T00:00:00.000Z"
	original := CreateAPIKeyResponse{
		ID:        "key_4",
		Name:      "Deploy bot",
		Key:       "sk_test_abc123",
		KeyPrefix: "sk_test_abc",
		Type:      "test",
		CreatedAt: "2026-02-02T00:00:00.000Z",
		APIKey: APIKey{
			ID:          "key_4",
			Name:        "Deploy bot",
			Type:        "test",
			Prefix:      "sk_test_abc",
			Permissions: []string{"sms:send", "sms:read"},
			CreatedAt:   "2026-02-02T00:00:00.000Z",
			LastUsedAt:  &lastUsed,
		},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded CreateAPIKeyResponse
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip changed the value:\n before %+v\n after  %+v", original, decoded)
	}
}

func TestAccountGetAPIKeyUsage_LegacyCreditsUsed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"keyId": "key_1", "keyName": "CLI", "creditsUsed": 34}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	usage, err := client.Account.GetAPIKeyUsage(context.Background(), "key_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.CreditsUsed != 34 {
		t.Errorf("expected CreditsUsed 34 to survive a payload without a summary, got %d", usage.CreditsUsed)
	}
}

func TestAPIKeyUsage_RoundTrip(t *testing.T) {
	lastUsed := "2026-02-01T00:00:00.000Z"
	original := APIKeyUsage{
		KeyID:   "key_1",
		KeyName: "CLI",
		Summary: APIKeyUsageSummary{
			TotalRequests: 12,
			TotalCredits:  34,
			LastUsed:      &lastUsed,
		},
		RecentRequests: []APIKeyUsageRequest{{
			Endpoint:    "/messages",
			Method:      "POST",
			StatusCode:  200,
			CreditsUsed: 2,
			CreatedAt:   lastUsed,
		}},
		EndpointBreakdown: []APIKeyUsageEndpoint{{Endpoint: "POST /messages", Count: 12}},
		CreditsUsed:       34,
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded APIKeyUsage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip changed the value:\n before %+v\n after  %+v", original, decoded)
	}
}

func TestGetCredits_CamelCaseWire(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		balance   int
		reserved  int
		available int
		mode      string
	}{
		{
			name:      "prepaid",
			body:      `{"balance":120,"reservedBalance":20,"availableBalance":100,"billingMode":"prepaid"}`,
			balance:   120,
			reserved:  20,
			available: 100,
			mode:      "prepaid",
		},
		{
			name:      "pooled",
			body:      `{"balance":-5,"reservedBalance":0,"availableBalance":45,"billingMode":"pooled"}`,
			balance:   -5,
			reserved:  0,
			available: 45,
			mode:      "pooled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/credits" {
					t.Errorf("expected path '/credits', got '%s'", r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient("test-api-key", WithBaseURL(server.URL))

			credits, err := client.Account.GetCredits(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if credits.Balance != tt.balance {
				t.Errorf("expected Balance %d, got %d", tt.balance, credits.Balance)
			}
			if credits.ReservedBalance != tt.reserved {
				t.Errorf("expected ReservedBalance %d, got %d", tt.reserved, credits.ReservedBalance)
			}
			if credits.AvailableBalance != tt.available {
				t.Errorf("expected AvailableBalance %d, got %d", tt.available, credits.AvailableBalance)
			}
			if credits.BillingMode != tt.mode {
				t.Errorf("expected BillingMode %q, got %q", tt.mode, credits.BillingMode)
			}
		})
	}
}

func TestGetCreditTransactions_TypesTheAPIRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"transactions":[
			{"id":"tx_1","amount":-50,"balance_after":150,"type":"transfer","description":"Transferred 50 credits to Acme East","created_at":"2026-09-25T10:00:00.000Z"},
			{"id":"tx_2","amount":100,"balance_after":200,"type":"admin_grant","description":"Goodwill credit","created_at":"2026-09-24T10:00:00.000Z"},
			{"id":"tx_3","amount":100,"balance_after":100,"type":"admin_seed","description":null,"created_at":"2026-09-23T10:00:00.000Z"}
		]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	transactions, err := client.Account.GetCreditTransactions(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []TransactionType{TransactionTypeTransfer, TransactionTypeAdminGrant, TransactionTypeAdminSeed}
	if len(transactions) != len(want) {
		t.Fatalf("expected %d transactions, got %d", len(want), len(transactions))
	}
	for i, tx := range transactions {
		if tx.Type != want[i] {
			t.Errorf("transaction %d: expected type %q, got %q", i, want[i], tx.Type)
		}
	}
	if transactions[0].Amount != -50 || transactions[0].BalanceAfter != 150 {
		t.Errorf("unexpected amounts: %+v", transactions[0])
	}
}

func TestAccountCreateAPIKey_ReadsExpiresAt(t *testing.T) {
	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "key_5",
			"name": "Nightly job",
			"key": "sk_test_abc123",
			"keyPrefix": "sk_test_abc",
			"type": "test",
			"createdAt": "2026-09-25T10:00:00.000Z",
			"expiresAt": "2026-12-31T00:00:00.000Z",
			"apiKey": {"id": "key_5", "name": "Nightly job", "type": "test", "prefix": "sk_test_abc...", "scopes": ["sms:send"], "permissions": ["sms:send"], "isActive": true, "isRevoked": false, "createdAt": "2026-09-25T10:00:00.000Z", "lastUsedAt": null, "expiresAt": "2026-12-31T00:00:00.000Z"}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	expiresAt := "2026-12-31T00:00:00Z"
	resp, err := client.Account.CreateAPIKeyWithOptions(context.Background(), CreateAPIKeyRequest{Name: "Nightly job", Scopes: []string{"sms:send"}, ExpiresAt: &expiresAt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["expiresAt"] != expiresAt || body["type"] != "test" {
		t.Errorf("unexpected request body: %v", body)
	}
	if resp.ExpiresAt == nil || *resp.ExpiresAt != "2026-12-31T00:00:00.000Z" {
		t.Errorf("expected ExpiresAt, got %v", resp.ExpiresAt)
	}
	if len(resp.APIKey.Permissions) != 1 || resp.APIKey.Permissions[0] != "sms:send" {
		t.Errorf("expected the nested key's permissions, got %v", resp.APIKey.Permissions)
	}
}

func TestAccountRotateAPIKey_ReadsTheKeyRecords(t *testing.T) {
	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/account/keys/key_old/rotate" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"newKey": {"id": "key_new", "userId": "usr_1", "organizationId": "org_1", "name": "CI (rotated)", "keyId": "key_Q2xhdWRlUm90YXRl", "keyPrefix": "sk_live_v1_k", "type": "live", "scopes": ["sms:send", "sms:read"], "source": "manual", "expiresAt": null, "lastUsedAt": null, "isActive": true, "revokedAt": null, "revokedReason": null, "rotatedFromId": "key_old", "gracePeriodHours": 24, "isEnterpriseMaster": false, "createdAt": "2026-09-25T10:00:00.000Z", "key": "sk_live_v1_key_Q2xhdWRlUm90YXRl_abcdefghijklmnopqrstuvwxyz012345", "warning": "This key will only be shown once. Store it securely."},
			"oldKey": {"id": "key_old", "userId": "usr_1", "organizationId": "org_1", "name": "CI", "keyId": "key_T2xkS2V5T2xkS2V5", "keyPrefix": "sk_live_v1_k", "type": "live", "scopes": ["sms:send", "sms:read"], "source": "manual", "expiresAt": "2026-09-27T10:00:00.000Z", "lastUsedAt": "2026-09-24T08:00:00.000Z", "isActive": true, "revokedAt": null, "revokedReason": null, "rotatedFromId": null, "gracePeriodHours": 24, "isEnterpriseMaster": false, "createdAt": "2026-09-01T10:00:00.000Z"},
			"message": "Old key will expire in 48 hours"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.Account.RotateAPIKey(context.Background(), "key_old", &RotateAPIKeyRequest{GracePeriodHours: 48})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["gracePeriodHours"] != float64(48) {
		t.Errorf("unexpected request body: %v", body)
	}

	scopes := []string{"sms:send", "sms:read"}
	newKey := resp.NewKey
	if newKey.ID != "key_new" || newKey.Name != "CI (rotated)" || newKey.Type != "live" || newKey.CreatedAt != "2026-09-25T10:00:00.000Z" {
		t.Errorf("unexpected new key: %+v", newKey)
	}
	if newKey.Prefix != "sk_live_v1_k..." {
		t.Errorf("expected NewKey.Prefix %q, got %q", "sk_live_v1_k...", newKey.Prefix)
	}
	if !reflect.DeepEqual(newKey.Permissions, scopes) {
		t.Errorf("expected NewKey.Permissions %v, got %v", scopes, newKey.Permissions)
	}
	if newKey.IsRevoked || newKey.ExpiresAt != nil || newKey.LastUsedAt != nil {
		t.Errorf("expected an active new key with no expiry, got %+v", newKey)
	}
	if newKey.Key != "sk_live_v1_key_Q2xhdWRlUm90YXRl_abcdefghijklmnopqrstuvwxyz012345" || newKey.Warning == "" {
		t.Errorf("expected the one-time key and warning, got %q / %q", newKey.Key, newKey.Warning)
	}

	oldKey := resp.OldKey
	if oldKey.ID != "key_old" || oldKey.Name != "CI" || oldKey.Type != "live" {
		t.Errorf("unexpected old key: %+v", oldKey)
	}
	if oldKey.Prefix != "sk_live_v1_k..." {
		t.Errorf("expected OldKey.Prefix %q, got %q", "sk_live_v1_k...", oldKey.Prefix)
	}
	if !reflect.DeepEqual(oldKey.Permissions, scopes) {
		t.Errorf("expected OldKey.Permissions %v, got %v", scopes, oldKey.Permissions)
	}
	if oldKey.IsRevoked {
		t.Errorf("expected the old key to stay active through its grace period")
	}
	if oldKey.ExpiresAt == nil || *oldKey.ExpiresAt != "2026-09-27T10:00:00.000Z" {
		t.Errorf("expected OldKey.ExpiresAt, got %v", oldKey.ExpiresAt)
	}
	if oldKey.LastUsedAt == nil || *oldKey.LastUsedAt != "2026-09-24T08:00:00.000Z" {
		t.Errorf("expected OldKey.LastUsedAt, got %v", oldKey.LastUsedAt)
	}
	if resp.Message != "Old key will expire in 48 hours" {
		t.Errorf("unexpected message %q", resp.Message)
	}
}
