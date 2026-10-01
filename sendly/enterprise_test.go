package sendly

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
)

func TestEnterpriseAnalyticsOverview_FractionalDeliveryRate(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/enterprise/analytics/overview" {
			t.Errorf("expected path '/enterprise/analytics/overview', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"totalWorkspaces":2,"totalMessages":3,"totalDelivered":2,"totalFailed":1,"deliveryRate":66.67,"totalCredits":10,"deliveredMessages":2,"failedMessages":1,"totalCreditsUsed":6,"activeWorkspaces":2,"suspendedWorkspaces":0,"totalMessagesSent":3,"totalMessagesDelivered":2,"totalCreditsRemaining":10}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	overview, err := client.Enterprise.Analytics.Overview(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 1 {
		t.Errorf("expected 1 request, got %d", requests)
	}
	if overview.DeliveryRate != 67 {
		t.Errorf("expected DeliveryRate 67, got %d", overview.DeliveryRate)
	}
	if overview.DeliveryRatePercent != 66.67 {
		t.Errorf("expected DeliveryRatePercent 66.67, got %v", overview.DeliveryRatePercent)
	}
	if overview.TotalWorkspaces != 2 || overview.TotalCredits != 10 {
		t.Errorf("unexpected workspace or balance totals: %+v", overview)
	}
	if overview.TotalMessages != 3 || overview.DeliveredMessages != 2 || overview.FailedMessages != 1 {
		t.Errorf("unexpected message counts: %+v", overview)
	}
	if overview.TotalCreditsUsed != 6 || overview.ActiveWorkspaces != 2 {
		t.Errorf("unexpected credit or workspace counts: %+v", overview)
	}
}

func TestAnalyticsOverview_DecodesTotalsWithoutTheTwinKeys(t *testing.T) {
	var overview AnalyticsOverview
	body := `{"totalWorkspaces":2,"totalMessages":3,"totalDelivered":2,"totalFailed":1,"deliveryRate":66.67,"totalCredits":10}`
	if err := json.Unmarshal([]byte(body), &overview); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overview.DeliveredMessages != 2 || overview.FailedMessages != 1 {
		t.Errorf("expected DeliveredMessages 2 and FailedMessages 1, got %+v", overview)
	}
	if overview.DeliveryRate != 67 || overview.DeliveryRatePercent != 66.67 {
		t.Errorf("expected rates 67 and 66.67, got %d and %v", overview.DeliveryRate, overview.DeliveryRatePercent)
	}
}

func TestAnalyticsOverview_WholeNumberRate(t *testing.T) {
	var overview AnalyticsOverview
	if err := json.Unmarshal([]byte(`{"totalMessages":0,"deliveryRate":0}`), &overview); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overview.DeliveryRate != 0 || overview.DeliveryRatePercent != 0 {
		t.Errorf("expected zero rates, got %d and %v", overview.DeliveryRate, overview.DeliveryRatePercent)
	}
	if err := json.Unmarshal([]byte(`{"deliveryRate":100}`), &overview); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overview.DeliveryRate != 100 || overview.DeliveryRatePercent != 100 {
		t.Errorf("expected rates 100 and 100, got %d and %v", overview.DeliveryRate, overview.DeliveryRatePercent)
	}
}

func TestAnalyticsOverview_RoundTrip(t *testing.T) {
	original := AnalyticsOverview{
		TotalMessages:       3,
		DeliveredMessages:   2,
		FailedMessages:      1,
		DeliveryRate:        67,
		TotalCreditsUsed:    6,
		ActiveWorkspaces:    2,
		DeliveryRatePercent: 66.67,
		TotalWorkspaces:     2,
		TotalDelivered:      2,
		TotalFailed:         1,
		TotalCredits:        10,
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded AnalyticsOverview
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip changed the value:\n before %+v\n after  %+v", original, decoded)
	}
}

func TestEnterpriseAnalyticsCredits_ReadsTotals(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/enterprise/analytics/credits" {
			t.Errorf("expected path '/enterprise/analytics/credits', got '%s'", r.URL.Path)
		}
		if r.URL.Query().Get("period") != "30d" {
			t.Errorf("expected period=30d, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"period":"30d","totalBalance":500,"totalLifetime":900,"totalUsed":400,"workspaceCount":3}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.Enterprise.Analytics.Credits(context.Background(), &AnalyticsCreditsOptions{Period: "30d"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Period != "30d" {
		t.Errorf("expected Period 30d, got %q", resp.Period)
	}
	if resp.TotalBalance != 500 || resp.TotalLifetime != 900 || resp.TotalUsed != 400 || resp.WorkspaceCount != 3 {
		t.Errorf("expected totals 500/900/400/3, got %+v", resp)
	}
}

func TestEnterpriseWebhooksSet_ReturnsSigningSecretOnce(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		if calls == 1 {
			w.Write([]byte(`{"url":"https://hooks.example.com/s","events":null,"workspaces":null,"signingSecret":"abc123"}`))
			return
		}
		w.Write([]byte(`{"url":"https://hooks.example.com/s","events":null,"workspaces":null}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	first, err := client.Enterprise.Webhooks.Set(context.Background(), "https://hooks.example.com/s")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.URL != "https://hooks.example.com/s" || first.SigningSecret != "abc123" {
		t.Errorf("expected the URL and signing secret, got %+v", first)
	}

	second, err := client.Enterprise.Webhooks.Set(context.Background(), "https://hooks.example.com/s")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second.SigningSecret != "" {
		t.Errorf("expected no signing secret on a second Set, got %q", second.SigningSecret)
	}
}

func TestEnterpriseWebhooksGet_DecodesEventsAndWorkspaces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/enterprise/webhooks" {
			t.Errorf("expected GET /enterprise/webhooks, got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"url":"https://hooks.example.com/s","events":["message.delivered"],"workspaces":["org_1"]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	webhook, err := client.Enterprise.Webhooks.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(webhook.Events, []string{"message.delivered"}) || !reflect.DeepEqual(webhook.Workspaces, []string{"org_1"}) {
		t.Errorf("expected events and workspaces to decode, got %+v", webhook)
	}
}

func TestEnterpriseWebhooksSetWithOptions_SendsFilters(t *testing.T) {
	var body map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"url":"https://hooks.example.com/s","events":["message.delivered"],"workspaces":null}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	webhook, err := client.Enterprise.Webhooks.SetWithOptions(context.Background(), &SetEnterpriseWebhookRequest{
		URL:    "https://hooks.example.com/s",
		Events: []string{"message.delivered"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body["url"]) != `"https://hooks.example.com/s"` || string(body["events"]) != `["message.delivered"]` || string(body["workspaces"]) != `null` {
		t.Errorf("unexpected request body: %v", body)
	}
	if !reflect.DeepEqual(webhook.Events, []string{"message.delivered"}) || webhook.Workspaces != nil {
		t.Errorf("unexpected webhook: %+v", webhook)
	}

	if _, err := client.Enterprise.Webhooks.SetWithOptions(context.Background(), &SetEnterpriseWebhookRequest{}); !IsValidationError(err) {
		t.Errorf("expected a ValidationError without a URL, got %v", err)
	}
}

func TestWorkspacesInheritVerification_PurchaseNewNumber(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/enterprise/workspaces/org_new/verification/inherit" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(raw))
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"verificationId":"bv_1","status":"pending","type":"toll_free","tollFreeNumber":"+18885550100","inheritedFrom":"org_src","newNumber":true}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	if _, err := client.Enterprise.Workspaces.InheritVerification(context.Background(), "org_new", "org_src"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp, err := client.Enterprise.Workspaces.InheritVerificationWithOptions(context.Background(), "org_new", &InheritVerificationRequest{
		SourceWorkspaceID: "org_src",
		PurchaseNewNumber: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bodies) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(bodies))
	}
	if bodies[0] != `{"sourceWorkspaceId":"org_src"}` {
		t.Errorf("expected the default call to send only sourceWorkspaceId, got %s", bodies[0])
	}
	if bodies[1] != `{"sourceWorkspaceId":"org_src","purchaseNewNumber":true}` {
		t.Errorf("expected purchaseNewNumber in the body, got %s", bodies[1])
	}
	if !resp.NewNumber || resp.InheritedFrom != "org_src" || resp.TollFreeNumber == nil || *resp.TollFreeNumber != "+18885550100" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestWorkspacesInheritVerificationWithOptions_NoNumberBought(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"verificationId":"bv_2","status":"pending","type":"toll_free","tollFreeNumber":null,"inheritedFrom":"org_src","newNumber":true}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.Enterprise.Workspaces.InheritVerificationWithOptions(context.Background(), "org_new", &InheritVerificationRequest{
		SourceWorkspaceID: "org_src",
		PurchaseNewNumber: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.NewNumber || resp.TollFreeNumber != nil || resp.Status != "pending" || resp.VerificationID != "bv_2" {
		t.Errorf("expected NewNumber with no number and a pending status, got %+v", resp)
	}
}

func TestWorkspacesInheritVerificationWithOptions_ChecksArguments(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	if _, err := client.Enterprise.Workspaces.InheritVerificationWithOptions(ctx, "", &InheritVerificationRequest{SourceWorkspaceID: "org_src"}); !IsValidationError(err) {
		t.Errorf("expected a ValidationError without a workspace ID, got %v", err)
	}
	if _, err := client.Enterprise.Workspaces.InheritVerificationWithOptions(ctx, "org_new", nil); !IsValidationError(err) {
		t.Errorf("expected a ValidationError for a nil request, got %v", err)
	}
	if _, err := client.Enterprise.Workspaces.InheritVerificationWithOptions(ctx, "org_new", &InheritVerificationRequest{PurchaseNewNumber: true}); !IsValidationError(err) {
		t.Errorf("expected a ValidationError without a source workspace ID, got %v", err)
	}
	if requests != 0 {
		t.Errorf("expected no request, got %d", requests)
	}
}

func TestUploadVerificationDocument_SendsTheFileContentType(t *testing.T) {
	var seen []string
	server := multerFilterServer(t, []string{"image/png", "image/jpeg", "image/jpg", "image/gif", "image/webp", "application/pdf"},
		"Only images (PNG, JPG, GIF, WEBP) and PDFs are allowed",
		`{"url":"https://files.example.com/verification-docs/ein.pdf","id":"file_2"}`, &seen)
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	pdf := []byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n1 0 obj\n<<>>\nendobj\n")
	resp, err := client.Enterprise.UploadVerificationDocument(context.Background(), "ein.pdf", bytes.NewReader(pdf), "org_1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "file_2" {
		t.Errorf("expected ID file_2, got %q", resp.ID)
	}
	if len(seen) != 1 || seen[0] != "ein.pdf application/pdf" {
		t.Errorf("expected the part 'ein.pdf application/pdf', got %v", seen)
	}
}

func TestWorkspacesProvisionBulk_AcceptsTheAPILimit(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body BulkProvisionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if len(body.Workspaces) > 100 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Maximum 100 workspaces per bulk provision"}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"results":[],"summary":{"total":60,"succeeded":60,"failed":0},"totalRequested":60,"totalCreated":60,"totalFailed":0}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	workspaces := make([]BulkProvisionWorkspace, 60)
	for i := range workspaces {
		workspaces[i] = BulkProvisionWorkspace{Name: "Workspace " + strconv.Itoa(i)}
	}
	result, err := client.Enterprise.Workspaces.ProvisionBulk(context.Background(), workspaces)
	if err != nil {
		t.Fatalf("expected 60 workspaces to be sent, got %v", err)
	}
	if result.Summary.Succeeded != 60 || requests != 1 {
		t.Errorf("unexpected result %+v after %d requests", result.Summary, requests)
	}

	if _, err := client.Enterprise.Workspaces.ProvisionBulk(context.Background(), make([]BulkProvisionWorkspace, 101)); !IsValidationError(err) {
		t.Errorf("expected a ValidationError for 101 workspaces, got %v", err)
	}
	if requests != 1 {
		t.Errorf("expected 101 workspaces to be refused before sending, got %d requests", requests)
	}
}
