package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRCSAgentsList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/agents" {
			t.Errorf("expected path '/rcs/agents', got '%s'", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"agents": [
				{
					"id": "rcsa_1",
					"name": "Acme Inc",
					"status": "approved",
					"useCase": "OTP",
					"sendable": true,
					"stage": "live",
					"createdAt": "2026-07-01T00:00:00Z"
				},
				{
					"id": "rcsa_2",
					"name": "Acme Promos",
					"status": "draft",
					"useCase": null,
					"sendable": false,
					"createdAt": "2026-07-02T00:00:00Z"
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	resp, err := client.RCS.Agents.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(resp.Agents))
	}
	if resp.Agents[0].ID != "rcsa_1" {
		t.Errorf("expected ID to be 'rcsa_1', got '%s'", resp.Agents[0].ID)
	}
	if resp.Agents[0].Name != "Acme Inc" {
		t.Errorf("expected Name to be 'Acme Inc', got '%s'", resp.Agents[0].Name)
	}
	if resp.Agents[0].Status != "approved" {
		t.Errorf("expected Status to be 'approved', got '%s'", resp.Agents[0].Status)
	}
	if resp.Agents[0].UseCase == nil || *resp.Agents[0].UseCase != "OTP" {
		t.Errorf("expected UseCase to be 'OTP', got %v", resp.Agents[0].UseCase)
	}
	if !resp.Agents[0].Sendable {
		t.Error("expected Sendable to be true")
	}
	if resp.Agents[0].Stage != RcsCustomerStageLive {
		t.Errorf("expected Stage to be 'live', got '%s'", resp.Agents[0].Stage)
	}
	if resp.Agents[1].Stage != "" {
		t.Errorf("expected Stage to be empty, got '%s'", resp.Agents[1].Stage)
	}
	if resp.Agents[1].UseCase != nil {
		t.Errorf("expected UseCase to be nil, got '%s'", *resp.Agents[1].UseCase)
	}
	if resp.Agents[1].Sendable {
		t.Error("expected Sendable to be false")
	}
}

func TestRCSAgentsList_NotEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIError{
			Code:    "rcs_not_enabled",
			Message: "RCS isn't set up on this workspace yet.",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.RCS.Agents.List(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestRCSCapability_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/rcs/capability" {
			t.Errorf("expected path '/rcs/capability', got '%s'", r.URL.Path)
		}

		query := r.URL.Query()
		if to := query.Get("to"); to != "+15551234567" {
			t.Errorf("expected to to be '+15551234567', got '%s'", to)
		}
		if agentID := query.Get("agentId"); agentID != "rcsa_1" {
			t.Errorf("expected agentId to be 'rcsa_1', got '%s'", agentID)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"to": "+15551234567",
			"agentId": "rcsa_1",
			"capable": true,
			"features": ["RICHCARD_STANDALONE", "ACTION_CREATE_CALENDAR_EVENT"]
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	capability, err := client.RCS.Capability(ctx, "+15551234567", "rcsa_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capability.To != "+15551234567" {
		t.Errorf("expected To to be '+15551234567', got '%s'", capability.To)
	}
	if capability.AgentID != "rcsa_1" {
		t.Errorf("expected AgentID to be 'rcsa_1', got '%s'", capability.AgentID)
	}
	if !capability.Capable {
		t.Error("expected Capable to be true")
	}
	if len(capability.Features) != 2 {
		t.Errorf("expected 2 features, got %v", capability.Features)
	}
}

func TestRCSCapability_OmitsEmptyAgentID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if _, present := query["agentId"]; present {
			t.Error("expected agentId to be omitted")
		}
		if to := query.Get("to"); to != "+15551234567" {
			t.Errorf("expected to to be '+15551234567', got '%s'", to)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"to": "+15551234567",
			"agentId": "rcsa_1",
			"capable": false,
			"features": []
		}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	capability, err := client.RCS.Capability(ctx, "+15551234567", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capability.Capable {
		t.Error("expected Capable to be false")
	}
	if len(capability.Features) != 0 {
		t.Errorf("expected no features, got %v", capability.Features)
	}
}

func TestRCSCapability_EmptyTo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with empty to")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := client.RCS.Capability(ctx, "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(err.Error(), "to is required") {
		t.Errorf("expected error to contain 'to is required', got '%s'", err.Error())
	}
}

func TestRCSCapability_RequiresLiveKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(APIError{
			Code:    "rcs_requires_live_key",
			Message: "RCS capability checks require a live API key. Test keys cannot query RCS capability.",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))
	ctx := context.Background()

	_, err := client.RCS.Capability(ctx, "+15551234567", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	sendlyErr, ok := err.(*SendlyError)
	if !ok {
		t.Fatalf("expected SendlyError, got %T", err)
	}
	if sendlyErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected StatusCode to be 403, got %d", sendlyErr.StatusCode)
	}
	if sendlyErr.Code != "rcs_requires_live_key" {
		t.Errorf("expected Code to be 'rcs_requires_live_key', got '%s'", sendlyErr.Code)
	}
}
