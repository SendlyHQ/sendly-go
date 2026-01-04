# Sendly Go SDK

Official Go SDK for the Sendly SMS API.

## Installation

```bash
go get github.com/SendlyHQ/sendly-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/SendlyHQ/sendly-go/sendly"
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
    "github.com/SendlyHQ/sendly-go/sendly"
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
transactions, err := client.Account.GetCreditTransactions(ctx)
for _, tx := range transactions.Data {
    fmt.Printf("%s: %d credits - %s\n", tx.Type, tx.Amount, tx.Description)
}

// List API keys
keys, err := client.Account.ListAPIKeys(ctx)
for _, key := range keys.Data {
    fmt.Printf("%s: %s*** (%s)\n", key.Name, key.Prefix, key.Type)
}

// Create a new API key
newKey, err := client.Account.CreateAPIKey(ctx, &sendly.CreateAPIKeyRequest{
    Name:   "Production Key",
    Type:   "live",
    Scopes: []string{"sms:send", "sms:read"},
})
fmt.Printf("New key: %s\n", newKey.Key) // Only shown once!

// Revoke an API key
err = client.Account.RevokeAPIKey(ctx, "key_xxx")
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
| Domestic | US, CA | 1 |
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

## Requirements

- Go 1.21+

## License

MIT
