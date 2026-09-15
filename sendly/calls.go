package sendly

import (
	"context"
	"net/url"
	"strconv"
)

// CallsService places, lists, inspects and ends phone calls.
//
// A call placed over the API is answered by one of the workspace's AI agents
// (created with VoiceService.Agents or in the dashboard under Calls → Agents);
// the agent talks to the person who picks up. The number you call from must
// have voice switched on (VoiceService.Numbers or the dashboard) and, for
// outbound calls, a registered emergency address.
// Calls are billed per started minute from the workspace's prepaid credits
// (an agent-handled outbound call costs 10 credits a minute), and only to
// US and Canadian numbers.
//
// Reads need the calls:read scope; Create and Hangup need calls:write and a
// live API key. While voice is not enabled for the workspace every method
// returns a *NotFoundError with Code CallErrorCodeVoiceNotEnabled.
type CallsService struct {
	client *Client
}

// CallKind is what kind of call this is, serialized as kind.
type CallKind string

const (
	// CallKindPstn is a phone call to or from a phone number.
	CallKindPstn CallKind = "pstn"
	// CallKindInternal is a browser-to-browser call between teammates.
	CallKindInternal CallKind = "internal"
)

// CallDirection is who placed the call, serialized as direction.
type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)

// CallStatus is where a call stands, serialized as status. Ringing and
// Active are live; every other value is terminal.
type CallStatus string

const (
	// CallStatusRinging means the far end has not answered yet.
	CallStatusRinging CallStatus = "ringing"
	// CallStatusActive means the call is connected.
	CallStatusActive CallStatus = "active"
	// CallStatusCompleted means the call was answered and has ended.
	CallStatusCompleted CallStatus = "completed"
	// CallStatusNoAnswer means nobody picked up before the ring deadline.
	CallStatusNoAnswer CallStatus = "no_answer"
	// CallStatusBusy means the far end was busy.
	CallStatusBusy CallStatus = "busy"
	// CallStatusCancelled means the caller hung up before it was answered.
	CallStatusCancelled CallStatus = "cancelled"
	// CallStatusDeclined means the far end declined the call.
	CallStatusDeclined CallStatus = "declined"
	// CallStatusFailed means the call could not be set up or was cut short by a fault.
	CallStatusFailed CallStatus = "failed"
	// CallStatusSuspended can appear on an internal call whose media dropped and may recover.
	CallStatusSuspended CallStatus = "suspended"
)

// CallHandledBy is who answers the call, serialized as handledBy.
type CallHandledBy string

const (
	// CallHandledByAgent means an AI agent is on the call.
	CallHandledByAgent CallHandledBy = "agent"
	// CallHandledByDashboard means a teammate answers in the dashboard.
	CallHandledByDashboard CallHandledBy = "dashboard"
)

// CallBilling is the billing state of a call, serialized as billing.
type CallBilling string

const (
	// CallBillingMetered means a phone call is in progress and charged per minute.
	CallBillingMetered CallBilling = "metered"
	// CallBillingSettled means the call has ended and its charge is final.
	CallBillingSettled CallBilling = "settled"
	// CallBillingUnbilled means the call was never charged (internal calls).
	CallBillingUnbilled CallBilling = "unbilled"
)

// CallRecordingStatus is the state of a call's recording. On a Call it is
// nil while there is nothing to report; on CallRecording the value
// CallRecordingStatusNone says the call has no recording at all.
type CallRecordingStatus string

const (
	// CallRecordingStatusNone means there is no recording for this call
	// (recording is off, or the call was never answered). Recording
	// endpoint only.
	CallRecordingStatusNone CallRecordingStatus = "none"
	// CallRecordingStatusRecording means the call is being recorded.
	CallRecordingStatusRecording CallRecordingStatus = "recording"
	// CallRecordingStatusReady means the recording can be downloaded.
	CallRecordingStatusReady CallRecordingStatus = "ready"
	// CallRecordingStatusFailed means the recording could not be produced.
	CallRecordingStatusFailed CallRecordingStatus = "failed"
)

// Error codes returned by the calls endpoints, readable from the Code field
// of the typed error (for example err.(*SendlyError).Code).
const (
	// CallErrorCodeVoiceNotEnabled (404, *NotFoundError): voice isn't
	// enabled for the workspace yet. Every calls endpoint answers this
	// while it is off.
	CallErrorCodeVoiceNotEnabled = "voice_not_enabled"
	// CallErrorCodeOutboundNotEnabled (404, *NotFoundError): calls to phone
	// numbers aren't enabled for the workspace yet.
	CallErrorCodeOutboundNotEnabled = "outbound_calls_not_enabled"
	// CallErrorCodeVoiceUnavailable (503, *SendlyError): outbound calling
	// isn't available right now.
	CallErrorCodeVoiceUnavailable = "voice_unavailable"
	// CallErrorCodeAgentsUnavailable (503, *SendlyError): AI agents aren't
	// switched on for this deployment yet.
	CallErrorCodeAgentsUnavailable = "agents_unavailable"
	// CallErrorCodeAgentRequired (400, *ValidationError): calls placed over
	// the API are answered by an AI agent, so AgentID is required.
	CallErrorCodeAgentRequired = "agent_required"
	// CallErrorCodeAgentNotFound (404, *NotFoundError): no agent with that ID
	// in this workspace.
	CallErrorCodeAgentNotFound = "agent_not_found"
	// CallErrorCodeAgentDisabled (409, *SendlyError): the agent is switched
	// off; switch it on before calling it.
	CallErrorCodeAgentDisabled = "agent_disabled"
	// CallErrorCodeInvalidMetadata (400, *ValidationError): Metadata has more
	// than 20 keys, a key outside 1-40 chars of [A-Za-z0-9_.:-], or a value
	// over 500 chars.
	CallErrorCodeInvalidMetadata = "invalid_metadata"
	// CallErrorCodeFromNumberRequired (400, *ValidationError): the workspace
	// has more than one voice-enabled number, so From must be given.
	CallErrorCodeFromNumberRequired = "from_number_required"
	// CallErrorCodeNoVoiceNumber (409, *SendlyError): no number in the
	// workspace has voice enabled.
	CallErrorCodeNoVoiceNumber = "no_voice_number"
	// CallErrorCodeNumberNotFound (404, *NotFoundError): From isn't a number
	// in this workspace.
	CallErrorCodeNumberNotFound = "number_not_found"
	// CallErrorCodeInvalidNumber (400, *ValidationError): To isn't a valid
	// phone number, or equals From.
	CallErrorCodeInvalidNumber = "invalid_number"
	// CallErrorCodeDestinationNotSupported (400, *ValidationError): calls can
	// only be placed to US and Canadian numbers.
	CallErrorCodeDestinationNotSupported = "destination_not_supported"
	// CallErrorCodeE911Required (428, *SendlyError): register an emergency
	// address for the From number (VoiceNumbersService.RegisterEmergencyAddress
	// or the dashboard) before placing calls.
	CallErrorCodeE911Required = "e911_required"
	// CallErrorCodeInsufficientCredits (402, *InsufficientCreditsError): the
	// balance doesn't cover the first minute.
	CallErrorCodeInsufficientCredits = "insufficient_credits"
	// CallErrorCodeLinesBusy (409, *SendlyError): every line is in use; try
	// again in a moment.
	CallErrorCodeLinesBusy = "lines_busy"
	// CallErrorCodeDailyCallLimit (429, *RateLimitError): today's calling
	// limit has been reached.
	CallErrorCodeDailyCallLimit = "daily_call_limit"
	// CallErrorCodeCallNotFound (404, *NotFoundError): no call with that ID
	// in this workspace.
	CallErrorCodeCallNotFound = "call_not_found"
	// CallErrorCodeLiveKeyRequired (403, *SendlyError): Create and Hangup
	// need a live API key.
	CallErrorCodeLiveKeyRequired = "live_key_required"
	// CallErrorCodeInternal (500, *SendlyError): something went wrong on Sendly's side.
	CallErrorCodeInternal = "voice_internal_error"
)

// Call is a phone call (or a browser-to-browser call between teammates).
type Call struct {
	// ID is the call's unique identifier.
	ID string `json:"id"`
	// Object is always "call".
	Object string `json:"object"`
	// Kind is "pstn" for a phone call, "internal" for a teammate call.
	Kind CallKind `json:"kind"`
	// Direction is "inbound" or "outbound".
	Direction CallDirection `json:"direction"`
	// Status is where the call stands; see CallStatus.
	Status CallStatus `json:"status"`
	// HandledBy is who answers: an AI agent or a teammate in the dashboard.
	HandledBy CallHandledBy `json:"handledBy"`
	// AgentID is the AI agent on the call, nil when a teammate handles it.
	AgentID *string `json:"agentId"`
	// From is the calling number in E.164 format, nil on internal calls.
	From *string `json:"from"`
	// To is the called number in E.164 format, nil on internal calls.
	To *string `json:"to"`
	// CallerName is a display name for the calling side, when known.
	CallerName *string `json:"callerName"`
	// CalleeName is a display name for the called side, when known.
	CalleeName *string `json:"calleeName"`
	// StartedAt is when the call started ringing (ISO 8601).
	StartedAt string `json:"startedAt"`
	// AnsweredAt is when the call was answered (ISO 8601), nil until then.
	AnsweredAt *string `json:"answeredAt"`
	// EndedAt is when the call ended (ISO 8601), nil while it is live.
	EndedAt *string `json:"endedAt"`
	// DurationSecs is the answered duration in seconds, 0 until the call ends.
	DurationSecs int `json:"durationSecs"`
	// CreditsCharged is the credits charged so far (final once Billing is settled).
	CreditsCharged int `json:"creditsCharged"`
	// Billing is the billing state; see CallBilling.
	Billing CallBilling `json:"billing"`
	// HangupClass says why the call ended, nil while it is live. Values are
	// grouped as normal endings (normal, caller_hung_up, callee_hung_up,
	// caller_left, peer_left, agent_ended, agent_agent_hangup,
	// agent_caller_left), never connected (ring_timeout, callee_declined,
	// callee_busy, caller_cancelled, room_closed_unanswered,
	// agent_left_unanswered, agent_caller_never_joined, invalid_number,
	// destination_rejected), cut short by the platform (max_duration,
	// credits_exhausted, media_aborted, peer_connection_lost, room_closed,
	// agent_left) and setup problems (setup_failed, agent_dispatch_failed,
	// agent_api_unreachable, agent_already_ended). Anything unrecognised is
	// reported as "ended".
	HangupClass *string `json:"hangupClass"`
	// RecordingStatus is the recording's state, nil when there is nothing to report.
	RecordingStatus *CallRecordingStatus `json:"recordingStatus"`
	// Metadata is the string map attached on Create; empty when none.
	Metadata map[string]string `json:"metadata"`
	// Transcript is what was said, in order. Only present on Get, and only
	// for agent-handled calls (empty when nothing was said).
	Transcript []CallTranscriptLine `json:"transcript,omitempty"`
}

// CallTranscriptLine is one utterance in an agent-handled call's transcript.
type CallTranscriptLine struct {
	// Speaker is "caller" or "agent".
	Speaker string `json:"speaker"`
	// Text is what was said.
	Text string `json:"text"`
	// AtMs is the offset from the start of the call, in milliseconds.
	AtMs int `json:"atMs"`
}

// CreateCallRequest is the body for CallsService.Create.
type CreateCallRequest struct {
	// To is the number to call, in E.164 format. US and Canada only.
	To string `json:"to"`
	// AgentID is the AI agent that handles the call. Required.
	AgentID string `json:"agentId"`
	// From is a voice-enabled number in the workspace to call from. Optional
	// when the workspace has exactly one voice-enabled number.
	From string `json:"from,omitempty"`
	// Context is appended to the agent's instructions for this call only
	// (up to 2000 chars). It is not echoed back.
	Context string `json:"context,omitempty"`
	// Metadata is up to 20 string pairs stored with the call and echoed on
	// every read and in every call.* webhook. Keys are 1-40 chars of
	// [A-Za-z0-9_.:-]; values up to 500 chars.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ListCallsRequest filters CallsService.List. Every field is optional.
type ListCallsRequest struct {
	// Limit is the maximum number of calls to return (1-100, default 50).
	Limit int
	// Offset is the number of calls to skip for pagination (default 0).
	Offset int
	// Status keeps calls in one status.
	Status CallStatus
	// Direction keeps inbound or outbound calls.
	Direction CallDirection
	// Kind keeps phone or internal calls.
	Kind CallKind
	// AgentID keeps calls handled by one agent.
	AgentID string
	// To keeps calls to one number (E.164, exact match).
	To string
	// From keeps calls from one number (E.164, exact match).
	From string
}

// CallPagination contains pagination info for call lists.
type CallPagination struct {
	// Total is the number of calls matching the filters.
	Total int `json:"total"`
	// Limit is the maximum number of calls returned.
	Limit int `json:"limit"`
	// Offset is the number of calls skipped.
	Offset int `json:"offset"`
	// HasMore indicates if more calls are available.
	HasMore bool `json:"hasMore"`
}

// CallListResponse is the response from listing calls.
type CallListResponse struct {
	// Data contains the calls, newest first.
	Data []Call `json:"data"`
	// Pagination contains pagination info.
	Pagination CallPagination `json:"pagination"`
}

// CallRecording is the response from CallsService.Recording.
type CallRecording struct {
	// CallID is the call the recording belongs to.
	CallID string `json:"callId"`
	// Status is the recording's state; URL is only set when it is
	// CallRecordingStatusReady.
	Status CallRecordingStatus `json:"status"`
	// URL is a signed download link, valid for five minutes. Nil unless ready.
	URL *string `json:"url"`
	// ExpiresAt is when URL stops working (ISO 8601). Nil unless ready.
	ExpiresAt *string `json:"expiresAt"`
	// ContentType is "audio/ogg" when ready, nil otherwise. Recordings are
	// Ogg/Opus; agent calls are recorded dual-channel, with the agent on the
	// left channel and the other party on the right.
	ContentType *string `json:"contentType"`
}

// WebhookCallData is the data.object of a call.started, call.completed or
// call.recording.ready event: the Call in snake_case. Decode it with
// WebhookEvent.DecodeObject.
type WebhookCallData struct {
	ID              string               `json:"id"`
	Object          string               `json:"object"`
	Kind            CallKind             `json:"kind"`
	Direction       CallDirection        `json:"direction"`
	Status          CallStatus           `json:"status"`
	HandledBy       CallHandledBy        `json:"handled_by"`
	AgentID         *string              `json:"agent_id"`
	From            *string              `json:"from"`
	To              *string              `json:"to"`
	CallerName      *string              `json:"caller_name"`
	CalleeName      *string              `json:"callee_name"`
	StartedAt       string               `json:"started_at"`
	AnsweredAt      *string              `json:"answered_at"`
	EndedAt         *string              `json:"ended_at"`
	DurationSecs    int                  `json:"duration_secs"`
	CreditsCharged  int                  `json:"credits_charged"`
	Billing         CallBilling          `json:"billing"`
	HangupClass     *string              `json:"hangup_class"`
	RecordingStatus *CallRecordingStatus `json:"recording_status"`
	Metadata        map[string]string    `json:"metadata"`
	OrganizationID  *string              `json:"organization_id,omitempty"`
}

// Create places a phone call that one of the workspace's AI agents handles.
// The call comes back ringing; poll Get or subscribe to call.started and
// call.completed to follow it. Requires a live API key and the calls:write
// scope. An Idempotency-Key is sent automatically (see WithIdempotencyKey to
// supply your own).
//
// Refusals carry a Code from the CallErrorCode* constants, for example
// CallErrorCodeE911Required (428) when the From number has no emergency
// address, or an *InsufficientCreditsError when the balance doesn't cover
// the first minute.
func (s *CallsService) Create(ctx context.Context, req *CreateCallRequest, opts ...RequestOption) (*Call, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.To == "" {
		return nil, &ValidationError{APIError: APIError{Message: "to is required"}}
	}
	if req.AgentID == "" {
		return nil, &ValidationError{APIError: APIError{Message: "agentId is required"}}
	}

	var resp Call
	if err := s.client.request(ctx, "POST", "/calls", req, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List returns the workspace's calls, newest first, with pagination. Live
// calls are reconciled before being returned, so a ring past its deadline
// shows as no_answer.
func (s *CallsService) List(ctx context.Context, req *ListCallsRequest) (*CallListResponse, error) {
	params := make(map[string]string)

	if req != nil {
		if req.Limit > 0 {
			params["limit"] = strconv.Itoa(req.Limit)
		}
		if req.Offset > 0 {
			params["offset"] = strconv.Itoa(req.Offset)
		}
		params["status"] = string(req.Status)
		params["direction"] = string(req.Direction)
		params["kind"] = string(req.Kind)
		params["agentId"] = req.AgentID
		params["to"] = req.To
		params["from"] = req.From
	}

	path := "/calls" + buildQueryString(params)

	var resp CallListResponse
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get returns one call. Agent-handled calls include their Transcript.
// Returns a *NotFoundError (Code CallErrorCodeCallNotFound) when the call is
// not in this workspace.
func (s *CallsService) Get(ctx context.Context, id string) (*Call, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "call id is required"}}
	}

	var resp Call
	if err := s.client.request(ctx, "GET", "/calls/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Hangup ends a call and returns it. A ringing call becomes cancelled
// (HangupClass caller_cancelled), an active one completed (HangupClass
// normal); a call that already ended is returned unchanged. Requires a live
// API key and the calls:write scope.
func (s *CallsService) Hangup(ctx context.Context, id string, opts ...RequestOption) (*Call, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "call id is required"}}
	}

	var resp Call
	if err := s.client.request(ctx, "POST", "/calls/"+url.PathEscape(id)+"/hangup", map[string]interface{}{}, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Recording returns the call's recording status and, once it is ready, a
// signed download URL valid for five minutes. Fetch it again for a fresh URL
// after ExpiresAt.
func (s *CallsService) Recording(ctx context.Context, id string) (*CallRecording, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "call id is required"}}
	}

	var resp CallRecording
	if err := s.client.request(ctx, "GET", "/calls/"+url.PathEscape(id)+"/recording", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
