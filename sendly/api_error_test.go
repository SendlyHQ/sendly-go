package sendly

import (
	"encoding/json"
	"testing"
)

func TestAPIErrorUnmarshal_ReadsErrorKey(t *testing.T) {
	apiErr, err := decodeAPIError([]byte(`{"error": "rcs_not_enabled", "message": "off"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr.Code != "rcs_not_enabled" {
		t.Errorf("expected Code to be 'rcs_not_enabled', got '%s'", apiErr.Code)
	}
	if apiErr.Message != "off" {
		t.Errorf("expected Message to be 'off', got '%s'", apiErr.Message)
	}
	if apiErr.Errors != nil {
		t.Errorf("expected no field errors, got %v", apiErr.Errors)
	}
}

func TestAPIErrorUnmarshal_CodeKeyWins(t *testing.T) {
	apiErr, err := decodeAPIError([]byte(`{"code": "explicit", "error": "fallback", "message": "m", "details": {"k": 1}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr.Code != "explicit" {
		t.Errorf("expected Code to be 'explicit', got '%s'", apiErr.Code)
	}
	if apiErr.Details["k"] != float64(1) {
		t.Errorf("expected Details to be kept, got %v", apiErr.Details)
	}
}

func TestAPIErrorUnmarshal_IgnoresNonStringErrorAndNonFieldErrors(t *testing.T) {
	body := `{"error": {"nested": true}, "message": "m", "errors": ["plain", "strings"]}`
	apiErr, err := decodeAPIError([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiErr.Code != "" {
		t.Errorf("expected Code to be empty, got '%s'", apiErr.Code)
	}
	if apiErr.Errors != nil {
		t.Errorf("expected Errors to be nil, got %v", apiErr.Errors)
	}
}

func TestAPIErrorUnmarshal_FieldErrors(t *testing.T) {
	var apiErr APIError
	body := `{"error": "rcs_invalid_content", "message": "m", "errors": [{"path": "devices", "message": "devices must be a list"}]}`
	if err := json.Unmarshal([]byte(body), &apiErr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apiErr.Errors) != 1 || apiErr.Errors[0].Path != "devices" || apiErr.Errors[0].Message != "devices must be a list" {
		t.Errorf("unexpected Errors: %v", apiErr.Errors)
	}
}

func TestAPIErrorMarshal_RoundTrip(t *testing.T) {
	in := APIError{Code: "c", Message: "m", Errors: []APIFieldError{{Path: "p", Message: "fm"}}}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out APIError
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Code != "c" || out.Message != "m" || len(out.Errors) != 1 || out.Errors[0].Path != "p" {
		t.Errorf("round trip mismatch: %+v", out)
	}
}

func TestTypedErrorsKeepTheirOwnFieldsWhenUnmarshalled(t *testing.T) {
	// APIError must not carry an UnmarshalJSON method: it is embedded in every
	// typed error, and a promoted method would decode only the embedded fields
	// and silently zero the wrapper's own.
	var rate RateLimitError
	if err := json.Unmarshal([]byte(`{"message": "slow down", "RetryAfter": 30}`), &rate); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate.RetryAfter != 30 {
		t.Errorf("expected RetryAfter 30, got %d", rate.RetryAfter)
	}
	if rate.Message != "slow down" {
		t.Errorf("expected the embedded Message to survive, got '%s'", rate.Message)
	}
}
