package hubspot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ListContactsRequest controls query options for listing contacts.
type ListContactsRequest struct {
	After      string
	Limit      int
	Properties []string
}

// Contact represents a minimal HubSpot contact record.
type Contact struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties,omitempty"`
}

// ContactFilter defines one search filter clause.
type ContactFilter struct {
	PropertyName string   `json:"propertyName"`
	Operator     string   `json:"operator"`
	Value        string   `json:"value,omitempty"`
	HighValue    string   `json:"highValue,omitempty"`
	Values       []string `json:"values,omitempty"`
}

// ContactFilterGroup defines an OR group of AND filters.
type ContactFilterGroup struct {
	Filters []ContactFilter `json:"filters"`
}

// ContactSearchRequest controls query options for the contacts search endpoint.
type ContactSearchRequest struct {
	Query        string               `json:"query,omitempty"`
	Limit        int                  `json:"limit,omitempty"`
	After        string               `json:"after,omitempty"`
	Sorts        []string             `json:"sorts,omitempty"`
	Properties   []string             `json:"properties,omitempty"`
	FilterGroups []ContactFilterGroup `json:"filterGroups,omitempty"`
}

// ContactsSearchResponse is the minimal response model from contacts search endpoint.
type ContactsSearchResponse struct {
	Total   int            `json:"total,omitempty"`
	Results []Contact      `json:"results"`
	Paging  *PagingDetails `json:"paging,omitempty"`
}

// GetContactRequest controls query options for reading a single contact.
type GetContactRequest struct {
	Properties   []string
	Associations []string
	Archived     bool
	IDProperty   string
}

// CreateContactRequest is the minimal payload for creating a contact.
type CreateContactRequest struct {
	Properties map[string]string `json:"properties"`
}

// UpdateContactRequest is the minimal payload for updating a contact.
type UpdateContactRequest struct {
	Properties map[string]string `json:"properties"`
}

// ContactsListResponse is the minimal response model returned by the contacts list endpoint.
type ContactsListResponse struct {
	Results []Contact      `json:"results"`
	Paging  *PagingDetails `json:"paging,omitempty"`
}

// PagingDetails mirrors HubSpot paging blocks for list endpoints.
type PagingDetails struct {
	Next *PagingNext `json:"next,omitempty"`
}

// PagingNext contains cursor details for the next page request.
type PagingNext struct {
	After string `json:"after,omitempty"`
	Link  string `json:"link,omitempty"`
}

// ListContacts fetches a page of contacts from HubSpot.
func (c *Client) ListContacts(ctx context.Context, req ListContactsRequest) (ContactsListResponse, error) {
	query := url.Values{}
	if req.After != "" {
		query.Set("after", req.After)
	}

	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}

	for _, property := range req.Properties {
		if property != "" {
			query.Add("properties", property)
		}
	}

	path := "/crm/v3/objects/contacts"
	if encoded := query.Encode(); encoded != "" {
		path = fmt.Sprintf("%s?%s", path, encoded)
	}

	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return ContactsListResponse{}, fmt.Errorf("list contacts: %w", err)
	}
	defer resp.Body.Close()

	var parsed ContactsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return ContactsListResponse{}, fmt.Errorf("decode contacts list response: %w", err)
	}

	return parsed, nil
}

// SearchContacts executes a contacts search request.
func (c *Client) SearchContacts(ctx context.Context, req ContactSearchRequest) (ContactsSearchResponse, error) {
	resp, err := c.doJSON(ctx, http.MethodPost, "/crm/v3/objects/contacts/search", req)
	if err != nil {
		return ContactsSearchResponse{}, fmt.Errorf("search contacts: %w", err)
	}
	defer resp.Body.Close()

	var parsed ContactsSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return ContactsSearchResponse{}, fmt.Errorf("decode contacts search response: %w", err)
	}

	return parsed, nil
}

// GetContact reads a single contact by id.
func (c *Client) GetContact(ctx context.Context, contactID string, req GetContactRequest) (Contact, error) {
	trimmedID := strings.TrimSpace(contactID)
	if trimmedID == "" {
		return Contact{}, errors.New("contact id is required")
	}

	query := url.Values{}
	for _, property := range req.Properties {
		if property != "" {
			query.Add("properties", property)
		}
	}

	for _, association := range req.Associations {
		if association != "" {
			query.Add("associations", association)
		}
	}

	if req.Archived {
		query.Set("archived", strconv.FormatBool(req.Archived))
	}

	if strings.TrimSpace(req.IDProperty) != "" {
		query.Set("idProperty", req.IDProperty)
	}

	path := "/crm/v3/objects/contacts/" + url.PathEscape(trimmedID)
	if encoded := query.Encode(); encoded != "" {
		path = fmt.Sprintf("%s?%s", path, encoded)
	}

	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return Contact{}, fmt.Errorf("get contact id=%s: %w", trimmedID, err)
	}
	defer resp.Body.Close()

	var parsed Contact
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Contact{}, fmt.Errorf("decode get contact response id=%s: %w", trimmedID, err)
	}

	return parsed, nil
}

// CreateContact creates a single contact.
func (c *Client) CreateContact(ctx context.Context, req CreateContactRequest) (Contact, error) {
	resp, err := c.doJSON(ctx, http.MethodPost, "/crm/v3/objects/contacts", req)
	if err != nil {
		return Contact{}, fmt.Errorf("create contact: %w", err)
	}
	defer resp.Body.Close()

	var parsed Contact
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Contact{}, fmt.Errorf("decode create contact response: %w", err)
	}

	return parsed, nil
}

// UpdateContact updates a single contact by id.
func (c *Client) UpdateContact(ctx context.Context, contactID string, req UpdateContactRequest) (Contact, error) {
	trimmedID := strings.TrimSpace(contactID)
	if trimmedID == "" {
		return Contact{}, errors.New("contact id is required")
	}

	resp, err := c.doJSON(ctx, http.MethodPatch, "/crm/v3/objects/contacts/"+url.PathEscape(trimmedID), req)
	if err != nil {
		return Contact{}, fmt.Errorf("update contact id=%s: %w", trimmedID, err)
	}
	defer resp.Body.Close()

	var parsed Contact
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Contact{}, fmt.Errorf("decode update contact response id=%s: %w", trimmedID, err)
	}

	return parsed, nil
}

// EditContact updates a single contact by id.
func (c *Client) EditContact(ctx context.Context, contactID string, req UpdateContactRequest) (Contact, error) {
	return c.UpdateContact(ctx, contactID, req)
}

// DeleteContact deletes a single contact by id.
func (c *Client) DeleteContact(ctx context.Context, contactID string) error {
	trimmedID := strings.TrimSpace(contactID)
	if trimmedID == "" {
		return errors.New("contact id is required")
	}

	resp, err := c.Do(ctx, http.MethodDelete, "/crm/v3/objects/contacts/"+url.PathEscape(trimmedID), nil, nil)
	if err != nil {
		return fmt.Errorf("delete contact id=%s: %w", trimmedID, err)
	}
	defer resp.Body.Close()

	return nil
}
