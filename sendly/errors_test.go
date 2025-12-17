package sendly

import (
	"errors"
	"testing"
)

func TestSendlyError_Error(t *testing.T) {
	err := &SendlyError{
		APIError: APIError{
			Code:    "TEST_ERROR",
			Message: "Test error message",
		},
		StatusCode: 500,
	}

	expected := "sendly: Test error message (code: TEST_ERROR, status: 500)"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestAuthenticationError_Error(t *testing.T) {
	err := &AuthenticationError{
		APIError: APIError{
			Code:    "UNAUTHORIZED",
			Message: "Invalid API key",
		},
	}

	expected := "sendly: authentication failed: Invalid API key"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestRateLimitError_Error(t *testing.T) {
	tests := []struct {
		name        string
		err         *RateLimitError
		expectedMsg string
	}{
		{
			name: "with retry after",
			err: &RateLimitError{
				APIError: APIError{
					Code:    "RATE_LIMIT_EXCEEDED",
					Message: "Too many requests",
				},
				RetryAfter: 60,
			},
			expectedMsg: "sendly: rate limit exceeded, retry after 60 seconds",
		},
		{
			name: "without retry after",
			err: &RateLimitError{
				APIError: APIError{
					Code:    "RATE_LIMIT_EXCEEDED",
					Message: "Too many requests",
				},
				RetryAfter: 0,
			},
			expectedMsg: "sendly: rate limit exceeded: Too many requests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expectedMsg {
				t.Errorf("expected error message '%s', got '%s'", tt.expectedMsg, tt.err.Error())
			}
		})
	}
}

func TestInsufficientCreditsError_Error(t *testing.T) {
	err := &InsufficientCreditsError{
		APIError: APIError{
			Code:    "INSUFFICIENT_CREDITS",
			Message: "Not enough credits",
		},
	}

	expected := "sendly: insufficient credits: Not enough credits"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name        string
		err         *ValidationError
		expectedMsg string
	}{
		{
			name: "with wrapped error",
			err: &ValidationError{
				APIError: APIError{
					Code:    "VALIDATION_ERROR",
					Message: "Invalid input",
				},
				Err: errors.New("underlying error"),
			},
			expectedMsg: "sendly: validation error: underlying error",
		},
		{
			name: "without wrapped error",
			err: &ValidationError{
				APIError: APIError{
					Code:    "VALIDATION_ERROR",
					Message: "Invalid input",
				},
			},
			expectedMsg: "sendly: validation error: Invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expectedMsg {
				t.Errorf("expected error message '%s', got '%s'", tt.expectedMsg, tt.err.Error())
			}
		})
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := &ValidationError{
		APIError: APIError{
			Code:    "VALIDATION_ERROR",
			Message: "Invalid input",
		},
		Err: underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("expected unwrapped error to be underlying error, got %v", unwrapped)
	}
}

func TestNotFoundError_Error(t *testing.T) {
	err := &NotFoundError{
		APIError: APIError{
			Code:    "NOT_FOUND",
			Message: "Resource not found",
		},
	}

	expected := "sendly: not found: Resource not found"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestNetworkError_Error(t *testing.T) {
	tests := []struct {
		name        string
		err         *NetworkError
		expectedMsg string
	}{
		{
			name: "with wrapped error",
			err: &NetworkError{
				Message: "connection failed",
				Err:     errors.New("dial tcp: connection refused"),
			},
			expectedMsg: "sendly: network error: connection failed: dial tcp: connection refused",
		},
		{
			name: "without wrapped error",
			err: &NetworkError{
				Message: "connection failed",
			},
			expectedMsg: "sendly: network error: connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expectedMsg {
				t.Errorf("expected error message '%s', got '%s'", tt.expectedMsg, tt.err.Error())
			}
		})
	}
}

func TestNetworkError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("dial tcp: connection refused")
	err := &NetworkError{
		Message: "connection failed",
		Err:     underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("expected unwrapped error to be underlying error, got %v", unwrapped)
	}
}

func TestIsAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "authentication error",
			err: &AuthenticationError{
				APIError: APIError{
					Code:    "UNAUTHORIZED",
					Message: "Invalid API key",
				},
			},
			expected: true,
		},
		{
			name: "validation error",
			err: &ValidationError{
				APIError: APIError{
					Code:    "VALIDATION_ERROR",
					Message: "Invalid input",
				},
			},
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthenticationError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "rate limit error",
			err: &RateLimitError{
				APIError: APIError{
					Code:    "RATE_LIMIT_EXCEEDED",
					Message: "Too many requests",
				},
			},
			expected: true,
		},
		{
			name: "authentication error",
			err: &AuthenticationError{
				APIError: APIError{
					Code:    "UNAUTHORIZED",
					Message: "Invalid API key",
				},
			},
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRateLimitError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsInsufficientCreditsError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "insufficient credits error",
			err: &InsufficientCreditsError{
				APIError: APIError{
					Code:    "INSUFFICIENT_CREDITS",
					Message: "Not enough credits",
				},
			},
			expected: true,
		},
		{
			name: "authentication error",
			err: &AuthenticationError{
				APIError: APIError{
					Code:    "UNAUTHORIZED",
					Message: "Invalid API key",
				},
			},
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInsufficientCreditsError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "validation error",
			err: &ValidationError{
				APIError: APIError{
					Code:    "VALIDATION_ERROR",
					Message: "Invalid input",
				},
			},
			expected: true,
		},
		{
			name: "authentication error",
			err: &AuthenticationError{
				APIError: APIError{
					Code:    "UNAUTHORIZED",
					Message: "Invalid API key",
				},
			},
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidationError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "not found error",
			err: &NotFoundError{
				APIError: APIError{
					Code:    "NOT_FOUND",
					Message: "Resource not found",
				},
			},
			expected: true,
		},
		{
			name: "authentication error",
			err: &AuthenticationError{
				APIError: APIError{
					Code:    "UNAUTHORIZED",
					Message: "Invalid API key",
				},
			},
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "network error",
			err: &NetworkError{
				Message: "connection failed",
				Err:     errors.New("dial tcp: connection refused"),
			},
			expected: true,
		},
		{
			name: "authentication error",
			err: &AuthenticationError{
				APIError: APIError{
					Code:    "UNAUTHORIZED",
					Message: "Invalid API key",
				},
			},
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAllErrorTypes_TypeAssertion(t *testing.T) {
	// Test that all error types implement the error interface
	var _ error = &SendlyError{}
	var _ error = &AuthenticationError{}
	var _ error = &RateLimitError{}
	var _ error = &InsufficientCreditsError{}
	var _ error = &ValidationError{}
	var _ error = &NotFoundError{}
	var _ error = &NetworkError{}
}

func TestErrorTypesWithDetails(t *testing.T) {
	details := map[string]interface{}{
		"field":  "phone",
		"reason": "invalid format",
	}

	err := &ValidationError{
		APIError: APIError{
			Code:    "VALIDATION_ERROR",
			Message: "Invalid phone number",
			Details: details,
		},
	}

	if err.Details == nil {
		t.Error("expected Details to be set")
	}
	if err.Details["field"] != "phone" {
		t.Errorf("expected field to be 'phone', got '%v'", err.Details["field"])
	}
	if err.Details["reason"] != "invalid format" {
		t.Errorf("expected reason to be 'invalid format', got '%v'", err.Details["reason"])
	}
}
