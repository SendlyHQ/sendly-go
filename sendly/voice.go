package sendly

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
)

// VoiceService configures voice from code: which numbers take phone calls and
// how they answer, the emergency address each number needs before it can
// place calls, and the AI agents that talk to callers. Place and follow calls
// with CallsService.
//
// Reads need the calls:read scope. Writes need calls:write and a live API
// key (a test key gets a 403 with Code CallErrorCodeLiveKeyRequired). In a
// team workspace, changing a number or its emergency address also needs a
// role with settings:write, and managing agents needs api_keys:write because
// each agent holds its own scoped sending key; other roles get a 403 with
// Code "forbidden". While voice is not enabled for the workspace every
// method returns a *NotFoundError with Code CallErrorCodeVoiceNotEnabled.
type VoiceService struct {
	client *Client
	// Numbers switches voice on or off for the workspace's numbers, sets how
	// they answer and registers their emergency addresses.
	Numbers *VoiceNumbersService
	// Agents creates and manages the AI agents that answer and place calls.
	Agents *VoiceAgentsService
	// Voices lists the voices an agent can speak with.
	Voices *VoicesService
}

// VoiceNumbersService reads and changes the voice settings of the
// workspace's numbers. A number is addressed by its ID or by its phone
// number in E.164 format.
type VoiceNumbersService struct {
	client *Client
}

// VoiceAgentsService creates and manages the workspace's AI agents. An agent
// answers real callers on every number pointed at it (VoiceModeAgent) and
// handles the calls placed with CallsService.Create. A workspace can have up
// to 20 agents.
type VoiceAgentsService struct {
	client *Client
}

// VoicesService lists the voices an agent can speak with.
type VoicesService struct {
	client *Client
}

// VoiceMode is how a number answers phone calls, serialized as voiceMode.
type VoiceMode string

const (
	// VoiceModeNone means voice is off and the number doesn't take calls.
	VoiceModeNone VoiceMode = "none"
	// VoiceModeRingDashboard means calls ring teammates in the dashboard.
	VoiceModeRingDashboard VoiceMode = "ring_dashboard"
	// VoiceModeAgent means an AI agent answers.
	VoiceModeAgent VoiceMode = "agent"
)

// VoiceAgentKind is who answers an agent's calls, serialized as kind. The
// type is a plain string, so a kind added later decodes as its own value.
type VoiceAgentKind string

const (
	// VoiceAgentKindSendly means Sendly writes the replies from the agent's
	// instructions. It is the default.
	VoiceAgentKindSendly VoiceAgentKind = "sendly"
	// VoiceAgentKindCustomLLM means Sendly listens and speaks, and your chat
	// completions endpoint writes each reply.
	VoiceAgentKindCustomLLM VoiceAgentKind = "custom_llm"
	// VoiceAgentKindSip means Sendly connects the caller to your agent's SIP
	// address. A sip agent answers incoming calls only.
	VoiceAgentKindSip VoiceAgentKind = "sip"
	// VoiceAgentKindWebsocket means Sendly streams the call audio to your
	// wss:// server and plays back what it sends. A websocket agent answers
	// incoming calls only.
	VoiceAgentKindWebsocket VoiceAgentKind = "websocket"
)

// VoiceAgentSipTransport is how a sip agent's call is carried, serialized as
// transport. A value added later decodes as its own value.
type VoiceAgentSipTransport string

const (
	VoiceAgentSipTransportAuto VoiceAgentSipTransport = "auto"
	VoiceAgentSipTransportUDP  VoiceAgentSipTransport = "udp"
	VoiceAgentSipTransportTCP  VoiceAgentSipTransport = "tcp"
	VoiceAgentSipTransportTLS  VoiceAgentSipTransport = "tls"
)

// VoiceAgentSipMediaEncryption is whether a sip agent's call audio is
// encrypted, serialized as mediaEncryption. A value added later decodes as
// its own value.
type VoiceAgentSipMediaEncryption string

const (
	VoiceAgentSipMediaEncryptionDisable VoiceAgentSipMediaEncryption = "disable"
	VoiceAgentSipMediaEncryptionAllow   VoiceAgentSipMediaEncryption = "allow"
	VoiceAgentSipMediaEncryptionRequire VoiceAgentSipMediaEncryption = "require"
)

// VoiceAgentAudioFormat is the audio format a websocket agent's server
// sends and receives, serialized as audioFormat. A value added later decodes
// as its own value.
type VoiceAgentAudioFormat string

const (
	// VoiceAgentAudioFormatPcm16At16000 is 16-bit PCM at 16 kHz, the default.
	VoiceAgentAudioFormatPcm16At16000 VoiceAgentAudioFormat = "pcm16_16000"
	// VoiceAgentAudioFormatPcm16At24000 is 16-bit PCM at 24 kHz.
	VoiceAgentAudioFormatPcm16At24000 VoiceAgentAudioFormat = "pcm16_24000"
	// VoiceAgentAudioFormatPcm16At8000 is 16-bit PCM at 8 kHz.
	VoiceAgentAudioFormatPcm16At8000 VoiceAgentAudioFormat = "pcm16_8000"
	// VoiceAgentAudioFormatMulawAt8000 is mu-law at 8 kHz.
	VoiceAgentAudioFormatMulawAt8000 VoiceAgentAudioFormat = "mulaw_8000"
)

// Error codes returned by the voice configuration endpoints, readable from
// the Code field of the typed error. They can also return
// CallErrorCodeVoiceNotEnabled, CallErrorCodeAgentNotFound,
// CallErrorCodeAgentDisabled, CallErrorCodeLiveKeyRequired,
// CallErrorCodeVoiceUnavailable and CallErrorCodeInternal, plus
// "invalid_request" (400, *ValidationError) when a field has the wrong type.
const (
	// VoiceErrorCodeNumberNotFound (404, *NotFoundError): the ID or phone
	// number isn't an active number in this workspace.
	VoiceErrorCodeNumberNotFound = "number_not_found"
	// VoiceErrorCodeAgentRequired (400, *ValidationError): VoiceModeAgent
	// needs an AgentID, sent with the update or already stored on the number.
	VoiceErrorCodeAgentRequired = "agent_required"
	// VoiceErrorCodeAgentInUse (409, *SendlyError): the agent still answers
	// one or more numbers, listed as E.164 strings in Extra["numbers"]. Point
	// them at another agent or back to the dashboard before deleting it.
	VoiceErrorCodeAgentInUse = "agent_in_use"
	// VoiceErrorCodeAgentLimit (409, *SendlyError): the workspace already has
	// the maximum of 20 agents.
	VoiceErrorCodeAgentLimit = "agent_limit"
	// VoiceErrorCodeInvalidVoiceMode (400, *ValidationError): VoiceMode isn't
	// none, ring_dashboard or agent.
	VoiceErrorCodeInvalidVoiceMode = "invalid_voice_mode"
	// VoiceErrorCodeInvalidAddress (*ValidationError): 400 when a required
	// field is missing, the state isn't a two-letter code, the ZIP or postal
	// code is malformed, or the country isn't US or CA; 422 when the address
	// couldn't be validated as a real place, with a corrected address in
	// Extra["suggested"] (an EmergencyAddress, or null when none was found).
	VoiceErrorCodeInvalidAddress = "invalid_address"
	// VoiceErrorCodeE911NotApplicable (400, *ValidationError): emergency
	// registration applies to US and Canadian numbers only.
	VoiceErrorCodeE911NotApplicable = "e911_not_applicable"
	// VoiceErrorCodeAttachFailed (503, *SendlyError): voice couldn't be
	// switched on for the number. Try again in a moment.
	VoiceErrorCodeAttachFailed = "voice_attach_failed"
	// VoiceErrorCodeCarrierRefused (503, *SendlyError): the emergency address
	// couldn't be registered. It isn't retried automatically, because every
	// attempt registers the address anew; retry once after a pause, unless the
	// message says the number couldn't be found for emergency registration:
	// contact support.
	VoiceErrorCodeCarrierRefused = "carrier_refused"
	// VoiceErrorCodeByoAgentsNotEnabled (403, *SendlyError): Kind is one of
	// your own (or Connection or RotateSigningSecret was sent) and the
	// workspace's bring-your-own-agent feature is off. Support switches it on.
	VoiceErrorCodeByoAgentsNotEnabled = "byo_agents_not_enabled"
)

// EmergencyAddress is the street address emergency services are sent to
// when someone dials 911 from a number. US and Canadian addresses only.
type EmergencyAddress struct {
	// Street is the street address, for example "500 Example Ave". Required.
	Street string `json:"street"`
	// Unit is the suite, apartment or floor. Optional.
	Unit string `json:"unit,omitempty"`
	// City is the city. Required.
	City string `json:"city"`
	// State is the two-letter state or province code, for example "TX". Required.
	State string `json:"state"`
	// Zip is the five-digit ZIP code, or a Canadian postal code like "A1A 1A1". Required.
	Zip string `json:"zip"`
	// Country is "US" or "CA". Defaults to "US" when registering.
	Country string `json:"country,omitempty"`
}

// VoiceNumberEmergencyAddress is the emergency address registered for a number.
type VoiceNumberEmergencyAddress struct {
	// Status is where the registration stands, for example "provisioning" or
	// "active". Any other value means the registration did not go through;
	// register the address again.
	Status string `json:"status"`
	// Address is the registered address, nil if none is stored.
	Address *EmergencyAddress `json:"address"`
}

// VoiceNumberRates are a number's per-minute call prices in credits.
type VoiceNumberRates struct {
	// Inbound is the rate for an inbound call a teammate answers.
	Inbound int `json:"inbound"`
	// Outbound is the rate for an outbound call.
	Outbound int `json:"outbound"`
	// Agent is the rate for an inbound call an AI agent answers.
	Agent int `json:"agent"`
}

// VoiceNumber is one of the workspace's numbers with its voice settings.
type VoiceNumber struct {
	// ID is the number's unique identifier.
	ID string `json:"id"`
	// Object is always "voice_number".
	Object string `json:"object"`
	// PhoneNumber is the number in E.164 format.
	PhoneNumber string `json:"phoneNumber"`
	// PhoneNumberType is the kind of number, for example "local" or "toll_free".
	PhoneNumberType *string `json:"phoneNumberType"`
	// CountryCode is the number's two-letter country code.
	CountryCode *string `json:"countryCode"`
	// IsDefault is true for the workspace's default number.
	IsDefault bool `json:"isDefault"`
	// VoiceEnabled is true when the number takes and places phone calls.
	VoiceEnabled bool `json:"voiceEnabled"`
	// VoiceMode is how the number answers; always VoiceModeNone when
	// VoiceEnabled is false.
	VoiceMode VoiceMode `json:"voiceMode"`
	// AgentID is the agent that answers when VoiceMode is VoiceModeAgent. In
	// other modes it is whatever agent was last stored, or nil.
	AgentID *string `json:"agentId"`
	// EmergencyAddress is the registered emergency address, nil when none
	// was ever registered.
	EmergencyAddress *VoiceNumberEmergencyAddress `json:"emergencyAddress"`
	// RatePerMinute is what calls on this number cost, in credits a minute.
	RatePerMinute VoiceNumberRates `json:"ratePerMinute"`
}

// VoiceNumberListResponse is the response from VoiceNumbersService.List.
type VoiceNumberListResponse struct {
	// Data contains the workspace's active numbers, the default number first.
	Data []VoiceNumber `json:"data"`
}

// UpdateVoiceNumberRequest is the body for VoiceNumbersService.Update. Only
// the fields you set are sent.
type UpdateVoiceNumberRequest struct {
	// VoiceEnabled switches voice on or off and, when set, wins over
	// VoiceMode: false switches voice off whatever the mode, and true with
	// VoiceModeNone, or switching it on without a VoiceMode, makes the number
	// ring the dashboard.
	VoiceEnabled *bool `json:"voiceEnabled,omitempty"`
	// VoiceMode sets how the number answers. Sent without VoiceEnabled,
	// VoiceModeRingDashboard or VoiceModeAgent switches voice on and
	// VoiceModeNone switches it off.
	VoiceMode VoiceMode `json:"voiceMode,omitempty"`
	// AgentID is the agent that answers in VoiceModeAgent. The agent must be
	// in the workspace and switched on. A pointer to an empty string clears
	// the stored agent.
	AgentID *string `json:"agentId,omitempty"`
}

// VoiceAgentTools are what an agent may do beyond talking.
type VoiceAgentTools struct {
	// SendSms is true when the agent may text the person it is talking to.
	SendSms bool `json:"sendSms"`
	// TransferTo is the number, in E.164 format, the agent transfers a
	// caller to when asked for a person, and the number CallsService.Transfer
	// uses when no To is given. Nil when the agent cannot transfer.
	TransferTo *string `json:"transferTo"`
}

// VoiceAgentToolsRequest sets an agent's tools on create or update. Only
// the fields you set are sent; on update the rest keep their stored values.
type VoiceAgentToolsRequest struct {
	// SendSms lets the agent text the person it is talking to. A new agent
	// may text unless this is set to false.
	SendSms *bool `json:"sendSms,omitempty"`
	// TransferTo is a phone number in E.164 format to store with the agent;
	// a pointer to an empty string clears it.
	TransferTo *string `json:"transferTo,omitempty"`
}

// VoiceAgent is an AI agent that answers and places phone calls.
type VoiceAgent struct {
	// ID is the agent's unique identifier.
	ID string `json:"id"`
	// Object is always "voice_agent".
	Object string `json:"object"`
	// Name is the agent's name.
	Name string `json:"name"`
	// Enabled is false when the agent is switched off; a switched-off agent
	// can't answer numbers or take calls.
	Enabled bool `json:"enabled"`
	// Voice is the ID of the voice the agent speaks with (see VoicesService.List).
	Voice string `json:"voice"`
	// VoiceLabel is the voice's display name, for example "Ashley (US, warm)".
	VoiceLabel string `json:"voiceLabel"`
	// Language is the language tag the agent speaks, for example "en-US".
	Language string `json:"language"`
	// Greeting is the first thing the agent says; empty when unset.
	Greeting string `json:"greeting"`
	// Instructions tell the agent what it is for and how to behave; empty when unset.
	Instructions string `json:"instructions"`
	// Kind is who answers the agent's calls; see VoiceAgentKind.
	Kind VoiceAgentKind `json:"kind"`
	// Connection is how Sendly reaches your own agent, nil for a
	// VoiceAgentKindSendly agent. Secrets are never returned.
	Connection *VoiceAgentConnection `json:"connection"`
	// HasSigningSecret is true when the agent has a signing secret. Your own
	// agents do: Sendly signs every request to them with it, in an
	// X-Sendly-Signature header of the form t=<unix seconds>,v1=<hex
	// HMAC-SHA256>.
	HasSigningSecret bool `json:"hasSigningSecret"`
	// SigningSecret is the agent's signing secret, set only on the response
	// that made it: Create of one of your own agents, Update that switches a
	// sendly agent to one of your own, and Update with RotateSigningSecret.
	// Save it now; it is never shown again.
	SigningSecret string `json:"signingSecret,omitempty"`
	// Tools are what the agent may do beyond talking.
	Tools VoiceAgentTools `json:"tools"`
	// CanSendSms is true when the agent holds its own scoped sending key. It
	// texts only when Tools.SendSms is also on.
	CanSendSms bool `json:"canSendSms"`
	// CallsHandled is the number of calls the agent has handled.
	CallsHandled int `json:"callsHandled"`
	// AvgDurationSecs is the average length of those calls in seconds.
	AvgDurationSecs int `json:"avgDurationSecs"`
	// CreatedAt is when the agent was created (ISO 8601).
	CreatedAt string `json:"createdAt"`
	// UpdatedAt is when the agent was last changed (ISO 8601).
	UpdatedAt string `json:"updatedAt"`
}

// VoiceAgentConnection is how Sendly reaches one of your own agents. Only
// the fields of the agent's Kind are set; secrets are never returned, only
// whether one is saved.
type VoiceAgentConnection struct {
	// BaseURL is your chat completions endpoint (VoiceAgentKindCustomLLM).
	// Sendly sends POST {BaseURL}/chat/completions with streaming on.
	BaseURL string `json:"baseUrl,omitempty"`
	// Model is the model name sent in each request (VoiceAgentKindCustomLLM).
	Model string `json:"model,omitempty"`
	// HeaderNames are the names of the extra headers sent with each request
	// (VoiceAgentKindCustomLLM). Their values are never returned.
	HeaderNames []string `json:"headerNames,omitempty"`
	// HasAPIKey is true when an API key is saved (VoiceAgentKindCustomLLM).
	HasAPIKey bool `json:"hasApiKey,omitempty"`

	// URI is your agent's SIP address, such as
	// "sip:+15125550100@sip.example.com" (VoiceAgentKindSip).
	URI string `json:"uri,omitempty"`
	// Transport is how the call is carried (VoiceAgentKindSip).
	Transport VoiceAgentSipTransport `json:"transport,omitempty"`
	// MediaEncryption is whether the call audio is encrypted (VoiceAgentKindSip).
	MediaEncryption VoiceAgentSipMediaEncryption `json:"mediaEncryption,omitempty"`
	// Username is the digest auth username, nil when none (VoiceAgentKindSip).
	Username *string `json:"username,omitempty"`
	// HasPassword is true when a digest auth password is saved (VoiceAgentKindSip).
	HasPassword bool `json:"hasPassword,omitempty"`
	// Headers are the extra X- headers sent on the call (VoiceAgentKindSip).
	Headers map[string]string `json:"headers,omitempty"`

	// URL is your wss:// server (VoiceAgentKindWebsocket).
	URL string `json:"url,omitempty"`
	// AudioFormat is the audio format both ways (VoiceAgentKindWebsocket).
	AudioFormat VoiceAgentAudioFormat `json:"audioFormat,omitempty"`
	// HasAuthToken is true when an auth token is saved (VoiceAgentKindWebsocket).
	HasAuthToken bool `json:"hasAuthToken,omitempty"`
	// CustomParameters are sent to your server in the start message
	// (VoiceAgentKindWebsocket).
	CustomParameters map[string]string `json:"customParameters,omitempty"`
}

// VoiceAgentConnectionRequest sets how Sendly reaches one of your own agents
// on create or update. Set the fields of the agent's Kind; only the fields
// you set are sent. On an update that keeps the same Kind, the connection is
// merged into the saved one, so fields left out keep their values. Hosts
// must resolve to public internet addresses; a field that breaks a rule is
// refused with a *ValidationError whose Message reads
// "connection.<field>: <reason>".
//
// A secret (APIKey, Password, AuthToken) left nil keeps the saved one, and a
// pointer to an empty string removes it (it is sent as null). A non-nil
// Headers or CustomParameters map replaces the saved set, so an empty map
// removes them.
type VoiceAgentConnectionRequest struct {
	// BaseURL is your chat completions endpoint (VoiceAgentKindCustomLLM):
	// an https:// URL of up to 2048 characters, with no credentials or #. A
	// trailing / is dropped. Required when the agent becomes custom_llm.
	BaseURL string
	// Model is the model name sent in each request (VoiceAgentKindCustomLLM):
	// 1 to 200 letters, digits and _ . : / @ + -.
	Model string
	// APIKey is sent as Authorization: Bearer (VoiceAgentKindCustomLLM),
	// 1 to 2048 printable ASCII characters.
	APIKey *string

	// URI is your agent's SIP address (VoiceAgentKindSip): sip: or sips:
	// with a user part and a hostname (not an IP address), such as
	// "sip:+15125550100@sip.example.com". Only ;transport= may follow.
	URI string
	// Transport defaults to VoiceAgentSipTransportTCP; a sips: address always
	// uses TLS (VoiceAgentKindSip).
	Transport VoiceAgentSipTransport
	// MediaEncryption defaults to VoiceAgentSipMediaEncryptionAllow;
	// VoiceAgentSipMediaEncryptionRequire needs the TLS transport
	// (VoiceAgentKindSip).
	MediaEncryption VoiceAgentSipMediaEncryption
	// Username is the digest auth username (VoiceAgentKindSip). A pointer to
	// an empty string removes it.
	Username *string
	// Password is the digest auth password (VoiceAgentKindSip).
	Password *string

	// Headers are extra headers with secret values. For
	// VoiceAgentKindCustomLLM, up to 10 sent with each request; Authorization,
	// Content-Type, Content-Length, Host, Connection, Transfer-Encoding,
	// User-Agent and X-Sendly-* can't be set. For VoiceAgentKindSip, up to 10
	// named X-... (not X-Sendly-...) sent on the call.
	Headers map[string]string

	// URL is your server (VoiceAgentKindWebsocket): a wss:// URL of up to
	// 2048 characters, with no credentials or #.
	URL string
	// AudioFormat defaults to VoiceAgentAudioFormatPcm16At16000
	// (VoiceAgentKindWebsocket).
	AudioFormat VoiceAgentAudioFormat
	// AuthToken is sent as Authorization: Bearer (VoiceAgentKindWebsocket).
	AuthToken *string
	// CustomParameters are up to 20 values sent to your server in the start
	// message (VoiceAgentKindWebsocket). Keys are 1-40 letters, digits and
	// _ . : -; values up to 500 characters.
	CustomParameters map[string]string
}

// MarshalJSON sends only the fields that are set, a secret set to an empty
// string as null so it is removed, and an empty map as {} so it replaces the
// saved set.
func (r VoiceAgentConnectionRequest) MarshalJSON() ([]byte, error) {
	body := struct {
		BaseURL          string                       `json:"baseUrl,omitempty"`
		Model            string                       `json:"model,omitempty"`
		APIKey           json.RawMessage              `json:"apiKey,omitempty"`
		URI              string                       `json:"uri,omitempty"`
		Transport        VoiceAgentSipTransport       `json:"transport,omitempty"`
		MediaEncryption  VoiceAgentSipMediaEncryption `json:"mediaEncryption,omitempty"`
		Username         *string                      `json:"username,omitempty"`
		Password         json.RawMessage              `json:"password,omitempty"`
		Headers          *map[string]string           `json:"headers,omitempty"`
		URL              string                       `json:"url,omitempty"`
		AudioFormat      VoiceAgentAudioFormat        `json:"audioFormat,omitempty"`
		AuthToken        json.RawMessage              `json:"authToken,omitempty"`
		CustomParameters *map[string]string           `json:"customParameters,omitempty"`
	}{
		BaseURL:         r.BaseURL,
		Model:           r.Model,
		URI:             r.URI,
		Transport:       r.Transport,
		MediaEncryption: r.MediaEncryption,
		Username:        r.Username,
		URL:             r.URL,
		AudioFormat:     r.AudioFormat,
	}
	var err error
	if body.APIKey, err = connectionSecretJSON(r.APIKey); err != nil {
		return nil, err
	}
	if body.Password, err = connectionSecretJSON(r.Password); err != nil {
		return nil, err
	}
	if body.AuthToken, err = connectionSecretJSON(r.AuthToken); err != nil {
		return nil, err
	}
	if r.Headers != nil {
		body.Headers = &r.Headers
	}
	if r.CustomParameters != nil {
		body.CustomParameters = &r.CustomParameters
	}
	return json.Marshal(body)
}

func connectionSecretJSON(value *string) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	if *value == "" {
		return json.RawMessage("null"), nil
	}
	return json.Marshal(*value)
}

// VoiceAgentListResponse is the response from VoiceAgentsService.List.
type VoiceAgentListResponse struct {
	// Data contains the workspace's agents.
	Data []VoiceAgent `json:"data"`
}

// CreateVoiceAgentRequest is the body for VoiceAgentsService.Create.
type CreateVoiceAgentRequest struct {
	// Name is the agent's name, up to 80 characters. Required.
	Name string `json:"name"`
	// Enabled defaults to true.
	Enabled *bool `json:"enabled,omitempty"`
	// Voice is a voice ID from VoicesService.List. An empty or unknown ID
	// uses the default voice.
	Voice string `json:"voice,omitempty"`
	// Language is the language tag the agent speaks. Defaults to "en-US".
	Language string `json:"language,omitempty"`
	// Greeting is the first thing the agent says, up to 500 characters.
	Greeting string `json:"greeting,omitempty"`
	// Instructions tell the agent what it is for and how to behave, up to
	// 4000 characters.
	Instructions string `json:"instructions,omitempty"`
	// Tools sets what the agent may do beyond talking. Texting is on by
	// default: leave Tools nil and the agent may text the people it talks
	// to. Set Tools.SendSms to false to stop it.
	Tools *VoiceAgentToolsRequest `json:"tools,omitempty"`
	// Kind is who answers the agent's calls. Empty means
	// VoiceAgentKindSendly. Your own kinds need the workspace's
	// bring-your-own-agent feature (VoiceErrorCodeByoAgentsNotEnabled).
	Kind VoiceAgentKind `json:"kind,omitempty"`
	// Connection is how Sendly reaches your own agent, matching Kind.
	// Required when Kind is one of your own.
	Connection *VoiceAgentConnectionRequest `json:"connection,omitempty"`
}

// UpdateVoiceAgentRequest is the body for VoiceAgentsService.Update. Only
// the fields you set change: empty Name, Voice and Language and nil pointers
// are left out of the request.
type UpdateVoiceAgentRequest struct {
	// Name is the agent's name, up to 80 characters.
	Name string `json:"name,omitempty"`
	// Enabled switches the agent on or off.
	Enabled *bool `json:"enabled,omitempty"`
	// Voice is a voice ID from VoicesService.List. An unknown ID uses the
	// default voice.
	Voice string `json:"voice,omitempty"`
	// Language is the language tag the agent speaks.
	Language string `json:"language,omitempty"`
	// Greeting is the first thing the agent says, up to 500 characters. A
	// pointer to an empty string removes it.
	Greeting *string `json:"greeting,omitempty"`
	// Instructions tell the agent what it is for and how to behave, up to
	// 4000 characters. A pointer to an empty string removes them.
	Instructions *string `json:"instructions,omitempty"`
	// Tools changes the tools you set and keeps the rest.
	Tools *VoiceAgentToolsRequest `json:"tools,omitempty"`
	// Kind switches who answers the agent's calls. Switching to one of your
	// own needs Connection; switching to VoiceAgentKindSendly removes the
	// connection and the signing secret.
	Kind VoiceAgentKind `json:"kind,omitempty"`
	// Connection changes how Sendly reaches your own agent. With the same
	// Kind it is merged into the saved connection.
	Connection *VoiceAgentConnectionRequest `json:"connection,omitempty"`
	// RotateSigningSecret makes a new signing secret for one of your own
	// agents; the response carries it once in SigningSecret. The old secret
	// stops working at once. Only true is sent. Refused with a
	// *ValidationError on a VoiceAgentKindSendly agent.
	RotateSigningSecret bool `json:"rotateSigningSecret,omitempty"`
}

// TestVoiceAgentRequest is the body for VoiceAgentsService.Test. Leave it nil
// to test the saved connection; only the fields you set are sent.
type TestVoiceAgentRequest struct {
	// Kind is the kind of the settings in Connection. Empty means the agent's
	// own kind. Needed to test settings of another kind, such as on a
	// VoiceAgentKindSendly agent before switching it.
	Kind VoiceAgentKind `json:"kind,omitempty"`
	// Connection is settings to test instead of the saved ones. Nothing is
	// saved. For the agent's own kind, a secret left out is taken from the
	// saved connection, while the host stays the same.
	Connection *VoiceAgentConnectionRequest `json:"connection,omitempty"`
	// Instructions is the system prompt to send (VoiceAgentKindCustomLLM),
	// up to 4000 characters. Empty means the agent's instructions.
	Instructions string `json:"instructions,omitempty"`
}

// VoiceAgentTestOutcome is what a connection test found, serialized as
// outcome. A value added later decodes as its own value.
type VoiceAgentTestOutcome string

const (
	// VoiceAgentTestOutcomeOK means the agent answered the way a call needs.
	VoiceAgentTestOutcomeOK VoiceAgentTestOutcome = "ok"
	// VoiceAgentTestOutcomeBlockedDestination means the host isn't a public
	// address.
	VoiceAgentTestOutcomeBlockedDestination VoiceAgentTestOutcome = "blocked_destination"
	VoiceAgentTestOutcomeUnreachable        VoiceAgentTestOutcome = "unreachable"
	VoiceAgentTestOutcomeTimeout            VoiceAgentTestOutcome = "timeout"
	VoiceAgentTestOutcomeTLSFailed          VoiceAgentTestOutcome = "tls_failed"
	// VoiceAgentTestOutcomeAuthRequired means the SIP server wants
	// credentials the agent doesn't have.
	VoiceAgentTestOutcomeAuthRequired VoiceAgentTestOutcome = "auth_required"
	// VoiceAgentTestOutcomeAuthFailed means the key, token or credentials
	// were refused.
	VoiceAgentTestOutcomeAuthFailed VoiceAgentTestOutcome = "auth_failed"
	// VoiceAgentTestOutcomeNotFound means a wrong path or SIP user.
	VoiceAgentTestOutcomeNotFound VoiceAgentTestOutcome = "not_found"
	// VoiceAgentTestOutcomeRejected means any other refusal.
	VoiceAgentTestOutcomeRejected VoiceAgentTestOutcome = "rejected"
	// VoiceAgentTestOutcomeBadReply means an answer outside the protocol.
	VoiceAgentTestOutcomeBadReply VoiceAgentTestOutcome = "bad_reply"
)

// VoiceAgentTestStepName is one step of a connection test, serialized as
// step. A value added later decodes as its own value.
type VoiceAgentTestStepName string

const (
	// VoiceAgentTestStepAddress means the host resolved to a public address.
	VoiceAgentTestStepAddress VoiceAgentTestStepName = "address"
	// VoiceAgentTestStepConnect means a connection opened.
	VoiceAgentTestStepConnect VoiceAgentTestStepName = "connect"
	// VoiceAgentTestStepLogin means the SIP server accepted the credentials.
	VoiceAgentTestStepLogin VoiceAgentTestStepName = "login"
	// VoiceAgentTestStepReply means the agent answered as it should.
	VoiceAgentTestStepReply VoiceAgentTestStepName = "reply"
)

// VoiceAgentTestStep is one step of a connection test.
type VoiceAgentTestStep struct {
	// Step is which step this is; see VoiceAgentTestStepName.
	Step VoiceAgentTestStepName `json:"step"`
	// OK is whether the step worked.
	OK bool `json:"ok"`
	// Detail is what happened at this step, in plain words.
	Detail string `json:"detail"`
}

// VoiceAgentTest is the response from VoiceAgentsService.Test. A failed test
// is still a successful request: check OK.
type VoiceAgentTest struct {
	// OK is whether the agent answered the way a call needs.
	OK bool `json:"ok"`
	// Outcome is VoiceAgentTestOutcomeOK, or why not.
	Outcome VoiceAgentTestOutcome `json:"outcome"`
	// Message is one or two sentences to show a person: what happened and
	// what to change.
	Message string `json:"message"`
	// Status is the HTTP or SIP status the agent answered with, or nil.
	Status *int `json:"status"`
	// LatencyMs is how long the agent took to answer, or nil.
	LatencyMs *int `json:"latencyMs"`
	// Signed is whether the request carried X-Sendly-Signature, as calls do.
	// Always false for VoiceAgentKindSip.
	Signed bool `json:"signed"`
	// Steps are each step tried, in order, up to the first that failed.
	Steps []VoiceAgentTestStep `json:"steps"`
}

// DeletedVoiceAgent is the response from VoiceAgentsService.Delete.
type DeletedVoiceAgent struct {
	// ID is the deleted agent's ID.
	ID string `json:"id"`
	// Object is always "voice_agent".
	Object string `json:"object"`
	// Deleted is always true.
	Deleted bool `json:"deleted"`
}

// Voice is a voice an agent can speak with.
type Voice struct {
	// ID is what to pass as the agent's Voice.
	ID string `json:"id"`
	// Label is the voice's display name, for example "Ashley (US, warm)".
	Label string `json:"label"`
	// Language is the language the voice speaks, for example "en".
	Language string `json:"language"`
}

// VoiceListResponse is the response from VoicesService.List.
type VoiceListResponse struct {
	// Data contains the available voices.
	Data []Voice `json:"data"`
}

func voicePathSegment(value string) string {
	return strings.ReplaceAll(url.PathEscape(value), "+", "%2B")
}

// List returns the workspace's active numbers with their voice settings,
// emergency addresses and per-minute rates, the default number first.
func (s *VoiceNumbersService) List(ctx context.Context) (*VoiceNumberListResponse, error) {
	var resp VoiceNumberListResponse
	if err := s.client.request(ctx, "GET", "/voice/numbers", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get returns one number's voice settings. number is the number's ID or its
// phone number in E.164 format. Returns a *NotFoundError (Code
// VoiceErrorCodeNumberNotFound) when it isn't an active number in this
// workspace.
func (s *VoiceNumbersService) Get(ctx context.Context, number string) (*VoiceNumber, error) {
	if number == "" {
		return nil, &ValidationError{APIError: APIError{Message: "number is required"}}
	}

	var resp VoiceNumber
	if err := s.client.request(ctx, "GET", "/voice/numbers/"+voicePathSegment(number), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update switches voice on or off for a number and sets how it answers, and
// returns the number. number is the number's ID or its phone number in E.164
// format. This changes how real phone calls to the number are answered.
//
// A mode alone is enough: VoiceModeRingDashboard or VoiceModeAgent without
// VoiceEnabled switches voice on, and VoiceModeNone without it switches voice
// off. VoiceEnabled false still wins and sets the mode to VoiceModeNone.
// VoiceModeAgent needs a switched-on agent, sent as AgentID or already stored
// on the number. Requires a live API key and the calls:write scope (and
// settings:write in a team workspace). Switching voice on, a mode alone
// included, can be refused with VoiceErrorCodeAgentRequired,
// CallErrorCodeAgentNotFound, CallErrorCodeAgentDisabled,
// CallErrorCodeVoiceUnavailable or VoiceErrorCodeAttachFailed; an unknown
// mode is VoiceErrorCodeInvalidVoiceMode. Pass WithIdempotencyKey to make the
// update replay-safe.
func (s *VoiceNumbersService) Update(ctx context.Context, number string, req *UpdateVoiceNumberRequest, opts ...RequestOption) (*VoiceNumber, error) {
	if number == "" {
		return nil, &ValidationError{APIError: APIError{Message: "number is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	var resp VoiceNumber
	if err := s.client.request(ctx, "PATCH", "/voice/numbers/"+voicePathSegment(number), req, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RegisterEmergencyAddress registers the address emergency services are sent
// to when someone dials 911 from the number, and returns the number. An
// emergency address is required before a number can place calls in the US
// and Canada, and costs 1.50 USD a month; registering a new address for a
// number that already has one replaces it at no extra charge. Requires a
// live API key and the calls:write scope (and settings:write in a team
// workspace). An Idempotency-Key is sent automatically.
//
// A malformed address fails with a *ValidationError (400, Code
// VoiceErrorCodeInvalidAddress), and so does one that couldn't be validated
// (422), with any corrected address in Extra["suggested"].
// VoiceErrorCodeE911NotApplicable means the number isn't a US or Canadian
// number. A 5xx, such as VoiceErrorCodeCarrierRefused, is returned without
// the automatic retries other requests get, because every attempt registers
// the address anew.
func (s *VoiceNumbersService) RegisterEmergencyAddress(ctx context.Context, number string, req *EmergencyAddress, opts ...RequestOption) (*VoiceNumber, error) {
	if number == "" {
		return nil, &ValidationError{APIError: APIError{Message: "number is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.Street == "" {
		return nil, &ValidationError{APIError: APIError{Message: "street is required"}}
	}
	if req.City == "" {
		return nil, &ValidationError{APIError: APIError{Message: "city is required"}}
	}
	if req.State == "" {
		return nil, &ValidationError{APIError: APIError{Message: "state is required"}}
	}
	if req.Zip == "" {
		return nil, &ValidationError{APIError: APIError{Message: "zip is required"}}
	}

	var resp VoiceNumber
	opts = append([]RequestOption{withoutServerErrorRetries()}, opts...)
	if err := s.client.request(ctx, "POST", "/voice/numbers/"+voicePathSegment(number)+"/emergency-address", req, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List returns the workspace's agents with how many calls each has handled.
func (s *VoiceAgentsService) List(ctx context.Context) (*VoiceAgentListResponse, error) {
	var resp VoiceAgentListResponse
	if err := s.client.request(ctx, "GET", "/voice/agents", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Create adds an agent that answers real callers once a number points at it.
// Only Name is required; the agent starts switched on, with the default voice
// and "en-US", and may text callers: it gets its own scoped sending key, and
// Tools.SendSms is on unless you set it to false. Requires a live API key and
// the calls:write scope (and api_keys:write in a team workspace). Returns a
// *SendlyError (409, Code VoiceErrorCodeAgentLimit) at 20 agents. An
// Idempotency-Key is sent automatically.
//
// Set Kind and Connection to bring your own agent. The response's
// SigningSecret is the only time the agent's signing secret is shown.
// Returns a *SendlyError (403, Code VoiceErrorCodeByoAgentsNotEnabled) when
// the workspace's bring-your-own-agent feature is off.
func (s *VoiceAgentsService) Create(ctx context.Context, req *CreateVoiceAgentRequest, opts ...RequestOption) (*VoiceAgent, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.Name == "" {
		return nil, &ValidationError{APIError: APIError{Message: "name is required"}}
	}

	var resp VoiceAgent
	if err := s.client.request(ctx, "POST", "/voice/agents", req, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get returns one agent. Returns a *NotFoundError (Code
// CallErrorCodeAgentNotFound) when the agent is not in this workspace.
func (s *VoiceAgentsService) Get(ctx context.Context, id string) (*VoiceAgent, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "agent id is required"}}
	}

	var resp VoiceAgent
	if err := s.client.request(ctx, "GET", "/voice/agents/"+voicePathSegment(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update changes the fields you set and returns the agent. Changes apply to
// the next call the agent handles. Requires a live API key and the
// calls:write scope (and api_keys:write in a team workspace). Pass
// WithIdempotencyKey to make the update replay-safe.
//
// The response's SigningSecret is set when the update made one: switching a
// sendly agent to one of your own, or RotateSigningSecret. Save it then; it
// is never shown again.
func (s *VoiceAgentsService) Update(ctx context.Context, id string, req *UpdateVoiceAgentRequest, opts ...RequestOption) (*VoiceAgent, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "agent id is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	var resp VoiceAgent
	if err := s.client.request(ctx, "PATCH", "/voice/agents/"+voicePathSegment(id), req, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Test reaches one of your own agents the way a call would, without placing
// one, and reports what happened. Requires a live API key and the
// calls:write scope.
//
// A VoiceAgentKindCustomLLM agent gets a signed, streamed chat completion
// with a test message. A VoiceAgentKindWebsocket agent gets a signed
// connection with connected, a start carrying test: true, half a second of
// silence and a stop with reason test_finished. A VoiceAgentKindSip agent
// gets an OPTIONS request, answered with the saved credentials if the server
// asks; a SIP server answering shows the address and credentials work, not
// which agent picks up.
//
// Pass req with Connection to test settings before saving them, or nil to
// test the saved ones. A failed test is still a successful call: check the
// response's OK. A workspace can run 10 tests a minute; the 11th returns a
// *RateLimitError. Returns a *ValidationError (400) when the agent is a
// VoiceAgentKindSendly agent and no Kind was sent, a *SendlyError (403, Code
// VoiceErrorCodeByoAgentsNotEnabled) when the workspace's
// bring-your-own-agent feature is off, and a *NotFoundError (Code
// CallErrorCodeAgentNotFound) when the agent is not in this workspace. No
// Idempotency-Key is sent.
func (s *VoiceAgentsService) Test(ctx context.Context, id string, req *TestVoiceAgentRequest) (*VoiceAgentTest, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "agent id is required"}}
	}
	if req == nil {
		req = &TestVoiceAgentRequest{}
	}

	var resp VoiceAgentTest
	if err := s.client.request(ctx, "POST", "/voice/agents/"+voicePathSegment(id)+"/test", req, &resp, withoutAutoIdempotencyKey()); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Delete removes an agent and revokes its sending key. An agent that still
// answers a number can't be removed: the call fails with a *SendlyError (409,
// Code VoiceErrorCodeAgentInUse) until those numbers point at another agent
// or back to the dashboard. Requires a live API key and the calls:write scope
// (and api_keys:write in a team workspace).
func (s *VoiceAgentsService) Delete(ctx context.Context, id string, opts ...RequestOption) (*DeletedVoiceAgent, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "agent id is required"}}
	}

	var resp DeletedVoiceAgent
	if err := s.client.request(ctx, "DELETE", "/voice/agents/"+voicePathSegment(id), nil, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List returns the voices an agent can speak with.
func (s *VoicesService) List(ctx context.Context) (*VoiceListResponse, error) {
	var resp VoiceListResponse
	if err := s.client.request(ctx, "GET", "/voice/voices", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
