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

// A lifecycle payload that reuses a message field name at an incompatible type
// used to fail the whole parse, so RawObject never reached the caller — which
// defeated the point of adding it. Verified against the published v3.40.0
// before this fix: both returned "cannot unmarshal ... WebhookMessageData".
func TestLifecycleTypeCollisionStillReachable(t *testing.T) {
	cases := map[string]string{
		"status as object": `{"id":"evt_1","type":"call.completed","api_version":"2024-01","created":1,"livemode":true,"data":{"object":{"id":"call_1","status":{"code":200},"hangup_class":"normal"}}}`,
		"id as number":     `{"id":"evt_2","type":"number.activated","api_version":"2024-01","created":1,"livemode":true,"data":{"object":{"id":12345,"phone":"+15555550100"}}}`,
	}
	for name, payload := range cases {
		sig, ts := signed(t, payload)
		event, err := Webhooks{}.ParseEvent(payload, sig, testSecret, ts)
		if err != nil {
			t.Fatalf("%s: ParseEvent should not fail on a lifecycle payload: %v", name, err)
		}
		if len(event.RawObject) == 0 {
			t.Fatalf("%s: RawObject must carry the payload", name)
		}
	}
}

// The message view is decoded only for events that carry one. opt_in/opt_out
// share the message.* prefix but carry an opt-out record, so decoding them as a
// message would invent to/from/segments the server never sent.
func TestOptOutIsNotDecodedAsAMessage(t *testing.T) {
	payload := `{"id":"evt_3","type":"message.opt_out","api_version":"2024-01","created":1,"livemode":true,"data":{"object":{"phone_number":"+15555550100","keyword":"STOP","from_number":"+15555550199","timestamp":"2026-01-01T00:00:00Z"}}}`
	sig, ts := signed(t, payload)
	event, err := Webhooks{}.ParseEvent(payload, sig, testSecret, ts)
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if event.Data.Segments != 0 || event.Data.To != "" {
		t.Fatalf("invented message fields on an opt-out: %+v", event.Data)
	}
	var obj map[string]any
	if err := event.DecodeObject(&obj); err != nil || obj["keyword"] != "STOP" {
		t.Fatalf("opt-out payload unreachable: %v %v", err, obj)
	}
}

// Regression: the message_id fallback once sat OUTSIDE the message-event gate,
// so a legacy flat-payload contact.auto_flagged had its ID filled from
// message_id — a different row entirely. That is the wrong-record bug this
// release exists to remove, and it must not come back through the legacy path.
func TestLegacyFlatPayloadDoesNotBorrowMessageID(t *testing.T) {
	payload := `{"id":"evt_9","type":"contact.auto_flagged","api_version":"2024-01","created":1,"livemode":true,"data":{"id":"contact_ABC","message_id":"msg_REAL","source":"send_failure"}}`
	sig, ts := signed(t, payload)
	event, err := Webhooks{}.ParseEvent(payload, sig, testSecret, ts)
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if event.Data.ID != "" {
		t.Fatalf("a non-message event must not carry a message id, got %q", event.Data.ID)
	}
	var obj map[string]any
	if err := event.DecodeObject(&obj); err != nil {
		t.Fatalf("DecodeObject: %v", err)
	}
	if obj["id"] != "contact_ABC" || obj["message_id"] != "msg_REAL" {
		t.Fatalf("payload not reachable verbatim: %v", obj)
	}
}

// A message event whose object genuinely does not decode must still surface the
// error rather than returning a half-populated struct.
func TestMalformedMessageEventStillErrors(t *testing.T) {
	payload := `{"id":"evt_10","type":"message.received","api_version":"2024-01","created":1,"livemode":true,"data":{"object":{"id":999,"to":"+15555550100","segments":3}}}`
	sig, ts := signed(t, payload)
	w := Webhooks{}
	if _, err := w.ParseEvent(payload, sig, testSecret, ts); err == nil {
		t.Fatal("a message event with a numeric id should error, not decode partially")
	}
}
