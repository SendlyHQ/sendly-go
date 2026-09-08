package sendly

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

const testSecret = "whsec_test_secret"

func signed(t *testing.T, payload string) (string, string) {
	t.Helper()
	w := Webhooks{}
	// Signature verification enforces a 300s tolerance, so this must be live.
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	return w.GenerateSignature(payload, testSecret, ts), ts
}

// A lifecycle event's data.object is not message-shaped. Before RawObject there
// was no way to reach agent_id, name or stage at all: they were decoded into
// WebhookMessageData, which has no such fields, and silently dropped.
func TestParseEventExposesLifecycleObject(t *testing.T) {
	payload := `{"id":"evt_1","type":"rcs_agent.live","api_version":"2024-01","created":1757000000,"livemode":true,` +
		`"data":{"object":{"agent_id":"agt_123","name":"Acme Support","stage":"live","organization_id":"org_1"}}}`

	sig, ts := signed(t, payload)
	event, err := Webhooks{}.ParseEvent(payload, sig, testSecret, ts)
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if event.Type != WebhookEventRcsAgentLive {
		t.Fatalf("type = %q, want rcs_agent.live", event.Type)
	}

	var agent struct {
		AgentID string `json:"agent_id"`
		Name    string `json:"name"`
		Stage   string `json:"stage"`
	}
	if err := event.DecodeObject(&agent); err != nil {
		t.Fatalf("DecodeObject: %v", err)
	}
	if agent.AgentID != "agt_123" || agent.Name != "Acme Support" || agent.Stage != "live" {
		t.Fatalf("decoded %+v, want agt_123/Acme Support/live", agent)
	}

	// The message view is empty for this event, which is why RawObject exists.
	if event.Data.ID != "" {
		t.Fatalf("Data.ID = %q, want empty for a lifecycle event", event.Data.ID)
	}
}

func TestParseEventStillDecodesMessageEvents(t *testing.T) {
	payload := `{"id":"evt_2","type":"message.delivered","api_version":"2024-01","created":1757000000,"livemode":true,` +
		`"data":{"object":{"id":"msg_1","to":"+15551234567","from":"+15559876543","status":"delivered","segments":1,"credits_used":2}}}`

	sig, ts := signed(t, payload)
	event, err := Webhooks{}.ParseEvent(payload, sig, testSecret, ts)
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if event.Data.ID != "msg_1" || event.Data.To != "+15551234567" || event.Data.CreditsUsed != 2 {
		t.Fatalf("message decode wrong: %+v", event.Data)
	}
	// RawObject is populated for message events too.
	var m map[string]any
	if err := json.Unmarshal(event.RawObject, &m); err != nil {
		t.Fatalf("RawObject not valid json: %v", err)
	}
	if m["id"] != "msg_1" {
		t.Fatalf("RawObject id = %v", m["id"])
	}
}

func TestDecodeObjectErrorsWhenAbsent(t *testing.T) {
	var e WebhookEvent
	var v map[string]any
	if err := e.DecodeObject(&v); err == nil {
		t.Fatal("expected an error when the event carries no data.object")
	}
}

// An event type this build does not know must not fail the parse.
func TestParseEventAcceptsUnknownType(t *testing.T) {
	payload := `{"id":"evt_3","type":"something.invented_later","api_version":"2024-01","created":1757000000,"livemode":true,` +
		`"data":{"object":{"foo":"bar"}}}`
	sig, ts := signed(t, payload)
	event, err := Webhooks{}.ParseEvent(payload, sig, testSecret, ts)
	if err != nil {
		t.Fatalf("ParseEvent on unknown type: %v", err)
	}
	if event.Type != WebhookEventType("something.invented_later") {
		t.Fatalf("type = %q", event.Type)
	}
}
