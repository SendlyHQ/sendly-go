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
