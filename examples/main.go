package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/SendlyHQ/sendly-go/sendly"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("SENDLY_API_KEY")
	if apiKey == "" {
		apiKey = "sk_test_v1_example" // For demonstration
	}

	// Create client with options
	client := sendly.NewClient(apiKey,
		sendly.WithTimeout(30*time.Second),
		sendly.WithMaxRetries(3),
	)

	ctx := context.Background()

	// Example 1: Send an SMS
	fmt.Println("=== Sending SMS ===")
	message, err := client.Messages.Send(ctx, &sendly.SendMessageRequest{
		To:   "+15551234567",
		Text: "Hello from Sendly Go SDK!",
	})
	if err != nil {
		handleError(err)
	} else {
		fmt.Printf("Message sent!\n")
		fmt.Printf("  ID: %s\n", message.ID)
		fmt.Printf("  Status: %s\n", message.Status)
		fmt.Printf("  Credits used: %d\n", message.CreditsUsed)
	}

	// Example 2: List messages
	fmt.Println("\n=== Listing Messages ===")
	listResp, err := client.Messages.List(ctx, &sendly.ListMessagesRequest{
		Limit: 10,
	})
	if err != nil {
		handleError(err)
	} else {
		fmt.Printf("Found %d messages\n", len(listResp.Data))
		for _, msg := range listResp.Data {
			fmt.Printf("  - %s: %s (%s)\n", msg.ID, msg.To, msg.Status)
		}
	}

	// Example 3: Get a specific message
	fmt.Println("\n=== Getting Message ===")
	if message != nil {
		msg, err := client.Messages.Get(ctx, message.ID)
		if err != nil {
			handleError(err)
		} else {
			fmt.Printf("Message details:\n")
			fmt.Printf("  ID: %s\n", msg.ID)
			fmt.Printf("  To: %s\n", msg.To)
			fmt.Printf("  Text: %s\n", msg.Text)
			fmt.Printf("  Status: %s\n", msg.Status)
			fmt.Printf("  Created: %s\n", msg.CreatedAt)
		}
	}
}

func handleError(err error) {
	switch {
	case sendly.IsAuthenticationError(err):
		log.Printf("Authentication failed: %v", err)
	case sendly.IsRateLimitError(err):
		log.Printf("Rate limit exceeded: %v", err)
	case sendly.IsInsufficientCreditsError(err):
		log.Printf("Insufficient credits: %v", err)
	case sendly.IsValidationError(err):
		log.Printf("Validation error: %v", err)
	case sendly.IsNotFoundError(err):
		log.Printf("Not found: %v", err)
	case sendly.IsNetworkError(err):
		log.Printf("Network error: %v", err)
	default:
		log.Printf("Error: %v", err)
	}
}
