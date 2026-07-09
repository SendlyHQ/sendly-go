package sendly

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// LinksService handles branded URL-shortening operations.
//
// Mint branded short links for a destination URL, list the links your
// workspace has created (with click analytics), and enable or disable an
// individual link (a per-link kill switch). Branded, owned-domain short links
// improve deliverability — carriers filter public shorteners — and give you
// click data for analytics.
//
// These endpoints live under the versioned /api/v1 base and authenticate with
// your API key. URL shortening is gated behind the url_shortener rollout flag;
// calls return a not_found error until the flag is on for your account.
type LinksService struct {
	client *Client
}

// Create mints a branded short link for a destination URL. Uses the
// workspace's brand slug when one is configured. destinationURL must be an
// http:// or https:// URL.
func (s *LinksService) Create(ctx context.Context, destinationURL string) (*ShortLink, error) {
	if destinationURL == "" {
		return nil, &ValidationError{APIError: APIError{Message: "url is required"}}
	}
	if !strings.HasPrefix(destinationURL, "http://") && !strings.HasPrefix(destinationURL, "https://") {
		return nil, &ValidationError{APIError: APIError{Message: "url must be an http:// or https:// URL"}}
	}

	body := map[string]string{"url": destinationURL}

	var resp ShortLink
	err := s.client.request(ctx, "POST", "/links", body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// List retrieves the short links the workspace has created, newest first, with
// click counts and a 14-day daily click histogram.
func (s *LinksService) List(ctx context.Context, req *ListShortLinksRequest) (*ShortLinkListResponse, error) {
	params := make(map[string]string)

	if req != nil {
		if req.Limit > 0 {
			params["limit"] = strconv.Itoa(req.Limit)
		}
		if req.Offset > 0 {
			params["offset"] = strconv.Itoa(req.Offset)
		}
	}

	path := "/links" + buildQueryString(params)

	var resp ShortLinkListResponse
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Update enables or disables a short link (a per-link kill switch). A disabled
// link's redirect returns 404 until it is re-enabled.
func (s *LinksService) Update(ctx context.Context, code string, disabled bool) (*ShortLinkDisabledResponse, error) {
	if code == "" {
		return nil, &ValidationError{APIError: APIError{Message: "link code is required"}}
	}

	path := "/links/" + url.PathEscape(code)
	body := map[string]bool{"disabled": disabled}

	var resp ShortLinkDisabledResponse
	err := s.client.request(ctx, "PATCH", path, body, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Disable disables a short link (its redirect returns 404 until re-enabled).
// Convenience wrapper over Update.
func (s *LinksService) Disable(ctx context.Context, code string) (*ShortLinkDisabledResponse, error) {
	return s.Update(ctx, code, true)
}

// Enable re-enables a previously disabled short link. Convenience wrapper over
// Update.
func (s *LinksService) Enable(ctx context.Context, code string) (*ShortLinkDisabledResponse, error) {
	return s.Update(ctx, code, false)
}
