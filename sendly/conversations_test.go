package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const wireConversation = `{"id":"conv_1","userId":"usr_1","organizationId":null,"phoneNumber":"+15551230001","status":"active","unreadCount":2,"messageCount":10,"lastMessageText":"Thanks","lastMessageAt":"2026-09-25T10:00:00.000Z","lastMessageDirection":"inbound","channel":"sms","metadata":{},"tags":[],"contactId":"ct_1","createdAt":"2026-09-20T10:00:00.000Z","updatedAt":"2026-09-25T10:00:00.000Z","isGroup":false}`

func labelsServer(t *testing.T, requests *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == "POST" && r.URL.Path == "/conversations/conv_1/labels":
			var body struct {
				LabelIDs []string `json:"labelIds"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.LabelIDs) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"invalid_request","message":"labelIds array is required"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":[{"id":"lbl_1","name":"VIP","color":"#f00","createdAt":"2026-09-01T00:00:00.000Z"}]}`))
		case r.Method == "DELETE" && r.URL.Path == "/conversations/conv_1/labels/lbl_1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == "GET" && r.URL.Path == "/conversations/conv_1":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(wireConversation))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestConversationsAddLabels_ReturnsTheConversation(t *testing.T) {
	var requests []string
	server := labelsServer(t, &requests)
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	conversation, err := client.Conversations.AddLabels(context.Background(), "conv_1", []string{"lbl_1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conversation.ID != "conv_1" || conversation.PhoneNumber != "+15551230001" || conversation.UnreadCount != 2 {
		t.Errorf("expected conversation conv_1, got %+v", conversation)
	}
	want := []string{"POST /conversations/conv_1/labels", "GET /conversations/conv_1"}
	if !reflect.DeepEqual(requests, want) {
		t.Errorf("expected requests %v, got %v", want, requests)
	}
}

func TestConversationsRemoveLabel_ReturnsTheConversation(t *testing.T) {
	var requests []string
	server := labelsServer(t, &requests)
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	conversation, err := client.Conversations.RemoveLabel(context.Background(), "conv_1", "lbl_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conversation.ID != "conv_1" || conversation.Status != ConversationStatusActive {
		t.Errorf("expected conversation conv_1, got %+v", conversation)
	}
	want := []string{"DELETE /conversations/conv_1/labels/lbl_1", "GET /conversations/conv_1"}
	if !reflect.DeepEqual(requests, want) {
		t.Errorf("expected requests %v, got %v", want, requests)
	}
}

func TestConversationsAddLabels_KeepsTheIDWhenTheReadFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":[{"id":"lbl_1","name":"VIP"}]}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"insufficient_scope","message":"This API key lacks the sms:read scope"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	conversation, err := client.Conversations.AddLabels(context.Background(), "conv_1", []string{"lbl_1"})
	if err != nil {
		t.Fatalf("expected the labels to be added without an error, got %v", err)
	}
	if conversation.ID != "conv_1" {
		t.Errorf("expected ID conv_1, got %q", conversation.ID)
	}
}

func TestConversationsSuggestReplies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/conversations/conv_1/suggest-replies" {
			t.Errorf("expected POST /conversations/conv_1/suggest-replies, got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"suggestions":[{"text":"Yes, we open at 9am.","tone":"professional"},{"text":"See you at 9!","tone":"friendly"}],"basedOnMessageId":"msg_in_1","model":"gpt-4o-mini"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.Conversations.SuggestReplies(context.Background(), "conv_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Suggestions) != 2 || resp.Suggestions[1].Tone != "friendly" {
		t.Errorf("expected 2 suggestions, the second friendly, got %+v", resp.Suggestions)
	}
	if resp.BasedOnMessageId != "msg_in_1" || resp.Model != "gpt-4o-mini" {
		t.Errorf("expected basedOnMessageId and model, got %q %q", resp.BasedOnMessageId, resp.Model)
	}
	if resp.Disabled || resp.Reason != "" {
		t.Errorf("expected suggestions enabled, got Disabled=%v Reason=%q", resp.Disabled, resp.Reason)
	}
}

func TestConversationsSuggestReplies_DisabledByEnterprise(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"suggestions":[],"disabled":true,"reason":"enterprise_policy"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	resp, err := client.Conversations.SuggestReplies(context.Background(), "conv_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Suggestions) != 0 {
		t.Errorf("expected no suggestions, got %+v", resp.Suggestions)
	}
	if !resp.Disabled || resp.Reason != "enterprise_policy" {
		t.Errorf("expected Disabled with reason enterprise_policy, got %v %q", resp.Disabled, resp.Reason)
	}
}

func TestConversationsList_FiltersByLabel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/conversations" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("label") != "lbl_vip" || q.Get("status") != "active" || q.Get("limit") != "10" {
			t.Errorf("unexpected query %s", r.URL.RawQuery)
		}
		w.Write([]byte(`{"data":[],"pagination":{"total":0,"limit":10,"offset":0,"hasMore":false}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.Conversations.List(context.Background(), &ListConversationsRequest{
		Limit:  10,
		Status: "active",
		Label:  "lbl_vip",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConversationsList_OmitsLabelWhenUnset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["label"]; ok {
			t.Errorf("expected no label, got %s", r.URL.RawQuery)
		}
		w.Write([]byte(`{"data":[],"pagination":{"total":0,"limit":20,"offset":0,"hasMore":false}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.Conversations.List(context.Background(), &ListConversationsRequest{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
