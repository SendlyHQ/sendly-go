package sendly

import "fmt"

// SendlyError is the base error type for Sendly API errors.
type SendlyError struct {
	APIError
	StatusCode int
}

func (e *SendlyError) Error() string {
	return fmt.Sprintf("sendly: %s (code: %s, status: %d)", e.Message, e.Code, e.StatusCode)
}

// AuthenticationError indicates invalid or missing API credentials.
type AuthenticationError struct {
	APIError
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("sendly: authentication failed: %s", e.Message)
}

// RateLimitError indicates the rate limit has been exceeded.
type RateLimitError struct {
	APIError
	// RetryAfter is the number of seconds to wait before retrying.
	RetryAfter int
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("sendly: rate limit exceeded, retry after %d seconds", e.RetryAfter)
	}
	return fmt.Sprintf("sendly: rate limit exceeded: %s", e.Message)
}

// InsufficientCreditsError indicates the account has insufficient credits.
type InsufficientCreditsError struct {
	APIError
}

func (e *InsufficientCreditsError) Error() string {
	return fmt.Sprintf("sendly: insufficient credits: %s", e.Message)
}

// ValidationError indicates invalid request parameters.
type ValidationError struct {
	APIError
	Err error
}

func (e *ValidationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("sendly: validation error: %v", e.Err)
	}
	return fmt.Sprintf("sendly: validation error: %s", e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// NotFoundError indicates the requested resource was not found.
type NotFoundError struct {
	APIError
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("sendly: not found: %s", e.Message)
}

// NetworkError indicates a network-level error.
type NetworkError struct {
	Message string
	Err     error
}

func (e *NetworkError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("sendly: network error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("sendly: network error: %s", e.Message)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// IsAuthenticationError checks if the error is an authentication error.
func IsAuthenticationError(err error) bool {
	_, ok := err.(*AuthenticationError)
	return ok
}

// IsRateLimitError checks if the error is a rate limit error.
func IsRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

// IsInsufficientCreditsError checks if the error is an insufficient credits error.
func IsInsufficientCreditsError(err error) bool {
	_, ok := err.(*InsufficientCreditsError)
	return ok
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// IsNotFoundError checks if the error is a not found error.
func IsNotFoundError(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// IsNetworkError checks if the error is a network error.
func IsNetworkError(err error) bool {
	_, ok := err.(*NetworkError)
	return ok
}
