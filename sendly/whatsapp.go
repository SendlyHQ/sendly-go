package sendly

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// WhatsAppService connects numbers to WhatsApp and manages the channel:
// start a signup, list senders, manage Meta-reviewed message templates, and
// check 24-hour customer-service windows. Send WhatsApp messages with
// MessagesService.SendWhatsApp.
//
// Connecting a number is a one-time $19 setup (no monthly fee). The first
// number always ends with a human step: Signup.Create returns a connect URL a
// person must open in a browser and complete with a Facebook login to link
// their WhatsApp Business Account. Further numbers can join an account the
// workspace already connected without that step: Signup.CreateWithOptions
// with BusinessAccountID has WhatsApp send the number a code, which
// Signup.Verify submits.
//
// Scopes and keys: sends go through MessagesService.SendWhatsApp and need
// sms:send, not whatsapp:write, and a live key. Reads (signup status,
// templates, the window, senders, sender profiles and their ice breakers and
// commands) need whatsapp:read and accept test keys. Signup (including
// Verify and Resend), template create/edit/delete, and sender edits (the
// profile and its photo, ice breakers and commands, and calling) need
// whatsapp:write and a live key (otherwise 403 whatsapp_requires_live_key).
// In a team workspace, connecting and sender edits need an owner or admin
// (settings:write), and template writes need an owner, admin or member
// (templates:write); a missing role returns 403 insufficient_permissions.
//
// WhatsApp is enabled per person: the user who owns the API key, not the
// workspace. While it is off, sends return 403 whatsapp_not_enabled and every
// method on this service gets 404 not_found.
//
// Pricing: free-form text or media inside the 24-hour window costs 1 credit
// each for the first 1,000 per sending number per calendar month (UTC), then
// the destination's utility template price; countries without a listed price
// use the default utility price of 12 credits. Templates are priced by
// category and destination country; countries without a listed price use 33
// (marketing), 12 (utility) and 12 (authentication) credits. A failed send
// gives its slot back.
type WhatsAppService struct {
	client *Client
	// Signup connects numbers to WhatsApp.
	Signup *WhatsAppSignupService
	// Senders lists the numbers connected (or connecting) to WhatsApp.
	Senders *WhatsAppSendersService
	// Templates manages Meta-reviewed message templates.
	Templates *WhatsAppTemplatesService
}

// WhatsAppSignupService connects numbers to WhatsApp.
type WhatsAppSignupService struct {
	client *Client
}

// WhatsAppSendersService lists the numbers connected (or connecting) to
// WhatsApp and manages their business profiles and photos, ice breakers and
// commands, and WhatsApp calling. Every method that takes a phoneNumber
// returns 404 whatsapp_sender_not_connected for a number that isn't
// connected.
type WhatsAppSendersService struct {
	client *Client
}

// WhatsAppTemplatesService manages Meta-reviewed WhatsApp message templates.
type WhatsAppTemplatesService struct {
	client *Client
}

// WhatsApp signup statuses, as sent in WhatsAppSignup.Status and
// WhatsAppSignupSession.Status. Compare with these rather than literals; the
// field is a plain string, so a status added later still decodes.
const (
	// WhatsAppSignupStatusInitiated means the signup is waiting for the
	// Facebook step.
	WhatsAppSignupStatusInitiated = "initiated"
	// WhatsAppSignupStatusRegistering means WhatsApp is activating the
	// number after the Facebook step.
	WhatsAppSignupStatusRegistering = "registering"
	// WhatsAppSignupStatusVerifying means a number being added to a
	// connected WhatsApp Business Account is waiting for its verification
	// code (see WhatsAppSignupService.Verify).
	WhatsAppSignupStatusVerifying = "verifying"
	// WhatsAppSignupStatusActive means the number is connected.
	WhatsAppSignupStatusActive = "active"
	// WhatsAppSignupStatusFailed means the signup failed; FailureReasons
	// says why.
	WhatsAppSignupStatusFailed = "failed"
)

// How WhatsApp delivers the verification code when a number is added to a
// connected WhatsApp Business Account.
const (
	// WhatsAppVerificationMethodSMS sends the code by text message. It is
	// the default.
	WhatsAppVerificationMethodSMS = "sms"
	// WhatsAppVerificationMethodVoice reads the code out in a phone call.
	WhatsAppVerificationMethodVoice = "voice"
)

// Error codes returned by the WhatsApp endpoints and WhatsApp sends,
// readable from the Code field of the typed error (for example
// err.(*SendlyError).Code). A 400 or 422 is a *ValidationError, a 404 a
// *NotFoundError, a 429 a *RateLimitError, and the rest a *SendlyError
// carrying StatusCode.
const (
	// WhatsAppErrorCodeSendFailed comes from MessagesService.SendWhatsApp.
	// As a 422 (*ValidationError) WhatsApp refused the message: it is final,
	// the API caches it under the idempotency key and replays it for 24
	// hours. As a 502 (*SendlyError) the message provably never reached the
	// carrier, so it was not sent and is safe to send again; it is never
	// cached, and the client retries it like any 5xx under the same key.
	WhatsAppErrorCodeSendFailed = "whatsapp_send_failed"
	// WhatsAppErrorCodeSendUnconfirmed (409, *SendlyError) comes from
	// MessagesService.SendWhatsApp when the outcome is unknown: the message
	// was marked failed and refunded but may still be delivered, so check
	// before sending it again (it could arrive twice). It is not retried
	// automatically. The body's errorCode, in Extra, is a classifier code
	// such as "E024", not the machine code.
	WhatsAppErrorCodeSendUnconfirmed = "whatsapp_send_unconfirmed"
	// WhatsAppErrorCodeFileRequired (400): UploadProfilePhoto sent no file.
	WhatsAppErrorCodeFileRequired = "file_required"
	// WhatsAppErrorCodeProfilePhotoInvalid (400): the photo isn't a JPEG
	// or PNG, judged by its bytes.
	WhatsAppErrorCodeProfilePhotoInvalid = "whatsapp_profile_photo_invalid"
	// WhatsAppErrorCodeProfilePhotoTooLarge (413, *SendlyError): the photo
	// is over 5 MB.
	WhatsAppErrorCodeProfilePhotoTooLarge = "whatsapp_profile_photo_too_large"
	// WhatsAppErrorCodeProfileUpdateFailed (502): WhatsApp refused the
	// profile change or couldn't be reached. For a photo, retry after
	// checking it is square and at least 192 pixels wide.
	WhatsAppErrorCodeProfileUpdateFailed = "whatsapp_profile_update_failed"
	// WhatsAppErrorCodeConversationalComponentsFetch (502): the ice
	// breakers and commands couldn't be fetched.
	WhatsAppErrorCodeConversationalComponentsFetch = "whatsapp_conversational_components_fetch_failed"
	// WhatsAppErrorCodeConversationalComponentsUpdate (502): the ice
	// breakers and commands couldn't be saved.
	WhatsAppErrorCodeConversationalComponentsUpdate = "whatsapp_conversational_components_update_failed"
	// WhatsAppErrorCodeCallingUnavailable (422): WhatsApp didn't allow
	// calling on the number. Meta enables it only once the account may
	// message at least 2,000 people a day and the number's display name is
	// approved.
	WhatsAppErrorCodeCallingUnavailable = "whatsapp_calling_unavailable"
	// WhatsAppErrorCodeCallingUpdateFailed (502): calling couldn't be
	// changed right now; retry.
	WhatsAppErrorCodeCallingUpdateFailed = "whatsapp_calling_update_failed"
	// WhatsAppErrorCodeBusinessAccountNotFound (404): no WhatsApp Business
	// Account with that BusinessAccountID is connected in the workspace with
	// at least one active number.
	WhatsAppErrorCodeBusinessAccountNotFound = "whatsapp_business_account_not_found"
	// WhatsAppErrorCodeDisplayNameRequired (400): no DisplayName was given
	// and the account has no sender display name or business name to use.
	WhatsAppErrorCodeDisplayNameRequired = "display_name_required"
	// WhatsAppErrorCodeSignupInProgress (409): a Facebook connection for
	// the number is in flight. The body's id, in Extra, is that signup.
	WhatsAppErrorCodeSignupInProgress = "whatsapp_signup_in_progress"
	// WhatsAppErrorCodeAlreadyEnabled (409): the number is already
	// connected to WhatsApp.
	WhatsAppErrorCodeAlreadyEnabled = "whatsapp_already_enabled"
	// WhatsAppErrorCodeVerificationStartFailed: WhatsApp wouldn't send the
	// code (422, final) or couldn't be reached (502). Either way the signup
	// failed and its fee is refunded; start again.
	WhatsAppErrorCodeVerificationStartFailed = "whatsapp_verification_start_failed"
	// WhatsAppErrorCodeVerificationInProgress (409): Create was called
	// without BusinessAccountID for a number that is waiting for its code;
	// the body's id, in Extra, is that signup. CreateWithOptions with
	// BusinessAccountID can also return it, without an id, while an expired
	// attempt for the number is being replaced; try again in a moment.
	WhatsAppErrorCodeVerificationInProgress = "whatsapp_verification_in_progress"
	// WhatsAppErrorCodeInvalidVerificationCode (400): the code isn't 6
	// digits.
	WhatsAppErrorCodeInvalidVerificationCode = "invalid_verification_code"
	// WhatsAppErrorCodeVerificationCodeInvalid (422): WhatsApp didn't
	// accept the code. The body's attemptsRemaining, in Extra, says how many
	// tries are left.
	WhatsAppErrorCodeVerificationCodeInvalid = "whatsapp_verification_code_invalid"
	// WhatsAppErrorCodeVerificationFailed (409): 5 wrong codes. The signup
	// failed and its fee is refunded.
	WhatsAppErrorCodeVerificationFailed = "whatsapp_verification_failed"
	// WhatsAppErrorCodeVerificationBusy (409): another code for the number
	// is being checked; retry in a moment.
	WhatsAppErrorCodeVerificationBusy = "whatsapp_verification_busy"
	// WhatsAppErrorCodeVerificationUnavailable (502): WhatsApp couldn't
	// check the code. The attempt isn't counted; retry.
	WhatsAppErrorCodeVerificationUnavailable = "whatsapp_verification_unavailable"
	// WhatsAppErrorCodeActivationPending (502): WhatsApp accepted the code
	// but Sendly couldn't finish connecting the number. Sendly is alerted;
	// check the signup again shortly.
	WhatsAppErrorCodeActivationPending = "whatsapp_activation_pending"
	// WhatsAppErrorCodeSignupNotActive (409): the signup isn't waiting for
	// a code (it failed, or began more than 3 hours ago).
	WhatsAppErrorCodeSignupNotActive = "signup_not_active"
	// WhatsAppErrorCodeSignupNotFound (404): no signup with that ID in the
	// workspace.
	WhatsAppErrorCodeSignupNotFound = "signup_not_found"
	// WhatsAppErrorCodeVerificationResendTooSoon (429, *RateLimitError):
	// codes are at least 30 seconds apart, counted from the signup's last
	// change, a code submission included. RetryAfter holds the seconds to
	// wait; the client returns it at once rather than waiting.
	WhatsAppErrorCodeVerificationResendTooSoon = "whatsapp_verification_resend_too_soon"
	// WhatsAppErrorCodeVerificationResendFailed: WhatsApp wouldn't send
	// another code yet (422) or couldn't be reached (502).
	WhatsAppErrorCodeVerificationResendFailed = "whatsapp_verification_resend_failed"
)

// WhatsAppSignupSession is the response from starting a WhatsApp signup.
// For a Facebook connection, hand ConnectURL to a human: they open it in a
// browser and log in with Facebook to link their WhatsApp Business Account.
// Poll WhatsAppSignupService.Get with ID until the status is "active". For a
// number added by code (CreateWithOptions with BusinessAccountID), ConnectURL
// is empty, Status is "verifying" and the signup fields below are filled;
// submit the code with WhatsAppSignupService.Verify.
type WhatsAppSignupSession struct {
	// ID is the unique signup identifier.
	ID string `json:"id"`
	// ConnectURL is the hosted connect page URL. A person must open this in
	// a browser. Empty for a number added by code.
	ConnectURL string `json:"connectUrl"`
	// Status is "initiated", "registering", "verifying", "active", or
	// "failed" (see the WhatsAppSignupStatus constants). The API does not
	// send "expired".
	Status string `json:"status"`
	// PhoneNumber is the number being connected, in E.164 format. Set for a
	// number added by code.
	PhoneNumber string `json:"phoneNumber,omitempty"`
	// BusinessAccountID is the WhatsApp Business Account the number is
	// joining. Set for a number added by code.
	BusinessAccountID *string `json:"businessAccountId,omitempty"`
	// VerificationMethod is "sms" or "voice" while Status is "verifying".
	VerificationMethod string `json:"verificationMethod,omitempty"`
	// VerificationAttemptsRemaining is how many wrong codes are left while
	// Status is "verifying"; nil otherwise.
	VerificationAttemptsRemaining *int `json:"verificationAttemptsRemaining,omitempty"`
	// FailureReasons is why the signup failed; see WhatsAppSignup.
	FailureReasons []string `json:"failureReasons,omitempty"`
	// UpdatedAt is the ISO 8601 timestamp of the last status change. Set for
	// a number added by code.
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// CreateWhatsAppSignupRequest starts a WhatsApp signup. PhoneNumber is
// required. Leave BusinessAccountID empty for a Facebook connection, the
// same request Create sends.
//
// Set BusinessAccountID to add the number to a WhatsApp Business Account
// the workspace already connected, without the Facebook step: WhatsApp sends
// the number a 6-digit code, and WhatsAppSignupService.Verify submits it.
type CreateWhatsAppSignupRequest struct {
	// PhoneNumber is an active number in the workspace, in E.164 format.
	PhoneNumber string `json:"phoneNumber"`
	// BusinessAccountID is the WhatsApp Business Account id, as
	// WhatsAppSender.BusinessAccountID and WhatsAppSignup.BusinessAccountID
	// report it. The account must be connected in this workspace with at
	// least one active number.
	BusinessAccountID string `json:"businessAccountId,omitempty"`
	// VerificationMethod is how the code arrives: "sms" (the default) or
	// "voice". Only used with BusinessAccountID.
	VerificationMethod string `json:"verificationMethod,omitempty"`
	// DisplayName is the name WhatsApp shows for the number, at most 512
	// characters. Only used with BusinessAccountID. Defaults to the
	// account's existing sender display name, else its business name.
	DisplayName string `json:"displayName,omitempty"`
}

// WhatsAppSignup is the status of a WhatsApp signup.
type WhatsAppSignup struct {
	// ID is the unique signup identifier.
	ID string `json:"id"`
	// Status is "initiated", "registering", "verifying", "active", or
	// "failed" (see the WhatsAppSignupStatus constants). After the Facebook
	// step it stays "registering" while WhatsApp activates the number.
	// Activation usually takes a few minutes but can take hours. If it
	// hasn't finished about 6 hours after the session began, the session
	// fails with registration_timeout and the fee is refunded. A number
	// added by code is "verifying" until its code is accepted. The API does
	// not send "expired".
	Status string `json:"status"`
	// PhoneNumber is the number being connected, in E.164 format.
	PhoneNumber string `json:"phoneNumber"`
	// BusinessAccountID is the customer's WhatsApp Business Account id. It
	// is set while Status is "active", or "verifying" for a number added by
	// code, and nil otherwise.
	BusinessAccountID *string `json:"businessAccountId,omitempty"`
	// FailureReasons is why the signup failed, when Status is "failed". It
	// holds one code: "setup_fee_payment_failed", "signup_abandoned",
	// "meta_exchange_failed", "registration_failed", "waba_already_connected",
	// "waba_mismatch" (the WhatsApp Business Account chosen in the Facebook
	// step doesn't hold the verified number), "registration_timeout"
	// (activation hadn't finished about 6 hours after the session began),
	// "phone_number_mismatch", or, for a number added by code,
	// "verification_start_failed" (WhatsApp wouldn't send the code),
	// "verification_failed" (5 wrong codes) or "verification_expired" (no
	// code was accepted and the signup was untouched for an hour, or it began
	// more than 3 hours ago when the number was added again). If the
	// connection fails, the $19 fee is refunded automatically.
	FailureReasons []string `json:"failureReasons,omitempty"`
	// UpdatedAt is the ISO 8601 timestamp of the last status change.
	UpdatedAt string `json:"updatedAt"`
	// VerificationMethod is how the code is sent, "sms" or "voice", while
	// Status is "verifying"; empty otherwise.
	VerificationMethod string `json:"verificationMethod,omitempty"`
	// VerificationAttemptsRemaining is how many wrong codes are left while
	// Status is "verifying"; nil otherwise.
	VerificationAttemptsRemaining *int `json:"verificationAttemptsRemaining,omitempty"`
	// VerificationCode is the code WhatsApp texted to the number, read from
	// the workspace's inbound messages, once it has arrived. Only Get fills
	// it, only while Status is "verifying", and nil until a code arrives.
	// Until a code has been submitted it is the newest code that has arrived
	// since the signup started, so after a resend it still shows the earlier
	// code until the new one arrives. Once WhatsApp has checked a code, only
	// a code that arrived after the last submission or resend is returned. A
	// submission answered with 502 whatsapp_verification_unavailable is not
	// counted, so the same unchecked code can come back, and submitting it
	// again is safe.
	VerificationCode *string `json:"verificationCode,omitempty"`
}

// WhatsAppSender is a number connected (or connecting) to WhatsApp.
type WhatsAppSender struct {
	// PhoneNumber is the sender, in E.164 format.
	PhoneNumber string `json:"phoneNumber"`
	// DisplayName is the name recipients see — chosen during the connect
	// flow and reviewed by Meta; nil until set.
	DisplayName *string `json:"displayName,omitempty"`
	// Status is "pending", "active", or "suspended".
	Status string `json:"status"`
	// QualityRating is the Meta quality rating (e.g. "GREEN"), nil before the first rating.
	QualityRating *string `json:"qualityRating,omitempty"`
	// BusinessAccountID is the WhatsApp Business Account the sender belongs
	// to; nil while the sender is pending. Pass it as
	// CreateWhatsAppSignupRequest.BusinessAccountID to add another number to
	// the same account.
	BusinessAccountID *string `json:"businessAccountId,omitempty"`
	// BusinessName is the WhatsApp Business Account's business name; nil
	// while the sender is pending or when Meta hasn't reported one.
	BusinessName *string `json:"businessName,omitempty"`
	// CallingEnabled is true when WhatsApp calling is switched on for the
	// sender (see WhatsAppSendersService.SetCalling).
	CallingEnabled bool `json:"callingEnabled,omitempty"`
	// OutboundCallingAllowed is false for +1 numbers (the US, Canada and
	// the rest of the North American numbering plan), Egypt (+20), Vietnam
	// (+84) and Nigeria (+234), where Meta forbids business-initiated
	// WhatsApp calls.
	OutboundCallingAllowed bool `json:"outboundCallingAllowed,omitempty"`
	// CreatedAt is the ISO 8601 timestamp when the sender was connected.
	CreatedAt string `json:"createdAt"`
}

// WhatsAppCommand is a command a customer sees when they type "/" in a chat
// with the business.
type WhatsAppCommand struct {
	// Command is letters, digits and underscores, 1-32 characters. A
	// leading "/" is stripped.
	Command string `json:"command"`
	// Description says what the command does, 1-256 characters.
	Description string `json:"description"`
}

// WhatsAppConversationalComponents are a connected sender's ice breakers
// and commands.
type WhatsAppConversationalComponents struct {
	// PhoneNumber is the sender, in E.164 format.
	PhoneNumber string `json:"phoneNumber"`
	// IceBreakers are tappable suggestions shown when someone opens a chat
	// with the business for the first time.
	IceBreakers []string `json:"iceBreakers"`
	// Commands are shown when the customer types "/".
	Commands []WhatsAppCommand `json:"commands"`
}

// UpdateWhatsAppConversationalComponentsRequest replaces a sender's ice
// breakers, commands, or both. A nil list is left unchanged; a non-nil list
// replaces the stored one, so an empty list ([]string{} or
// []WhatsAppCommand{}) clears it. At least one list must be set.
//
// The API allows at most 4 ice breakers, each 1-80 characters after
// trimming and different from the others ignoring case, and at most 30
// commands with no duplicates. Anything else is refused with a 400
// invalid_request whose Message says what to fix.
type UpdateWhatsAppConversationalComponentsRequest struct {
	IceBreakers []string
	Commands    []WhatsAppCommand
}

// MarshalJSON sends only the lists that are set, keeping an empty list as
// [] so it clears the stored one.
func (r UpdateWhatsAppConversationalComponentsRequest) MarshalJSON() ([]byte, error) {
	body := struct {
		IceBreakers *[]string          `json:"iceBreakers,omitempty"`
		Commands    *[]WhatsAppCommand `json:"commands,omitempty"`
	}{}
	if r.IceBreakers != nil {
		body.IceBreakers = &r.IceBreakers
	}
	if r.Commands != nil {
		body.Commands = &r.Commands
	}
	return json.Marshal(body)
}

// WhatsAppSenderCalling is a sender's WhatsApp calling setting.
type WhatsAppSenderCalling struct {
	// PhoneNumber is the sender, in E.164 format.
	PhoneNumber string `json:"phoneNumber"`
	// CallingEnabled is true when WhatsApp users can call the number.
	CallingEnabled bool `json:"callingEnabled"`
	// OutboundCallingAllowed is false for +1 numbers (the US, Canada and
	// the rest of the North American numbering plan), Egypt (+20), Vietnam
	// (+84) and Nigeria (+234), where Meta forbids business-initiated
	// WhatsApp calls.
	OutboundCallingAllowed bool `json:"outboundCallingAllowed"`
}

// WhatsAppSenderListResponse wraps the workspace's WhatsApp senders.
type WhatsAppSenderListResponse struct {
	Senders []WhatsAppSender `json:"senders"`
}

// WhatsAppSenderProfile is the WhatsApp Business profile recipients see
// for a connected sender.
type WhatsAppSenderProfile struct {
	// PhoneNumber is the sender, in E.164 format.
	PhoneNumber string `json:"phoneNumber"`
	// DisplayName is the name recipients see; nil until set.
	DisplayName *string `json:"displayName,omitempty"`
	// ProfilePhotoURL is the profile photo URL; nil until set. Change it
	// with WhatsAppSendersService.UploadProfilePhoto and
	// DeleteProfilePhoto.
	ProfilePhotoURL *string `json:"profilePhotoUrl,omitempty"`
	// Category is the business category; nil until set.
	Category *string `json:"category,omitempty"`
	// About is the short line under the profile name; nil until set.
	About *string `json:"about,omitempty"`
	// Description is the longer business description; nil until set.
	Description *string `json:"description,omitempty"`
	// Email is the public contact email; nil until set.
	Email *string `json:"email,omitempty"`
	// Website is the public website URL; nil until set.
	Website *string `json:"website,omitempty"`
	// Address is the public business address; nil until set.
	Address *string `json:"address,omitempty"`
}

// UpdateWhatsAppSenderProfileRequest edits a connected sender's WhatsApp
// Business profile. Supply only the fields to change — at least one is
// required; omitted fields keep their current value. About is capped at
// 139 characters and Description at 512.
type UpdateWhatsAppSenderProfileRequest struct {
	DisplayName string `json:"displayName,omitempty"`
	About       string `json:"about,omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Email       string `json:"email,omitempty"`
	Website     string `json:"website,omitempty"`
	Address     string `json:"address,omitempty"`
}

// WhatsAppTemplateButton is a button on a WhatsApp template. Type is "url"
// (URL is required and may contain a {{1}} placeholder — supply Example
// values for review), "quick_reply" (e.g. a "Stop promotions" opt-out,
// recommended on marketing templates), or "otp" (copy-code button, required
// on AUTHENTICATION templates).
type WhatsAppTemplateButton struct {
	Type string `json:"type"`
	Text string `json:"text"`
	// URL is the link target (url buttons only); may contain a {{1}} placeholder.
	URL string `json:"url,omitempty"`
	// Example values for a url placeholder, for Meta review.
	Example []string `json:"example,omitempty"`
}

// CreateWhatsAppTemplateRequest creates a template and submits it to Meta
// for review. Sender (the WhatsApp-connected sending number in E.164),
// Name (lowercase letters, digits, and underscores, e.g. "order_shipped"),
// Language (e.g. "en_US"), Category, and Body are required. Use {{1}},
// {{2}}, … in Body for variables; every placeholder needs an example value
// in Examples.
type CreateWhatsAppTemplateRequest struct {
	Sender string `json:"sender"`
	Name   string `json:"name"`
	// Language is the template language code (e.g. "en_US").
	Language string `json:"language"`
	// Category is "AUTHENTICATION", "UTILITY", or "MARKETING" (the server
	// uppercases it). It is required, with no default: leaving it out returns
	// 400 template_category_invalid. It drives Meta review rules and pricing,
	// and Update can't change it.
	Category string `json:"category"`
	Body     string `json:"body"`
	Footer   string `json:"footer,omitempty"`
	// Header is fixed text. A header containing {{n}} is refused with
	// template_header_variable_unsupported, because sends fill only body and
	// button variables.
	Header  string                   `json:"header,omitempty"`
	Buttons []WhatsAppTemplateButton `json:"buttons,omitempty"`
	// Examples are example values for body placeholders, keyed by
	// placeholder number: {"1": "Acme Inc", "2": "#4821"}. Required when the
	// body has variables.
	Examples map[string]string `json:"examples,omitempty"`
}

// UpdateWhatsAppTemplateRequest edits a template. Supply only the fields to
// change; omitted fields keep their current value.
type UpdateWhatsAppTemplateRequest struct {
	Body   string `json:"body,omitempty"`
	Footer string `json:"footer,omitempty"`
	// Header can't contain {{n}} variables
	// (template_header_variable_unsupported).
	Header   string                   `json:"header,omitempty"`
	Buttons  []WhatsAppTemplateButton `json:"buttons,omitempty"`
	Examples map[string]string        `json:"examples,omitempty"`
}

// WhatsAppTemplate is a WhatsApp message template.
type WhatsAppTemplate struct {
	// ID is the unique template identifier.
	ID string `json:"id"`
	// Name is the template name.
	Name string `json:"name"`
	// Language is the template language code.
	Language string `json:"language"`
	// Category is "AUTHENTICATION", "UTILITY", or "MARKETING" (Meta may
	// reclassify; this value drives pricing).
	Category string `json:"category"`
	// Status is Meta's review status in uppercase, for example "PENDING"
	// (Meta review usually takes 24-48h), "APPROVED", "REJECTED" (edit with
	// Update to resubmit), "PAUSED" or "DISABLED". Meta may report others.
	Status string `json:"status"`
	// QualityRating is the Meta quality rating (e.g. "GREEN"), nil before the first rating.
	QualityRating *string `json:"qualityRating,omitempty"`
	// RejectionReason is why Meta rejected the template, when Status is "REJECTED".
	RejectionReason *string `json:"rejectionReason,omitempty"`
	// CreatedAt is the ISO 8601 timestamp when the template was created.
	CreatedAt string `json:"createdAt"`
	// UpdatedAt is the ISO 8601 timestamp when the template was last updated.
	UpdatedAt string `json:"updatedAt"`
	// Warnings are non-blocking submission warnings (e.g. a marketing
	// template without an opt-out button). Present on create responses when
	// applicable.
	Warnings []string `json:"warnings,omitempty"`
}

// WhatsAppTemplateListResponse wraps the workspace's WhatsApp templates.
type WhatsAppTemplateListResponse struct {
	Templates []WhatsAppTemplate `json:"templates"`
}

// WhatsAppTemplateDeletedResponse confirms a template deletion.
type WhatsAppTemplateDeletedResponse struct {
	// ID is the deleted template's id.
	ID string `json:"id"`
	// Deleted is always true.
	Deleted bool `json:"deleted"`
}

// WhatsAppWindow reports whether a 24-hour customer-service window is open.
type WhatsAppWindow struct {
	// Open is true when a 24-hour customer-service window is currently open.
	Open bool `json:"open"`
	// ExpiresAt is when the window closes (ISO 8601). After it closes this is
	// the past expiry, with Open false. Nil when Sendly has no window on
	// record for the pair; a free-form send may still go through then if
	// WhatsApp reports an open window, and otherwise fails with
	// whatsapp_window_closed.
	ExpiresAt *string `json:"expiresAt,omitempty"`
}

// Create starts connecting a number to WhatsApp. The number must be an
// active number in your workspace, in E.164 format.
//
// Charges a one-time $19 setup fee (no monthly fee) and returns a connect
// URL. Completing the connection requires a human: hand the URL to your
// user — they open it in a browser and log in with Facebook to link their
// WhatsApp Business Account. Then poll Get until the status is "active".
// Calling again for a number with an in-flight signup returns the existing
// signup (same ConnectURL) without charging again. Requires a live API key
// with the whatsapp:write scope (a test key gets 403
// whatsapp_requires_live_key) and, in a team workspace, an owner or admin
// with settings:write (otherwise a 403 insufficient_permissions). After the
// Facebook step the signup stays "registering" while WhatsApp activates the
// number. Activation usually takes a few minutes but can take hours. If it
// hasn't finished about 6 hours after the session began, the session fails
// with registration_timeout and the fee is refunded. If the connection
// fails, the $19 fee is refunded automatically; once the number has
// connected there is no refund, and a later disconnect gets nothing back.
//
// While WhatsApp connections are unavailable the API answers 503
// whatsapp_unavailable before charging anything, with retryAfter: 3600 in
// the body and a Retry-After: 3600 header. Only signup returns it; no send
// does. The client retries it like any 5xx before returning it as a
// *SendlyError. After 5 failed, charged signups in 24 hours it answers 429
// whatsapp_signup_limit_reached, returned as a *RateLimitError and not
// retried; try again the next day.
//
// To add a number to a WhatsApp Business Account the workspace already
// connected, without the Facebook step, use CreateWithOptions with
// BusinessAccountID. Calling Create for a number that is waiting for its
// code that way returns 409 whatsapp_verification_in_progress, with the
// signup's id in the error's Extra.
func (s *WhatsAppSignupService) Create(ctx context.Context, phoneNumber string) (*WhatsAppSignupSession, error) {
	if phoneNumber == "" {
		return nil, &ValidationError{APIError: APIError{Message: "phoneNumber is required"}}
	}

	body := map[string]string{"phoneNumber": phoneNumber}
	var resp WhatsAppSignupSession
	if err := s.client.request(ctx, "POST", "/whatsapp/signup", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateWithOptions starts a WhatsApp signup. With only PhoneNumber set it
// sends the same request as Create, a Facebook connection.
//
// With BusinessAccountID set it adds the number to a WhatsApp Business
// Account the workspace already connected, with no Facebook step. The number
// must pass the same checks as for a Facebook connection, and the account
// must be connected in this workspace with at least one active number
// (otherwise 404 whatsapp_business_account_not_found). The same one-time $19
// fee is charged before WhatsApp is asked for the code and refunded
// automatically if the signup fails; 402 payment_method_required and
// payment_failed and 429 whatsapp_signup_limit_reached apply as for Create.
// The signup comes back "verifying", with no ConnectURL, and WhatsApp sends
// the number a 6-digit code by text (or a voice call with
// VerificationMethod "voice"). Submit it with Verify; while the session is
// verifying, Get returns the code in VerificationCode once its text has
// arrived on the number. Calling again for a number that is verifying
// returns the same signup without charging again or sending a second code;
// a verifying signup that began more than 3 hours ago is expired and
// refunded, and a new one is started.
//
// Errors for a number added by code: 409 whatsapp_signup_in_progress (a
// Facebook connection for the number is in flight), 409
// whatsapp_already_enabled, 400 display_name_required, and
// whatsapp_verification_start_failed when WhatsApp refused to send the code
// (422, final) or couldn't be reached (502). Either way that signup failed
// and its fee is refunded; start again. A 5xx, a timeout or a network error
// is returned without a retry here, since the request may have run and each
// retry could start, and charge, a new signup. Nor does Go's HTTP transport
// resend the request when a reused keep-alive connection drops. Check with
// Senders.List, or call again with the same key passed to
// WithIdempotencyKey. A retryable 429 is still retried, because the API
// refused it before running it.
//
// Requires a live API key with the whatsapp:write scope and, in a team
// workspace, an owner or admin with settings:write. An Idempotency-Key is
// sent automatically (see WithIdempotencyKey to supply your own).
func (s *WhatsAppSignupService) CreateWithOptions(ctx context.Context, req *CreateWhatsAppSignupRequest, opts ...RequestOption) (*WhatsAppSignupSession, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.PhoneNumber == "" {
		return nil, &ValidationError{APIError: APIError{Message: "phoneNumber is required"}}
	}
	if req.BusinessAccountID != "" && strings.TrimSpace(req.BusinessAccountID) == "" {
		return nil, &ValidationError{APIError: APIError{Message: "businessAccountId must be a non-empty string"}}
	}
	if req.BusinessAccountID != "" {
		opts = append([]RequestOption{withoutUnknownOutcomeRetries()}, opts...)
	}

	var resp WhatsAppSignupSession
	if err := s.client.request(ctx, "POST", "/whatsapp/signup", req, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Verify submits the 6-digit code WhatsApp sent to a number being added
// with CreateWithOptions (spaces and dashes are ignored). An accepted code
// connects the number: the signup comes back "active" and a
// whatsapp_account.connected webhook fires. Verify on a signup that is
// already active returns it unchanged.
//
// A code that isn't 6 digits is a 400 invalid_verification_code. A wrong
// code is a 422 whatsapp_verification_code_invalid whose Extra
// attemptsRemaining says how many tries are left; after 5 wrong codes the
// signup fails with 409 whatsapp_verification_failed, the fee is refunded
// and whatsapp_account.failed fires. 409 whatsapp_verification_busy means
// another code is being checked; retry in a moment. 502
// whatsapp_verification_unavailable means WhatsApp couldn't check the code
// and the attempt isn't counted; retry. 502 whatsapp_activation_pending means
// the code was accepted but connecting the number didn't finish; Sendly is
// alerted, so check the signup with Get shortly. A 5xx, a timeout or a
// network error is returned without a retry, because the code may have been
// checked and each attempt submits it again and can use up one of the 5
// tries; check the signup with Get before submitting again. Nor does Go's
// HTTP transport resend the request when a reused keep-alive connection
// drops. A retryable 429 is still retried, because the API refused it before
// running it. 409 signup_not_active means the signup isn't waiting for a
// code (it failed, or began more than 3 hours ago); 404 signup_not_found
// means there is no such signup.
//
// Requires a live API key with the whatsapp:write scope and, in a team
// workspace, an owner or admin with settings:write.
func (s *WhatsAppSignupService) Verify(ctx context.Context, id, code string) (*WhatsAppSignup, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "signup ID is required"}}
	}
	if code == "" {
		return nil, &ValidationError{APIError: APIError{Message: "code is required"}}
	}

	body := map[string]string{"code": code}
	var resp WhatsAppSignup
	if err := s.client.request(ctx, "POST", "/whatsapp/signup/"+url.PathEscape(id)+"/verify", body, &resp, withoutUnknownOutcomeRetries()); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Resend asks WhatsApp for a new code for a number being added with
// CreateWithOptions and returns the signup. verificationMethod is "sms" or
// "voice"; an empty string sends "sms", whatever the previous method was.
// Resend on a signup that is already active returns it unchanged.
//
// Codes are at least 30 seconds apart, counted from the signup's last
// change, a code submission included. Sooner than that the API answers 429
// whatsapp_verification_resend_too_soon, returned at once as a
// *RateLimitError with RetryAfter set to the seconds left. WhatsApp
// refusing to send another code yet is a 422
// whatsapp_verification_resend_failed, and being unreachable a 502 with the
// same code. 409 signup_not_active means the signup isn't waiting for a
// code.
//
// Requires a live API key with the whatsapp:write scope and, in a team
// workspace, an owner or admin with settings:write.
func (s *WhatsAppSignupService) Resend(ctx context.Context, id, verificationMethod string) (*WhatsAppSignup, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "signup ID is required"}}
	}

	body := map[string]string{}
	if verificationMethod != "" {
		body["verificationMethod"] = verificationMethod
	}
	var resp WhatsAppSignup
	if err := s.client.request(ctx, "POST", "/whatsapp/signup/"+url.PathEscape(id)+"/resend", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get fetches the status of a WhatsApp signup. Poll this after Create until
// the status is "active" ("failed" sets FailureReasons). After the Facebook
// step it stays "registering": activation usually takes a few minutes but
// can take hours, and a session that hasn't finished about 6 hours after it
// began fails with registration_timeout. A number added by code stays
// "verifying" until Verify accepts its code; meanwhile Get fills
// VerificationMethod, VerificationAttemptsRemaining and, once the code's
// text has arrived on the number, VerificationCode. Needs the whatsapp:read
// scope; test keys work.
func (s *WhatsAppSignupService) Get(ctx context.Context, id string) (*WhatsAppSignup, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "signup ID is required"}}
	}

	var resp WhatsAppSignup
	if err := s.client.request(ctx, "GET", "/whatsapp/signup/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List returns the numbers connected (or connecting) to WhatsApp on the
// workspace, newest first, with connection status and quality rating. An
// empty list means no number is connected yet — start one with
// WhatsAppSignupService.Create. Needs the whatsapp:read scope; test keys
// work.
func (s *WhatsAppSendersService) List(ctx context.Context) (*WhatsAppSenderListResponse, error) {
	var resp WhatsAppSenderListResponse
	if err := s.client.request(ctx, "GET", "/whatsapp/senders", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetProfile fetches the WhatsApp Business profile recipients see for a
// connected sender (E.164). The number must have an active WhatsApp
// connection. Needs the whatsapp:read scope; test keys work.
func (s *WhatsAppSendersService) GetProfile(ctx context.Context, phoneNumber string) (*WhatsAppSenderProfile, error) {
	if phoneNumber == "" {
		return nil, &ValidationError{APIError: APIError{Message: "phoneNumber is required"}}
	}

	var resp WhatsAppSenderProfile
	if err := s.client.request(ctx, "GET", "/whatsapp/senders/"+url.PathEscape(phoneNumber)+"/profile", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateProfile edits a connected sender's WhatsApp Business profile and
// returns the updated profile. Supply only the fields to change — at least
// one is required. Requires a live API key with the whatsapp:write scope
// and, in a team workspace, an owner or admin (settings:write).
func (s *WhatsAppSendersService) UpdateProfile(ctx context.Context, phoneNumber string, req *UpdateWhatsAppSenderProfileRequest) (*WhatsAppSenderProfile, error) {
	if phoneNumber == "" {
		return nil, &ValidationError{APIError: APIError{Message: "phoneNumber is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	var resp WhatsAppSenderProfile
	if err := s.client.request(ctx, "PATCH", "/whatsapp/senders/"+url.PathEscape(phoneNumber)+"/profile", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadProfilePhoto sets a connected sender's WhatsApp Business profile
// photo and returns the updated profile. The photo is sent as the multipart
// field "file", read in full from file, and must be a JPEG or PNG (judged by
// its bytes) of at most 5 MB. WhatsApp wants it square and at least 192
// pixels wide (640 recommended).
//
// A file that isn't a JPEG or PNG is a 400 whatsapp_profile_photo_invalid,
// one over 5 MB a 413 whatsapp_profile_photo_too_large (*SendlyError), and
// no file a 400 file_required. A 502 whatsapp_profile_update_failed means
// WhatsApp refused the photo or couldn't be reached; check it is square and
// large enough, then retry. The upload is sent once and never retried, not
// even by Go's HTTP transport when a reused keep-alive connection drops, so
// any 429, 5xx, timeout or network error comes straight back. Free.
// Requires a live API key with the whatsapp:write scope and, in a team
// workspace, an owner or admin (settings:write).
func (s *WhatsAppSendersService) UploadProfilePhoto(ctx context.Context, phoneNumber, filename string, file io.Reader) (*WhatsAppSenderProfile, error) {
	if phoneNumber == "" {
		return nil, &ValidationError{APIError: APIError{Message: "phoneNumber is required"}}
	}
	if file == nil {
		return nil, &ValidationError{APIError: APIError{Message: "file is required"}}
	}
	if filename == "" {
		return nil, &ValidationError{APIError: APIError{Message: "filename is required"}}
	}
	path := "/whatsapp/senders/" + url.PathEscape(phoneNumber) + "/profile/photo"
	if err := checkPathSegments(path); err != nil {
		return nil, err
	}

	if err := s.client.rateLimiter.Wait(ctx); err != nil {
		return nil, &NetworkError{Message: "rate limiter error", Err: err}
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, &NetworkError{Message: "failed to write file data", Err: err}
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := createFilePart(writer, "file", filepath.Base(filename), data)
	if err != nil {
		return nil, &NetworkError{Message: "failed to create form file", Err: err}
	}
	if _, err := part.Write(data); err != nil {
		return nil, &NetworkError{Message: "failed to write file data", Err: err}
	}
	if err := writer.Close(); err != nil {
		return nil, &NetworkError{Message: "failed to close multipart writer", Err: err}
	}

	fullURL := s.client.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, &buf)
	if err != nil {
		return nil, &NetworkError{Message: "failed to create request", Err: err}
	}
	req.GetBody = nil

	req.Header.Set("Authorization", "Bearer "+s.client.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sendly-go/"+Version)
	req.Header.Set("Idempotency-Key", generateIdempotencyKey())
	if s.client.OrganizationID != "" {
		req.Header.Set("X-Organization-Id", s.client.OrganizationID)
	}

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, &NetworkError{Message: "request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &NetworkError{Message: "failed to read response body", Err: err}
	}
	if resp.StatusCode >= 400 {
		return nil, s.client.handleErrorResponse(resp, respBody)
	}

	var result WhatsAppSenderProfile
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, &NetworkError{Message: "failed to unmarshal response", Err: err}
	}
	return &result, nil
}

// DeleteProfilePhoto removes a connected sender's WhatsApp Business profile
// photo and returns the updated profile, normally with ProfilePhotoURL nil
// (it is what WhatsApp reports after the change). A 502
// whatsapp_profile_update_failed means WhatsApp couldn't remove it; retry
// shortly. Free. Requires a live API key with the whatsapp:write scope and,
// in a team workspace, an owner or admin (settings:write).
func (s *WhatsAppSendersService) DeleteProfilePhoto(ctx context.Context, phoneNumber string) (*WhatsAppSenderProfile, error) {
	if phoneNumber == "" {
		return nil, &ValidationError{APIError: APIError{Message: "phoneNumber is required"}}
	}

	var resp WhatsAppSenderProfile
	if err := s.client.request(ctx, "DELETE", "/whatsapp/senders/"+url.PathEscape(phoneNumber)+"/profile/photo", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetConversationalComponents fetches a connected sender's ice breakers
// (tappable suggestions shown when someone opens a chat with the business
// for the first time) and commands (shown when the customer types "/"). A
// 502 whatsapp_conversational_components_fetch_failed means WhatsApp
// couldn't be reached. Needs the whatsapp:read scope; test keys work.
func (s *WhatsAppSendersService) GetConversationalComponents(ctx context.Context, phoneNumber string) (*WhatsAppConversationalComponents, error) {
	if phoneNumber == "" {
		return nil, &ValidationError{APIError: APIError{Message: "phoneNumber is required"}}
	}

	var resp WhatsAppConversationalComponents
	if err := s.client.request(ctx, "GET", "/whatsapp/senders/"+url.PathEscape(phoneNumber)+"/conversational_components", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateConversationalComponents replaces a connected sender's ice
// breakers, commands, or both, and returns what is stored. Each list that is
// set replaces the stored one, and an empty list clears it; see
// UpdateWhatsAppConversationalComponentsRequest for the limits. A 502
// whatsapp_conversational_components_update_failed means WhatsApp couldn't
// save them. Free. Requires a live API key with the whatsapp:write scope
// and, in a team workspace, an owner or admin (settings:write).
func (s *WhatsAppSendersService) UpdateConversationalComponents(ctx context.Context, phoneNumber string, req *UpdateWhatsAppConversationalComponentsRequest) (*WhatsAppConversationalComponents, error) {
	if phoneNumber == "" {
		return nil, &ValidationError{APIError: APIError{Message: "phoneNumber is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	var resp WhatsAppConversationalComponents
	if err := s.client.request(ctx, "PATCH", "/whatsapp/senders/"+url.PathEscape(phoneNumber)+"/conversational_components", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetCalling switches WhatsApp calling on or off for a connected sender and
// returns the setting. Once it is on, a WhatsApp user calling the number
// rings exactly like a phone call (a dashboard ring or an AI agent, per the
// number's voice settings), billed at the normal inbound rate. There is no
// API for placing WhatsApp calls. The switch itself is free.
//
// Turning it on needs voice switched on for the number first (voice mode
// ring_dashboard or agent), otherwise a 409 voice_not_enabled
// (CallErrorCodeVoiceNotEnabled, as a *SendlyError). A 422
// whatsapp_calling_unavailable means Meta refused: it enables calling only
// once the account may message at least 2,000 people a day and the
// number's display name is approved. A 502 whatsapp_calling_update_failed
// means WhatsApp couldn't be reached; retry. Requires a live API key with
// the whatsapp:write scope and, in a team workspace, an owner or admin
// (settings:write).
func (s *WhatsAppSendersService) SetCalling(ctx context.Context, phoneNumber string, enabled bool) (*WhatsAppSenderCalling, error) {
	if phoneNumber == "" {
		return nil, &ValidationError{APIError: APIError{Message: "phoneNumber is required"}}
	}

	body := map[string]bool{"enabled": enabled}
	var resp WhatsAppSenderCalling
	if err := s.client.request(ctx, "PATCH", "/whatsapp/senders/"+url.PathEscape(phoneNumber)+"/calling", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List returns the workspace's WhatsApp templates with review status and
// quality rating. Needs the whatsapp:read scope; test keys work.
func (s *WhatsAppTemplatesService) List(ctx context.Context) (*WhatsAppTemplateListResponse, error) {
	var resp WhatsAppTemplateListResponse
	if err := s.client.request(ctx, "GET", "/whatsapp/templates", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Create creates a template and submits it to Meta for review. Review
// usually takes 24-48h; the template is usable once its status is
// "APPROVED". Requires a live API key with the whatsapp:write scope and, in a
// team workspace, an owner, admin or member (templates:write). Category is
// required, with no default. A sender that isn't connected gets 404
// whatsapp_sender_not_connected, checked first. A template that fails the
// API's checks is refused with a 400 *ValidationError whose Code is the
// template_* reason and whose Message says what to fix:
// template_category_invalid (category missing or not one of the three),
// template_authentication_otp_button_required,
// template_authentication_no_links (a link in the body or a URL button on an
// authentication template) or template_header_variable_unsupported (a header
// containing {{n}}). A marketing template without an opt-out button only
// gets a warning.
func (s *WhatsAppTemplatesService) Create(ctx context.Context, req *CreateWhatsAppTemplateRequest) (*WhatsAppTemplate, error) {
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}
	if req.Sender == "" {
		return nil, &ValidationError{APIError: APIError{Message: "sender is required"}}
	}
	if req.Name == "" {
		return nil, &ValidationError{APIError: APIError{Message: "name is required"}}
	}
	if req.Language == "" {
		return nil, &ValidationError{APIError: APIError{Message: "language is required"}}
	}
	if req.Category == "" {
		return nil, &ValidationError{APIError: APIError{Message: "category is required"}}
	}
	if req.Body == "" {
		return nil, &ValidationError{APIError: APIError{Message: "body is required"}}
	}

	var resp WhatsAppTemplate
	if err := s.client.request(ctx, "POST", "/whatsapp/templates", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update edits an APPROVED or REJECTED template and resubmits it for
// review. This is the recovery path for rejections: template names are
// locked for ~30 days after deletion, so editing a rejected template
// (rather than deleting and re-creating it) is the way to fix it. The
// updated template goes back to "PENDING" review. The category can't be
// changed. Requires a live API key with the whatsapp:write scope and, in a
// team workspace, an owner, admin or member (templates:write).
func (s *WhatsAppTemplatesService) Update(ctx context.Context, id string, req *UpdateWhatsAppTemplateRequest) (*WhatsAppTemplate, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "template ID is required"}}
	}
	if req == nil {
		return nil, &ValidationError{APIError: APIError{Message: "request is required"}}
	}

	var resp WhatsAppTemplate
	if err := s.client.request(ctx, "PATCH", "/whatsapp/templates/"+url.PathEscape(id), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Delete deletes a template. Meta locks a deleted template's name for ~30
// days — re-creating it fails until the lock lifts. To fix a rejected
// template, prefer Update. Requires a live API key with the whatsapp:write
// scope and, in a team workspace, an owner, admin or member
// (templates:write).
func (s *WhatsAppTemplatesService) Delete(ctx context.Context, id string) (*WhatsAppTemplateDeletedResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "template ID is required"}}
	}

	var resp WhatsAppTemplateDeletedResponse
	if err := s.client.request(ctx, "DELETE", "/whatsapp/templates/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Window checks whether a 24-hour customer-service window is open between
// one of your WhatsApp senders (from) and a recipient (to), both in E.164
// format. Free-form text and media only deliver while a window is open (it
// opens when the recipient messages you and lasts 24h from their last
// inbound message). Outside a window, send an approved template. Needs the
// whatsapp:read scope; test keys work. The response is exactly
// { open, expiresAt }: with no window on record Open is false and ExpiresAt
// is nil; after a window has expired Open is false and ExpiresAt is the past
// expiry.
func (s *WhatsAppService) Window(ctx context.Context, from, to string) (*WhatsAppWindow, error) {
	if from == "" {
		return nil, &ValidationError{APIError: APIError{Message: "from is required"}}
	}
	if to == "" {
		return nil, &ValidationError{APIError: APIError{Message: "to is required"}}
	}

	path := "/whatsapp/window" + buildQueryString(map[string]string{"from": from, "to": to})
	var resp WhatsAppWindow
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
