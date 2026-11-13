package sendly

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRulesUpdate_SendsEnabledFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/rules/rule_abc123" {
			t.Errorf("expected PATCH /rules/rule_abc123, got %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]interface{}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("bad body: %v", err)
		}
		if v, ok := body["enabled"]; !ok || v != false {
			t.Errorf("expected enabled false in body, got %s", raw)
		}
		if _, ok := body["name"]; ok {
			t.Errorf("expected no name in body, got %s", raw)
		}
		w.Write([]byte(`{"id":"rule_abc123","name":"Urgent","conditions":{},"actions":{},"priority":0,"enabled":false,"createdAt":"2026-11-01T10:00:00.000Z","updatedAt":"2026-11-13T10:00:00.000Z"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	off := false
	rule, err := client.Rules.Update(context.Background(), "rule_abc123", &UpdateRuleRequest{Enabled: &off})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Enabled == nil || *rule.Enabled {
		t.Errorf("expected Enabled false, got %v", rule.Enabled)
	}
}

func TestRulesUpdate_OmitsEnabledWhenUnset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]interface{}
		json.Unmarshal(raw, &body)
		if _, ok := body["enabled"]; ok {
			t.Errorf("expected no enabled in body, got %s", raw)
		}
		w.Write([]byte(`{"id":"rule_abc123","name":"Renamed","conditions":{},"actions":{},"createdAt":"2026-11-01T10:00:00.000Z","updatedAt":"2026-11-13T10:00:00.000Z"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.Rules.Update(context.Background(), "rule_abc123", &UpdateRuleRequest{Name: "Renamed"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
