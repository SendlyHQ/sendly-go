package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
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

const wireDeliveries = `{"deliveries":[{"id":"del_1","webhook_id":"whk_1","event_id":"evt_1","event_type":"message.delivered","status":"delivered","success":true,"response_status_code":200,"http_status":200,"response_time":87,"response_time_ms":87,"response_body":"ok","error_message":null,"error_code":null,"attempt_number":1,"max_attempts":6,"next_retry_at":null,"created_at":"2026-09-25T10:00:00.000Z","delivered_at":"2026-09-25T10:00:00.087Z"}],"pagination":{"limit":50,"offset":0}}`

func TestWebhooksGetDeliveries_DecodesEnvelope(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/webhooks/whk_1/deliveries" {
			t.Errorf("expected path '/webhooks/whk_1/deliveries', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(wireDeliveries))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	deliveries, err := client.Webhooks.GetDeliveries(context.Background(), "whk_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 1 {
		t.Errorf("expected 1 request, got %d", requests)
	}
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(deliveries))
	}
	d := deliveries[0]
	if d.ID != "del_1" || d.WebhookID != "whk_1" || d.EventID != "evt_1" || d.EventType != "message.delivered" {
		t.Errorf("unexpected delivery identity: %+v", d)
	}
	if d.Status != DeliveryStatusDelivered || d.AttemptNumber != 1 || d.MaxAttempts != 6 {
		t.Errorf("unexpected delivery status fields: %+v", d)
	}
	if d.ResponseStatusCode == nil || *d.ResponseStatusCode != 200 {
		t.Errorf("expected ResponseStatusCode 200, got %v", d.ResponseStatusCode)
	}
	if d.ResponseTimeMs == nil || *d.ResponseTimeMs != 87 {
		t.Errorf("expected ResponseTimeMs 87, got %v", d.ResponseTimeMs)
	}
	if d.DeliveredAt == nil || *d.DeliveredAt != "2026-09-25T10:00:00.087Z" {
		t.Errorf("expected DeliveredAt to decode, got %v", d.DeliveredAt)
	}
}

func TestWebhooksGetDeliveries_NoResponseTimeWithoutAResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"deliveries":[{"id":"del_2","webhook_id":"whk_1","event_id":"evt_2","event_type":"message.delivered","status":"failed","success":false,"response_status_code":null,"http_status":0,"response_time":null,"response_time_ms":0,"response_body":null,"error_message":"Request timeout","error_code":"timeout","attempt_number":6,"max_attempts":6,"next_retry_at":null,"created_at":"2026-09-25T10:00:00.000Z","delivered_at":null}],"pagination":{"limit":50,"offset":0}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	deliveries, err := client.Webhooks.GetDeliveries(context.Background(), "whk_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(deliveries))
	}
	d := deliveries[0]
	if d.ResponseTimeMs != nil {
		t.Errorf("expected no ResponseTimeMs for a delivery that timed out, got %d", *d.ResponseTimeMs)
	}
	if d.ResponseStatusCode != nil {
		t.Errorf("expected no ResponseStatusCode, got %d", *d.ResponseStatusCode)
	}
	if d.Status != DeliveryStatusFailed || d.ErrorCode == nil || *d.ErrorCode != "timeout" {
		t.Errorf("unexpected delivery: %+v", d)
	}
}

func TestWebhooksGetDeliveriesWithOptions_SendsFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("limit") != "10" || query.Get("offset") != "20" || query.Get("status") != "failed" {
			t.Errorf("expected limit=10&offset=20&status=failed, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"deliveries":[],"pagination":{"limit":10,"offset":20}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	deliveries, err := client.Webhooks.GetDeliveriesWithOptions(context.Background(), "whk_1", &ListWebhookDeliveriesOptions{
		Limit:  10,
		Offset: 20,
		Status: DeliveryStatusFailed,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deliveries == nil || len(deliveries) != 0 {
		t.Errorf("expected an empty, non-nil list, got %#v", deliveries)
	}
}

func TestWebhooksTest_ReadsTheTestDelivery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/webhooks/whk_1/test" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"message":"Test webhook delivered successfully in 87ms","delivery":{"id":"del_1","delivery_id":"del_1","webhook_url":"https://example.com/hook","event_type":"webhook.test","status":"delivered","response_time":87,"status_code":200,"response_body":"ok","delivered_at":"2026-09-25T10:00:00.087Z"}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	result, err := client.Webhooks.Test(context.Background(), "whk_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected Success")
	}
	if result.StatusCode == nil || *result.StatusCode != 200 {
		t.Errorf("expected StatusCode 200, got %v", result.StatusCode)
	}
	if result.ResponseTimeMs == nil || *result.ResponseTimeMs != 87 {
		t.Errorf("expected ResponseTimeMs 87, got %v", result.ResponseTimeMs)
	}
	if result.Error != nil {
		t.Errorf("expected no Error, got %q", *result.Error)
	}
	if result.Message != "Test webhook delivered successfully in 87ms" {
		t.Errorf("unexpected Message %q", result.Message)
	}
	if result.Delivery == nil || result.Delivery.ID != "del_1" || result.Delivery.Status != "delivered" || result.Delivery.WebhookURL != "https://example.com/hook" {
		t.Errorf("unexpected Delivery: %+v", result.Delivery)
	}
}

func TestWebhooksTest_FailedTestIsAValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"message":"Test webhook failed: HTTP 500 Internal Server Error"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	_, err := client.Webhooks.Test(context.Background(), "whk_1")
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if validationErr.Message != "Test webhook failed: HTTP 500 Internal Server Error" {
		t.Errorf("unexpected Message %q", validationErr.Message)
	}
}

func TestWebhookTestResult_RoundTrip(t *testing.T) {
	statusCode, responseTime := 200, 87
	original := WebhookTestResult{
		Success:        true,
		StatusCode:     &statusCode,
		ResponseTimeMs: &responseTime,
		Message:        "Test webhook delivered successfully in 87ms",
		Delivery: &WebhookTestDelivery{
			ID:             "del_1",
			EventType:      "webhook.test",
			Status:         "delivered",
			ResponseTimeMs: &responseTime,
			StatusCode:     &statusCode,
		},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded WebhookTestResult
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round-trip changed the value:\n before %+v\n after  %+v", original, decoded)
	}
}

func TestWebhooksRotateSecret_ReadsTheRotation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/webhooks/whk_1/rotate-secret" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"id":"whk_1","secret":"whsec_new","new_secret":"whsec_new","new_secret_version":1,"grace_period_hours":24,"rotated_at":"2026-09-25T10:00:00.000Z","message":"Webhook secret rotated successfully. Save this secret - it won't be shown again."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	rotation, err := client.Webhooks.RotateSecret(context.Background(), "whk_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rotation.NewSecret != "whsec_new" {
		t.Errorf("expected NewSecret whsec_new, got %q", rotation.NewSecret)
	}
	if rotation.Webhook.ID != "whk_1" {
		t.Errorf("expected Webhook.ID whk_1, got %q", rotation.Webhook.ID)
	}
	if rotation.RotatedAt != "2026-09-25T10:00:00.000Z" || rotation.GracePeriodHours != 24 || rotation.NewSecretVersion != 1 {
		t.Errorf("unexpected rotation fields: %+v", rotation)
	}
	if rotation.OldSecretExpiresAt != "" {
		t.Errorf("expected no OldSecretExpiresAt, got %q", rotation.OldSecretExpiresAt)
	}
	if rotation.Message == "" {
		t.Errorf("expected the message")
	}
}

func TestWebhookAndKeyMethods_EscapeIDs(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.EscapedPath()+"?"+r.URL.RawQuery)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))
	ctx := context.Background()

	client.Webhooks.Get(ctx, "whk_1/../../account/keys")
	client.Webhooks.Update(ctx, "whk_1?mode=live", UpdateWebhookRequest{})
	client.Webhooks.Delete(ctx, "whk_1#x")
	client.Webhooks.Test(ctx, "whk_1/x")
	client.Webhooks.ResetCircuit(ctx, "whk_1/x")
	client.Webhooks.Redeliver(ctx, "whk_1/x", nil)
	client.Webhooks.Backfill(ctx, "whk_1/x", nil)
	client.Webhooks.RotateSecret(ctx, "whk_1/x")
	client.Account.GetAPIKey(ctx, "../../webhooks")
	client.Account.GetAPIKeyUsage(ctx, "key_1?x=1")
	client.Account.RevokeAPIKey(ctx, "key_1/x")
	client.Account.RotateAPIKey(ctx, "key_1/x", nil)

	want := []string{
		"GET /webhooks/whk_1%2F..%2F..%2Faccount%2Fkeys?",
		"PATCH /webhooks/whk_1%3Fmode=live?",
		"DELETE /webhooks/whk_1%23x?",
		"POST /webhooks/whk_1%2Fx/test?",
		"POST /webhooks/whk_1%2Fx/reset-circuit?",
		"POST /webhooks/whk_1%2Fx/redeliver?",
		"POST /webhooks/whk_1%2Fx/backfill?",
		"POST /webhooks/whk_1%2Fx/rotate-secret?",
		"GET /account/keys/..%2F..%2Fwebhooks?",
		"GET /account/keys/key_1%3Fx=1/usage?",
		"PATCH /account/keys/key_1%2Fx/revoke?",
		"POST /account/keys/key_1%2Fx/rotate?",
	}
	if len(paths) != len(want) {
		t.Fatalf("expected %d requests, got %d: %v", len(want), len(paths), paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("request %d: expected %q, got %q", i, want[i], paths[i])
		}
	}
}
