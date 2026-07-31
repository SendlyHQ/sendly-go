<p align="center">
  <img src="https://raw.githubusercontent.com/SendlyHQ/sendly-go/main/.github/header.svg" alt="Sendly Go SDK" />
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/SendlyHQ/sendly-go/v3"><img src="https://pkg.go.dev/badge/github.com/SendlyHQ/sendly-go/v3.svg" alt="Go Reference" /></a>
  <a href="https://github.com/SendlyHQ/sendly-go/blob/main/LICENSE"><img src="https://img.shields.io/github/license/SendlyHQ/sendly-go?style=flat-square" alt="license" /></a>
</p>

# Sendly Go SDK

Official Go SDK for the Sendly SMS API.

## Installation

```bash
go get github.com/SendlyHQ/sendly-go/v3
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/SendlyHQ/sendly-go/v3/sendly"
)

func main() {
    // Create a client
    client := sendly.NewClient("sk_live_v1_your_api_key")
    ctx := context.Background()

    // Send an SMS
    message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
        To:   "+15551234567",
        Text: "Hello from Sendly!",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Message sent: %s\n", message.ID)
}
```

## Prerequisites for Live Messaging

Before sending live SMS messages, you need:

1. **Business Verification** - Complete verification in the [Sendly dashboard](https://sendly.live/dashboard)
   - **International**: Instant approval (just provide Sender ID)
   - **US/Canada**: Requires carrier approval (3-7 business days)

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

## Configuration

```go
import (
    "time"
    "github.com/SendlyHQ/sendly-go/v3/sendly"
)

// Create client with options
client := sendly.NewClient("sk_live_v1_xxx",
    sendly.WithBaseURL("https://sendly.live/api/v1"),
    sendly.WithTimeout(60*time.Second),
    sendly.WithMaxRetries(5),
    sendly.WithDebug(true),
)
```

## Messages

### Send an SMS

```go
// Marketing message (default)
message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:   "+15551234567",
    Text: "Check out our new features!",
})
if err != nil {
    log.Fatal(err)
}

// Transactional message (bypasses quiet hours)
message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:          "+15551234567",
    Text:        "Your verification code is: 123456",
    MessageType: "transactional",
})

// With custom metadata (max 4KB)
message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:   "+15551234567",
    Text: "Your order #12345 has shipped!",
    Metadata: map[string]interface{}{
        "order_id":    "12345",
        "customer_id": "cust_abc",
    },
})

// Send from one of your owned numbers (or an alphanumeric sender ID).
// Omit From to use your default sender.
message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:   "+15551234567",
    Text: "Hello from our team!",
    From: "+447111111111",
})

fmt.Printf("ID: %s\n", message.ID)
fmt.Printf("Status: %s\n", message.Status)
fmt.Printf("Credits: %d\n", message.CreditsUsed)
```

### List Messages

```go
resp, err := client.Messages.List(ctx, &sendly.ListMessagesRequest{
    Limit:  50,
    Offset: 0,
    Status: sendly.MessageStatusDelivered,
    To:     "+15551234567",
})
if err != nil {
    log.Fatal(err)
}

for _, msg := range resp.Data {
    fmt.Printf("%s: %s (%s)\n", msg.ID, msg.To, msg.Status)
}
```

### Get a Message

```go
message, err := client.Messages.Get(ctx, "msg_abc123")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("To: %s\n", message.To)
fmt.Printf("Text: %s\n", message.Text)
fmt.Printf("Status: %s\n", message.Status)
```

### Scheduling Messages

```go
// Schedule a message for future delivery
scheduled, err := client.Messages.Schedule(ctx, &sendly.ScheduleMessageRequest{
    To:          "+15551234567",
    Text:        "Your appointment is tomorrow!",
    ScheduledAt: "2025-01-15T10:00:00Z",
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Scheduled: %s\n", scheduled.ID)
fmt.Printf("Will send at: %s\n", scheduled.ScheduledAt)

// List scheduled messages
resp, err := client.Messages.ListScheduled(ctx, nil)
for _, msg := range resp.Data {
    fmt.Printf("%s: %s\n", msg.ID, msg.ScheduledAt)
}

// Get a specific scheduled message
msg, err := client.Messages.GetScheduled(ctx, "sched_xxx")

// Cancel a scheduled message (refunds credits)
result, err := client.Messages.CancelScheduled(ctx, "sched_xxx")
fmt.Printf("Refunded: %d credits\n", result.CreditsRefunded)
```

### Batch Messages

```go
// Send multiple messages in one API call (up to 1000)
batch, err := client.Messages.SendBatch(ctx, &sendly.SendBatchRequest{
    Messages: []sendly.BatchMessageItem{
        {To: "+15551234567", Text: "Hello User 1!"},
        {To: "+15559876543", Text: "Hello User 2!"},
        {To: "+15551112222", Text: "Hello User 3!"},
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Batch ID: %s\n", batch.BatchID)
fmt.Printf("Queued: %d\n", batch.Queued)
fmt.Printf("Failed: %d\n", batch.Failed)
fmt.Printf("Credits used: %d\n", batch.CreditsUsed)

// Get batch status
status, err := client.Messages.GetBatch(ctx, "batch_xxx")

// List all batches
batches, err := client.Messages.ListBatches(ctx, nil)

// Preview batch (dry run) - validates without sending
preview, err := client.Messages.PreviewBatch(ctx, &sendly.SendBatchRequest{
    Messages: []sendly.BatchMessageItem{
        {To: "+15551234567", Text: "Hello User 1!"},
        {To: "+447700900123", Text: "Hello UK!"},
    },
})
fmt.Printf("Total credits needed: %d\n", preview.TotalCredits)
fmt.Printf("Valid: %d, Invalid: %d\n", preview.Valid, preview.Invalid)
```

### Group MMS

Send one MMS to 2-8 US/Canada recipients who all share a thread. Group
messaging is an A2P 10DLC capability — the sending number must be an
MMS-enabled, 10DLC-registered number you own. Omit `From` to use your
default sender.

```go
group, err := client.Messages.SendGroup(ctx, &sendly.SendGroupMessageRequest{
    To:   []string{"+15551234567", "+15559876543"},
    Text: "Dinner at 7 tonight?",
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Group message: %s (%s)\n", group.ID, group.Status)
if group.GroupMessageID != "" {
    fmt.Printf("Thread: %s\n", group.GroupMessageID)
}
```

### AI Enhance

Rewrite a draft message for clarity, compliance, and send-readiness. Provide
`Text`, `MessageType`, or both.

```go
enhanced, err := client.Messages.Enhance(ctx, &sendly.EnhanceMessageRequest{
    Text:        "hey wanna buy our stuff its on sale",
    MessageType: "marketing",
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(enhanced.Enhanced)    // the rewritten message
fmt.Println(enhanced.Explanation) // a short note on what changed
```

## WhatsApp

Connect a number you own to WhatsApp, create Meta-reviewed message templates,
and send with `client.Messages.SendWhatsApp`. Connecting is a one-time $19
setup (no monthly fee) and always ends with a human step: the connect URL must
be opened in a browser and completed with a Facebook login. Free-form text and
media only deliver inside an open 24-hour customer-service window — outside
it, send an approved template.

```go
// 1. Connect a number ($19 one-time). A human must open the connect URL.
signup, err := client.WhatsApp.Signup.Create(ctx, "+15559876543")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Have your user open: %s\n", signup.ConnectURL)

// 2. Poll until active
status, err := client.WhatsApp.Signup.Get(ctx, signup.ID)
fmt.Println(status.Status) // "initiated" -> "registering" -> "active"

// 3. List your WhatsApp senders
senders, err := client.WhatsApp.Senders.List(ctx)
for _, s := range senders.Senders {
    fmt.Printf("%s: %s\n", s.PhoneNumber, s.Status)
}

// 4. Create a template (Meta reviews it, usually 24-48h)
template, err := client.WhatsApp.Templates.Create(ctx, &sendly.CreateWhatsAppTemplateRequest{
    Sender:   "+15559876543",
    Name:     "order_shipped",
    Language: "en_US",
    Category: "UTILITY",
    Body:     "Hi {{1}}, your order {{2}} has shipped!",
    Examples: map[string]string{"1": "Sam", "2": "#4821"},
})
fmt.Println(template.Status) // "PENDING"

// 5. Check the 24-hour window, then send
window, err := client.WhatsApp.Window(ctx, "+15559876543", "+15551234567")
if window.Open {
    // Free-form text (or media with a caption via MediaUrls + Text)
    msg, err := client.Messages.SendWhatsApp(ctx, &sendly.SendWhatsAppMessageRequest{
        To:   "+15551234567",
        From: "+15559876543",
        Text: "Your table is ready!",
    })
    fmt.Println(msg.ID)
} else {
    // Approved template — works regardless of the window
    msg, err := client.Messages.SendWhatsApp(ctx, &sendly.SendWhatsAppMessageRequest{
        To:   "+15551234567",
        From: "+15559876543",
        Template: &sendly.WhatsAppTemplateSendParams{
            Name:      "order_shipped",
            Language:  "en_US",
            Variables: map[string]string{"1": "Sam", "2": "#4821"},
        },
    })
    fmt.Println(msg.WhatsApp.Kind) // "template"
}
```

## Webhooks

```go
// Create a webhook endpoint
webhook, err := client.Webhooks.Create(ctx, &sendly.CreateWebhookRequest{
    URL:    "https://example.com/webhooks/sendly",
    Events: []string{"message.delivered", "message.failed"},
})
fmt.Printf("Webhook ID: %s\n", webhook.ID)
fmt.Printf("Secret: %s\n", webhook.Secret) // Store securely!

// List all webhooks
webhooks, err := client.Webhooks.List(ctx)

// Get a specific webhook
wh, err := client.Webhooks.Get(ctx, "whk_xxx")

// Update a webhook
client.Webhooks.Update(ctx, "whk_xxx", &sendly.UpdateWebhookRequest{
    URL:    "https://new-endpoint.example.com/webhook",
    Events: []string{"message.delivered", "message.failed", "message.sent"},
})

// Test a webhook
result, err := client.Webhooks.Test(ctx, "whk_xxx")

// Rotate webhook secret
rotation, err := client.Webhooks.RotateSecret(ctx, "whk_xxx")

// Delete a webhook
err = client.Webhooks.Delete(ctx, "whk_xxx")
```

## Numbers

```go
// List the numbers attached to your workspace
owned, err := client.Numbers.List(ctx)
for _, n := range owned.Numbers {
    fmt.Printf("%s: %s (%s)\n", n.ID, n.PhoneNumber, n.Status)
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
fmt.Printf("Default: %v\n", updated.IsDefault)

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

## Account & Credits

```go
// Get account information
account, err := client.Account.Get(ctx)
fmt.Printf("Email: %s\n", account.Email)

// Check credit balance
credits, err := client.Account.GetCredits(ctx)
fmt.Printf("Available: %d credits\n", credits.AvailableBalance)
fmt.Printf("Reserved: %d credits\n", credits.ReservedBalance)
fmt.Printf("Total: %d credits\n", credits.Balance)

// View credit transaction history
transactions, err := client.Account.GetCreditTransactions(ctx, nil)
for _, tx := range transactions {
    fmt.Printf("%s: %d credits - %s\n", tx.Type, tx.Amount, tx.Description)
}

// List API keys
keys, err := client.Account.ListAPIKeys(ctx)
for _, key := range keys {
    fmt.Printf("%s: %s*** (%s)\n", key.Name, key.Prefix, key.Type)
}

// Create a new API key
newKey, err := client.Account.CreateAPIKey(ctx, "Production Key")
fmt.Printf("New key: %s\n", newKey.Key) // Only shown once!

// Revoke an API key
err = client.Account.RevokeAPIKey(ctx, "key_xxx")

// Rotate an API key — issues a new secret and keeps the old one valid for a
// grace period (24-168 hours, default 24) so running code keeps working.
rotated, err := client.Account.RotateAPIKey(ctx, "key_xxx", &sendly.RotateAPIKeyRequest{
    GracePeriodHours: 48,
})
fmt.Printf("New key: %s\n", rotated.NewKey.Key) // Only shown once!
fmt.Println(rotated.Message)                    // e.g. when the old key expires
```

## Branded Links

Mint branded short links for a destination URL, list them with click
analytics, and flip a per-link kill switch. Requires the `url_shortener`
feature on your account.

```go
// Create a short link (destination must be an http:// or https:// URL)
link, err := client.Links.Create(ctx, "https://example.com/spring-sale")
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

## Error Handling

```go
message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:   "+15551234567",
    Text: "Hello!",
})
if err != nil {
    switch {
    case sendly.IsAuthenticationError(err):
        log.Fatal("Invalid API key")
    case sendly.IsRateLimitError(err):
        rateLimitErr := err.(*sendly.RateLimitError)
        log.Printf("Rate limited, retry after %d seconds", rateLimitErr.RetryAfter)
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

## Message Status

| Status | Description |
|--------|-------------|
| `queued` | Message is queued for delivery |
| `sending` | Message is being sent |
| `sent` | Message was sent to carrier |
| `delivered` | Message was delivered |
| `failed` | Message delivery failed |

## Pricing Tiers

| Tier | Countries | Credits per SMS |
|------|-----------|-----------------|
| Domestic | US, CA | 2 |
| Tier 1 | GB, PL, IN, etc. | 8 |
| Tier 2 | FR, JP, AU, etc. | 12 |
| Tier 3 | DE, IT, MX, etc. | 16 |

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

## Enterprise

The Enterprise API lets you programmatically manage workspaces, verification, credits, and API keys for multi-tenant platforms. Requires an enterprise master key (`sk_live_v1_master_*`).

### Quick Provision

Create a fully configured workspace in a single call:

```go
client := sendly.NewClient("sk_live_v1_master_YOUR_KEY")

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

fmt.Println(result.Workspace.ID)
fmt.Println(result.Key.Key)
```

Three provisioning modes:

| Mode | Params | Description |
|------|--------|-------------|
| **Inherit** | `SourceWorkspaceID` | Shares toll-free number from verified workspace |
| **Inherit + New Number** | `SourceWorkspaceID` + `InheritWithNewNumber: true` | Copies business info, purchases new number |
| **Fresh** | `Verification: sendly.VerificationData{...}` | Full business details, new number + carrier approval |

### Workspace Management

```go
ws, _ := client.Enterprise.Workspaces.Create(ctx, "Acme Insurance", "")
list, _ := client.Enterprise.Workspaces.List(ctx)
detail, _ := client.Enterprise.Workspaces.Get(ctx, "ws_xxx")
_ = client.Enterprise.Workspaces.Delete(ctx, "ws_xxx")
```

### Credits & API Keys

```go
result, _ := client.Enterprise.Workspaces.TransferCredits(ctx, "ws_dest", "ws_source", 5000)

key, _ := client.Enterprise.Workspaces.CreateKey(ctx, "ws_xxx", "Production", "live")
fmt.Println(key.Key)

_ = client.Enterprise.Workspaces.RevokeKey(ctx, "ws_xxx", "key_abc")
```

### Webhooks & Analytics

```go
webhook, _ := client.Enterprise.Webhooks.Set(ctx, "https://yourapp.com/webhooks")
overview, _ := client.Enterprise.Analytics.Overview(ctx)
messages, _ := client.Enterprise.Analytics.Messages(ctx, &sendly.AnalyticsMessagesOptions{Period: "30d"})
delivery, _ := client.Enterprise.Analytics.Delivery(ctx)
```

Full enterprise docs: [sendly.live/docs/enterprise](https://sendly.live/docs/enterprise)

---

## Requirements

- Go 1.21+

## License

MIT
