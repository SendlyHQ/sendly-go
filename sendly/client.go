// Package sendly provides a Go client for the Sendly SMS API.
package sendly

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

const (
	// DefaultBaseURL is the default Sendly API base URL.
	DefaultBaseURL = "https://sendly.live/api/v1"
	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second
	// Version is the SDK version.
	Version = "1.2.0"
)

// Client is the Sendly API client.
type Client struct {
	// BaseURL is the API base URL.
	BaseURL string
	// APIKey is the authentication key.
	APIKey string
	// HTTPClient is the underlying HTTP client.
	HTTPClient *http.Client
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int
	// Timeout is the request timeout.
	Timeout time.Duration
	// Debug enables debug logging.
	Debug bool

	// Messages provides access to message operations.
	Messages *MessagesService

	rateLimiter *rate.Limiter
}

// ClientOption is a function that configures the client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.BaseURL = baseURL
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.Timeout = timeout
		c.HTTPClient.Timeout = timeout
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.HTTPClient = httpClient
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		c.MaxRetries = maxRetries
	}
}

// WithDebug enables debug mode.
func WithDebug(debug bool) ClientOption {
	return func(c *Client) {
		c.Debug = debug
	}
}

// NewClient creates a new Sendly API client.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		BaseURL: DefaultBaseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		MaxRetries:  3,
		Timeout:     DefaultTimeout,
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 10), // 10 requests per second
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Messages = &MessagesService{client: c}

	return c
}

// request performs an HTTP request with retries and rate limiting.
func (c *Client) request(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return &NetworkError{Message: "rate limiter error", Err: err}
	}

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := c.doRequest(ctx, method, path, body, result)
		if err == nil {
			return nil
		}

		// Don't retry on certain errors
		if _, ok := err.(*AuthenticationError); ok {
			return err
		}
		if _, ok := err.(*ValidationError); ok {
			return err
		}
		if _, ok := err.(*NotFoundError); ok {
			return err
		}
		if _, ok := err.(*InsufficientCreditsError); ok {
			return err
		}

		lastErr = err

		// Check for rate limit error with Retry-After
		if rateLimitErr, ok := err.(*RateLimitError); ok {
			if rateLimitErr.RetryAfter > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(rateLimitErr.RetryAfter) * time.Second):
				}
			}
		}
	}

	return lastErr
}

// doRequest performs a single HTTP request.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	fullURL := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return &ValidationError{APIError: APIError{Message: "failed to marshal request body"}, Err: err}
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return &NetworkError{Message: "failed to create request", Err: err}
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sendly-go/"+Version)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return &NetworkError{Message: "request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &NetworkError{Message: "failed to read response body", Err: err}
	}

	if resp.StatusCode >= 400 {
		return c.handleErrorResponse(resp, respBody)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return &NetworkError{Message: "failed to unmarshal response", Err: err}
		}
	}

	return nil
}

// handleErrorResponse converts HTTP error responses to typed errors.
func (c *Client) handleErrorResponse(resp *http.Response, body []byte) error {
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		apiErr = APIError{
			Code:    "UNKNOWN_ERROR",
			Message: string(body),
		}
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return &AuthenticationError{
			APIError: apiErr,
		}
	case http.StatusTooManyRequests:
		retryAfter := 0
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			retryAfter, _ = strconv.Atoi(ra)
		}
		return &RateLimitError{
			APIError:   apiErr,
			RetryAfter: retryAfter,
		}
	case http.StatusPaymentRequired:
		return &InsufficientCreditsError{
			APIError: apiErr,
		}
	case http.StatusNotFound:
		return &NotFoundError{
			APIError: apiErr,
		}
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return &ValidationError{
			APIError: apiErr,
		}
	default:
		return &SendlyError{
			APIError:   apiErr,
			StatusCode: resp.StatusCode,
		}
	}
}

// buildQueryString builds a query string from parameters.
func buildQueryString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	values := url.Values{}
	for k, v := range params {
		if v != "" {
			values.Set(k, v)
		}
	}

	if len(values) == 0 {
		return ""
	}

	return "?" + values.Encode()
}
