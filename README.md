# Sendly Go SDK

Official Go SDK for the Sendly SMS API.

## Installation

```bash
go get github.com/sendly-live/sendly-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sendly-live/sendly-go/sendly"
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
    "github.com/sendly-live/sendly-go/sendly"
)

// Create client with options
client := sendly.NewClient("sk_live_v1_xxx",
    sendly.WithBaseURL("https://api.sendly.live/v1"),
    sendly.WithTimeout(60*time.Second),
    sendly.WithMaxRetries(5),
    sendly.WithDebug(true),
)
```

## Messages

### Send an SMS

```go
message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
    To:   "+15551234567",
    Text: "Hello from Sendly!",
})
if err != nil {
    log.Fatal(err)
}

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
| +15550001234 | Success |
| +15550001001 | Invalid number |
| +15550001002 | Carrier rejected |
| +15550001003 | No credits |
| +15550001004 | Rate limited |

## Requirements

- Go 1.21+

## License

MIT
