package sendly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

const voiceNumberAgentJSON = `{
	"id": "5f0c1c2e-2a44-4d4b-9d51-0a9b0f6f4a11",
	"object": "voice_number",
	"phoneNumber": "+15555550188",
	"phoneNumberType": "local",
	"countryCode": "US",
	"isDefault": true,
	"voiceEnabled": true,
	"voiceMode": "agent",
	"agentId": "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
	"emergencyAddress": {
		"status": "active",
		"address": {"street": "500 Example Ave", "unit": "Suite 2", "city": "Austin", "state": "TX", "zip": "78701", "country": "US"}
	},
	"ratePerMinute": {"inbound": 2, "outbound": 2, "agent": 10}
}`

const voiceNumberOffJSON = `{
	"id": "7a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d",
	"object": "voice_number",
	"phoneNumber": "+15555550199",
	"phoneNumberType": null,
	"countryCode": null,
	"isDefault": false,
	"voiceEnabled": false,
	"voiceMode": "none",
	"agentId": null,
	"emergencyAddress": null,
	"ratePerMinute": {"inbound": 2, "outbound": 2, "agent": 10}
}`

const voiceAgentJSON = `{
	"id": "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
	"object": "voice_agent",
	"name": "Front desk",
	"enabled": true,
	"voice": "ashley",
	"voiceLabel": "Ashley (US, warm)",
	"language": "en-US",
	"greeting": "Thanks for calling Acme, how can I help?",
	"instructions": "Answer questions about opening hours.",
	"tools": {"sendSms": true, "transferTo": "+15555550190"},
	"canSendSms": true,
	"callsHandled": 12,
	"avgDurationSecs": 74,
	"createdAt": "2026-09-14T17:00:00.000Z",
	"updatedAt": "2026-09-14T17:05:00.000Z"
}`

func TestVoiceNumbersList_UnwrapsData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/voice/numbers" {
			t.Errorf("expected path '/voice/numbers', got '%s'", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "" {
			t.Errorf("expected no Idempotency-Key on GET, got '%s'", key)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [` + voiceNumberAgentJSON + `, ` + voiceNumberOffJSON + `]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.Voice.Numbers.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 numbers, got %d", len(resp.Data))
	}

	first := resp.Data[0]
	if first.Object != "voice_number" || first.PhoneNumber != "+15555550188" || !first.IsDefault {
		t.Errorf("unexpected number %+v", first)
	}
	if !first.VoiceEnabled || first.VoiceMode != VoiceModeAgent {
		t.Errorf("unexpected VoiceEnabled/VoiceMode %v/%q", first.VoiceEnabled, first.VoiceMode)
	}
	if first.PhoneNumberType == nil || *first.PhoneNumberType != "local" || first.CountryCode == nil || *first.CountryCode != "US" {
		t.Errorf("unexpected PhoneNumberType/CountryCode %v/%v", first.PhoneNumberType, first.CountryCode)
	}
	if first.AgentID == nil || *first.AgentID != "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b" {
		t.Errorf("unexpected AgentID %v", first.AgentID)
	}
	if first.EmergencyAddress == nil || first.EmergencyAddress.Status != "active" {
		t.Fatalf("unexpected EmergencyAddress %+v", first.EmergencyAddress)
	}
	address := first.EmergencyAddress.Address
	if address == nil {
		t.Fatal("expected an address")
	}
	if address.Street != "500 Example Ave" || address.Unit != "Suite 2" || address.City != "Austin" ||
		address.State != "TX" || address.Zip != "78701" || address.Country != "US" {
		t.Errorf("unexpected address %+v", address)
	}
	if first.RatePerMinute != (VoiceNumberRates{Inbound: 2, Outbound: 2, Agent: 10}) {
		t.Errorf("unexpected RatePerMinute %+v", first.RatePerMinute)
	}

	second := resp.Data[1]
	if second.VoiceEnabled || second.VoiceMode != VoiceModeNone {
		t.Errorf("unexpected VoiceEnabled/VoiceMode %v/%q", second.VoiceEnabled, second.VoiceMode)
	}
	if second.AgentID != nil || second.EmergencyAddress != nil || second.PhoneNumberType != nil || second.CountryCode != nil {
		t.Errorf("expected nil AgentID/EmergencyAddress/PhoneNumberType/CountryCode, got %+v", second)
	}
}

func TestVoiceNumbersGet_EncodesPhoneNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.RequestURI != "/voice/numbers/%2B15555550188" {
			t.Errorf("expected the leading + to be percent-encoded, got '%s'", r.RequestURI)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(voiceNumberAgentJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	number, err := client.Voice.Numbers.Get(context.Background(), "+15555550188")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if number.ID != "5f0c1c2e-2a44-4d4b-9d51-0a9b0f6f4a11" {
		t.Errorf("unexpected ID %q", number.ID)
	}
}

func TestVoiceNumbersGet_ByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI != "/voice/numbers/5f0c1c2e-2a44-4d4b-9d51-0a9b0f6f4a11" {
			t.Errorf("unexpected path '%s'", r.RequestURI)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(voiceNumberAgentJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.Voice.Numbers.Get(context.Background(), "5f0c1c2e-2a44-4d4b-9d51-0a9b0f6f4a11"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVoiceNumbersGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"number_not_found","message":"This number isn't in your workspace."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.Voice.Numbers.Get(context.Background(), "+15555550100")
	if !IsNotFoundError(err) {
		t.Fatalf("expected NotFoundError, got %T", err)
	}
	if err.(*NotFoundError).Code != VoiceErrorCodeNumberNotFound {
		t.Errorf("expected Code number_not_found, got %q", err.(*NotFoundError).Code)
	}
}

func TestVoiceNumbersUpdate_WireBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH request, got %s", r.Method)
		}
		if r.RequestURI != "/voice/numbers/%2B15555550188" {
			t.Errorf("unexpected path '%s'", r.RequestURI)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "" {
			t.Errorf("expected no auto-generated Idempotency-Key on PATCH, got '%s'", key)
		}
		body := decodeRequestBody(t, r)
		if len(body) != 3 {
			t.Errorf("expected exactly voiceEnabled, voiceMode and agentId, got %v", body)
		}
		if body["voiceEnabled"] != true {
			t.Errorf("expected voiceEnabled true, got %v", body["voiceEnabled"])
		}
		if body["voiceMode"] != "agent" {
			t.Errorf("expected voiceMode 'agent', got %v", body["voiceMode"])
		}
		if body["agentId"] != "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b" {
			t.Errorf("expected agentId, got %v", body["agentId"])
		}
		for _, key := range []string{"voiceAgentId", "voice_agent_id", "agent_id", "voice_enabled", "voice_mode"} {
			if _, present := body[key]; present {
				t.Errorf("unexpected wire key %q in body", key)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(voiceNumberAgentJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	enabled := true
	agentID := "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b"
	number, err := client.Voice.Numbers.Update(context.Background(), "+15555550188", &UpdateVoiceNumberRequest{
		VoiceEnabled: &enabled,
		VoiceMode:    VoiceModeAgent,
		AgentID:      &agentID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if number.VoiceMode != VoiceModeAgent {
		t.Errorf("expected VoiceMode agent, got %q", number.VoiceMode)
	}
}

func TestVoiceNumbersUpdate_ClearsAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeRequestBody(t, r)
		if len(body) != 2 {
			t.Errorf("expected exactly voiceMode and agentId, got %v", body)
		}
		if body["voiceMode"] != "ring_dashboard" {
			t.Errorf("expected voiceMode 'ring_dashboard', got %v", body["voiceMode"])
		}
		if value, present := body["agentId"]; !present || value != "" {
			t.Errorf("expected agentId to be sent as an empty string, got %v", body["agentId"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(voiceNumberAgentJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	noAgent := ""
	if _, err := client.Voice.Numbers.Update(context.Background(), "+15555550188", &UpdateVoiceNumberRequest{
		VoiceMode: VoiceModeRingDashboard,
		AgentID:   &noAgent,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVoiceNumbersUpdate_DisableSendsFalseOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if key := r.Header.Get("Idempotency-Key"); key != "number-off-1" {
			t.Errorf("expected caller-supplied Idempotency-Key, got '%s'", key)
		}
		body := decodeRequestBody(t, r)
		if len(body) != 1 {
			t.Errorf("expected only voiceEnabled, got %v", body)
		}
		if value, present := body["voiceEnabled"]; !present || value != false {
			t.Errorf("expected voiceEnabled false, got %v", body["voiceEnabled"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(voiceNumberOffJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	disabled := false
	number, err := client.Voice.Numbers.Update(context.Background(), "7a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d", &UpdateVoiceNumberRequest{
		VoiceEnabled: &disabled,
	}, WithIdempotencyKey("number-off-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if number.VoiceEnabled || number.VoiceMode != VoiceModeNone {
		t.Errorf("unexpected VoiceEnabled/VoiceMode %v/%q", number.VoiceEnabled, number.VoiceMode)
	}
}

func TestVoiceNumbersRegisterEmergencyAddress_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.RequestURI != "/voice/numbers/%2B15555550188/emergency-address" {
			t.Errorf("unexpected path '%s'", r.RequestURI)
		}
		if key := r.Header.Get("Idempotency-Key"); !autoKeyPattern.MatchString(key) {
			t.Errorf("expected an auto-generated Idempotency-Key, got '%s'", key)
		}
		body := decodeRequestBody(t, r)
		expected := map[string]string{
			"street":  "500 Example Ave",
			"unit":    "Suite 2",
			"city":    "Austin",
			"state":   "TX",
			"zip":     "78701",
			"country": "US",
		}
		for key, want := range expected {
			if body[key] != want {
				t.Errorf("expected %s=%q, got %v", key, want, body[key])
			}
		}
		if len(body) != len(expected) {
			t.Errorf("unexpected body keys: %v", body)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(voiceNumberAgentJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	number, err := client.Voice.Numbers.RegisterEmergencyAddress(context.Background(), "+15555550188", &EmergencyAddress{
		Street:  "500 Example Ave",
		Unit:    "Suite 2",
		City:    "Austin",
		State:   "TX",
		Zip:     "78701",
		Country: "US",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if number.EmergencyAddress == nil || number.EmergencyAddress.Status != "active" {
		t.Errorf("unexpected EmergencyAddress %+v", number.EmergencyAddress)
	}
}

func TestVoiceNumbersRegisterEmergencyAddress_OmitsUnsetOptionalKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeRequestBody(t, r)
		if len(body) != 4 {
			t.Errorf("expected exactly street, city, state and zip, got %v", body)
		}
		for _, key := range []string{"unit", "country"} {
			if _, present := body[key]; present {
				t.Errorf("expected %q to be omitted when unset", key)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(voiceNumberAgentJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.Voice.Numbers.RegisterEmergencyAddress(context.Background(), "+15555550188", &EmergencyAddress{
		Street: "500 Example Ave",
		City:   "Austin",
		State:  "TX",
		Zip:    "78701",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVoiceNumbersRegisterEmergencyAddress_UnvalidatedAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error":"invalid_address","message":"We couldn't validate that address.","suggested":{"street":"500 Example Avenue","city":"Austin","state":"TX","zip":"78701","country":"US"}}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.Voice.Numbers.RegisterEmergencyAddress(context.Background(), "+15555550188", &EmergencyAddress{
		Street: "500 Example Ave",
		City:   "Austin",
		State:  "TX",
		Zip:    "78701",
	})
	if !IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	validationErr := err.(*ValidationError)
	if validationErr.Code != VoiceErrorCodeInvalidAddress {
		t.Errorf("expected Code invalid_address, got %q", validationErr.Code)
	}
	var suggested EmergencyAddress
	if err := json.Unmarshal(validationErr.Extra["suggested"], &suggested); err != nil {
		t.Fatalf("expected a suggested address in Extra, got %v (%v)", validationErr.Extra, err)
	}
	if suggested != (EmergencyAddress{Street: "500 Example Avenue", City: "Austin", State: "TX", Zip: "78701", Country: "US"}) {
		t.Errorf("unexpected suggested address %+v", suggested)
	}
}

func TestVoiceNumbersRegisterEmergencyAddress_ServerErrorIsNotRetried(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"carrier_refused","message":"The address couldn't be registered. Try again in a moment.","suggested":null}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.Voice.Numbers.RegisterEmergencyAddress(context.Background(), "+15555550188", &EmergencyAddress{
		Street: "500 Example Ave",
		City:   "Austin",
		State:  "TX",
		Zip:    "78701",
	})
	sendlyErr, ok := err.(*SendlyError)
	if !ok {
		t.Fatalf("expected *SendlyError, got %T", err)
	}
	if sendlyErr.StatusCode != http.StatusBadGateway || sendlyErr.Code != VoiceErrorCodeCarrierRefused {
		t.Errorf("unexpected error %d/%q", sendlyErr.StatusCode, sendlyErr.Code)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("expected exactly 1 request (no automatic retry of a 5xx), got %d", got)
	}
}

func TestVoiceAgentsCreate_WireBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/voice/agents" {
			t.Errorf("expected path '/voice/agents', got '%s'", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); !autoKeyPattern.MatchString(key) {
			t.Errorf("expected an auto-generated Idempotency-Key, got '%s'", key)
		}
		body := decodeRequestBody(t, r)
		expected := map[string]interface{}{
			"name":         "Front desk",
			"enabled":      false,
			"voice":        "ashley",
			"language":     "en-US",
			"greeting":     "Thanks for calling Acme, how can I help?",
			"instructions": "Answer questions about opening hours.",
		}
		for key, want := range expected {
			if body[key] != want {
				t.Errorf("expected %s=%v, got %v", key, want, body[key])
			}
		}
		tools, ok := body["tools"].(map[string]interface{})
		if !ok || tools["sendSms"] != true || tools["transferTo"] != "+15555550190" || len(tools) != 2 {
			t.Errorf("unexpected tools %v", body["tools"])
		}
		if len(body) != len(expected)+1 {
			t.Errorf("unexpected body keys: %v", body)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(voiceAgentJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	enabled := false
	sendSms := true
	transferTo := "+15555550190"
	agent, err := client.Voice.Agents.Create(context.Background(), &CreateVoiceAgentRequest{
		Name:         "Front desk",
		Enabled:      &enabled,
		Voice:        "ashley",
		Language:     "en-US",
		Greeting:     "Thanks for calling Acme, how can I help?",
		Instructions: "Answer questions about opening hours.",
		Tools:        &VoiceAgentToolsRequest{SendSms: &sendSms, TransferTo: &transferTo},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID != "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b" || agent.Object != "voice_agent" {
		t.Errorf("unexpected ID/Object %q/%q", agent.ID, agent.Object)
	}
	if agent.Name != "Front desk" || !agent.Enabled || agent.Voice != "ashley" || agent.VoiceLabel != "Ashley (US, warm)" {
		t.Errorf("unexpected agent %+v", agent)
	}
	if agent.Language != "en-US" || agent.Greeting == "" || agent.Instructions == "" {
		t.Errorf("unexpected Language/Greeting/Instructions %+v", agent)
	}
	if !agent.Tools.SendSms || agent.Tools.TransferTo == nil || *agent.Tools.TransferTo != "+15555550190" {
		t.Errorf("unexpected Tools %+v", agent.Tools)
	}
	if !agent.CanSendSms || agent.CallsHandled != 12 || agent.AvgDurationSecs != 74 {
		t.Errorf("unexpected CanSendSms/CallsHandled/AvgDurationSecs %+v", agent)
	}
	if agent.CreatedAt != "2026-09-14T17:00:00.000Z" || agent.UpdatedAt != "2026-09-14T17:05:00.000Z" {
		t.Errorf("unexpected timestamps %q/%q", agent.CreatedAt, agent.UpdatedAt)
	}
}

func TestVoiceAgentsCreate_MinimalBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeRequestBody(t, r)
		if len(body) != 1 || body["name"] != "Front desk" {
			t.Errorf("expected only name, got %v", body)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(voiceAgentJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.Voice.Agents.Create(context.Background(), &CreateVoiceAgentRequest{Name: "Front desk"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVoiceAgentsCreate_AgentLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":"agent_limit","message":"You've reached the agent limit for this workspace."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.Voice.Agents.Create(context.Background(), &CreateVoiceAgentRequest{Name: "Front desk"})
	sendlyErr, ok := err.(*SendlyError)
	if !ok {
		t.Fatalf("expected *SendlyError, got %T", err)
	}
	if sendlyErr.StatusCode != http.StatusConflict || sendlyErr.Code != VoiceErrorCodeAgentLimit {
		t.Errorf("unexpected error %d/%q", sendlyErr.StatusCode, sendlyErr.Code)
	}
}

func TestVoiceAgentsList_UnwrapsData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/voice/agents" {
			t.Errorf("expected path '/voice/agents', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": [` + voiceAgentJSON + `]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.Voice.Agents.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Name != "Front desk" {
		t.Errorf("unexpected agents %+v", resp.Data)
	}
}

func TestVoiceAgentsGet_EscapesID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.RequestURI != "/voice/agents/agent%2Fwith%20space%2B1" {
			t.Errorf("expected escaped path, got '%s'", r.RequestURI)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(voiceAgentJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	if _, err := client.Voice.Agents.Get(context.Background(), "agent/with space+1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVoiceAgentsUpdate_SendsOnlySetFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH request, got %s", r.Method)
		}
		if r.URL.Path != "/voice/agents/3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b" {
			t.Errorf("unexpected path '%s'", r.URL.Path)
		}
		body := decodeRequestBody(t, r)
		if len(body) != 2 {
			t.Errorf("expected exactly greeting and tools, got %v", body)
		}
		if body["greeting"] != "Thanks for calling Acme. This call may be recorded." {
			t.Errorf("unexpected greeting %v", body["greeting"])
		}
		tools, ok := body["tools"].(map[string]interface{})
		if !ok || len(tools) != 1 {
			t.Fatalf("expected tools with only transferTo, got %v", body["tools"])
		}
		if value, present := tools["transferTo"]; !present || value != "" {
			t.Errorf("expected transferTo to be sent as an empty string, got %v", tools["transferTo"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(voiceAgentJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	greeting := "Thanks for calling Acme. This call may be recorded."
	noTransfer := ""
	if _, err := client.Voice.Agents.Update(context.Background(), "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b", &UpdateVoiceAgentRequest{
		Greeting: &greeting,
		Tools:    &VoiceAgentToolsRequest{TransferTo: &noTransfer},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVoiceAgentsUpdate_ClearsGreetingAndInstructions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeRequestBody(t, r)
		if len(body) != 2 {
			t.Errorf("expected exactly greeting and instructions, got %v", body)
		}
		for _, key := range []string{"greeting", "instructions"} {
			if value, present := body[key]; !present || value != "" {
				t.Errorf("expected %s to be sent as an empty string, got %v", key, body[key])
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(voiceAgentJSON))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	empty := ""
	if _, err := client.Voice.Agents.Update(context.Background(), "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b", &UpdateVoiceAgentRequest{
		Greeting:     &empty,
		Instructions: &empty,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVoiceAgentsDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/voice/agents/3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b" {
			t.Errorf("unexpected path '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b","object":"voice_agent","deleted":true}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	deleted, err := client.Voice.Agents.Delete(context.Background(), "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted.ID != "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b" || deleted.Object != "voice_agent" || !deleted.Deleted {
		t.Errorf("unexpected response %+v", deleted)
	}
}

func TestVoiceAgentsDelete_AgentInUse(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":"agent_in_use","message":"This agent answers 1 number. Point it elsewhere first.","numbers":["+15555550188"]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.Voice.Agents.Delete(context.Background(), "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	sendlyErr, ok := err.(*SendlyError)
	if !ok {
		t.Fatalf("expected *SendlyError, got %T", err)
	}
	if sendlyErr.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", sendlyErr.StatusCode)
	}
	if sendlyErr.Code != VoiceErrorCodeAgentInUse {
		t.Errorf("expected Code agent_in_use, got %q", sendlyErr.Code)
	}
	if sendlyErr.Message != "This agent answers 1 number. Point it elsewhere first." {
		t.Errorf("expected the API message, got %q", sendlyErr.Message)
	}
	var numbers []string
	if err := json.Unmarshal(sendlyErr.Extra["numbers"], &numbers); err != nil {
		t.Fatalf("expected numbers in Extra, got %v (%v)", sendlyErr.Extra, err)
	}
	if len(numbers) != 1 || numbers[0] != "+15555550188" {
		t.Errorf("unexpected numbers %v", numbers)
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Errorf("expected exactly 1 request (no retries on 409), got %d", got)
	}
}

func TestVoicesList_UnwrapsData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/voice/voices" {
			t.Errorf("expected path '/voice/voices', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[{"id":"ashley","label":"Ashley (US, warm)","language":"en"},{"id":"diego","label":"Diego (Spanish, MX)","language":"es"}]}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	resp, err := client.Voice.Voices.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 voices, got %d", len(resp.Data))
	}
	if resp.Data[0] != (Voice{ID: "ashley", Label: "Ashley (US, warm)", Language: "en"}) {
		t.Errorf("unexpected voice %+v", resp.Data[0])
	}
	if resp.Data[1].ID != "diego" || resp.Data[1].Language != "es" {
		t.Errorf("unexpected voice %+v", resp.Data[1])
	}
}

func TestVoice_VoiceNotEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"voice_not_enabled","message":"Voice is not enabled for your account."}`))
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	_, err := client.Voice.Agents.List(context.Background())
	if !IsNotFoundError(err) {
		t.Fatalf("expected NotFoundError, got %T", err)
	}
	if err.(*NotFoundError).Code != CallErrorCodeVoiceNotEnabled {
		t.Errorf("expected Code voice_not_enabled, got %q", err.(*NotFoundError).Code)
	}
}

func TestVoice_ClientSideValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should reach the server on a client-side validation failure")
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))
	ctx := context.Background()
	enabled := true
	address := func(mutate func(*EmergencyAddress)) *EmergencyAddress {
		a := &EmergencyAddress{Street: "500 Example Ave", City: "Austin", State: "TX", Zip: "78701"}
		mutate(a)
		return a
	}

	checks := map[string]error{}
	_, checks["numbers get without number"] = client.Voice.Numbers.Get(ctx, "")
	_, checks["numbers update without number"] = client.Voice.Numbers.Update(ctx, "", &UpdateVoiceNumberRequest{VoiceEnabled: &enabled})
	_, checks["numbers update without request"] = client.Voice.Numbers.Update(ctx, "+15555550188", nil)
	_, checks["emergency address without number"] = client.Voice.Numbers.RegisterEmergencyAddress(ctx, "", address(func(*EmergencyAddress) {}))
	_, checks["emergency address without request"] = client.Voice.Numbers.RegisterEmergencyAddress(ctx, "+15555550188", nil)
	_, checks["emergency address without street"] = client.Voice.Numbers.RegisterEmergencyAddress(ctx, "+15555550188", address(func(a *EmergencyAddress) { a.Street = "" }))
	_, checks["emergency address without city"] = client.Voice.Numbers.RegisterEmergencyAddress(ctx, "+15555550188", address(func(a *EmergencyAddress) { a.City = "" }))
	_, checks["emergency address without state"] = client.Voice.Numbers.RegisterEmergencyAddress(ctx, "+15555550188", address(func(a *EmergencyAddress) { a.State = "" }))
	_, checks["emergency address without zip"] = client.Voice.Numbers.RegisterEmergencyAddress(ctx, "+15555550188", address(func(a *EmergencyAddress) { a.Zip = "" }))
	_, checks["agents create without request"] = client.Voice.Agents.Create(ctx, nil)
	_, checks["agents create without name"] = client.Voice.Agents.Create(ctx, &CreateVoiceAgentRequest{Voice: "ashley"})
	_, checks["agents get without id"] = client.Voice.Agents.Get(ctx, "")
	_, checks["agents update without id"] = client.Voice.Agents.Update(ctx, "", &UpdateVoiceAgentRequest{Name: "Front desk"})
	_, checks["agents update without request"] = client.Voice.Agents.Update(ctx, "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b", nil)
	_, checks["agents delete without id"] = client.Voice.Agents.Delete(ctx, "")

	for name, err := range checks {
		if !IsValidationError(err) {
			t.Errorf("%s: expected ValidationError, got %T (%v)", name, err, err)
		}
	}
}
