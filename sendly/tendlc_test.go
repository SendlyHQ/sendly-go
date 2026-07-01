package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTenDlcListBrands_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/tendlc/brands" {
			t.Errorf("expected path '/tendlc/brands', got '%s'", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-api-key" {
			t.Errorf("expected Authorization header to be 'Bearer test-api-key', got '%s'", auth)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":         "brd_123",
					"legalName":  "Acme Holdings LLC",
					"dba":        nil,
					"entityType": "PRIVATE_PROFIT",
					"ein":        "12-3456789",
					"status":     "verified",
					"createdAt":  "2026-01-01T00:00:00Z",
					"updatedAt":  "2026-01-02T00:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.TenDlc.ListBrands(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 brand, got %d", len(resp.Data))
	}
	brand := resp.Data[0]
	if brand.ID != "brd_123" {
		t.Errorf("expected ID to be 'brd_123', got '%s'", brand.ID)
	}
	if brand.LegalName != "Acme Holdings LLC" {
		t.Errorf("expected LegalName to be 'Acme Holdings LLC', got '%s'", brand.LegalName)
	}
	if brand.DBA != nil {
		t.Errorf("expected DBA to be nil, got '%s'", *brand.DBA)
	}
	if brand.EIN == nil || *brand.EIN != "12-3456789" {
		t.Errorf("expected EIN to be '12-3456789', got %v", brand.EIN)
	}
	if brand.Status != "verified" {
		t.Errorf("expected Status to be 'verified', got '%s'", brand.Status)
	}
}

func TestTenDlcCreateBrand_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/tendlc/brands" {
			t.Errorf("expected path '/tendlc/brands', got '%s'", r.URL.Path)
		}

		var req CreateTenDlcBrandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.LegalName != "Acme Holdings LLC" {
			t.Errorf("expected LegalName to be 'Acme Holdings LLC', got '%s'", req.LegalName)
		}
		if req.EIN != "12-3456789" {
			t.Errorf("expected EIN to be '12-3456789', got '%s'", req.EIN)
		}
		if req.Website != "https://acme.example" {
			t.Errorf("expected Website to be 'https://acme.example', got '%s'", req.Website)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"id":         "brd_123",
				"legalName":  req.LegalName,
				"entityType": "PRIVATE_PROFIT",
				"status":     "pending",
				"createdAt":  "2026-01-01T00:00:00Z",
				"updatedAt":  "2026-01-01T00:00:00Z",
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.TenDlc.CreateBrand(ctx, &CreateTenDlcBrandRequest{
		LegalName: "Acme Holdings LLC",
		EIN:       "12-3456789",
		Website:   "https://acme.example",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Data.ID != "brd_123" {
		t.Errorf("expected ID to be 'brd_123', got '%s'", resp.Data.ID)
	}
	if resp.Data.Status != "pending" {
		t.Errorf("expected Status to be 'pending', got '%s'", resp.Data.Status)
	}
}

func TestTenDlcCreateBrand_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	tests := []struct {
		name        string
		req         *CreateTenDlcBrandRequest
		expectedErr string
	}{
		{
			name:        "nil request",
			req:         nil,
			expectedErr: "request is required",
		},
		{
			name:        "empty legalName",
			req:         &CreateTenDlcBrandRequest{EIN: "12-3456789"},
			expectedErr: "legalName is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.TenDlc.CreateBrand(ctx, tt.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error to contain '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestTenDlcGetBrand_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/tendlc/brands/brd_123" {
			t.Errorf("expected path '/tendlc/brands/brd_123', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"id":             "brd_123",
				"legalName":      "Acme Holdings LLC",
				"entityType":     "PRIVATE_PROFIT",
				"status":         "failed",
				"failureReasons": []string{"Business address could not be confirmed"},
				"createdAt":      "2026-01-01T00:00:00Z",
				"updatedAt":      "2026-01-02T00:00:00Z",
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.TenDlc.GetBrand(ctx, "brd_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Data.Status != "failed" {
		t.Errorf("expected Status to be 'failed', got '%s'", resp.Data.Status)
	}
	if len(resp.Data.FailureReasons) != 1 {
		t.Fatalf("expected 1 failure reason, got %d", len(resp.Data.FailureReasons))
	}
}

func TestTenDlcGetBrand_EmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.TenDlc.GetBrand(ctx, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestTenDlcGetBrand_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.TenDlc.GetBrand(ctx, "brd_missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestTenDlcQualify_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/tendlc/brands/brd_123/qualify/MIXED" {
			t.Errorf("expected path '/tendlc/brands/brd_123/qualify/MIXED', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"useCase":   "MIXED",
				"qualified": true,
				"reason":    nil,
				"throughput": map[string]interface{}{
					"tier":          "Standard",
					"carriersReady": 3,
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.TenDlc.Qualify(ctx, "brd_123", "MIXED")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Data.UseCase != "MIXED" {
		t.Errorf("expected UseCase to be 'MIXED', got '%s'", resp.Data.UseCase)
	}
	if !resp.Data.Qualified {
		t.Error("expected Qualified to be true")
	}
	if resp.Data.Throughput == nil || resp.Data.Throughput.Tier != "Standard" {
		t.Errorf("expected Throughput.Tier to be 'Standard', got %v", resp.Data.Throughput)
	}
	if resp.Data.Throughput.CarriersReady != 3 {
		t.Errorf("expected CarriersReady to be 3, got %d", resp.Data.Throughput.CarriersReady)
	}
}

func TestTenDlcQualify_NotQualified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"useCase":    "MARKETING",
				"qualified":  false,
				"reason":     "Use case not accepted for this brand",
				"throughput": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.TenDlc.Qualify(ctx, "brd_123", "MARKETING")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Data.Qualified {
		t.Error("expected Qualified to be false")
	}
	if resp.Data.Reason == nil || *resp.Data.Reason != "Use case not accepted for this brand" {
		t.Errorf("expected Reason to be set, got %v", resp.Data.Reason)
	}
	if resp.Data.Throughput != nil {
		t.Errorf("expected Throughput to be nil, got %v", resp.Data.Throughput)
	}
}

func TestTenDlcQualify_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	if _, err := client.TenDlc.Qualify(ctx, "", "MIXED"); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty brand ID, got %T", err)
	}
	if _, err := client.TenDlc.Qualify(ctx, "brd_123", ""); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty use case, got %T", err)
	}
}

func TestTenDlcListCampaigns_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/tendlc/campaigns" {
			t.Errorf("expected path '/tendlc/campaigns', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":             "cmp_123",
					"brandId":        "brd_123",
					"useCase":        "MIXED",
					"subUseCases":    []string{},
					"status":         "active",
					"sampleMessages": []string{"Your order #123 has shipped!"},
					"createdAt":      "2026-01-01T00:00:00Z",
					"updatedAt":      "2026-01-02T00:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.TenDlc.ListCampaigns(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 campaign, got %d", len(resp.Data))
	}
	if resp.Data[0].BrandID != "brd_123" {
		t.Errorf("expected BrandID to be 'brd_123', got '%s'", resp.Data[0].BrandID)
	}
	if resp.Data[0].Status != "active" {
		t.Errorf("expected Status to be 'active', got '%s'", resp.Data[0].Status)
	}
}

func TestTenDlcCreateCampaign_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/tendlc/campaigns" {
			t.Errorf("expected path '/tendlc/campaigns', got '%s'", r.URL.Path)
		}

		var req CreateTenDlcCampaignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.BrandID != "brd_123" {
			t.Errorf("expected BrandID to be 'brd_123', got '%s'", req.BrandID)
		}
		if req.UseCase != "MIXED" {
			t.Errorf("expected UseCase to be 'MIXED', got '%s'", req.UseCase)
		}
		if req.MessageFlow != "Customers opt in at checkout on acme.example" {
			t.Errorf("unexpected MessageFlow: '%s'", req.MessageFlow)
		}
		if len(req.SampleMessages) != 1 || req.SampleMessages[0] != "Your order #123 has shipped!" {
			t.Errorf("unexpected SampleMessages: %v", req.SampleMessages)
		}
		if req.OptOutKeywords != "STOP" {
			t.Errorf("expected OptOutKeywords to be 'STOP', got '%s'", req.OptOutKeywords)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"id":             "cmp_123",
				"brandId":        req.BrandID,
				"useCase":        req.UseCase,
				"subUseCases":    []string{},
				"status":         "pending",
				"sampleMessages": req.SampleMessages,
				"createdAt":      "2026-01-01T00:00:00Z",
				"updatedAt":      "2026-01-01T00:00:00Z",
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.TenDlc.CreateCampaign(ctx, &CreateTenDlcCampaignRequest{
		BrandID:        "brd_123",
		UseCase:        "MIXED",
		Description:    "Order updates and support replies for Acme customers",
		MessageFlow:    "Customers opt in at checkout on acme.example",
		SampleMessages: []string{"Your order #123 has shipped!"},
		OptOutKeywords: "STOP",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Data.ID != "cmp_123" {
		t.Errorf("expected ID to be 'cmp_123', got '%s'", resp.Data.ID)
	}
	if resp.Data.Status != "pending" {
		t.Errorf("expected Status to be 'pending', got '%s'", resp.Data.Status)
	}
}

func TestTenDlcCreateCampaign_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	valid := CreateTenDlcCampaignRequest{
		BrandID:        "brd_123",
		UseCase:        "MIXED",
		Description:    "Order updates",
		MessageFlow:    "Customers opt in at checkout",
		SampleMessages: []string{"Your order #123 has shipped!"},
	}

	tests := []struct {
		name        string
		mutate      func(*CreateTenDlcCampaignRequest)
		expectedErr string
	}{
		{
			name:        "missing brandId",
			mutate:      func(r *CreateTenDlcCampaignRequest) { r.BrandID = "" },
			expectedErr: "brandId is required",
		},
		{
			name:        "missing useCase",
			mutate:      func(r *CreateTenDlcCampaignRequest) { r.UseCase = "" },
			expectedErr: "useCase is required",
		},
		{
			name:        "missing description",
			mutate:      func(r *CreateTenDlcCampaignRequest) { r.Description = "" },
			expectedErr: "description is required",
		},
		{
			name:        "missing messageFlow",
			mutate:      func(r *CreateTenDlcCampaignRequest) { r.MessageFlow = "" },
			expectedErr: "messageFlow is required",
		},
		{
			name:        "missing sampleMessages",
			mutate:      func(r *CreateTenDlcCampaignRequest) { r.SampleMessages = nil },
			expectedErr: "sampleMessages is required",
		},
	}

	if _, err := client.TenDlc.CreateCampaign(ctx, nil); !IsValidationError(err) {
		t.Errorf("expected ValidationError for nil request, got %T", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			tt.mutate(&req)
			_, err := client.TenDlc.CreateCampaign(ctx, &req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error to contain '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestTenDlcGetCampaign_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/tendlc/campaigns/cmp_123" {
			t.Errorf("expected path '/tendlc/campaigns/cmp_123', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"id":             "cmp_123",
				"brandId":        "brd_123",
				"useCase":        "MIXED",
				"subUseCases":    []string{},
				"status":         "active",
				"sampleMessages": []string{"Your order #123 has shipped!"},
				"throughput": map[string]interface{}{
					"tier":          "Standard",
					"carriersReady": 4,
				},
				"createdAt": "2026-01-01T00:00:00Z",
				"updatedAt": "2026-01-02T00:00:00Z",
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.TenDlc.GetCampaign(ctx, "cmp_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Data.Status != "active" {
		t.Errorf("expected Status to be 'active', got '%s'", resp.Data.Status)
	}
	if resp.Data.Throughput == nil || resp.Data.Throughput.CarriersReady != 4 {
		t.Errorf("expected Throughput.CarriersReady to be 4, got %v", resp.Data.Throughput)
	}
}

func TestTenDlcAssignNumber_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/tendlc/campaigns/cmp_123/assign" {
			t.Errorf("expected path '/tendlc/campaigns/cmp_123/assign', got '%s'", r.URL.Path)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["phoneNumber"] != "+15551234567" {
			t.Errorf("expected phoneNumber to be '+15551234567', got '%s'", body["phoneNumber"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"id":          "asg_123",
				"campaignId":  "cmp_123",
				"phoneNumber": "+15551234567",
				"status":      "Under review",
				"assignedAt":  nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.TenDlc.AssignNumber(ctx, "cmp_123", "+15551234567")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Data.ID != "asg_123" {
		t.Errorf("expected ID to be 'asg_123', got '%s'", resp.Data.ID)
	}
	if resp.Data.Status != "Under review" {
		t.Errorf("expected Status to be 'Under review', got '%s'", resp.Data.Status)
	}
	if resp.Data.AssignedAt != nil {
		t.Errorf("expected AssignedAt to be nil, got '%s'", *resp.Data.AssignedAt)
	}
}

func TestTenDlcAssignNumber_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with validation error")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	if _, err := client.TenDlc.AssignNumber(ctx, "", "+15551234567"); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty campaign ID, got %T", err)
	}
	if _, err := client.TenDlc.AssignNumber(ctx, "cmp_123", ""); !IsValidationError(err) {
		t.Errorf("expected ValidationError for empty phone number, got %T", err)
	}
}

func TestTenDlcAssignNumber_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "number_already_assigned",
			"message": "This number is already assigned to another campaign",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))
	ctx := context.Background()

	_, err := client.TenDlc.AssignNumber(ctx, "cmp_123", "+15551234567")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	sendlyErr, ok := err.(*SendlyError)
	if !ok {
		t.Fatalf("expected SendlyError, got %T", err)
	}
	if sendlyErr.StatusCode != http.StatusConflict {
		t.Errorf("expected StatusCode to be 409, got %d", sendlyErr.StatusCode)
	}
}

func TestTenDlcListAssignments_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/tendlc/assignments" {
			t.Errorf("expected path '/tendlc/assignments', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":          "asg_123",
					"campaignId":  "cmp_123",
					"phoneNumber": "+15551234567",
					"status":      "Active",
					"assignedAt":  "2026-01-03T00:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.TenDlc.ListAssignments(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(resp.Data))
	}
	if resp.Data[0].PhoneNumber != "+15551234567" {
		t.Errorf("expected PhoneNumber to be '+15551234567', got '%s'", resp.Data[0].PhoneNumber)
	}
	if resp.Data[0].Status != "Active" {
		t.Errorf("expected Status to be 'Active', got '%s'", resp.Data[0].Status)
	}
	if resp.Data[0].AssignedAt == nil || *resp.Data[0].AssignedAt != "2026-01-03T00:00:00Z" {
		t.Errorf("expected AssignedAt to be '2026-01-03T00:00:00Z', got %v", resp.Data[0].AssignedAt)
	}
}
