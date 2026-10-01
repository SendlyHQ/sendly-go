package sendly

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func newPathSegmentServer(t *testing.T) (*Client, *int32, *string) {
	t.Helper()
	var requests int32
	var lastPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		lastPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)
	return NewClient("test-api-key", WithBaseURL(server.URL), WithMaxRetries(0)), &requests, &lastPath
}

func TestDotSegmentIDsAreRefusedBeforeSending(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		call func(c *Client) error
	}{
		{"revoke key ..", func(c *Client) error { return c.Enterprise.Workspaces.RevokeKey(ctx, "ws_1", "..") }},
		{"revoke key .", func(c *Client) error { return c.Enterprise.Workspaces.RevokeKey(ctx, "ws_1", ".") }},
		{"workspace ..", func(c *Client) error { _, err := c.Enterprise.Workspaces.GetVerification(ctx, ".."); return err }},
		{"contact empty", func(c *Client) error { _, err := c.Contacts.Get(ctx, ""); return err }},
		{"contact ..", func(c *Client) error { return c.Contacts.Delete(ctx, "..") }},
		{"upgrade start ..", func(c *Client) error {
			_, err := c.BusinessUpgrade.Start(ctx, "..", &StartUpgradeParams{}, nil)
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, requests, _ := newPathSegmentServer(t)
			err := tc.call(client)
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected *ValidationError, got %T: %v", err, err)
			}
			if n := atomic.LoadInt32(requests); n != 0 {
				t.Errorf("expected no request, got %d", n)
			}
		})
	}
}

func TestOrdinaryIDsWithDotsStillSend(t *testing.T) {
	ctx := context.Background()
	client, requests, lastPath := newPathSegmentServer(t)

	if err := client.Enterprise.Workspaces.RevokeKey(ctx, "ws_1", "key_1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *lastPath != "/enterprise/workspaces/ws_1/keys/key_1" {
		t.Errorf("unexpected path %q", *lastPath)
	}

	if err := client.Contacts.Delete(ctx, "..."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *lastPath != "/contacts/..." {
		t.Errorf("unexpected path %q", *lastPath)
	}

	if err := client.Contacts.Delete(ctx, "a.b"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *lastPath != "/contacts/a.b" {
		t.Errorf("unexpected path %q", *lastPath)
	}

	if n := atomic.LoadInt32(requests); n != 3 {
		t.Errorf("expected 3 requests, got %d", n)
	}
}
