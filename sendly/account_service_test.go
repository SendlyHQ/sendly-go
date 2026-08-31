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
