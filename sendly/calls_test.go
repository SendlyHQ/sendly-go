package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const callRingingJSON = `{
	"id": "6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
	"object": "call",
	"kind": "pstn",
	"direction": "outbound",
	"status": "ringing",
	"handledBy": "agent",
	"agentId": "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
	"from": "+15555550188",
	"to": "+15555550123",
	"callerName": "Front Desk",
	"calleeName": "+15555550123",
	"startedAt": "2026-09-12T14:03:11.000Z",
	"answeredAt": null,
	"endedAt": null,
	"durationSecs": 0,
	"creditsCharged": 0,
	"billing": "metered",
	"hangupClass": null,
	"recordingStatus": null,
	"metadata": {"crmId": "lead_8812"}
}`

const callCompletedJSON = `{
	"id": "6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f",
	"object": "call",
	"kind": "pstn",
	"direction": "outbound",
	"status": "completed",
	"handledBy": "agent",
	"agentId": "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
	"from": "+15555550188",
	"to": "+15555550123",
	"callerName": "Front Desk",
	"calleeName": "+15555550123",
	"startedAt": "2026-09-12T14:03:11.000Z",
	"answeredAt": "2026-09-12T14:03:19.000Z",
	"endedAt": "2026-09-12T14:05:02.000Z",
	"durationSecs": 103,
	"creditsCharged": 20,
	"billing": "settled",
	"hangupClass": "normal",
	"recordingStatus": "ready",
	"metadata": {},
	"transcript": [
		{"speaker": "agent", "text": "Hi Jordan, this is the front desk.", "atMs": 1200},
		{"speaker": "caller", "text": "Hi, yes, Tuesday works.", "atMs": 4800}
	]
}`

const callInboundDashboardJSON = `{
	"id": "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d",
	"object": "call",
	"kind": "pstn",
	"direction": "inbound",
	"status": "completed",
	"handledBy": "dashboard",
	"agentId": null,
	"from": "+15555550177",
	"to": "+15555550188",
	"callerName": null,
	"calleeName": "Front Desk",
	"startedAt": "2026-09-12T13:00:00.000Z",
	"answeredAt": "2026-09-12T13:00:06.000Z",
	"endedAt": "2026-09-12T13:01:00.000Z",
	"durationSecs": 54,
	"creditsCharged": 2,
	"billing": "settled",
	"hangupClass": "caller_hung_up",
	"recordingStatus": null,
	"metadata": {}
}`

func TestCallsCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/calls" {
			t.Errorf("expected path '/calls', got '%s'", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-api-key" {
			t.Errorf("expected Authorization header to be 'Bearer test-api-key', got '%s'", auth)
		}
		if key := r.Header.Get("Idempotency-Key"); !autoKeyPattern.MatchString(key) {
			t.Errorf("expected an auto-generated Idempotency-Key, got '%s'", key)
		}

		body := decodeRequestBody(t, r)
		if body["to"] != "+15555550123" {
			t.Errorf("expected to '+15555550123', got %v", body["to"])
		}
		if body["agentId"] != "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b" {
			t.Errorf("expected agentId, got %v", body["agentId"])
		}
		if body["from"] != "+15555550188" {
			t.Errorf("expected from '+15555550188', got %v", body["from"])
		}
		if body["context"] != "Confirm the 3pm appointment on Tuesday." {
			t.Errorf("unexpected context %v", body["context"])
		}
		metadata, ok := body["metadata"].(map[string]interface{})
		if !ok || metadata["crmId"] != "lead_8812" {
			t.Errorf("expected metadata.crmId 'lead_8812', got %v", body["metadata"])
		}
		for _, key := range []string{"agent_id", "from_number", "toNumber"} {
			if _, present := body[key]; present {
				t.Errorf("unexpected wire key %q in body", key)
			}
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(callRingingJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	call, err := client.Calls.Create(ctx, &CreateCallRequest{
		To:       "+15555550123",
		AgentID:  "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
		From:     "+15555550188",
		Context:  "Confirm the 3pm appointment on Tuesday.",
		Metadata: map[string]string{"crmId": "lead_8812"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if call.ID != "6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f" {
		t.Errorf("unexpected ID %q", call.ID)
	}
	if call.Object != "call" {
		t.Errorf("expected Object 'call', got %q", call.Object)
	}
	if call.Status != CallStatusRinging {
		t.Errorf("expected Status ringing, got %q", call.Status)
	}
	if call.HandledBy != CallHandledByAgent {
		t.Errorf("expected HandledBy agent, got %q", call.HandledBy)
	}
	if call.Kind != CallKindPstn || call.Direction != CallDirectionOutbound {
		t.Errorf("unexpected Kind/Direction %q/%q", call.Kind, call.Direction)
	}
	if call.Billing != CallBillingMetered {
		t.Errorf("expected Billing metered, got %q", call.Billing)
	}
	if call.CreditsCharged != 0 || call.DurationSecs != 0 {
		t.Errorf("expected zero charge and duration, got %d/%d", call.CreditsCharged, call.DurationSecs)
	}
	if call.AgentID == nil || *call.AgentID != "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b" {
		t.Errorf("unexpected AgentID %v", call.AgentID)
	}
	if call.From == nil || *call.From != "+15555550188" || call.To == nil || *call.To != "+15555550123" {
		t.Errorf("unexpected From/To %v/%v", call.From, call.To)
	}
	if call.AnsweredAt != nil || call.EndedAt != nil || call.HangupClass != nil || call.RecordingStatus != nil {
		t.Errorf("expected nil AnsweredAt/EndedAt/HangupClass/RecordingStatus on a ringing call")
	}
	if call.Metadata["crmId"] != "lead_8812" {
		t.Errorf("expected Metadata.crmId 'lead_8812', got %v", call.Metadata)
	}
	if call.Transcript != nil {
		t.Errorf("expected no Transcript on Create, got %v", call.Transcript)
	}
}

func TestCallsCreate_MinimalBodyOmitsOptionalKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeRequestBody(t, r)
		if len(body) != 2 {
			t.Errorf("expected exactly to and agentId, got %v", body)
		}
		for _, key := range []string{"from", "context", "metadata"} {
			if _, present := body[key]; present {
				t.Errorf("expected %q to be omitted when unset", key)
			}
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(callRingingJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.Calls.Create(context.Background(), &CreateCallRequest{
		To:      "+15555550123",
		AgentID: "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCallsCreate_CallerIdempotencyKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if key := r.Header.Get("Idempotency-Key"); key != "call-lead-8812" {
			t.Errorf("expected caller-supplied Idempotency-Key, got '%s'", key)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(callRingingJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.Calls.Create(context.Background(), &CreateCallRequest{
		To:      "+15555550123",
		AgentID: "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
	}, WithIdempotencyKey("call-lead-8812")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCallsCreate_Validation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should reach the server on a client-side validation failure")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	cases := map[string]*CreateCallRequest{
		"nil request":   nil,
		"missing to":    {AgentID: "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b"},
		"missing agent": {To: "+15555550123"},
	}
	for name, req := range cases {
		_, err := client.Calls.Create(ctx, req)
		if err == nil {
			t.Errorf("%s: expected a validation error", name)
			continue
		}
		if !IsValidationError(err) {
			t.Errorf("%s: expected ValidationError, got %T", name, err)
		}
	}
}

func TestCallsCreate_InsufficientCredits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte(`{"error":"insufficient_credits","message":"Calls cost 10 credits a minute. Current balance: 4.","creditsNeeded":10,"currentBalance":4}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.Calls.Create(context.Background(), &CreateCallRequest{
		To:      "+15555550123",
		AgentID: "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsInsufficientCreditsError(err) {
		t.Fatalf("expected InsufficientCreditsError, got %T", err)
	}
	creditsErr := err.(*InsufficientCreditsError)
	if creditsErr.Code != CallErrorCodeInsufficientCredits {
		t.Errorf("expected Code insufficient_credits, got %q", creditsErr.Code)
	}
	if !strings.Contains(creditsErr.Message, "Current balance: 4") {
		t.Errorf("expected the API message, got %q", creditsErr.Message)
	}
}

func TestCallsCreate_E911RequiredIsSendlyErrorWithCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPreconditionRequired)
		w.Write([]byte(`{"error":"e911_required","message":"Register an emergency address for this number before placing calls. It's required by US law."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0))
	_, err := client.Calls.Create(context.Background(), &CreateCallRequest{
		To:      "+15555550123",
		AgentID: "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	sendlyErr, ok := err.(*SendlyError)
	if !ok {
		t.Fatalf("expected *SendlyError, got %T", err)
	}
	if sendlyErr.StatusCode != http.StatusPreconditionRequired {
		t.Errorf("expected status 428, got %d", sendlyErr.StatusCode)
	}
	if sendlyErr.Code != CallErrorCodeE911Required {
		t.Errorf("expected Code e911_required, got %q", sendlyErr.Code)
	}
}

func TestCallsCreate_E911RequiredIsNotRetried(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusPreconditionRequired)
		w.Write([]byte(`{"error":"e911_required","message":"Register an emergency address for this number before placing calls."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	start := time.Now()
	_, err := client.Calls.Create(context.Background(), &CreateCallRequest{
		To:      "+15555550123",
		AgentID: "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	sendlyErr, ok := err.(*SendlyError)
	if !ok {
		t.Fatalf("expected *SendlyError, got %T", err)
	}
	if sendlyErr.StatusCode != http.StatusPreconditionRequired {
		t.Errorf("expected status 428, got %d", sendlyErr.StatusCode)
	}
	if sendlyErr.Code != CallErrorCodeE911Required {
		t.Errorf("expected Code e911_required, got %q", sendlyErr.Code)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("expected exactly 1 request (no retries on 428), got %d", got)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Errorf("expected an immediate return, took %s", elapsed)
	}
}

func TestCallsCreate_VoiceNotEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"voice_not_enabled","message":"Voice is not enabled for your account."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.Calls.Create(context.Background(), &CreateCallRequest{
		To:      "+15555550123",
		AgentID: "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
	})
	if !IsNotFoundError(err) {
		t.Fatalf("expected NotFoundError, got %T (%v)", err, err)
	}
	if err.(*NotFoundError).Code != CallErrorCodeVoiceNotEnabled {
		t.Errorf("expected Code voice_not_enabled, got %q", err.(*NotFoundError).Code)
	}
}

func TestCallsList_NoFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/calls" {
			t.Errorf("expected path '/calls', got '%s'", r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query parameters, got '%s'", r.URL.RawQuery)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "" {
			t.Errorf("expected no Idempotency-Key on GET, got '%s'", key)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [` + callRingingJSON + `, ` + callInboundDashboardJSON + `], "pagination": {"total": 132, "limit": 50, "offset": 0, "hasMore": true}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.Calls.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(resp.Data))
	}
	if resp.Pagination.Total != 132 || resp.Pagination.Limit != 50 || resp.Pagination.Offset != 0 {
		t.Errorf("unexpected pagination %+v", resp.Pagination)
	}
	if !resp.Pagination.HasMore {
		t.Error("expected HasMore to be true")
	}
	inbound := resp.Data[1]
	if inbound.Direction != CallDirectionInbound || inbound.HandledBy != CallHandledByDashboard {
		t.Errorf("unexpected inbound call %+v", inbound)
	}
	if inbound.AgentID != nil || inbound.CallerName != nil {
		t.Errorf("expected nil AgentID and CallerName, got %v %v", inbound.AgentID, inbound.CallerName)
	}
	if inbound.HangupClass == nil || *inbound.HangupClass != "caller_hung_up" {
		t.Errorf("unexpected HangupClass %v", inbound.HangupClass)
	}
	if inbound.Billing != CallBillingSettled || inbound.CreditsCharged != 2 || inbound.DurationSecs != 54 {
		t.Errorf("unexpected billing fields %+v", inbound)
	}
	if len(inbound.Metadata) != 0 {
		t.Errorf("expected empty Metadata, got %v", inbound.Metadata)
	}
}

func TestCallsList_Filters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		expected := map[string]string{
			"limit":     "20",
			"offset":    "40",
			"status":    "completed",
			"direction": "outbound",
			"kind":      "pstn",
			"agentId":   "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
			"to":        "+15555550123",
			"from":      "+15555550188",
		}
		for key, want := range expected {
			if got := q.Get(key); got != want {
				t.Errorf("expected %s=%q, got %q", key, want, got)
			}
		}
		if len(q) != len(expected) {
			t.Errorf("unexpected query keys: %v", q)
		}
		if !strings.Contains(r.URL.RawQuery, "to=%2B15555550123") {
			t.Errorf("expected the leading + to be percent-encoded, got %q", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [], "pagination": {"total": 0, "limit": 20, "offset": 40, "hasMore": false}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.Calls.List(context.Background(), &ListCallsRequest{
		Limit:     20,
		Offset:    40,
		Status:    CallStatusCompleted,
		Direction: CallDirectionOutbound,
		Kind:      CallKindPstn,
		AgentID:   "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
		To:        "+15555550123",
		From:      "+15555550188",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 0 || resp.Pagination.HasMore {
		t.Errorf("unexpected response %+v", resp)
	}
}

func TestCallsList_PartialFiltersOmitEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "status=ringing" {
			t.Errorf("expected only status=ringing, got %q", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [], "pagination": {"total": 0, "limit": 50, "offset": 0, "hasMore": false}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.Calls.List(context.Background(), &ListCallsRequest{Status: CallStatusRinging}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCallsGet_AgentCallHasTranscript(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/calls/6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f" {
			t.Errorf("unexpected path '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(callCompletedJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	call, err := client.Calls.Get(context.Background(), "6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if call.Status != CallStatusCompleted || call.Billing != CallBillingSettled {
		t.Errorf("unexpected Status/Billing %q/%q", call.Status, call.Billing)
	}
	if call.DurationSecs != 103 || call.CreditsCharged != 20 {
		t.Errorf("unexpected DurationSecs/CreditsCharged %d/%d", call.DurationSecs, call.CreditsCharged)
	}
	if call.AnsweredAt == nil || *call.AnsweredAt != "2026-09-12T14:03:19.000Z" {
		t.Errorf("unexpected AnsweredAt %v", call.AnsweredAt)
	}
	if call.HangupClass == nil || *call.HangupClass != "normal" {
		t.Errorf("unexpected HangupClass %v", call.HangupClass)
	}
	if call.RecordingStatus == nil || *call.RecordingStatus != CallRecordingStatusReady {
		t.Errorf("unexpected RecordingStatus %v", call.RecordingStatus)
	}
	if len(call.Transcript) != 2 {
		t.Fatalf("expected 2 transcript lines, got %d", len(call.Transcript))
	}
	if call.Transcript[0].Speaker != "agent" || call.Transcript[0].AtMs != 1200 {
		t.Errorf("unexpected first line %+v", call.Transcript[0])
	}
	if call.Transcript[1].Speaker != "caller" || call.Transcript[1].Text != "Hi, yes, Tuesday works." {
		t.Errorf("unexpected second line %+v", call.Transcript[1])
	}
}

func TestCallsGet_DashboardCallHasNoTranscript(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(callInboundDashboardJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	call, err := client.Calls.Get(context.Background(), "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if call.Transcript != nil {
		t.Errorf("expected nil Transcript on a dashboard-handled call, got %v", call.Transcript)
	}
}

func TestCallsGet_EscapesID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/calls/call%2Fwith%20space" {
			t.Errorf("expected escaped path, got '%s'", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(callRingingJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.Calls.Get(context.Background(), "call/with space"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCallsGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"call_not_found","message":"No call with that id is in this workspace."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.Calls.Get(context.Background(), "missing")
	if !IsNotFoundError(err) {
		t.Fatalf("expected NotFoundError, got %T", err)
	}
	if err.(*NotFoundError).Code != CallErrorCodeCallNotFound {
		t.Errorf("expected Code call_not_found, got %q", err.(*NotFoundError).Code)
	}
}

func TestCallsIDMethods_RequireID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should reach the server on a client-side validation failure")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()

	if _, err := client.Calls.Get(ctx, ""); !IsValidationError(err) {
		t.Errorf("Get: expected ValidationError, got %T", err)
	}
	if _, err := client.Calls.Hangup(ctx, ""); !IsValidationError(err) {
		t.Errorf("Hangup: expected ValidationError, got %T", err)
	}
	if _, err := client.Calls.Recording(ctx, ""); !IsValidationError(err) {
		t.Errorf("Recording: expected ValidationError, got %T", err)
	}
}

func TestCallsHangup_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/calls/6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f/hangup" {
			t.Errorf("unexpected path '%s'", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); !autoKeyPattern.MatchString(key) {
			t.Errorf("expected an auto-generated Idempotency-Key, got '%s'", key)
		}
		body := decodeRequestBody(t, r)
		if len(body) != 0 {
			t.Errorf("expected an empty JSON object body, got %v", body)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strings.Replace(strings.Replace(callRingingJSON, `"status": "ringing"`, `"status": "cancelled"`, 1), `"hangupClass": null`, `"hangupClass": "caller_cancelled"`, 1)))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	call, err := client.Calls.Hangup(context.Background(), "6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if call.Status != CallStatusCancelled {
		t.Errorf("expected Status cancelled, got %q", call.Status)
	}
	if call.HangupClass == nil || *call.HangupClass != "caller_cancelled" {
		t.Errorf("unexpected HangupClass %v", call.HangupClass)
	}
}

func TestCallsRecording_Ready(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/calls/6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f/recording" {
			t.Errorf("unexpected path '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"callId":"6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f","status":"ready","url":"https://example.com/recordings/abc?sig=xyz","expiresAt":"2026-09-12T14:10:00.000Z","contentType":"audio/ogg"}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	rec, err := client.Calls.Recording(context.Background(), "6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.CallID != "6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f" {
		t.Errorf("unexpected CallID %q", rec.CallID)
	}
	if rec.Status != CallRecordingStatusReady {
		t.Errorf("expected Status ready, got %q", rec.Status)
	}
	if rec.URL == nil || *rec.URL != "https://example.com/recordings/abc?sig=xyz" {
		t.Errorf("unexpected URL %v", rec.URL)
	}
	if rec.ExpiresAt == nil || *rec.ExpiresAt != "2026-09-12T14:10:00.000Z" {
		t.Errorf("unexpected ExpiresAt %v", rec.ExpiresAt)
	}
	if rec.ContentType == nil || *rec.ContentType != "audio/ogg" {
		t.Errorf("unexpected ContentType %v", rec.ContentType)
	}
}

func TestCallsRecording_NoneHasNilURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"callId":"9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d","status":"none","url":null,"expiresAt":null,"contentType":null}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	rec, err := client.Calls.Recording(context.Background(), "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Status != CallRecordingStatusNone {
		t.Errorf("expected Status none, got %q", rec.Status)
	}
	if rec.URL != nil || rec.ExpiresAt != nil || rec.ContentType != nil {
		t.Errorf("expected nil URL/ExpiresAt/ContentType, got %v %v %v", rec.URL, rec.ExpiresAt, rec.ContentType)
	}
}

func TestWebhookCallDataRoundTrips(t *testing.T) {
	payload := `{"id":"evt_9","type":"call.completed","api_version":"2024-01","created":1757000000,"livemode":true,"data":{"object":{` +
		`"id":"6f1c2d3e-4a5b-4c6d-8e9f-0a1b2c3d4e5f","object":"call","kind":"pstn","direction":"outbound","status":"completed",` +
		`"handled_by":"agent","agent_id":"3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b","from":"+15555550188","to":"+15555550123",` +
		`"caller_name":"Front Desk","callee_name":"+15555550123","started_at":"2026-09-12T14:03:11.000Z",` +
		`"answered_at":"2026-09-12T14:03:19.000Z","ended_at":"2026-09-12T14:05:02.000Z","duration_secs":103,"credits_charged":20,` +
		`"billing":"settled","hangup_class":"normal","recording_status":"ready","metadata":{"crmId":"lead_8812"},"organization_id":"org_1"}}}`
	sig, ts := signed(t, payload)
	event, err := Webhooks{}.ParseEvent(payload, sig, testSecret, ts)
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if event.Type != WebhookEventCallCompleted {
		t.Fatalf("type = %q", event.Type)
	}

	var call WebhookCallData
	if err := event.DecodeObject(&call); err != nil {
		t.Fatalf("DecodeObject: %v", err)
	}
	if call.Billing != CallBillingSettled {
		t.Errorf("expected Billing settled, got %q", call.Billing)
	}
	if call.Metadata["crmId"] != "lead_8812" {
		t.Errorf("expected Metadata.crmId 'lead_8812', got %v", call.Metadata)
	}
	if call.HandledBy != CallHandledByAgent || call.DurationSecs != 103 || call.CreditsCharged != 20 {
		t.Errorf("unexpected call %+v", call)
	}
	if call.HangupClass == nil || *call.HangupClass != "normal" {
		t.Errorf("unexpected HangupClass %v", call.HangupClass)
	}
	if call.RecordingStatus == nil || *call.RecordingStatus != CallRecordingStatusReady {
		t.Errorf("unexpected RecordingStatus %v", call.RecordingStatus)
	}
	if call.OrganizationID == nil || *call.OrganizationID != "org_1" {
		t.Errorf("unexpected OrganizationID %v", call.OrganizationID)
	}

	var keys map[string]json.RawMessage
	if err := json.Unmarshal(event.RawObject, &keys); err != nil {
		t.Fatalf("raw object: %v", err)
	}
	if len(keys) != 21 {
		t.Errorf("expected the 21-key call object, got %d keys", len(keys))
	}
}

func TestNumbersList_VoiceFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"numbers":[` +
			`{"id":"num_1","phoneNumber":"+15555550188","status":"active","source":"purchased","pendingCancellation":false,"voiceEnabled":true,"voiceMode":"agent"},` +
			`{"id":"num_2","phoneNumber":"+15555550199","status":"active","source":"purchased","pendingCancellation":false,"voiceEnabled":false,"voiceMode":"none"}` +
			`]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.Numbers.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Numbers) != 2 {
		t.Fatalf("expected 2 numbers, got %d", len(resp.Numbers))
	}
	first := resp.Numbers[0]
	if first.VoiceEnabled == nil || !*first.VoiceEnabled || first.VoiceMode == nil || *first.VoiceMode != "agent" {
		t.Errorf("unexpected voice fields on num_1: %v %v", first.VoiceEnabled, first.VoiceMode)
	}
	second := resp.Numbers[1]
	if second.VoiceEnabled == nil || *second.VoiceEnabled || second.VoiceMode == nil || *second.VoiceMode != "none" {
		t.Errorf("unexpected voice fields on num_2: %v %v", second.VoiceEnabled, second.VoiceMode)
	}
}
