<p align="center">
  <img src="https://raw.githubusercontent.com/SendlyHQ/sendly-go/main/.github/header.svg" alt="Sendly Go SDK" />
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/SendlyHQ/sendly-go/v4"><img src="https://pkg.go.dev/badge/github.com/SendlyHQ/sendly-go/v4.svg" alt="Go Reference" /></a>
  <a href="https://github.com/SendlyHQ/sendly-go/blob/main/LICENSE"><img src="https://img.shields.io/github/license/SendlyHQ/sendly-go?style=flat-square" alt="license" /></a>
</p>

# Sendly Go SDK

Official Go SDK for the Sendly API: SMS and MMS, group messaging, campaigns,
contacts, conversations, verification, numbers, local-number (10DLC)
registration, WhatsApp, RCS, phone calls with AI agents, branded links,
webhooks and the enterprise multi-workspace API.

## Installation

```bash
go get github.com/SendlyHQ/sendly-go/v4
```

The module path carries the major version, so the import is
`github.com/SendlyHQ/sendly-go/v4/sendly`.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/SendlyHQ/sendly-go/v4/sendly"
)

func main() {
    // Create a client
    client := sendly.NewClient("sk_live_v1_your_api_key")
    ctx := context.Background()

    // Send an SMS
    message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
        To:   "+15125550123",
        Text: "Hello from Sendly!",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Message sent: %s\n", message.ID)
}
```

The snippets below are fragments: they assume the imports above, a `ctx`, a
`client`, and whatever standard-library packages that snippet uses (`os`,
`errors`, `encoding/json` and so on).

## Prerequisites for Live Messaging

Before sending live SMS messages, you need:

1. **Business Verification** - Complete verification in the [Sendly dashboard](https://sendly.live/dashboard)
   - **International**: verification is quicker, and an alphanumeric sender ID
     can be used where the destination country allows one
   - **US/Canada**: Requires carrier approval — toll-free verification and
     local-number (10DLC) registration are both reviewed by the carriers

2. **Credits** - Add credits to your account
   - Test keys (`sk_test_*`) work without credits (sandbox mode)
   - Live keys (`sk_live_*`) require credits for each message

3. **Live API Key** - Generate after verification + credits
   - Dashboard → API Keys → Create Live Key

### Test vs Live Keys

| Key Type | Prefix | Credits Required | Verification Required | Use Case |
|----------|--------|------------------|----------------------|----------|
| Test | `sk_test_v1_*` | No | No | Development, testing |
| Live | `sk_live_v1_*` | Yes | Yes | Production messaging |

> **Note**: You can start development immediately with a test key. Messages to sandbox test numbers are free and don't require verification.

Some features refuse a test key outright rather than simulating: RCS sends
(`403 rcs_requires_live_key`), WhatsApp sends and connections
(`403 whatsapp_requires_live_key`), and `Calls.Create` / `Calls.Hangup`
(`403 live_key_required`).

## Configuration

```go
import (
    "net/http"
    "time"

    "github.com/SendlyHQ/sendly-go/v4/sendly"
)

// Create client with options
client := sendly.NewClient("sk_live_v1_xxx",
    sendly.WithBaseURL("https://sendly.live/api/v1"),
    sendly.WithTimeout(60*time.Second),
    sendly.WithMaxRetries(5),
    sendly.WithDebug(true),
    sendly.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
    sendly.WithOrganizationID("ws_xxx"),
)
```

Defaults: base URL `https://sendly.live/api/v1`, a 30-second timeout and 3
retries. `sendly.Version` is the SDK version sent in the `User-Agent`.

`WithOrganizationID` (or the `SENDLY_ORG_ID` environment variable, read when
the option is not set) sends `X-Organization-Id` on every request, which
selects the workspace a key with multi-workspace access acts in.

## Rate Limits

The API allows **60 requests per minute** on a test key, **600** on a live
key, and **3,000** on an enterprise master key. Over the limit the API
answers `429`, which the SDK surfaces as `*sendly.RateLimitError` with
`RetryAfter` in seconds.

The client also paces itself locally: after a burst of 10 requests it sends
about one request a second.

A 429 is waited out and retried only when waiting can help, and only when the
wait is 60 seconds or less: an ordinary `rate_limit_exceeded`, a 429 with no
code, the per-minute `provision_rate_limit` from enterprise workspace
provisioning (120 requests a minute), and `too_many_concurrent_verifications` (too many
first-time API key checks running at once, retried after 1 second). The wait
comes from the `Retry-After` header, or from the body's `retryAfter` when there
is no header, and the retry goes out after exactly that wait. After the last
attempt the client returns without waiting. Every other 429 comes back at once
as a `*sendly.RateLimitError` whose `Code` names it:

- `too_many_failed_key_attempts`: repeated wrong API keys from your address
  locked that address out for up to 5 minutes. Fix the key, then wait
  `RetryAfter` seconds; until the lockout ends even the right key can be
  refused.
- `rate_limit_exceeded` from `Verify.Send` or `Verify.Resend` when the
  recipient has had 5 codes in 10 minutes or 20 in a day, with `RetryAfter`
  set (a wait of a minute or less near the end of the window is waited out
  like any rate limit).
- `max_attempts_exceeded` from `Verify.Check`, `daily_call_limit` from
  `Calls.Create`, `quota_exceeded` (the workspace's monthly message quota is
  used up), `whatsapp_signup_limit_reached`,
  `whatsapp_verification_resend_too_soon` from `WhatsApp.Signup.Resend`
  (codes are 30 seconds apart, with `RetryAfter` set), and the hourly
  `provision_rate_limit` (1,000 provisioning requests an hour), which carries
  `RetryAfter` so you can pace provisioning.

`Error()` on a 429 returned at once includes the API's message and code, plus
the wait when there is one.

## Retries

POSTs, GETs and the rest retry up to `MaxRetries` times (default 3) with
exponential backoff of 1s, 2s and 4s. Only 5xx responses, request timeouts
and network failures are retried, plus the 429s listed under
[Rate Limits](#rate-limits). **Every other 4xx returns immediately**: a 409
`lines_busy`, a 428 `e911_required` or a 422 validation failure is handed back
without burning the backoff. A 2xx whose body does not decode into the result
type is returned at once as a `*sendly.NetworkError` (the JSON error is in
`Err`) and is not retried.

`Voice.Numbers.RegisterEmergencyAddress` opts out of 5xx retries on purpose,
because every attempt registers the address anew. Two WhatsApp calls return
a 5xx, a timeout or a network error without retrying, because the request
may have run: `WhatsApp.Signup.CreateWithOptions` with a `BusinessAccountID`,
where every attempt could start, and charge, a new signup, and
`WhatsApp.Signup.Verify`, where every attempt submits the code again and can
use up one of its 5 tries. Both still wait out and retry a retryable 429,
which the API refused before running. File uploads (`Media.Upload`,
`Enterprise.UploadVerificationDocument`, `BusinessUpgrade.Start`,
`BusinessUpgrade.Resubmit` and `WhatsApp.Senders.UploadProfilePhoto`) are
sent once and never retried, so any 429, 5xx, timeout or network error there
comes straight back. Nor does Go's HTTP transport resend those two WhatsApp
calls or `UploadProfilePhoto` when a reused keep-alive connection drops.

## Idempotency

POSTs carry an automatically generated `Idempotency-Key`, and the client sends
the same key on every retry of that call: after a 5xx, a timeout, a network
error or a waited-out 429. The API never records a 5xx or a 429 under the key,
so those retries run the request again; when an earlier attempt did finish (a
timeout after the API answered, or a gateway 5xx in front of a recorded
answer), the retry gets the recorded answer back instead of sending and
charging twice. The API deduplicates on sends, batch, group, schedule,
conversation replies, draft approval, verify, number purchase, credit
transfers, enterprise provisioning, WhatsApp signup and template
creation, calls, and RCS and short-code writes; other POSTs ignore the key, so
a retried call there can run twice. Pass your own key with
`sendly.WithIdempotencyKey` when the guarantee needs to outlive the process,
such as a job queue that re-runs after a crash or your own retry loop:

```go
message, err := client.Messages.SendWithOptions(ctx, &sendly.SendMessageRequest{
    To:   "+15125550123",
    Text: "Your order has shipped!",
}, sendly.WithIdempotencyKey("order-4821-shipped"))
if err != nil {
    log.Fatal(err)
}
```

Keys are 1-255 printable ASCII characters. Reusing a key within 24 hours
returns the original response, and reusing it with a different body returns
`422 idempotency_key_mismatch`, so derive keys from something stable in your
domain, like an order id. On the endpoints that deduplicate, a 2xx and any
4xx other than a 429 are recorded under the key, so repeating a refused request with the same key replays the
refusal: use a fresh key to run it again. A 5xx or a 429 is never recorded,
so retry those under the same key.

Each `Messages` send method has a `WithOptions` twin that accepts request
options — `SendWithOptions`, `ScheduleWithOptions`, `SendBatchWithOptions`,
`SendGroupWithOptions`, `SendRcsWithOptions`, `SendWhatsAppWithOptions` — and
the RCS registration, voice and calls methods take options directly. Other
writes, such as `Verify.Send`, `Campaigns.Send` or `Numbers.Buy`, take no
options: their POSTs still carry the automatic key, but you can't pass your own.
`SendBatch` sends no automatic key, because the API already deduplicates
identical batches by their contents.

Full details: https://sendly.live/docs/idempotency

## Messages

### Send an SMS

```go
// Marketing message (default)
message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:   "+15125550123",
    Text: "Check out our new features!",
})
if err != nil {
    log.Fatal(err)
}

// Transactional message (bypasses quiet hours)
message, err = client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:          "+15125550123",
    Text:        "Your verification code is: 123456",
    MessageType: sendly.MessageTypeTransactional,
})

// With custom metadata (max 4KB)
message, err = client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:   "+15125550123",
    Text: "Your order #12345 has shipped!",
    Metadata: map[string]interface{}{
        "order_id":    "12345",
        "customer_id": "cust_abc",
    },
})

// Send from one of your owned numbers (or an alphanumeric sender ID).
// Omit From to use your default sender.
message, err = client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:   "+15125550123",
    Text: "Hello from our team!",
    From: "+15125550100",
})

fmt.Printf("ID: %s\n", message.ID)
fmt.Printf("Status: %s\n", message.Status)
fmt.Printf("Credits: %d\n", message.CreditsUsed)
```

### Send an MMS

Attach `MediaUrls` to turn a message into an MMS. Media has to be uploaded
through the media endpoint first — arbitrary third-party URLs are rejected.
`Media.Upload` takes a JPEG, PNG or GIF of up to 600 KB and labels the file
with the type its content shows (or, failing that, its extension). Any other
type is refused with a `*sendly.SendlyError` (HTTP 415,
`unsupported_media_type`), a larger file with a `*sendly.SendlyError`
(HTTP 413, `file_too_large`), and a file whose content does not match its
image type with a `*sendly.ValidationError` (`invalid_file`).

```go
file, err := os.Open("promo.jpg")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

media, err := client.Media.Upload(ctx, "promo.jpg", file)
if err != nil {
    log.Fatal(err)
}

message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:        "+15125550123",
    Text:      "This week's specials",
    MediaUrls: []string{media.URL},
})

// Remove an uploaded file when you no longer need it
err = client.Media.Delete(ctx, media.ID)
```

### List Messages

```go
resp, err := client.Messages.List(ctx, &sendly.ListMessagesRequest{
    Limit:  50,
    Offset: 0,
    Status: sendly.MessageStatusDelivered,
    To:     "+15125550123",
})
if err != nil {
    log.Fatal(err)
}

for _, msg := range resp.Data {
    fmt.Printf("%s: %s (%s)\n", msg.ID, msg.To, msg.Status)
}
fmt.Printf("%d on this page\n", resp.Count)
if resp.Pagination != nil {
    fmt.Printf("page %d of %d, %d in all\n",
        resp.Pagination.Page, resp.Pagination.TotalPages, resp.Pagination.Total)
    if resp.Pagination.HasMore {
        // fetch the next page with Offset: resp.Pagination.Offset + resp.Pagination.Limit
    }
}
```

`Status` and `To` filter on the server. `Limit` defaults to 50 and is capped
at 100. A test key lists sandbox messages only, and a live key production
messages only.

### Get a Message

```go
message, err := client.Messages.Get(ctx, "4a7c1e2f-9b3d-4c8a-91f2-7d5e6a0b3c19")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("To: %s\n", message.To)
fmt.Printf("Text: %s\n", message.Text)
fmt.Printf("Status: %s\n", message.Status)

// Messages.Get does not return AI classification (AiMetadata stays nil).
// Inbound messages carry it on client.Conversations.Get with IncludeMessages.
```

### Scheduling Messages

```go
// Schedule a message for future delivery
scheduled, err := client.Messages.Schedule(ctx, &sendly.ScheduleMessageRequest{
    To:          "+15125550123",
    Text:        "Your appointment is tomorrow!",
    ScheduledAt: time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Scheduled: %s\n", scheduled.ID)
fmt.Printf("Will send at: %s\n", scheduled.ScheduledAt)
fmt.Printf("Credits reserved: %d\n", scheduled.CreditsReserved)

// List scheduled messages
resp, err := client.Messages.ListScheduled(ctx, &sendly.ListScheduledMessagesRequest{
    Status: sendly.ScheduledMessageStatusScheduled,
})
for _, msg := range resp.Data {
    fmt.Printf("%s: %s\n", msg.ID, msg.ScheduledAt)
}

// Get a specific scheduled message
msg, err := client.Messages.GetScheduled(ctx, "schd_xxx")

// Cancel a scheduled message (refunds the reserved credits)
result, err := client.Messages.CancelScheduled(ctx, "schd_xxx")
fmt.Printf("Refunded: %d credits\n", result.CreditsRefunded)
```

### Batch Messages

```go
// Send many messages in one API call
batch, err := client.Messages.SendBatch(ctx, &sendly.SendBatchRequest{
    Messages: []sendly.BatchMessageItem{
        {To: "+15125550123", Text: "Hello User 1!"},
        {To: "+15125550124", Text: "Hello User 2!"},
        {To: "+15125550125", Text: "Hello User 3!"},
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Batch ID: %s\n", batch.BatchID)
fmt.Printf("Status: %s\n", batch.Status)
fmt.Printf("Total: %d, sent: %d, failed: %d, retrying: %d\n",
    batch.Total, batch.Sent, batch.Failed, batch.Retrying)
fmt.Printf("Skipped: %d opted out, %d unable to receive SMS\n",
    batch.OptedOutSkipped, batch.InvalidSkipped)
fmt.Printf("Credits used: %d\n", batch.CreditsUsed)

// Read a batch back, or list them
status, err := client.Messages.GetBatch(ctx, batch.BatchID)
fmt.Println(status.Delivered, status.CreditsReserved, status.CreditsRefunded)
batches, err := client.Messages.ListBatches(ctx, &sendly.ListBatchesRequest{Limit: 20})
fmt.Printf("%d batches on this page\n", batches.Count)

// Preview batch (dry run) - validates without sending
preview, err := client.Messages.PreviewBatch(ctx, &sendly.SendBatchRequest{
    Messages: []sendly.BatchMessageItem{
        {To: "+15125550123", Text: "Hello User 1!"},
        {To: "+447700900123", Text: "Hello UK!"},
    },
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%d of %d sendable, %d credits needed, balance %d\n",
    preview.Sendable, preview.Total, preview.CreditsNeeded, preview.CreditBalance)
for country, part := range preview.ByCountry {
    fmt.Printf("%s: %d messages, %d credits, allowed %t\n", country, part.Count, part.Credits, part.Allowed)
}
for _, blocked := range preview.BlockedMessages {
    fmt.Printf("message %d to %s blocked: %s\n", blocked.Index, blocked.To, blocked.Reason)
}
for _, warning := range preview.Warnings {
    fmt.Println(warning)
}
if !preview.CanSend {
    log.Fatal("this batch would be refused")
}
```

`Retrying` counts messages the send is retrying after a temporary carrier
error; they are in neither `Sent` nor `Failed`, and their credits stay
charged. `GetBatch` and `ListBatches` fill `Delivered` and `CreditsReserved`;
a send response leaves them 0, and only a send response carries the skipped
counts and `Retrying`.

`CanSend` is true when the preview found nothing that stops a send: something
is sendable, the batch has at most 10,000 messages, nothing is blocked except
opted-out recipients (a live send skips those but rejects the whole batch for
any other block), the key has the `sms:send` scope (`HasWriteScope`), and
`HasSufficientCredits` is true unless the key is a test key. A test key's send
skips the verification and destination checks, so it can go through while
`CanSend` is false. A send can still be refused for things the preview does
not check, such as a suspended workspace or the monthly message quota, and it
charges a repeated recipient for every message where the preview counts it
once. `Compliance` breaks down the opt-out, restricted-content and quiet-hours
checks. `TotalMessages`, `WillSend`, `CurrentBalance` and `HasEnoughCredits`
are deprecated copies of `Total`, `Sendable`, `CreditBalance` and
`HasSufficientCredits`.

The API rejects a batch of more than 10,000 messages with `batch_too_large`,
and `PreviewBatch` says so in `Warnings` too.

### Group MMS

Send one MMS to 2-8 US/Canada recipients who all share a thread. Group
messaging is an A2P 10DLC capability — the sending number must be an
MMS-enabled, 10DLC-registered number you own. Omit `From` to use your
default sender.

```go
group, err := client.Messages.SendGroup(ctx, &sendly.SendGroupMessageRequest{
    To:   []string{"+14155550101", "+14155550102"},
    Text: "Dinner at 7 tonight?",
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Group message: %s (%s)\n", group.ID, group.Status)
if group.GroupMessageID != "" {
    fmt.Printf("Thread: %s\n", group.GroupMessageID)
}
for _, r := range group.Recipients {
    fmt.Printf("%s: %s\n", r.PhoneNumber, r.Status)
}
```

`To` holds the recipients' phone numbers. A live send also lists each
recipient with its status in `Recipients`; a simulated send (a test key, or
before domestic verification) sets `Simulated` and leaves `Recipients` empty.

### AI Enhance

Rewrite a draft message for clarity, compliance, and send-readiness. Provide
`Text`, `MessageType`, or both.

```go
enhanced, err := client.Messages.Enhance(ctx, &sendly.EnhanceMessageRequest{
    Text:        "hey wanna buy our stuff its on sale",
    MessageType: sendly.MessageTypeMarketing,
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(enhanced.Enhanced)    // the rewritten message
fmt.Println(enhanced.Explanation) // a short note on what changed
```

## Verification (OTP)

Send a one-time code, check it, and follow the attempt. Sandbox sends return
the code on the response so tests never need a real handset.

```go
sent, err := client.Verify.Send(ctx, &sendly.SendVerificationRequest{
    To:          "+15125550123",
    AppName:     "Acme",
    CodeLength:  6,
    TimeoutSecs: 300,
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(sent.ID, sent.Status, sent.ExpiresAt)
if sent.Sandbox {
    fmt.Println("sandbox code:", sent.SandboxCode)
}

// Check the code the user typed. A wrong code is a *sendly.ValidationError
// with Code "invalid_code"; its Extra["remaining_attempts"] says how many tries are left.
// Once the attempts run out, Check returns a *sendly.RateLimitError with Code
// "max_attempts_exceeded" straight away: send a new code.
checked, err := client.Verify.Check(ctx, sent.ID, &sendly.CheckVerificationRequest{
    Code: "123456",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(checked.Status, checked.VerifiedAt)

// Resend, read one back, or list recent attempts
resent, err := client.Verify.Resend(ctx, sent.ID)
record, err := client.Verify.Get(ctx, sent.ID)
list, err := client.Verify.List(ctx, &sendly.VerificationListOptions{Limit: 20, Status: "verified"})
for _, v := range list.Verifications {
    fmt.Println(v.ID, v.Phone, v.Status, v.DeliveryStatus)
}
```

A recipient gets at most 5 codes in 10 minutes and 20 in a day. Past that,
`Send` and `Resend` return a `*sendly.RateLimitError` with Code
`rate_limit_exceeded` and `RetryAfter` set. The client waits it out only when
the wait is a minute or less.

### Hosted verification sessions

Hand the whole flow to a Sendly-hosted page and validate the token it returns.

```go
session, err := client.Verify.Sessions.Create(ctx, &sendly.CreateSessionRequest{
    SuccessURL: "https://acme.example/verified",
    CancelURL:  "https://acme.example/cancelled",
    BrandName:  "Acme",
    BrandColor: "#FF5500",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("send the user to:", session.URL)

// After the redirect, validate the token that comes back
valid, err := client.Verify.Sessions.Validate(ctx, &sendly.ValidateSessionRequest{
    Token: "tok_from_redirect",
})
if err != nil {
    // An unknown, already-used or unverified token is a 400
    // (*sendly.ValidationError), never a response with Valid false.
    log.Fatal(err)
}
fmt.Println("verified:", valid.Phone, valid.VerifiedAt)
```

## Contacts and Lists

```go
// Create and list contacts
contact, err := client.Contacts.Create(ctx, &sendly.CreateContactRequest{
    PhoneNumber: "+15125550123",
    Name:        "Ada Lovelace",
    Email:       "ada@acme.example",
})

contacts, err := client.Contacts.List(ctx, &sendly.ListContactsRequest{
    Limit:  50,
    Search: "ada",
})
for _, c := range contacts.Contacts {
    fmt.Println(c.ID, c.PhoneNumber, c.OptedOut != nil && *c.OptedOut)
}

// Update, read and delete. Update's result leaves OptedOut and CreatedAt
// empty; Get returns the whole contact.
contact, err = client.Contacts.Update(ctx, contact.ID, &sendly.UpdateContactRequest{Name: "Ada L."})
contact, err = client.Contacts.Get(ctx, contact.ID)
err = client.Contacts.Delete(ctx, contact.ID)

// Import in bulk
imported, err := client.Contacts.Import(ctx, &sendly.ImportContactsRequest{
    Contacts: []sendly.ImportContactItem{
        {Phone: "+15125550123", Name: "Grace"},
        {Phone: "+15125550124", Name: "Alan"},
    },
    ListID: "lst_xxx",
})
fmt.Println(imported.Imported, imported.SkippedDuplicates, imported.TotalErrors)
```

Contacts are auto-flagged as invalid when a send fails with a terminal
bad-number error or a carrier lookup says they can't receive SMS. Clear the
flag one at a time or in bulk, and trigger the lookup yourself:

```go
contact, err = client.Contacts.MarkValid(ctx, contact.ID)

cleared, err := client.Contacts.BulkMarkValid(ctx, sendly.BulkMarkValidRequest{
    ListID: "lst_xxx", // or IDs: []string{...} — one or the other, not both
})
fmt.Printf("%d contacts cleared\n", cleared.Cleared)

lookup, err := client.Contacts.CheckNumbers(ctx, &sendly.CheckNumbersRequest{ListID: "lst_xxx"})
fmt.Println(lookup.Success, lookup.AlreadyRunning)
```

Lists live under `client.Contacts.Lists`:

```go
list, err := client.Contacts.Lists.Create(ctx, &sendly.CreateContactListRequest{
    Name:        "Spring campaign",
    Description: "Opted in at checkout",
})
lists, err := client.Contacts.Lists.List(ctx)
detail, err := client.Contacts.Lists.Get(ctx, list.ID)
list, err = client.Contacts.Lists.Update(ctx, list.ID, &sendly.UpdateContactListRequest{Name: "Spring 2026"})
err = client.Contacts.Lists.AddContacts(ctx, list.ID, []string{contact.ID})
err = client.Contacts.Lists.RemoveContact(ctx, list.ID, contact.ID)
err = client.Contacts.Lists.Delete(ctx, list.ID)
```

## Campaigns

Send one message to whole contact lists, with a preview that prices the send
and flags what is blocked.

```go
campaign, err := client.Campaigns.Create(ctx, &sendly.CreateCampaignRequest{
    Name:           "Spring sale",
    Text:           "20% off this week only. Reply STOP to opt out.",
    ContactListIDs: []string{"lst_xxx"},
})
if err != nil {
    log.Fatal(err)
}

preview, err := client.Campaigns.Preview(ctx, campaign.ID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%d recipients, %.0f credits, balance %d (enough: %t)\n",
    preview.RecipientCount, preview.EstimatedCredits, preview.CurrentBalance, preview.HasEnoughCredits)
fmt.Printf("left out: %d opted out, %d invalid, %d unable to receive SMS (%d landlines)\n",
    preview.OptedOutCount, preview.InvalidCount, preview.InvalidNumberCount, preview.LandlineCount)

// Send now, or schedule it (not both: a sent campaign can no longer be
// scheduled or cancelled). Send returns the campaign read back after the send:
// sent, err := client.Campaigns.Send(ctx, campaign.ID)
// fmt.Println(sent.Status, sent.BatchID, sent.SentCount, sent.FailedCount)
campaign, err = client.Campaigns.Schedule(ctx, campaign.ID, &sendly.ScheduleCampaignRequest{
    ScheduledAt: time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
    Timezone:    "America/Chicago",
})

// Follow, cancel, clone or list
campaign, err = client.Campaigns.Get(ctx, campaign.ID)
campaign, err = client.Campaigns.Cancel(ctx, campaign.ID)
copied, err := client.Campaigns.Clone(ctx, campaign.ID)
list, err := client.Campaigns.List(ctx, &sendly.ListCampaignsRequest{
    Status: sendly.CampaignStatusCompleted,
    Limit:  20,
})
fmt.Println(len(list.Campaigns), "of", list.Total)
err = client.Campaigns.Delete(ctx, campaign.ID)
```

A campaign targets one contact list: `ContactListIDs` takes a single ID, and
more than one is refused with a 400. The only placeholders a campaign fills
are `{{name}}` and `{{brand_name}}`.

`Send` makes a second request to read the campaign back (that read needs the
`campaigns:read` scope). Its `Status` is the status the send reported for the
batch: `completed`, `partial_failure`, `failed` when every message failed, or
`processing` while messages are still going out. The campaign itself is
recorded as `completed` after any accepted send, which is what `Get` returns
afterwards, and `BatchID` names the batch for `Messages.GetBatch`. If the read
fails, the send has still happened and `Send` returns no error, with a
`Campaign` holding the ID, batch, status and counts the send reported. A sent
campaign has the status `CampaignStatusCompleted`; `CampaignStatusSent` and
`CampaignStatusPaused` are deprecated, and a `sent` filter on `List` is read
as `completed`.

## Templates

```go
template, err := client.Templates.Create(ctx, &sendly.CreateTemplateRequest{
    Name: "order_shipped",
    Text: "Hi {{name}}, order {{order}} has shipped.",
})
if err != nil {
    log.Fatal(err)
}

template, err = client.Templates.Publish(ctx, template.ID)

preview, err := client.Templates.Preview(ctx, template.ID, map[string]string{
    "name":  "Sam",
    "order": "#4821",
})
fmt.Println(preview.RenderedText, preview.SegmentCount)

all, err := client.Templates.List(ctx)
presets, err := client.Templates.Presets(ctx)
one, err := client.Templates.Get(ctx, template.ID)
draft, err := client.Templates.Clone(ctx, template.ID, &sendly.CloneTemplateRequest{Name: "order_shipped_v2"})
// Published templates are locked; edit the draft copy instead
draft, err = client.Templates.Update(ctx, draft.ID, &sendly.UpdateTemplateRequest{Text: "..."})
err = client.Templates.Delete(ctx, template.ID)

// Draft one with AI from a description
generated, err := client.Templates.Generate(ctx, &sendly.GenerateTemplateRequest{
    Description: "Let a customer know their pickup order is ready",
    Category:    "transactional",
})
fmt.Println(generated.Text, generated.Variables)
```

## Conversations

Threads of inbound and outbound messages, with labels, drafts and AI helpers.

```go
convos, err := client.Conversations.List(ctx, &sendly.ListConversationsRequest{
    Status: sendly.ConversationStatusActive,
    Limit:  25,
})
for _, c := range convos.Data {
    fmt.Printf("%s: %s (%d unread)\n", c.ID, c.PhoneNumber, c.UnreadCount)
}
fmt.Println(convos.Pagination.HasMore)

// Read one with its messages
convo, err := client.Conversations.Get(ctx, "conv_xxx", &sendly.GetConversationRequest{
    IncludeMessages: true,
    MessageLimit:    50,
})
if convo.Messages != nil {
    for _, m := range convo.Messages.Data {
        fmt.Println(m.Direction, m.Text)
    }
}

// Reply, then housekeeping
reply, err := client.Conversations.Reply(ctx, convo.ID, &sendly.ReplyToConversationRequest{
    Text: "We're on it!",
})
convo2, err := client.Conversations.MarkRead(ctx, convo.ID)
convo2, err = client.Conversations.Update(ctx, convo.ID, &sendly.UpdateConversationRequest{
    Tags: []string{"vip"},
})
convo2, err = client.Conversations.Close(ctx, convo.ID)
convo2, err = client.Conversations.Reopen(ctx, convo.ID)
// AddLabels and RemoveLabel read the conversation back with a second request
// (sms:read). If that read fails, the change still stands and the returned
// Conversation carries only its ID.
convo2, err = client.Conversations.AddLabels(ctx, convo.ID, []string{"lbl_xxx"})
convo2, err = client.Conversations.RemoveLabel(ctx, convo.ID, "lbl_xxx")

// Feed a thread to your own model, or ask Sendly for replies
threadContext, err := client.Conversations.GetContext(ctx, convo.ID, &sendly.GetConversationContextRequest{
    MaxMessages: 20,
})
fmt.Println(threadContext.Context, threadContext.TokenEstimate)

suggestions, err := client.Conversations.SuggestReplies(ctx, convo.ID)
for _, s := range suggestions.Suggestions {
    fmt.Printf("[%s] %s\n", s.Tone, s.Text)
}
```

### Labels

```go
label, err := client.Labels.Create(ctx, &sendly.CreateLabelRequest{
    Name:        "Billing",
    Color:       "#FF5500",
    Description: "Payment questions",
})
labels, err := client.Labels.List(ctx)
for _, l := range labels.Data {
    fmt.Println(l.ID, l.Name, l.Color)
}
err = client.Labels.Delete(ctx, label.ID)
```

### Drafts

Queue a reply for a human to approve before it sends.

```go
draft, err := client.Drafts.Create(ctx, &sendly.CreateDraftRequest{
    ConversationId: "conv_xxx",
    Text:           "Your refund is on its way.",
})
if err != nil {
    log.Fatal(err)
}

drafts, err := client.Drafts.List(ctx, &sendly.ListDraftsRequest{
    Status: sendly.DraftStatusPending,
    Limit:  50,
})
fmt.Println(len(drafts.Data), "of", drafts.Pagination.Total)

draft, err = client.Drafts.Get(ctx, draft.ID)
draft, err = client.Drafts.Update(ctx, draft.ID, &sendly.UpdateDraftRequest{Text: "Your refund is on its way today."})
// A pending draft is either approved (which sends it) or rejected, not both:
// once one succeeds, the other answers 404.
draft, err = client.Drafts.Approve(ctx, draft.ID)
if err != nil {
    log.Fatal(err)
}
if draft.MessageId != nil {
    fmt.Println("sent as", *draft.MessageId)
}
// Or turn it down instead:
// draft, err = client.Drafts.Reject(ctx, draft.ID, "Wrong tone")
```

### Rules

Auto-label inbound conversations from their AI classification (intent and
sentiment).

```go
priority := 10
rule, err := client.Rules.Create(ctx, &sendly.CreateRuleRequest{
    Name:       "Flag complaints",
    Conditions: map[string]interface{}{"intent": "complaint", "sentiment": "negative"},
    Actions:    map[string]interface{}{"addLabels": []string{"lbl_xxx"}},
    Priority:   &priority,
})
rules, err := client.Rules.List(ctx)
rule, err = client.Rules.Update(ctx, rule.ID, &sendly.UpdateRuleRequest{Name: "Flag unhappy customers"})
err = client.Rules.Delete(ctx, rule.ID)
```

## WhatsApp

Connect a number you own to WhatsApp, create Meta-reviewed message templates,
and send with `client.Messages.SendWhatsApp`. Connecting is a one-time $19
setup (no monthly fee). The first number always ends with a human step: the
connect URL must be opened in a browser and completed with a Facebook login.
Further numbers can join that WhatsApp Business Account with a code instead
(see [Adding a number by code](#adding-a-number-by-code)). Free-form text and
media only deliver inside an open 24-hour customer-service window — outside
it, send an approved template.

Sends go through `client.Messages.SendWhatsApp` (`POST /v1/messages` with
channel `whatsapp`) and need `sms:send`, not `whatsapp:write`. Reads
(`Signup.Get`, templates, the window, senders and sender profiles) need
`whatsapp:read` and accept test keys. Signup, template create/edit/delete and
profile edits need `whatsapp:write` and a live key (otherwise 403
`whatsapp_requires_live_key`). Sends need a live key too. In a team
workspace, connecting and profile edits need an owner or admin
(`settings:write`), and template writes need an owner, admin or member
(`templates:write`). A missing role returns 403 `insufficient_permissions`.

WhatsApp is enabled per person: the user who owns the API key, not the
workspace. While it is off, sends return 403 `whatsapp_not_enabled` and the
`/api/v1/whatsapp/*` management routes return 404 `not_found`.

While connections are unavailable, `Signup.Create` gets a 503
`whatsapp_unavailable` before anything is charged, with `retryAfter: 3600` in
the body and a `Retry-After: 3600` header (the client retries it like any 5xx
before returning a `*sendly.SendlyError`). Only signup returns it; no send
does. After 5 failed, charged signups in 24 hours it gets a 429
`whatsapp_signup_limit_reached`, which is final for the day. If the connection
fails, the $19 fee is refunded automatically; once a number has connected,
a later disconnect gets nothing back.

```go
// 1. Connect a number ($19 one-time). A human must open the connect URL.
signup, err := client.WhatsApp.Signup.Create(ctx, "+15125550123")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Have your user open: %s\n", signup.ConnectURL)

// 2. Poll until active. After the Facebook step the signup stays
//    "registering" while WhatsApp activates the number. Activation usually
//    takes a few minutes but can take hours. If it hasn't finished about 6
//    hours after the session began, the session fails with
//    registration_timeout and the fee is refunded. "failed" sets
//    FailureReasons.
status, err := client.WhatsApp.Signup.Get(ctx, signup.ID)
fmt.Println(status.Status) // "initiated" -> "registering" -> "active"

// 3. List your WhatsApp senders
senders, err := client.WhatsApp.Senders.List(ctx)
for _, s := range senders.Senders {
    fmt.Printf("%s: %s\n", s.PhoneNumber, s.Status)
}

// 4. Create a template (Meta reviews it, usually 24-48h). Category is
//    required (UTILITY, AUTHENTICATION or MARKETING) with no default, and
//    Update can't change it.
template, err := client.WhatsApp.Templates.Create(ctx, &sendly.CreateWhatsAppTemplateRequest{
    Sender:   "+15125550123",
    Name:     "order_shipped",
    Language: "en_US",
    Category: "UTILITY",
    Body:     "Hi {{1}}, your order {{2}} has shipped!",
    Examples: map[string]string{"1": "Sam", "2": "#4821"},
})
fmt.Println(template.Status) // "PENDING"

// 5. Check the 24-hour window, then send
window, err := client.WhatsApp.Window(ctx, "+15125550123", "+15125550100")
if err != nil {
    log.Fatal(err)
}
req := &sendly.SendWhatsAppMessageRequest{
    To:   "+15125550100",
    From: "+15125550123",
}
if window.Open {
    // Free-form text (or media with a caption via MediaUrls + Text)
    req.Text = "Your table is ready!"
} else {
    // Approved template — works regardless of the window
    req.Template = &sendly.WhatsAppTemplateSendParams{
        Name:      "order_shipped",
        Language:  "en_US",
        Variables: map[string]string{"1": "Sam", "2": "#4821"},
    }
}
msg, err := client.Messages.SendWhatsApp(ctx, req)
if err != nil {
    var sendlyErr *sendly.SendlyError
    if errors.As(err, &sendlyErr) {
        switch sendlyErr.Code {
        case sendly.WhatsAppErrorCodeSendFailed:
            // 502: the message provably never reached the carrier, so it was
            // not sent and is safe to send again.
            log.Fatal("not sent, safe to send again: ", sendlyErr.Message)
        case sendly.WhatsAppErrorCodeSendUnconfirmed:
            // 409: the outcome is unknown. The message was marked failed and
            // refunded but may still be delivered, so check before sending
            // it again (it could arrive twice).
            log.Fatal("WhatsApp send unconfirmed; check before resending: ", sendlyErr.Message)
        }
    }
    // A 422 whatsapp_send_failed (*sendly.ValidationError) means WhatsApp
    // refused the message; it is final, nothing was charged, and the API
    // replays it for 24 hours under the same idempotency key.
    log.Fatal(err)
}
fmt.Println(msg.ID, msg.WhatsApp.Kind, msg.CreditsUsed) // kind is "text", "media" or "template"
```

A failed send is one of three errors:

- 422 `whatsapp_send_failed`: WhatsApp refused the message. It is final,
  nothing was charged, and the API replays it for 24 hours under the same
  idempotency key.
- 502 `whatsapp_send_failed`: the message provably never reached the
  carrier, so it was not sent and is safe to send again. It is never cached,
  so the client retries it like any 5xx under the same idempotency key.
- 409 `whatsapp_send_unconfirmed`: the outcome is unknown. The message was
  marked failed and refunded but may still be delivered, so check before
  sending it again (it could arrive twice). It is not retried automatically.

Pricing: free-form text or media inside the 24-hour window costs 1 credit
each for the first 1,000 per sending number per calendar month (UTC), then
the destination's utility template price; countries without a listed price
use the default utility price of 12 credits. Templates are priced by category
and destination country; countries without a listed price use 33
(marketing), 12 (utility) and 12 (authentication) credits. A failed send
gives its slot back.

`Window` returns exactly `{ open, expiresAt }`: with no window on record
`Open` is false and `ExpiresAt` is nil; after a window has expired `Open` is
false and `ExpiresAt` is the past expiry.

Template pre-flight refusals are 400s: `template_category_invalid` (category
missing or not one of the three), `template_authentication_otp_button_required`,
`template_authentication_no_links` (a link in the body or a URL button on an
authentication template) and `template_header_variable_unsupported` (a header
is fixed text, since sends fill only body and button variables). On create,
404 `whatsapp_sender_not_connected` is checked first. A marketing template
without an opt-out button only gets a warning.

Templates can be listed, edited and deleted. Editing is the recovery path for
a rejection: Meta locks a deleted template's name for about 30 days, so fix a
`REJECTED` template with `Update` rather than deleting and re-creating it.

```go
templates, err := client.WhatsApp.Templates.List(ctx)
for _, t := range templates.Templates {
    fmt.Println(t.Name, t.Language, t.Status)
    if t.RejectionReason != nil {
        fmt.Println("rejected:", *t.RejectionReason)
    }
}

fixed, err := client.WhatsApp.Templates.Update(ctx, template.ID, &sendly.UpdateWhatsAppTemplateRequest{
    Body: "Hi {{1}}, order {{2}} is on its way.",
})
deleted, err := client.WhatsApp.Templates.Delete(ctx, template.ID)
fmt.Println(deleted.ID, deleted.Deleted)
```

Every connected sender has a WhatsApp Business profile — the name, photo, and
business details recipients see when they tap your number. Read it and edit it
in place (send only the fields you want to change; `About` is capped at 139
characters and `Description` at 512):

```go
profile, err := client.WhatsApp.Senders.GetProfile(ctx, "+15125550123")
if err != nil {
    log.Fatal(err)
}
if profile.DisplayName != nil {
    fmt.Println(*profile.DisplayName)
}

updated, err := client.WhatsApp.Senders.UpdateProfile(ctx, "+15125550123", &sendly.UpdateWhatsAppSenderProfileRequest{
    About:       "Fresh bread, daily.",
    Description: "Family bakery in Austin since 1998.",
    Email:       "hello@acme.example",
    Website:     "https://acme.example",
})
```

Set or remove the profile photo. It must be a JPEG or PNG of at most 5 MB
(checked by its bytes), and WhatsApp wants it square and at least 192 pixels
wide (640 recommended). A 502 `whatsapp_profile_update_failed` means WhatsApp
refused it or couldn't be reached. The upload is sent once and never
retried, not even by Go's HTTP transport when a reused keep-alive connection
drops.

```go
photo, err := os.Open("logo.jpg")
if err != nil {
    log.Fatal(err)
}
defer photo.Close()
profile, err = client.WhatsApp.Senders.UploadProfilePhoto(ctx, "+15125550123", "logo.jpg", photo)
if err != nil {
    log.Fatal(err)
}
if profile.ProfilePhotoURL != nil {
    fmt.Println(*profile.ProfilePhotoURL)
}

profile, err = client.WhatsApp.Senders.DeleteProfilePhoto(ctx, "+15125550123") // ProfilePhotoURL is normally nil
```

Ice breakers are tappable suggestions shown when someone opens a chat with
the business for the first time (at most 4, each up to 80 characters).
Commands are shown when the customer types "/" (at most 30; letters, digits
and underscores, up to 32 characters, with a description up to 256). Each
list you set replaces the stored one, an empty list clears it, and a nil list
is left alone:

```go
components, err := client.WhatsApp.Senders.GetConversationalComponents(ctx, "+15125550123")
fmt.Println(components.IceBreakers, components.Commands)

components, err = client.WhatsApp.Senders.UpdateConversationalComponents(ctx, "+15125550123", &sendly.UpdateWhatsAppConversationalComponentsRequest{
    IceBreakers: []string{"What are your opening hours?", "Book a table"},
    Commands: []sendly.WhatsAppCommand{
        {Command: "menu", Description: "See today's menu"},
    },
})

// Clear the commands, keep the ice breakers
components, err = client.WhatsApp.Senders.UpdateConversationalComponents(ctx, "+15125550123", &sendly.UpdateWhatsAppConversationalComponentsRequest{
    Commands: []sendly.WhatsAppCommand{},
})
```

With WhatsApp calling on, a WhatsApp user calling the number rings exactly
like a phone call (a dashboard ring or an AI agent, per the number's voice
settings), billed at the normal inbound rate. Voice must be on for the number
first, otherwise a 409 `voice_not_enabled`. A 422
`whatsapp_calling_unavailable` means Meta refused: it enables calling only once
the account may message at least 2,000 people a day and the number's display
name is approved. There is no API for placing WhatsApp calls. Senders report
`CallingEnabled` and `OutboundCallingAllowed` (false for +1 numbers, which
covers the US, Canada and the rest of the North American numbering plan, and
for +20 Egypt, +84 Vietnam and +234 Nigeria, where Meta forbids
business-initiated calls). Calls report `Channel` (`sendly.CallChannelWhatsApp` for a WhatsApp
call).

```go
calling, err := client.WhatsApp.Senders.SetCalling(ctx, "+15125550123", true)
fmt.Println(calling.CallingEnabled, calling.OutboundCallingAllowed)
```

Profile photos, ice breakers and commands, and the calling switch are free.
Their writes need `whatsapp:write`, a live key and, in a team workspace, an
owner or admin (`settings:write`); `GetConversationalComponents` needs
`whatsapp:read` and accepts test keys. A number that isn't connected to
WhatsApp gets 404 `whatsapp_sender_not_connected`.

### Adding a number by code

Once a WhatsApp Business Account is connected, add more of your numbers to it
without the Facebook step: pass its `BusinessAccountID` (from
`Senders.List`) to `Signup.CreateWithOptions`, and WhatsApp sends the number a
6-digit code by text (or a voice call with `VerificationMethod: "voice"`). The
same $19 one-time fee applies, charged before the code is requested and
refunded automatically if the signup fails. The signup comes back
`"verifying"`, with no connect URL. While it waits, `Signup.Get` returns the
code in `VerificationCode` once its text has arrived on the number, so you can
submit it without reading the phone.

```go
signup, err := client.WhatsApp.Signup.CreateWithOptions(ctx, &sendly.CreateWhatsAppSignupRequest{
    PhoneNumber:       "+15125550199",
    BusinessAccountID: "102938475610293", // a sender's BusinessAccountID from Senders.List
    DisplayName:       "Acme Bakery",     // optional: defaults to the account's existing name
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(signup.Status) // "verifying"

status, err := client.WhatsApp.Signup.Get(ctx, signup.ID)
if err != nil {
    log.Fatal(err)
}
if status.VerificationCode != nil {
    status, err = client.WhatsApp.Signup.Verify(ctx, signup.ID, *status.VerificationCode)
    var vErr *sendly.ValidationError
    if errors.As(err, &vErr) && vErr.Code == sendly.WhatsAppErrorCodeVerificationCodeInvalid {
        var left int
        json.Unmarshal(vErr.Extra["attemptsRemaining"], &left)
        log.Fatal("wrong code, attempts left: ", left)
    } else if err != nil {
        log.Fatal(err)
    }
    fmt.Println(status.Status) // "active"
}

// No code yet? Ask for another, at least 30 seconds after the last change.
status, err = client.WhatsApp.Signup.Resend(ctx, signup.ID, sendly.WhatsAppVerificationMethodVoice)
```

- The account must be connected in this workspace with at least one active
  number, otherwise 404 `whatsapp_business_account_not_found`. Without
  `DisplayName` the account's existing sender display name is used, else its
  business name, else 400 `display_name_required`. A Facebook connection
  already in flight for the number is a 409 `whatsapp_signup_in_progress`.
- WhatsApp refusing to send the code is a 422
  `whatsapp_verification_start_failed` and being unreachable a 502 with the
  same code; either way the signup failed and the fee is refunded, so start
  again. `CreateWithOptions` returns a 5xx, a timeout or a network error here
  without retrying, because each retry could start, and charge, a new
  signup. Check `Senders.List`, or call again with the same
  `WithIdempotencyKey`.
- Calling it again for a number that is verifying returns the same signup,
  with no new charge and no second code. A verifying signup that began more
  than 3 hours ago is expired and refunded, and a new one is started.
  `Signup.Create` (the Facebook flow) for that number gets a 409
  `whatsapp_verification_in_progress`, with the signup's `id` in `Extra`.
- A wrong code is a 422 `whatsapp_verification_code_invalid` with
  `attemptsRemaining`; after 5 wrong codes the signup fails with 409
  `whatsapp_verification_failed` and the fee is refunded. 502
  `whatsapp_verification_unavailable` doesn't count the attempt; retry. 502
  `whatsapp_activation_pending` means the code was accepted but connecting
  the number didn't finish; Sendly is alerted, so check it with `Signup.Get`
  shortly. `Verify` returns a 5xx, a timeout or a network error without
  retrying, because each attempt submits the code again; check the signup
  with `Signup.Get` before submitting again.
- `Resend` sooner than 30 seconds after the signup last changed (a code
  submission counts) is a 429 `whatsapp_verification_resend_too_soon`,
  returned at once as a `*sendly.RateLimitError` with `RetryAfter`. An empty
  method sends `"sms"`, whatever the previous method was.
- A connected number fires `whatsapp_account.connected`, and a failed signup
  `whatsapp_account.failed`, as for a Facebook connection. New failure
  reasons: `verification_start_failed`, `verification_failed` and
  `verification_expired` (the signup was untouched for an hour, or began more
  than 3 hours ago when the number was added again).

## RCS

RCS is the branded, rich upgrade to SMS: your verified agent name and logo
instead of a bare number, plus tappable suggestion chips and rich cards, on
Android and iOS 18+ handsets. Messages send through an RCS agent registered
for your brand. Registration is self-serve, from the dashboard or this SDK
(see [Registering an agent](#registering-an-agent)): Sendly reviews each
submission first, then the carrier network does. Sending RCS requires a live
API key — a test key is refused with `403 rcs_requires_live_key` rather than
simulated.

Text sends fall back to plain SMS automatically when the recipient's device or
network doesn't support RCS, so one call covers your whole list. The fallback
is billed as SMS and is visible on the response: check `FellBackTo` (or
`Channel`).

```go
// 1. Find your agents. Sendable means it can send right now.
agents, err := client.RCS.Agents.List(ctx)
if err != nil {
    log.Fatal(err)
}
for _, a := range agents.Agents {
    // Status is "draft", "submitted", "testing", "approved" or "suspended";
    // "testing" agents reach invited test devices only.
    fmt.Printf("%s: %s %s (sendable: %t)\n", a.ID, a.Name, a.Status, a.Sendable)
}

// 2. Optional pre-flight — sending handles the fallback on its own.
capability, err := client.RCS.Capability(ctx, "+15125550123", "")
fmt.Println(capability.Capable, capability.Features)

// 3. Text with tappable chips
msg, err := client.Messages.SendRcs(ctx, &sendly.SendRcsMessageRequest{
    To:   "+15125550123",
    Text: "Your order #4821 has shipped!",
    Suggestions: []sendly.RcsSuggestion{
        {Reply: &sendly.RcsSuggestedReply{Text: "Track it", PostbackData: "track_4821"}},
        {Action: &sendly.RcsSuggestedAction{
            Text:         "View receipt",
            PostbackData: "receipt_4821",
            URL:          "https://acme.example/receipts/4821",
        }},
    },
})
if err != nil {
    log.Fatal(err)
}

if msg.FellBackTo == "sms" {
    // Delivered as SMS — suggestions have no SMS form and were dropped.
    fmt.Println("fell back to SMS:", msg.RCS.SuggestionsDropped)
} else {
    fmt.Println("delivered over RCS from", msg.RCS.AgentName)
}

// 4. A rich card. Cards have no SMS form, so they never fall back —
//    a recipient without RCS gets a 422 (rcs_not_supported_for_recipient).
card, err := client.Messages.SendRcs(ctx, &sendly.SendRcsMessageRequest{
    To: "+15125550123",
    Card: &sendly.RcsCard{
        Title:       "Your table is ready",
        Description: "Head to the host stand — we'll hold it for 10 minutes.",
        MediaURL:    "https://acme.example/table.jpg",
        Orientation: "vertical",
        Suggestions: []sendly.RcsSuggestion{
            {Reply: &sendly.RcsSuggestedReply{Text: "On my way", PostbackData: "otw"}},
        },
    },
})
fmt.Println(card.RCS.Kind) // "card"

// Turn the fallback off to require RCS delivery (422 when unsupported).
rcsOnly := false
_, err = client.Messages.SendRcs(ctx, &sendly.SendRcsMessageRequest{
    To:            "+15125550123",
    Text:          "RCS only.",
    FallbackToSms: &rcsOnly,
})
```

### Registering an agent

Register from the dashboard or right here. Draft a brand (the business behind
the agent) and an agent (what recipients see), invite a few test devices, then
submit: Sendly reviews the submission, sends it on to the carrier network for
verification, and the agent enters testing. Once you have messaged an invited
device, fill in the campaign and request launch; after the launch review the
agent goes live. Follow progress with `client.RCS.Registration.Get` (or
`client.RCS.Agents.Get` for one agent), and read `ReviewNote` when changes are
requested.

Two things to know: registration is US-only for now (the brand address must be
in the US), and logo, hero and call-to-action media must be public `https://`
URLs. Uploading assets is dashboard-only, so host the files yourself or upload
them in the dashboard and use the URLs it gives you. Reads need the `rcs:read`
scope and writes `rcs:write`; every registration call answers
`404 rcs_not_enabled` until RCS is switched on for the account.

```go
// 1. Start from what Sendly already knows about the business.
dossier, err := client.RCS.Dossier.Get(ctx)
if err != nil {
    log.Fatal(err)
}
input := dossier.Brand
input.DisplayName = "Acme"
input.LegalEntityType = "LIMITED_LIABILITY_COMPANY"
if input.Address == nil {
    input.Address = &sendly.RcsBrandAddress{}
}
input.Address.CountryCode = "US"

brand, err := client.RCS.Brands.Create(ctx, &input)
if err != nil {
    log.Fatal(err)
}

// 2. Draft the agent recipients will see. Media must be public https URLs.
agent, err := client.RCS.Agents.Create(ctx, &sendly.CreateRcsAgentRequest{
    BrandID:     brand.Brand.ID,
    DisplayName: "Acme Support",
    UseCase:     "MULTI_USE",
    Basics: &sendly.RcsAgentBasics{
        Description:           "Order updates and support from Acme.",
        LogoURL:               "https://acme.example/rcs/logo.png",
        HeroURL:               "https://acme.example/rcs/hero.png",
        BrandColor:            "#FF5500",
        PrivacyPolicyURL:      "https://acme.example/privacy",
        TermsAndConditionsURL: "https://acme.example/terms",
        PhoneNumber:           &sendly.RcsAgentPhoneContact{Number: "+15125550123", Label: "Support"},
        Website:               &sendly.RcsAgentWebsiteContact{URL: "https://acme.example", Label: "Acme"},
    },
})
if err != nil {
    log.Fatal(err)
}

// 3. Invite test devices (the list is authoritative) and submit for review.
_, err = client.RCS.Agents.SetTestDevices(ctx, agent.Agent.ID, []sendly.RcsTestDeviceInput{
    {PhoneNumber: "+15125550123", Label: "Ada's phone"},
})
if err != nil {
    log.Fatal(err)
}
submitted, err := client.RCS.Agents.Submit(ctx, agent.Agent.ID)
if err != nil {
    if validationErr, ok := err.(*sendly.ValidationError); ok {
        for _, fieldErr := range validationErr.Errors {
            fmt.Println(fieldErr.Path, fieldErr.Message) // e.g. "brand.ein Enter a 9-digit EIN"
        }
    }
    log.Fatal(err)
}
fmt.Println(submitted.Stage) // "in_review"

// 4. Follow progress. Once the stage is "testing", message an invited device,
//    fill in the campaign, and request launch.
reg, err := client.RCS.Registration.Get(ctx)
if err != nil {
    log.Fatal(err)
}
if reg.Stage == sendly.RcsCustomerStageTesting {
    _, err = client.RCS.Agents.Update(ctx, reg.Agent.ID, &sendly.UpdateRcsAgentRequest{
        Campaign: &sendly.RcsCampaign{
            CompanyOverview: "Family bakery in Austin since 1998.",
            AgentOverview:   "Order confirmations, pickup reminders and support replies.",
            Interactions: []sendly.RcsInteraction{
                {InteractionType: "TRANSACTIONAL_UPDATES", Description: "Order and pickup updates"},
            },
            MessageExamples: []string{
                "Your order #4821 is confirmed.",
                "Your order is ready for pickup.",
                "Reply STOP to opt out.",
            },
            ConsentSettings: &sendly.RcsConsentSettings{
                OptInMethods:   []sendly.RcsOptInMethod{{MethodType: "WEBSITE", Description: "Checkbox at checkout"}},
                OptInMessage:   "Acme: you're in. Reply STOP to opt out.",
                HelpResponse:   "Acme support: hello@acme.example",
                OptOutResponse: "You've been opted out.",
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    launch, err := client.RCS.Agents.RequestLaunch(ctx, reg.Agent.ID, &sendly.RcsLaunchRequest{
        TestURL: "https://acme.example/rcs-test-notes",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(launch.Stage) // "launch_review"
}
```

Stages are the `sendly.RcsCustomerStage*` constants: `draft`, `in_review`,
`changes_requested`, `rejected`, `brand_verification`, `agent_review`,
`testing`, `launch_review`, `launching`, `launch_rejected`, `live`,
`suspended` and `failed`. The review state of a single brand or agent is a
`sendly.RcsReviewStatus*` constant. Refusals carry one of the
`sendly.RcsErrorCode*` codes — `rcs_not_enabled`, `rcs_not_found`,
`rcs_field_locked`, `rcs_us_only`, `rcs_invalid_content`,
`rcs_brand_not_verified`, `rcs_launch_not_ready` and `rcs_internal_error`.

Brands and agents lock while under review (`409 rcs_field_locked`) and unlock
again if Sendly requests changes. Pass `sendly.WithIdempotencyKey` on any
write to make it safe to retry across process restarts.

## Short Codes

The SDK has **no typed helpers for short codes**. Apply for one in the
dashboard, or call the REST endpoints directly with your own HTTP client and
the same `Authorization: Bearer <key>` header:

| Method | Path | Scope |
|--------|------|-------|
| `GET` | `/api/v1/short_codes` | `short_codes:read` |
| `POST` | `/api/v1/short_codes/requests` | `short_codes:write` |
| `GET` | `/api/v1/short_codes/application` | `short_codes:read` |
| `PUT` | `/api/v1/short_codes/application` | `short_codes:write` |
| `POST` | `/api/v1/short_codes/application/preflight` | `short_codes:read` |
| `POST` | `/api/v1/short_codes/application/submit` | `short_codes:write` |

Progress does reach you through webhooks: subscribe to
`sendly.WebhookEventShortCodeActionRequired`,
`sendly.WebhookEventShortCodeRejected`, `sendly.WebhookEventShortCodeFiled`
and `sendly.WebhookEventShortCodeLive`, and read the payload with
`event.DecodeObject(&v)` as with any other lifecycle event.

## Local Numbers (10DLC)

Register your business so you can text from local US numbers. The flow is
brand → campaign → number assignment, and the writes need a live API key.

```go
// 1. Register the brand and poll until it is verified
brand, err := client.TenDlc.CreateBrand(ctx, &sendly.CreateTenDlcBrandRequest{
    LegalName:  "Acme LLC",
    EIN:        "123456789",
    Website:    "https://acme.example",
    Email:      "hello@acme.example",
    Street:     "500 Example Ave",
    City:       "Austin",
    State:      "TX",
    PostalCode: "78701",
})
if err != nil {
    log.Fatal(err)
}
status, err := client.TenDlc.GetBrand(ctx, brand.Data.ID)
fmt.Println(status.Data.Status) // "pending" -> "verified" / "failed"

// 2. Check a use case qualifies, then create the campaign
qualified, err := client.TenDlc.Qualify(ctx, brand.Data.ID, "MIXED")
fmt.Println(qualified.Data.Qualified)
if qualified.Data.Reason != nil {
    fmt.Println(*qualified.Data.Reason)
}

campaign, err := client.TenDlc.CreateCampaign(ctx, &sendly.CreateTenDlcCampaignRequest{
    BrandID:     brand.Data.ID,
    UseCase:     "MIXED",
    Description: "Order updates and appointment reminders.",
    MessageFlow: "Customers opt in at checkout with a consent checkbox.",
    SampleMessages: []string{
        "Your order #4821 is confirmed.",
        "Reminder: your appointment is tomorrow at 3pm.",
    },
})

// 3. Assign a number you own once the campaign is active
assignment, err := client.TenDlc.AssignNumber(ctx, campaign.Data.ID, "+15125550123")
fmt.Println(assignment.Data.Status) // "Active" once the number can send

brands, err := client.TenDlc.ListBrands(ctx)
campaigns, err := client.TenDlc.ListCampaigns(ctx)
assignments, err := client.TenDlc.ListAssignments(ctx)
```

## Numbers

```go
// Browse what you can buy, then search live inventory
countries, err := client.Numbers.ListCountries(ctx)
for _, c := range countries.Countries {
    fmt.Println(c.Code, c.Name, c.NumberTypes)
}

available, err := client.Numbers.ListAvailable(ctx, &sendly.ListAvailableRequest{
    Country:  "US",
    Type:     "local",
    Contains: "512",
})
for _, n := range available.Numbers {
    fmt.Printf("%s: %s %s\n", n.PhoneNumber, n.MonthlyCost, n.Currency)
}

// Buy one. Some orders need documents or a payment method first: the
// response then carries an Action to hand to a human.
purchase, err := client.Numbers.Buy(ctx, &sendly.BuyNumberRequest{
    PhoneNumber:     "+15125550123",
    CountryCode:     "US",
    PhoneNumberType: "local",
})
if err != nil {
    log.Fatal(err)
}
if purchase.Action != nil {
    fmt.Printf("Open %s and enter code %s\n", purchase.Action.URL, purchase.Action.Code)
    // then call Buy again with ActionCode: purchase.Action.ActionCode
}

// List the numbers attached to your workspace
owned, err := client.Numbers.List(ctx)
for _, n := range owned.Numbers {
    fmt.Printf("%s: %s (%s)\n", n.ID, n.PhoneNumber, n.Status)
    // VoiceEnabled and VoiceMode ("none", "ring_dashboard", "agent") say
    // whether the number can take and place phone calls (see Voice Calls).
    if n.VoiceEnabled != nil && *n.VoiceEnabled && n.VoiceMode != nil {
        fmt.Printf("  voice: %s\n", *n.VoiceMode)
    }
}

// Get a single number (includes whether it is your default sender)
number, err := client.Numbers.Get(ctx, "num_xxx")
if number.IsDefault != nil && *number.IsDefault {
    fmt.Println("This is the default sender")
}

// Make a number your default sender (must be active)
isDefault := true
updated, err := client.Numbers.Update(ctx, "num_xxx", &sendly.UpdateNumberRequest{
    IsDefault: &isDefault,
})
fmt.Printf("Default: %v\n", *updated.IsDefault)

// Cancel a scheduled release ("keep this number")
keep := false
_, err = client.Numbers.Update(ctx, "num_xxx", &sendly.UpdateNumberRequest{
    PendingCancellation: &keep,
})

// Release a number. A live paid purchase is cancelled at the end of the paid
// period, in which case the response is scheduled rather than immediate.
result, err := client.Numbers.Release(ctx, "num_xxx")
if result.Scheduled {
    fmt.Printf("Releases at %s\n", *result.ScheduledReleaseAt)
} else {
    fmt.Println("Released")
}
```

## Toll-Free Entity Upgrade

When a customer forms a new legal entity, this flow reserves a new toll-free
number under it, submits it for carrier review, and swaps to it on approval —
the old number keeps sending throughout the 1-2 week review.

```go
report, err := client.BusinessUpgrade.Preflight(ctx, &sendly.PreflightCandidate{
    BusinessName: "Acme LLC",
    BRN:          "123456789",
    BRNType:      "EIN",
    BRNCountry:   "US",
    EntityType:   "PRIVATE_PROFIT",
    Website:      "https://acme.example",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(report.Verdict) // "ready", "warnings" or "blocked"
for _, issue := range report.Issues {
    fmt.Println(issue.Severity, issue.Field, issue.Message)
}

started, err := client.BusinessUpgrade.Start(ctx, "ws_xxx", &sendly.StartUpgradeParams{
    BusinessName: "Acme LLC",
    BRN:          "123456789",
    BRNType:      "EIN",
    BRNCountry:   "US",
    EntityType:   "PRIVATE_PROFIT",
}, nil) // pass an *EinDocument to attach a CP-575 PDF
fmt.Println(started.PendingVerificationID, started.Message) // the new number is reserved in the background; read it later from Status

state, err := client.BusinessUpgrade.Status(ctx, "ws_xxx")
if state.Pending != nil {
    fmt.Println(state.Pending.Status)
    if state.Pending.RejectionReason != nil {
        fmt.Println(*state.Pending.RejectionReason)
    }
}

// Fix a rejection with a partial resubmit, or back out entirely
resubmitted, err := client.BusinessUpgrade.Resubmit(ctx, "ws_xxx", &sendly.StartUpgradeParams{
    Website: "https://acme.example/about",
}, nil)
cancelled, err := client.BusinessUpgrade.Cancel(ctx, "ws_xxx")

// After approval, decide what happens to the old number
disposition, err := client.BusinessUpgrade.SetDisposition(ctx, "ws_xxx", &sendly.SetDispositionRequest{
    Disposition:       "moved",
    TargetWorkspaceID: "ws_other",
})
fmt.Println(disposition.Disposition)

prefill, err := client.BusinessUpgrade.BestPrefill(ctx)
fmt.Println(prefill.Prefill.UseCase, prefill.SourceWorkspaceCount)
```

## Webhooks

```go
// Create a webhook endpoint
webhook, err := client.Webhooks.Create(ctx, sendly.CreateWebhookRequest{
    URL:    "https://acme.example/webhooks/sendly",
    Events: []string{"message.delivered", "message.failed"},
    Mode:   sendly.WebhookModeAll, // or WebhookModeTest / WebhookModeLive
})
fmt.Printf("Webhook ID: %s\n", webhook.ID)
fmt.Printf("Secret: %s\n", webhook.Secret) // Store securely!

// List all webhooks
webhooks, err := client.Webhooks.List(ctx)

// Get a specific webhook
wh, err := client.Webhooks.Get(ctx, "whk_xxx")

// Update a webhook
newURL := "https://hooks.acme.example/sendly"
client.Webhooks.Update(ctx, "whk_xxx", sendly.UpdateWebhookRequest{
    URL:    &newURL,
    Events: []string{"message.delivered", "message.failed", "message.sent"},
})

// Send a test event. A failed test, including one for a webhook that does
// not exist, is a *sendly.ValidationError whose Message says why.
result, err := client.Webhooks.Test(ctx, "whk_xxx")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Message) // e.g. "Test webhook delivered successfully in 87ms"
if result.StatusCode != nil && result.ResponseTimeMs != nil {
    fmt.Println(*result.StatusCode, *result.ResponseTimeMs)
}

// Rotate the signing secret. Deliveries are signed with the new secret as
// soon as this returns, so have your endpoint accept both secrets while you
// deploy the new one.
rotation, err := client.Webhooks.RotateSecret(ctx, "whk_xxx")
fmt.Println(rotation.Webhook.ID, rotation.NewSecret, rotation.RotatedAt)

// Delete a webhook
err = client.Webhooks.Delete(ctx, "whk_xxx")
```

Webhook ids start with `whk_` and delivery ids with `del_`; the SDK checks
both client-side and returns an error before making the request if you pass
anything else. `Create` returns a `*WebhookCreatedResponse`, which embeds the
`Webhook` (so `webhook.ID` works) and adds `Secret` — the only time the
signing secret is shown.

### Deliveries and recovery

```go
// Delivery history, newest first (GetDeliveries returns the latest 50), and a
// retry of one failed attempt
deliveries, err := client.Webhooks.GetDeliveriesWithOptions(ctx, "whk_xxx", &sendly.ListWebhookDeliveriesOptions{
    Status: sendly.DeliveryStatusFailed,
    Limit:  100,
    Offset: 0,
})
for _, d := range deliveries {
    fmt.Println(d.ID, d.EventType, d.Status, d.AttemptNumber, "/", d.MaxAttempts)
    if d.ResponseTimeMs == nil {
        fmt.Println("  no response (timed out or could not connect)")
    }
}
err = client.Webhooks.RetryDelivery(ctx, "whk_xxx", "del_xxx")

// Repeated failures trip a circuit breaker; close it again before replaying
reset, err := client.Webhooks.ResetCircuit(ctx, "whk_xxx")

// Replay failed or cancelled deliveries from the audit log
limit := 500
replayed, err := client.Webhooks.Redeliver(ctx, "whk_xxx", &sendly.RedeliverOptions{
    Statuses: []string{"failed", "cancelled"},
    Limit:    &limit,
})
fmt.Println(replayed.Requeued, replayed.Skipped)

// Synthesize deliveries for events that never got an audit row at all
filled, err := client.Webhooks.Backfill(ctx, "whk_xxx", &sendly.BackfillOptions{
    EventTypes: []string{"message.delivered"},
})
fmt.Println(filled.Synthesized, filled.ByType)

// Everything you can subscribe to
types, err := client.Webhooks.ListEventTypes(ctx)
```

Both `Redeliver` and `Backfill` are rejected with `409` while the circuit is
open, so call `ResetCircuit` first. Redelivered and backfilled events carry
the event id the original dispatch used, so dedupe on `event.ID`. Do not
dedupe on `data.object.id`: a message's sent and delivered events share it.

### Receiving events

`sendly.Webhooks{}.ParseEvent` verifies the signature and returns the event. It
is a standalone helper, not a method on `client.Webhooks`. It takes the raw
request body plus the `X-Sendly-Signature` and `X-Sendly-Timestamp` headers
Sendly sends with every delivery, in that order: **payload, signature, secret,
timestamp**.

```go
import (
    "fmt"
    "io"
    "log"
    "net/http"
    "os"

    "github.com/SendlyHQ/sendly-go/v4/sendly"
)

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    event, err := sendly.Webhooks{}.ParseEvent(
        string(body),
        r.Header.Get("X-Sendly-Signature"),
        os.Getenv("SENDLY_WEBHOOK_SECRET"),
        r.Header.Get("X-Sendly-Timestamp"),
    )
    if err != nil {
        http.Error(w, "invalid signature", http.StatusBadRequest)
        return
    }

    switch event.Type {
    case sendly.WebhookEventMessageDelivered:
        fmt.Printf("%s delivered to %s\n", event.Data.ID, event.Data.To)
    case sendly.WebhookEventMessageFailed:
        fmt.Printf("%s failed: %s\n", event.Data.ID, event.Data.Error)
    }

    w.WriteHeader(http.StatusOK)
}
```

Signatures are `sha256=<hex>` HMACs over `timestamp + "." + payload`, and a
timestamp more than five minutes from now is rejected. Pass an empty
timestamp to sign the payload alone. `sendly.Webhooks{}.VerifySignature` runs
the same check on its own and returns a bool, and
`sendly.Webhooks{}.GenerateSignature(payload, secret, timestamp)` produces a
signature for your own tests.

### Lifecycle events

`event.Data` is the event's `data.object` decoded as a message, and only
`message.*` events carry one. Lifecycle events — `rcs_*`, `whatsapp_*`, `call.*`,
`brand.*`, `campaign.*`, `assignment.*`, `number.*`, `port*`, `short_code.*`,
`contact.*`, `conversation.*`, `draft.*` — carry a different object entirely, so
`event.Data` stays zeroed for them and no error is raised. Read those payloads
with `event.DecodeObject(&v)`, or straight off `event.RawObject`, which holds
the `data.object` exactly as it arrived and is populated for every event type,
including `message.*` and event types newer than your SDK build.

```go
switch event.Type {
case sendly.WebhookEventRcsAgentLive, sendly.WebhookEventRcsAgentRejected:
    var agent struct {
        AgentID string `json:"agent_id"`
        Name    string `json:"name"`
        Stage   string `json:"stage"`
        Reason  string `json:"reason,omitempty"`
    }
    if err := event.DecodeObject(&agent); err != nil {
        http.Error(w, "bad payload", http.StatusBadRequest)
        return
    }
    fmt.Printf("agent %s (%s) is %s\n", agent.AgentID, agent.Name, agent.Stage)

case sendly.WebhookEventContactAutoFlagged:
    var flagged struct {
        // ID is the CONTACT id, not a message id. The message that
        // triggered the flag is MessageID, and it can be null.
        ID            string  `json:"id"`
        PhoneNumber   string  `json:"phone_number"`
        InvalidReason string  `json:"invalid_reason"`
        Source        string  `json:"source"`
        MessageID     *string `json:"message_id"`
    }
    if err := event.DecodeObject(&flagged); err != nil {
        http.Error(w, "bad payload", http.StatusBadRequest)
        return
    }
    fmt.Printf("contact %s flagged: %s\n", flagged.ID, flagged.InvalidReason)

default:
    // Anything you have no typed handler for, including event types added
    // after this SDK build, is still readable in full.
    var object map[string]interface{}
    if err := event.DecodeObject(&object); err == nil {
        log.Printf("unhandled %s: %v", event.Type, object)
    }
}
```

`message.opt_in` and `message.opt_out` share the `message.` prefix but carry an
opt-out record (`phone_number`, `keyword`, `from_number`, `timestamp`), not a
message, so they are handled the same way — through `DecodeObject`.

`WebhookEventMessageQueued` and `WebhookEventMessageUndelivered` are deprecated.
The API has never emitted them and rejects them with a 400 when you subscribe;
drop them from your `Events` list.

## Account & Credits

```go
// Get account information
account, err := client.Account.Get(ctx)
fmt.Printf("Email: %s\n", account.Email)

// Check credit balance
credits, err := client.Account.GetCredits(ctx)
fmt.Printf("Balance: %d, reserved: %d, available: %d (%s)\n",
    credits.Balance, credits.ReservedBalance, credits.AvailableBalance, credits.BillingMode)

// View credit transaction history
transactions, err := client.Account.GetCreditTransactions(ctx, &sendly.ListCreditTransactionsOptions{Limit: 50})
for _, tx := range transactions {
    fmt.Printf("%s: %d credits - %s (balance %d)\n", tx.Type, tx.Amount, tx.Description, tx.BalanceAfter)
}

// Move credits to another workspace you control
moved, err := client.Account.TransferCredits(ctx, sendly.TransferCreditsRequest{
    TargetOrganizationID: "ws_xxx",
    Amount:               1000,
})
fmt.Println(moved.SourceBalance, moved.TargetBalance)
```

### API keys

```go
// List API keys
keys, err := client.Account.ListAPIKeys(ctx)
for _, key := range keys {
    fmt.Printf("%s: %s (%s) scopes %v\n", key.Name, key.Prefix, key.Type, key.Permissions)
}

// Read one, and see how it is being used
key, err := client.Account.GetAPIKey(ctx, "key_xxx")
usage, err := client.Account.GetAPIKeyUsage(ctx, "key_xxx")
fmt.Println(usage.Summary.TotalRequests, usage.Summary.TotalCredits)

// CreateAPIKey makes a TEST key. Use CreateAPIKeyWithOptions for a live one.
testKey, err := client.Account.CreateAPIKey(ctx, "Local development")
expiresAt := time.Now().AddDate(0, 3, 0).UTC().Format(time.RFC3339)
liveKey, err := client.Account.CreateAPIKeyWithOptions(ctx, sendly.CreateAPIKeyRequest{
    Name:      "Production",
    Type:      "live",
    Scopes:    []string{"sms:send", "sms:read"},
    ExpiresAt: &expiresAt,
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("New key: %s\n", liveKey.Key) // Only shown once!
fmt.Println(liveKey.KeyPrefix, *liveKey.ExpiresAt)

// Revoke an API key (a key cannot revoke itself)
err = client.Account.RevokeAPIKey(ctx, "key_xxx")

// Rotate an API key — issues a new secret and keeps the old one valid for a
// grace period (24-168 hours, default 24) so running code keeps working.
rotated, err := client.Account.RotateAPIKey(ctx, "key_xxx", &sendly.RotateAPIKeyRequest{
    GracePeriodHours: 48,
})
fmt.Printf("New key: %s\n", rotated.NewKey.Key) // Only shown once!
fmt.Println(rotated.NewKey.Prefix, rotated.NewKey.Permissions)
fmt.Println(rotated.Message) // e.g. when the old key expires
```

A key created without `Scopes` gets the scopes of the key that creates it, and
asking for a scope that key lacks is refused with
`403 insufficient_permissions`. A live key also needs a verified business
(`403 verification_required`) and a positive balance
(`402 credits_required`). `ExpiresAt` must be an ISO 8601 time in the future;
leave it nil for a key that never expires.

## Branded Links

Mint branded short links for a destination URL, list them with click
analytics, and flip a per-link kill switch. Requires the `url_shortener`
feature on your account; until it is on, these calls return a not-found error.

```go
// Create a short link (destination must be an http:// or https:// URL)
link, err := client.Links.Create(ctx, "https://acme.example/spring-sale")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%s -> %s\n", link.ShortURL, link.DestinationURL)

// List your links with click counts
list, err := client.Links.List(ctx, &sendly.ListShortLinksRequest{Limit: 50})
for _, l := range list.Links {
    fmt.Printf("%s: %d clicks\n", l.Code, l.ClickCount)
}

// Disable a link (its redirect returns 404 until re-enabled)
_, err = client.Links.Disable(ctx, link.Code)

// Re-enable it
_, err = client.Links.Enable(ctx, link.Code)
```

## Voice Calls

Place phone calls that one of your AI agents handles, follow them while they
ring and after they end, hang up early, and download recordings. Create
agents and switch voice on for the number you call from in code (see
[Configure voice](#configure-voice)) or in the dashboard; for outbound calls
the number also needs a registered emergency address. `Create` and `Hangup`
need a live API key with the `calls:write` scope; the reads need `calls:read`.

Calls are billed per started minute from your prepaid credits: 2 credits a
minute outbound, plus 8 a minute while an AI agent is on the line (so 10 for a
call placed here). Calls go from and to US and Canadian numbers only: any
other `From` is refused with `400 from_number_not_supported`
(`sendly.CallErrorCodeFromNumberNotSupported`) and any other `To` with
`400 destination_not_supported`. An unanswered call costs nothing. Today's
calling limit comes back as a `*sendly.RateLimitError` with Code
`daily_call_limit`, without a retry. Voice is being enabled workspace by
workspace; until it is on for yours, every method returns a `NotFoundError`
with code `voice_not_enabled`.

The snippets below use `errors` and `encoding/json` from the standard library
alongside the SDK import.

```go
// Place a call. It comes back ringing; the agent greets whoever answers.
call, err := client.Calls.Create(ctx, &sendly.CreateCallRequest{
    To:      "+15125550123",
    AgentID: "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
    From:    "+15125550100", // optional when exactly one number has voice on
    Context: "You are calling Jordan to confirm the 3pm appointment on Tuesday.",
    Metadata: map[string]string{"crmId": "lead_8812"},
})
if err != nil {
    var sendlyErr *sendly.SendlyError
    switch {
    case sendly.IsInsufficientCreditsError(err):
        log.Fatal("Top up: the balance doesn't cover the first minute")
    case errors.As(err, &sendlyErr) && sendlyErr.Code == sendly.CallErrorCodeE911Required:
        log.Fatal("Register an emergency address first: client.Voice.Numbers.RegisterEmergencyAddress")
    case errors.As(err, &sendlyErr) && sendlyErr.Code == sendly.CallErrorCodeLinesBusy:
        log.Fatal("Every line is in use, retry shortly")
    default:
        log.Fatal(err)
    }
}
fmt.Printf("%s is %s\n", call.ID, call.Status) // ringing

// Follow the call. Agent-handled calls include the transcript on Get.
call, err = client.Calls.Get(ctx, call.ID)
fmt.Println(call.Status, call.DurationSecs, call.CreditsCharged)
if call.HangupClass != nil {
    fmt.Println("ended:", *call.HangupClass)
}
for _, line := range call.Transcript {
    fmt.Printf("[%dms] %s: %s\n", line.AtMs, line.Speaker, line.Text)
}

// List calls, newest first, with filters and pagination
calls, err := client.Calls.List(ctx, &sendly.ListCallsRequest{
    Status:    sendly.CallStatusCompleted,
    Direction: sendly.CallDirectionOutbound,
    AgentID:   "3c4d5e6f-7081-4293-a4b5-c6d7e8f90a1b",
    Limit:     20,
})
for _, c := range calls.Data {
    fmt.Printf("%s %s -> %s (%d credits)\n", c.ID, *c.From, *c.To, c.CreditsCharged)
}
if calls.Pagination.HasMore {
    // fetch the next page with Offset: calls.Pagination.Offset + calls.Pagination.Limit
}

// Hang up. A ringing call becomes cancelled, an active one completed; a call
// that already ended is returned unchanged.
call, err = client.Calls.Hangup(ctx, call.ID)

// Download the recording once it is ready. The URL is signed and valid for
// five minutes; fetch again after ExpiresAt for a fresh one.
rec, err := client.Calls.Recording(ctx, call.ID)
if rec.Status == sendly.CallRecordingStatusReady {
    fmt.Println(*rec.URL, *rec.ContentType, *rec.ExpiresAt) // audio/ogg
}
```

To find a number to call from, look for `VoiceEnabled` on `client.Numbers.List`
(see [Numbers](#numbers)). The `call.started`, `call.completed` and
`call.recording.ready` webhooks carry the call in snake_case; decode it into a
`sendly.WebhookCallData` with `event.DecodeObject(&callData)`, where `Billing`
and `Metadata` round-trip alongside `HangupClass` and `CreditsCharged`.
Calls and call webhooks carry `Channel`: `"phone"`, `"whatsapp"` or
`"browser"` (`sendly.CallChannel`; a value added later decodes as itself).
Inbound WhatsApp calls may still read `"phone"` for now.

Recordings of agent calls are dual-channel: the agent is on the left channel
and the other party on the right.

### Configure voice

Everything a call depends on can be set up in code: switch voice on for a
number and choose how it answers, register the number's emergency address,
and create the AI agents that talk to callers. Reads need `calls:read`; writes
need a live API key with `calls:write`. In a team workspace, changing a number
or its emergency address also needs a role with `settings:write`, and managing
agents needs `api_keys:write`, because each agent holds its own scoped sending
key.

Switching voice on for a number changes how real phone calls to it are
answered, and an agent pointed at a number answers real callers. An emergency
address is required before a number can place calls in the US and Canada and
costs 1.50 USD a month.

```go
// See which voices an agent can speak with
voices, err := client.Voice.Voices.List(ctx)
for _, v := range voices.Data {
    fmt.Printf("%s: %s (%s)\n", v.ID, v.Label, v.Language)
}

// Create an agent. Only Name is required; it starts switched on and may text
// the people it talks to. Set SendSms to false to stop that.
sendSms := true
agent, err := client.Voice.Agents.Create(ctx, &sendly.CreateVoiceAgentRequest{
    Name:         "Front desk",
    Voice:        "ashley",
    Language:     "en-US",
    Greeting:     "Thanks for calling Acme, how can I help?",
    Instructions: "Answer questions about opening hours and take a message for anything else.",
    Tools:        &sendly.VoiceAgentToolsRequest{SendSms: &sendSms},
})
if err != nil {
    var sendlyErr *sendly.SendlyError
    if errors.As(err, &sendlyErr) && sendlyErr.Code == sendly.VoiceErrorCodeAgentLimit {
        log.Fatal("This workspace already has 20 agents")
    }
    log.Fatal(err)
}
fmt.Println(agent.ID, agent.VoiceLabel, agent.CanSendSms)

// List the agents, or read one back with its call stats
allAgents, err := client.Voice.Agents.List(ctx)
for _, a := range allAgents.Data {
    fmt.Println(a.Name, a.CallsHandled, a.AvgDurationSecs)
}
one, err := client.Voice.Agents.Get(ctx, agent.ID)

// Change only the fields you set. A pointer to "" removes the greeting or
// instructions.
greeting := "Thanks for calling Acme. This call may be recorded."
agent, err = client.Voice.Agents.Update(ctx, agent.ID, &sendly.UpdateVoiceAgentRequest{
    Greeting: &greeting,
})

// List the workspace's numbers with their voice settings and per-minute rates
numbers, err := client.Voice.Numbers.List(ctx)
for _, n := range numbers.Data {
    fmt.Printf("%s voice=%v mode=%s agent answers at %d credits/min\n",
        n.PhoneNumber, n.VoiceEnabled, n.VoiceMode, n.RatePerMinute.Agent)
}

// Let the agent answer a number. Pass the number's ID or its E.164 form.
enabled := true
number, err := client.Voice.Numbers.Update(ctx, "+15125550123", &sendly.UpdateVoiceNumberRequest{
    VoiceEnabled: &enabled,
    VoiceMode:    sendly.VoiceModeAgent,
    AgentID:      &agent.ID,
})

// Read one number's voice settings back
number, err = client.Voice.Numbers.Get(ctx, number.ID)

// Register the emergency address before placing calls from the number
number, err = client.Voice.Numbers.RegisterEmergencyAddress(ctx, number.ID, &sendly.EmergencyAddress{
    Street: "500 Example Ave",
    Unit:   "Suite 2",
    City:   "Austin",
    State:  "TX",
    Zip:    "78701",
})
if err != nil {
    var validationErr *sendly.ValidationError
    if errors.As(err, &validationErr) && validationErr.Code == sendly.VoiceErrorCodeInvalidAddress {
        var suggested sendly.EmergencyAddress
        if json.Unmarshal(validationErr.Extra["suggested"], &suggested) == nil && suggested.Street != "" {
            log.Fatalf("Did you mean %s, %s, %s %s?", suggested.Street, suggested.City, suggested.State, suggested.Zip)
        }
        log.Fatal("Check the address: ", validationErr.Message)
    }
    log.Fatal(err)
}
fmt.Println(number.EmergencyAddress.Status)

// Send calls back to the team, then remove the agent. Deleting an agent
// that still answers a number fails with code agent_in_use.
number, err = client.Voice.Numbers.Update(ctx, number.ID, &sendly.UpdateVoiceNumberRequest{
    VoiceMode: sendly.VoiceModeRingDashboard,
})
deleted, err := client.Voice.Agents.Delete(ctx, agent.ID)
if err != nil {
    var sendlyErr *sendly.SendlyError
    if errors.As(err, &sendlyErr) && sendlyErr.Code == sendly.VoiceErrorCodeAgentInUse {
        var numbers []string
        json.Unmarshal(sendlyErr.Extra["numbers"], &numbers)
        log.Fatal("Point these numbers at another agent or back to the team first: ", numbers)
    }
    log.Fatal(err)
}
fmt.Println(deleted.ID, deleted.Deleted)
```

A mode on its own is enough: `sendly.VoiceModeRingDashboard` or
`sendly.VoiceModeAgent` switches voice on (and can be refused like any
switch-on), and `sendly.VoiceModeNone` switches it off. `VoiceEnabled`, when
set, wins: false switches voice off whatever the mode, and true with
`sendly.VoiceModeNone` rings the dashboard. A pointer to an empty `AgentID`
clears the stored agent. `RegisterEmergencyAddress` is not retried automatically on a
5xx such as `carrier_refused`, because every attempt registers the address
anew. An agent's `TransferTo` number is stored
but calls are not transferred to it: when a caller asks for a person, the
agent offers to pass a message on and takes their details.

## Error Handling

```go
message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:   "+15125550123",
    Text: "Hello!",
})
if err != nil {
    switch {
    case sendly.IsAuthenticationError(err):
        log.Fatal("Invalid API key")
    case sendly.IsRateLimitError(err):
        rateLimitErr := err.(*sendly.RateLimitError)
        if rateLimitErr.Code == "too_many_failed_key_attempts" {
            // Retrying will not help: fix the API key, then wait out the lockout.
            log.Fatalf("Too many wrong API keys from this address; locked for %d seconds", rateLimitErr.RetryAfter)
        }
        log.Printf("Rate limited (%s), retry after %d seconds", rateLimitErr.Code, rateLimitErr.RetryAfter)
    case sendly.IsInsufficientCreditsError(err):
        log.Fatal("Add more credits to your account")
    case sendly.IsValidationError(err):
        log.Printf("Invalid request: %v", err)
    case sendly.IsNotFoundError(err):
        log.Fatal("Resource not found")
    case sendly.IsNetworkError(err):
        log.Printf("Network error: %v", err)
    default:
        log.Printf("Error: %v", err)
    }
    return
}
```

Which type you get follows the status code:

| Status | Type | Helper |
|--------|------|--------|
| 401 | `*sendly.AuthenticationError` | `IsAuthenticationError` |
| 402 | `*sendly.InsufficientCreditsError` | `IsInsufficientCreditsError` |
| 404 | `*sendly.NotFoundError` | `IsNotFoundError` |
| 400, 422 | `*sendly.ValidationError` | `IsValidationError` |
| 429 | `*sendly.RateLimitError` | `IsRateLimitError` |
| anything else | `*sendly.SendlyError` (carries `StatusCode`) | — |
| transport failure | `*sendly.NetworkError` | `IsNetworkError` |

Every type except `NetworkError` embeds `APIError`, so `Code`, `Message` and
`Details` are available on each of them; `NetworkError` carries only `Message`
and the wrapped `Err`. `Errors` lists the offending fields (`Path` and `Message`) when
the API reports them, and `Extra` keeps any other top-level key of the error
body undecoded as `json.RawMessage` — that is where `suggested` and `numbers`
come from in the voice examples above. `ValidationError` and `NetworkError`
also implement `Unwrap`.

Some errors come from the SDK before anything is sent: a `*sendly.ValidationError`
for a missing required field, an invalid idempotency key, or an id of `""`,
`.` or `..` (which would otherwise resolve to a different endpoint, such as
the parent collection). Every id is percent-encoded into the path, so an id
containing `/` or `?` cannot reach another endpoint either.

## Message Status

| Status | Constant | Description |
|--------|----------|-------------|
| `queued` | `MessageStatusQueued` | Message is queued for delivery |
| `sent` | `MessageStatusSent` | Message was sent to carrier |
| `delivered` | `MessageStatusDelivered` | Message was delivered |
| `read` | `MessageStatusRead` | Recipient read it (RCS and WhatsApp only) |
| `failed` | `MessageStatusFailed` | Message delivery failed |
| `bounced` | `MessageStatusBounced` | Carrier rejected the message |
| `retrying` | `MessageStatusRetrying` | Retrying after a transient failure |

Scheduled messages use their own set: `scheduled`, `sent`, `cancelled` and
`failed` (`ScheduledMessageStatus*`). Batches use `processing`, `completed`,
`partial_failure` and `failed` (`BatchStatus*`).

## Pricing Tiers

1 credit is $0.01, so the credit count is the price in cents.

| Tier | Countries | Credits per SMS |
|------|-----------|-----------------|
| Domestic | US, CA | 2 |
| Tier 1 | GB, AU, PL, BR, etc. | 8 |
| Tier 2 | FR, JP, IT, IN, ES, etc. | 12 |
| Tier 3 | DE, NL, MX, etc. | 16 |
| Tier 4 | UA, PA, VE, etc. | 24 |
| Tier 5 | IL, SI, SR, etc. | 48 |

Multi-segment messages cost the per-segment price times the segment count.
Enterprise accounts can have per-country or per-tier credit overrides, in
which case the override wins.

## Sandbox Testing

Use test API keys (`sk_test_v1_xxx`) with these test numbers:

| Number | Behavior |
|--------|----------|
| +15005550000 | Success (instant) |
| +15005550001 | Fails: invalid_number |
| +15005550002 | Fails: unroutable_destination |
| +15005550003 | Fails: queue_full |
| +15005550004 | Fails: rate_limit_exceeded |
| +15005550006 | Fails: carrier_violation |

Sandbox sends fire the matching `message.delivered` or `message.failed`
webhook, so an end-to-end integration can be exercised without a live key.

## Enterprise

The Enterprise API lets you programmatically manage workspaces, verification,
credits, and API keys for multi-tenant platforms. It requires an enterprise
master key — an ordinary live key (`sk_live_v1_…`) that has been marked as the
account's master key; create one from the dashboard. Without it these
endpoints answer 403. Master keys also get the higher rate limit of 3,000
requests a minute.

### Quick Provision

Create a fully configured workspace in a single call:

```go
client := sendly.NewClient("sk_live_v1_your_master_key")

generateOptIn := true
result, err := client.Enterprise.Provision(ctx, &sendly.ProvisionWorkspaceRequest{
    Name:                    "Acme Insurance - Austin",
    SourceWorkspaceID:       "ws_verified",
    CreditAmount:            5000,
    CreditSourceWorkspaceID: "SOURCE_WORKSPACE_ID",
    KeyName:                 "Production",
    KeyType:                 "live",
    GenerateOptInPage:       &generateOptIn,
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Workspace.ID)
if result.Key != nil {
    fmt.Println(result.Key.Key)
}
```

Three provisioning modes:

| Mode | Params | Description |
|------|--------|-------------|
| **Inherit** | `SourceWorkspaceID` | Shares toll-free number from verified workspace |
| **Inherit + New Number** | `SourceWorkspaceID` + `InheritWithNewNumber: true` | Copies business info, purchases new number |
| **Fresh** | `Verification: &sendly.ProvisionVerificationData{...}` | Full business details, new number + carrier approval |

Up to 100 workspaces can be created in one call with
`client.Enterprise.Workspaces.ProvisionBulk`:

```go
bulk, err := client.Enterprise.Workspaces.ProvisionBulk(ctx, []sendly.BulkProvisionWorkspace{
    {Name: "Acme Insurance - Dallas", SourceWorkspaceID: "ws_verified"},
    // CreditSourceWorkspaceID is the ID (a UUID) of the workspace the credits come from
    {Name: "Acme Insurance - Houston", SourceWorkspaceID: "ws_verified", CreditAmount: 1000, CreditSourceWorkspaceID: "5f0c2a9e-3b7d-4e1a-8c6f-2d9b0e4a7c13"},
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(bulk.Summary.Succeeded, "of", bulk.Summary.Total)
for _, r := range bulk.Results {
    if r.Error != nil {
        fmt.Println(r.Name, "failed:", *r.Error)
    }
}
```

Provisioning is limited to 120 requests a minute and 1,000 an hour, and a
bulk call counts as one request. The client waits out the per-minute limit; the hourly one comes back as a
`*sendly.RateLimitError` with Code `provision_rate_limit` and `RetryAfter` set.

### Workspace Management

```go
ws, _ := client.Enterprise.Workspaces.Create(ctx, "Acme Insurance", "")
account, _ := client.Enterprise.Workspaces.List(ctx)
for _, w := range account.Workspaces {
    fmt.Println(w.ID, w.Name, w.CreditBalance)
}
detail, _ := client.Enterprise.Workspaces.Get(ctx, "ws_xxx")
_ = client.Enterprise.Workspaces.Delete(ctx, "ws_xxx")

// Verification for a managed workspace: share a verified workspace's approval
_, _ = client.Enterprise.Workspaces.InheritVerification(ctx, "ws_xxx", "ws_verified")
status, _ := client.Enterprise.Workspaces.GetVerification(ctx, "ws_xxx")
fmt.Println(status.Status)

// Or copy its business details and file a toll-free verification of its own
own, _ := client.Enterprise.Workspaces.InheritVerificationWithOptions(ctx, "ws_other", &sendly.InheritVerificationRequest{
    SourceWorkspaceID: "ws_verified",
    PurchaseNewNumber: true,
})
if own.TollFreeNumber == nil {
    fmt.Println("no toll-free number could be bought yet")
}
fmt.Println(own.Status, own.NewNumber)

// Pause and restore a tenant
_, _ = client.Enterprise.Workspaces.Suspend(ctx, "ws_xxx", "non-payment")
_, _ = client.Enterprise.Workspaces.Resume(ctx, "ws_xxx")
```

Partial resubmits are supported: every field of `VerificationSubmitInput` is a
pointer, and unset fields are left out of the request so the server merges
them with the existing record.

```go
name := "Acme Insurance LLC"
_, _ = client.Enterprise.Workspaces.SubmitVerification(ctx, "ws_xxx", &sendly.VerificationSubmitInput{
    BusinessName: &name,
})
```

### Credits & API Keys

```go
result, _ := client.Enterprise.Workspaces.TransferCredits(ctx, "ws_dest", "ws_source", 5000)
balance, _ := client.Enterprise.Workspaces.GetCredits(ctx, "ws_dest")
fmt.Println(result.TargetBalance, balance.Balance)

key, _ := client.Enterprise.Workspaces.CreateKey(ctx, "ws_xxx", "Production", "live")
fmt.Println(key.Key)
keys, _ := client.Enterprise.Workspaces.ListKeys(ctx, "ws_xxx")

_ = client.Enterprise.Workspaces.RevokeKey(ctx, "ws_xxx", "key_abc")

// The shared credit pool behind the workspaces
pool, _ := client.Enterprise.Credits.Get(ctx)
```

### Webhooks & Analytics

```go
webhook, _ := client.Enterprise.Webhooks.SetWithOptions(ctx, &sendly.SetEnterpriseWebhookRequest{
    URL:        "https://acme.example/webhooks",
    Events:     []string{"message.delivered", "message.failed"},
    Workspaces: []string{"ws_xxx"},
})
if webhook.SigningSecret != "" {
    fmt.Println("store this secret now:", webhook.SigningSecret)
}
current, _ := client.Enterprise.Webhooks.Get(ctx)
fmt.Println(current.URL, current.Events, current.Workspaces)
test, _ := client.Enterprise.Webhooks.Test(ctx)
rotated, _ := client.Enterprise.Webhooks.RotateSecret(ctx)

overview, _ := client.Enterprise.Analytics.Overview(ctx)
fmt.Printf("%.2f%% delivered across %d workspaces (%d suspended)\n",
    overview.DeliveryRatePercent, overview.TotalWorkspaces, overview.SuspendedWorkspaces)
messages, _ := client.Enterprise.Analytics.Messages(ctx, &sendly.AnalyticsMessagesOptions{Period: "30d"})
delivery, _ := client.Enterprise.Analytics.Delivery(ctx)
creditUse, _ := client.Enterprise.Analytics.Credits(ctx, nil)
fmt.Println(creditUse.TotalBalance, creditUse.TotalLifetime, creditUse.TotalUsed, creditUse.WorkspaceCount)
```

The signing secret is returned once, by the account's first `Set` or
`SetWithOptions`; later calls leave `SigningSecret` empty, and `RotateSecret`
issues a new one. In `SetWithOptions`, a nil `Events` or `Workspaces` keeps
the current list and an empty, non-nil slice clears it, so every event or
every workspace is delivered. `Set(ctx, url)` changes only the URL.

`DeliveryRatePercent` is the rate as a percentage to two decimal places; the
deprecated `DeliveryRate` is rounded to a whole number. `Analytics.Credits` returns totals that are not
limited to the period, and its deprecated `Data` is always empty.

### Billing, settings and the rest

```go
breakdown, _ := client.Enterprise.Billing.GetBreakdown(ctx, &sendly.BillingBreakdownOptions{Period: "30d"})
fmt.Println(breakdown.Summary.TotalCost)

topUp, _ := client.Enterprise.Settings.GetAutoTopUp(ctx)
topUp, _ = client.Enterprise.Settings.UpdateAutoTopUp(ctx, &sendly.UpdateAutoTopUpRequest{
    Enabled:   true,
    Threshold: 1000,
    Amount:    5000,
})

acct, _ := client.Enterprise.GetAccount(ctx)
fmt.Println(acct.WorkspaceCount, "of", acct.MaxWorkspaces)
```

Per-workspace opt-in pages, hosted business pages, custom domains, member
invitations, message quotas and verification-document uploads are available
too — see `client.Enterprise.Workspaces` and `client.Enterprise` in the
[package reference](https://pkg.go.dev/github.com/SendlyHQ/sendly-go/v4/sendly).

Full enterprise docs: [sendly.live/docs/enterprise](https://sendly.live/docs/enterprise)

---

## Requirements

- Go 1.21+

## License

MIT
