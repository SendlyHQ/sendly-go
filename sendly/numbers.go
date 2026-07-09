package sendly

import (
	"context"
	"net/url"
)

// NumbersService provides access to buying and managing phone numbers.
type NumbersService struct {
	client *Client
}

// NumberCountry is a country in which numbers can be bought.
type NumberCountry struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	NumberTypes []string `json:"numberTypes"`
}

// NumberCountriesResponse wraps the buyable-countries list.
type NumberCountriesResponse struct {
	Countries []NumberCountry `json:"countries"`
}

// AvailableNumber is a purchasable number with its customer-facing monthly price.
type AvailableNumber struct {
	PhoneNumber string `json:"phoneNumber"`
	Country     string `json:"country"`
	NumberType  string `json:"numberType"`
	MonthlyCost string `json:"monthlyCost"`
	Currency    string `json:"currency"`
}

// AvailableNumbersResponse wraps an inventory search.
type AvailableNumbersResponse struct {
	Numbers []AvailableNumber `json:"numbers"`
}

// OwnedNumber is a number attached to the workspace.
type OwnedNumber struct {
	ID               string  `json:"id"`
	PhoneNumber      string  `json:"phoneNumber"`
	Status           string  `json:"status"`
	Source           string  `json:"source"`
	CountryCode      *string `json:"countryCode,omitempty"`
	PhoneNumberType  *string `json:"phoneNumberType,omitempty"`
	MonthlyCostCents *int    `json:"monthlyCostCents,omitempty"`
	// IsDefault is true when this is the workspace's default sending number.
	// Present on the single-number responses (Get, Update); omitted from List.
	IsDefault *bool `json:"isDefault,omitempty"`
	// RequirementsSubmittedAt is nil while the number still needs regulatory
	// documents; a value means documents were submitted and are under carrier review.
	RequirementsSubmittedAt *string `json:"requirementsSubmittedAt,omitempty"`
	// PendingCancellation is true if the number is scheduled for release at period end.
	PendingCancellation bool `json:"pendingCancellation"`
	// ScheduledReleaseAt is the time the number is scheduled for release, if any.
	ScheduledReleaseAt *string `json:"scheduledReleaseAt,omitempty"`
}

// OwnedNumbersResponse wraps the workspace's numbers.
type OwnedNumbersResponse struct {
	Numbers []OwnedNumber `json:"numbers"`
}

// UpdateNumberRequest is the body for NumbersService.Update. Supply at least
// one field; only these two mutations are supported:
//   - IsDefault = true: make this the workspace's default sender. The number
//     must be active, or the call fails with an invalid_state error.
//   - PendingCancellation = false: cancel a previously scheduled release and
//     keep the number.
//
// A body with neither field set fails with a no_supported_fields error.
type UpdateNumberRequest struct {
	IsDefault           *bool `json:"isDefault,omitempty"`
	PendingCancellation *bool `json:"pendingCancellation,omitempty"`
}

// ReleaseNumberResponse is the result of NumbersService.Release. An immediate
// release returns only Success=true; a live paid purchase is instead cancelled
// at the end of the paid period, in which case Scheduled is true and
// ScheduledReleaseAt carries the effective time.
type ReleaseNumberResponse struct {
	Success            bool    `json:"success"`
	Scheduled          bool    `json:"scheduled,omitempty"`
	ScheduledReleaseAt *string `json:"scheduledReleaseAt,omitempty"`
}

// ListAvailableRequest filters an inventory search. Country + Type are required.
type ListAvailableRequest struct {
	Country  string
	Type     string
	Contains string
}

// BuyNumberRequest places a purchase. After completing a hosted action, re-call
// Buy with the same fields plus ActionCode set to the completed action's code.
type BuyNumberRequest struct {
	PhoneNumber     string `json:"phoneNumber"`
	CountryCode     string `json:"countryCode"`
	PhoneNumberType string `json:"phoneNumberType"`
	MonthlyCost     string `json:"monthlyCost,omitempty"`
	ActionCode      string `json:"actionCode,omitempty"`
}

// NumberAction is the secure hosted hand-off returned when a buy needs
// registration documents or a payment method. Open URL, sign in, enter Code to
// prove terminal access, complete the step, then re-call Buy with ActionCode.
type NumberAction struct {
	URL        string `json:"url"`
	ActionCode string `json:"actionCode"`
	Code       string `json:"code"`
	ExpiresAt  int64  `json:"expiresAt"`
}

// BuyNumberResponse — Status is "provisioning", "documents_required", or
// "payment_required". When documents_required/payment_required, Action is set.
type BuyNumberResponse struct {
	Status       string        `json:"status"`
	Number       *OwnedNumber  `json:"number,omitempty"`
	Requirements []interface{} `json:"requirements,omitempty"`
	Action       *NumberAction `json:"action,omitempty"`
	Message      string        `json:"message,omitempty"`
}

// ListCountries returns the countries (and number types) available to buy.
func (s *NumbersService) ListCountries(ctx context.Context) (*NumberCountriesResponse, error) {
	var resp NumberCountriesResponse
	if err := s.client.request(ctx, "GET", "/numbers/countries", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListAvailable searches live inventory for a country + number type.
func (s *NumbersService) ListAvailable(ctx context.Context, req *ListAvailableRequest) (*AvailableNumbersResponse, error) {
	params := make(map[string]string)
	if req != nil {
		if req.Country != "" {
			params["country"] = req.Country
		}
		if req.Type != "" {
			params["type"] = req.Type
		}
		if req.Contains != "" {
			params["contains"] = req.Contains
		}
	}
	path := "/numbers/available" + buildQueryString(params)
	var resp AvailableNumbersResponse
	if err := s.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List returns the numbers attached to the workspace.
func (s *NumbersService) List(ctx context.Context) (*OwnedNumbersResponse, error) {
	var resp OwnedNumbersResponse
	if err := s.client.request(ctx, "GET", "/numbers", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get returns a single owned number by ID, including whether it is the
// workspace's default sender (IsDefault). Returns a NotFoundError if no such
// number exists in the workspace.
func (s *NumbersService) Get(ctx context.Context, id string) (*OwnedNumber, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "number ID is required"}}
	}
	var resp OwnedNumber
	if err := s.client.request(ctx, "GET", "/numbers/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update mutates a single owned number and returns its full record. Supply at
// least one supported field on req (see UpdateNumberRequest): set IsDefault to
// true to make it the workspace's default sender, and/or set PendingCancellation
// to false to cancel a scheduled release ("keep this number").
func (s *NumbersService) Update(ctx context.Context, id string, req *UpdateNumberRequest) (*OwnedNumber, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "number ID is required"}}
	}
	if req == nil || (req.IsDefault == nil && req.PendingCancellation == nil) {
		return nil, &ValidationError{APIError: APIError{Message: "provide at least one of isDefault or pendingCancellation"}}
	}
	var resp OwnedNumber
	if err := s.client.request(ctx, "PATCH", "/numbers/"+url.PathEscape(id), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Release releases a number you own. A live paid purchase is cancelled at the
// end of the paid period — the response then carries Scheduled=true plus a
// ScheduledReleaseAt; everything else is released immediately (Success=true).
func (s *NumbersService) Release(ctx context.Context, id string) (*ReleaseNumberResponse, error) {
	if id == "" {
		return nil, &ValidationError{APIError: APIError{Message: "number ID is required"}}
	}
	var resp ReleaseNumberResponse
	if err := s.client.request(ctx, "DELETE", "/numbers/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Buy purchases a number. Inspect the returned Status; on documents_required or
// payment_required, hand the user Action.URL + Action.Code, then re-call Buy
// with ActionCode set once the hosted action completes.
func (s *NumbersService) Buy(ctx context.Context, req *BuyNumberRequest) (*BuyNumberResponse, error) {
	var resp BuyNumberResponse
	if err := s.client.request(ctx, "POST", "/numbers/buy", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
