package sendly

import (
	"context"
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
	ID          string                 `json:"id"`
	PhoneNumber string                 `json:"phone_number"`
	Name        *string                `json:"name,omitempty"`
	Email       *string                `json:"email,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
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
