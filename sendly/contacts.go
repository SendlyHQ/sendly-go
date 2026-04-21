package sendly

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

type ContactsService struct {
	client *Client
	Lists  *ContactListsService
}

type ContactListsService struct {
	client *Client
}

type Contact struct {
	ID                 string                 `json:"id"`
	PhoneNumber        string                 `json:"phone_number"`
	Name               *string                `json:"name,omitempty"`
	Email              *string                `json:"email,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	OptedOut           *bool                  `json:"opted_out,omitempty"`
	LineType           *string                `json:"line_type,omitempty"`
	CarrierName        *string                `json:"carrier_name,omitempty"`
	LineTypeCheckedAt  *string                `json:"line_type_checked_at,omitempty"`
	InvalidReason      *string                `json:"invalid_reason,omitempty"`
	InvalidatedAt      *string                `json:"invalidated_at,omitempty"`
	UserMarkedValidAt  *string                `json:"user_marked_valid_at,omitempty"`
	CreatedAt          string                 `json:"created_at"`
	UpdatedAt          string                 `json:"updated_at"`
}

type CheckNumbersRequest struct {
	ListID string `json:"listId,omitempty"`
	Force  bool   `json:"force,omitempty"`
}

type CheckNumbersResponse struct {
	Success        bool   `json:"success"`
	AlreadyRunning bool   `json:"alreadyRunning,omitempty"`
	Message        string `json:"message,omitempty"`
}

// BulkMarkValidRequest scopes a bulk mark-valid call. Pass either IDs (up to
// 10,000 per call) OR ListID — not both. Foreign ids silently no-op via the
// per-organization filter.
type BulkMarkValidRequest struct {
	IDs    []string `json:"ids,omitempty"`
	ListID string   `json:"listId,omitempty"`
}

// BulkMarkValidResponse reports how many contacts actually had their invalid
// flag cleared.
type BulkMarkValidResponse struct {
	Cleared int `json:"cleared"`
}

type ContactList struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  *string `json:"description,omitempty"`
	ContactCount int     `json:"contact_count"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type ContactListResponse struct {
	Contacts []Contact `json:"contacts"`
	Total    int       `json:"total"`
	Limit    int       `json:"limit"`
	Offset   int       `json:"offset"`
}

type ContactListsResponse struct {
	Lists  []ContactList `json:"lists"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type ListContactsRequest struct {
	Limit  int
	Offset int
	Search string
	ListID string
}

type CreateContactRequest struct {
	PhoneNumber string                 `json:"phone_number"`
	Name        string                 `json:"name,omitempty"`
	Email       string                 `json:"email,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateContactRequest struct {
	PhoneNumber string                 `json:"phone_number,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Email       string                 `json:"email,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type CreateContactListRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type UpdateContactListRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type AddContactsRequest struct {
	ContactIDs []string `json:"contact_ids"`
}

type ImportContactItem struct {
	Phone     string `json:"phone"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	OptedInAt string `json:"optedInAt,omitempty"`
}

type ImportContactsRequest struct {
	Contacts  []ImportContactItem `json:"contacts"`
	ListID    string              `json:"listId,omitempty"`
	OptedInAt string              `json:"optedInAt,omitempty"`
}

type ImportContactsError struct {
	Index int    `json:"index"`
	Phone string `json:"phone"`
	Error string `json:"error"`
}

type ImportContactsResponse struct {
	Imported          int                   `json:"imported"`
	SkippedDuplicates int                   `json:"skippedDuplicates"`
	Errors            []ImportContactsError `json:"errors"`
	TotalErrors       int                   `json:"totalErrors"`
}

func (s *ContactsService) List(ctx context.Context, req *ListContactsRequest) (*ContactListResponse, error) {
	params := make(map[string]string)
	if req != nil {
		if req.Limit > 0 {
			params["limit"] = strconv.Itoa(req.Limit)
		}
		if req.Offset > 0 {
			params["offset"] = strconv.Itoa(req.Offset)
		}
		if req.Search != "" {
			params["search"] = req.Search
		}
		if req.ListID != "" {
			params["list_id"] = req.ListID
		}
	}

	path := "/contacts" + buildQueryString(params)
	var resp ContactListResponse
	err := s.client.request(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactsService) Get(ctx context.Context, id string) (*Contact, error) {
	var resp Contact
	err := s.client.request(ctx, "GET", fmt.Sprintf("/contacts/%s", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactsService) Create(ctx context.Context, req *CreateContactRequest) (*Contact, error) {
	var resp Contact
	err := s.client.request(ctx, "POST", "/contacts", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactsService) Update(ctx context.Context, id string, req *UpdateContactRequest) (*Contact, error) {
	var resp Contact
	err := s.client.request(ctx, "PATCH", fmt.Sprintf("/contacts/%s", id), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactsService) Delete(ctx context.Context, id string) error {
	return s.client.request(ctx, "DELETE", fmt.Sprintf("/contacts/%s", id), nil, nil)
}

// MarkValid clears the invalid flag on a contact so future campaigns include it again.
// Contacts get auto-flagged as invalid when a send fails with a terminal bad-number
// error (landline, invalid number) or when a carrier lookup reports they can't receive SMS.
func (s *ContactsService) MarkValid(ctx context.Context, id string) (*Contact, error) {
	var resp Contact
	err := s.client.request(ctx, "POST", fmt.Sprintf("/contacts/%s/mark-valid", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// BulkMarkValid clears the invalid flag on many contacts at once — the escape
// hatch for when auto-flag misclassifies at scale. Pass either an explicit id
// array (up to 10,000 per call) OR a ListID, not both. Foreign ids silently
// no-op via the per-organization filter.
//
// Returns the number of contacts whose flag was actually cleared. Already-clean
// contacts and foreign ids don't count.
func (s *ContactsService) BulkMarkValid(ctx context.Context, req BulkMarkValidRequest) (*BulkMarkValidResponse, error) {
	hasIDs := len(req.IDs) > 0
	hasListID := req.ListID != ""
	if !hasIDs && !hasListID {
		return nil, errors.New("BulkMarkValid requires either IDs or ListID")
	}
	if hasIDs && hasListID {
		return nil, errors.New("BulkMarkValid accepts IDs OR ListID, not both")
	}

	var body interface{}
	if hasIDs {
		body = struct {
			IDs []string `json:"ids"`
		}{IDs: req.IDs}
	} else {
		body = struct {
			ListID string `json:"listId"`
		}{ListID: req.ListID}
	}

	var resp BulkMarkValidResponse
	err := s.client.request(ctx, "POST", "/contacts/bulk-mark-valid", body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CheckNumbers triggers a background carrier lookup across your contacts. Landlines
// and other non-SMS-capable numbers are auto-excluded from future campaigns. Results
// populate the LineType, CarrierName, and InvalidReason fields on affected contacts.
// Idempotent: re-triggering while a lookup is running for the same scope is a no-op.
// Pass nil to check all un-checked contacts, or scope via &CheckNumbersRequest{ListID: "lst_xxx"}.
func (s *ContactsService) CheckNumbers(ctx context.Context, req *CheckNumbersRequest) (*CheckNumbersResponse, error) {
	var resp CheckNumbersResponse
	if req == nil {
		req = &CheckNumbersRequest{}
	}
	err := s.client.request(ctx, "POST", "/contacts/lookup", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactsService) Import(ctx context.Context, req *ImportContactsRequest) (*ImportContactsResponse, error) {
	var resp ImportContactsResponse
	err := s.client.request(ctx, "POST", "/contacts/import", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactListsService) List(ctx context.Context) (*ContactListsResponse, error) {
	var resp ContactListsResponse
	err := s.client.request(ctx, "GET", "/contact-lists", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactListsService) Get(ctx context.Context, id string) (*ContactList, error) {
	var resp ContactList
	err := s.client.request(ctx, "GET", fmt.Sprintf("/contact-lists/%s", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactListsService) Create(ctx context.Context, req *CreateContactListRequest) (*ContactList, error) {
	var resp ContactList
	err := s.client.request(ctx, "POST", "/contact-lists", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactListsService) Update(ctx context.Context, id string, req *UpdateContactListRequest) (*ContactList, error) {
	var resp ContactList
	err := s.client.request(ctx, "PATCH", fmt.Sprintf("/contact-lists/%s", id), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *ContactListsService) Delete(ctx context.Context, id string) error {
	return s.client.request(ctx, "DELETE", fmt.Sprintf("/contact-lists/%s", id), nil, nil)
}

func (s *ContactListsService) AddContacts(ctx context.Context, listID string, contactIDs []string) error {
	req := &AddContactsRequest{ContactIDs: contactIDs}
	return s.client.request(ctx, "POST", fmt.Sprintf("/contact-lists/%s/contacts", listID), req, nil)
}

func (s *ContactListsService) RemoveContact(ctx context.Context, listID, contactID string) error {
	return s.client.request(ctx, "DELETE", fmt.Sprintf("/contact-lists/%s/contacts/%s", listID, contactID), nil, nil)
}
