package hubspot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListContacts_BuildsQueryAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"results": [{"id": "123", "properties": {"email": "a@example.com"}}],
				"paging": {"next": {"after": "cursor-2", "link": "https://example.test"}}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{
		Token:      "token-123",
		HTTPClient: clientImpl,
		Retry: RetryPolicy{
			MaxAttempts: 1,
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.ListContacts(context.Background(), ListContactsRequest{
		After:      "cursor-1",
		Limit:      50,
		Properties: []string{"email", "firstname"},
	})
	if err != nil {
		t.Fatalf("ListContacts returned error: %v", err)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	requestURL := clientImpl.requests[0].URL
	query := requestURL.Query()
	if query.Get("after") != "cursor-1" {
		t.Fatalf("unexpected after query: %s", query.Get("after"))
	}

	if query.Get("limit") != "50" {
		t.Fatalf("unexpected limit query: %s", query.Get("limit"))
	}

	properties := query["properties"]
	if len(properties) != 2 || properties[0] != "email" || properties[1] != "firstname" {
		t.Fatalf("unexpected properties query: %#v", properties)
	}

	if len(resp.Results) != 1 || resp.Results[0].ID != "123" {
		t.Fatalf("unexpected parsed results: %#v", resp.Results)
	}

	if resp.Paging == nil || resp.Paging.Next == nil || resp.Paging.Next.After != "cursor-2" {
		t.Fatalf("unexpected parsed paging: %#v", resp.Paging)
	}
}

func TestListContacts_ReturnsDecodeError(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("{")),
			Header:     make(http.Header),
		}},
	}

	client, err := New(Config{
		Token:      "token-123",
		HTTPClient: clientImpl,
		Retry: RetryPolicy{
			MaxAttempts: 1,
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.ListContacts(context.Background(), ListContactsRequest{})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}

	if !strings.Contains(err.Error(), "decode contacts list response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchContacts_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"total": 1,
				"results": [{"id": "c-1", "properties": {"email": "a@example.com"}}]
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{
		Token:      "token-123",
		HTTPClient: clientImpl,
		Retry:      RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.SearchContacts(context.Background(), ContactSearchRequest{
		Query:      "example.com",
		Limit:      10,
		Properties: []string{"email"},
		Sorts:      []string{"-createdate"},
		FilterGroups: []ContactFilterGroup{{
			Filters: []ContactFilter{{
				PropertyName: "lifecyclestage",
				Operator:     "EQ",
				Value:        "lead",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("SearchContacts returned error: %v", err)
	}

	if len(resp.Results) != 1 || resp.Results[0].ID != "c-1" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPost {
		t.Fatalf("unexpected method: %s", req.Method)
	}

	if req.URL.Path != "/crm/v3/objects/contacts/search" {
		t.Fatalf("unexpected path: %s", req.URL.Path)
	}

	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected json content-type, got %q", got)
	}

	var payload ContactSearchRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if payload.Query != "example.com" || payload.Limit != 10 {
		t.Fatalf("unexpected search payload: %#v", payload)
	}
}

func TestGetContact_BuildsPathQueryAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "123",
				"properties": {"email": "a@example.com"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.GetContact(context.Background(), "123", GetContactRequest{
		Properties:   []string{"email", "firstname"},
		Associations: []string{"companies"},
		Archived:     true,
		IDProperty:   "email",
	})
	if err != nil {
		t.Fatalf("GetContact returned error: %v", err)
	}

	if resp.ID != "123" || resp.Properties["email"] != "a@example.com" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodGet {
		t.Fatalf("unexpected method: %s", req.Method)
	}

	if req.URL.Path != "/crm/v3/objects/contacts/123" {
		t.Fatalf("unexpected path: %s", req.URL.Path)
	}

	query := req.URL.Query()
	if query.Get("idProperty") != "email" {
		t.Fatalf("unexpected idProperty: %s", query.Get("idProperty"))
	}

	if query.Get("archived") != "true" {
		t.Fatalf("unexpected archived: %s", query.Get("archived"))
	}

	if len(query["properties"]) != 2 {
		t.Fatalf("unexpected properties query: %#v", query["properties"])
	}

	if len(query["associations"]) != 1 || query["associations"][0] != "companies" {
		t.Fatalf("unexpected associations query: %#v", query["associations"])
	}
}

func TestGetContact_RequiresContactID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.GetContact(context.Background(), "  ", GetContactRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestCreateContact_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "created-1",
				"properties": {"email": "new@example.com"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.CreateContact(context.Background(), CreateContactRequest{
		Properties: map[string]string{"email": "new@example.com"},
	})
	if err != nil {
		t.Fatalf("CreateContact returned error: %v", err)
	}

	if resp.ID != "created-1" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/crm/v3/objects/contacts" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	var payload CreateContactRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if payload.Properties["email"] != "new@example.com" {
		t.Fatalf("unexpected create payload: %#v", payload)
	}
}

func TestUpdateContact_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "123",
				"properties": {"firstname": "Updated"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.UpdateContact(context.Background(), "123", UpdateContactRequest{
		Properties: map[string]string{"firstname": "Updated"},
	})
	if err != nil {
		t.Fatalf("UpdateContact returned error: %v", err)
	}

	if resp.ID != "123" || resp.Properties["firstname"] != "Updated" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPatch || req.URL.Path != "/crm/v3/objects/contacts/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestUpdateContact_RequiresContactID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.UpdateContact(context.Background(), "", UpdateContactRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestEditContact_DelegatesToUpdateContact(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"123"}`)),
			Header:     make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.EditContact(context.Background(), "123", UpdateContactRequest{Properties: map[string]string{"firstname": "Updated"}})
	if err != nil {
		t.Fatalf("EditContact returned error: %v", err)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPatch || req.URL.Path != "/crm/v3/objects/contacts/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestDeleteContact_UsesDeleteEndpoint(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := client.DeleteContact(context.Background(), "123"); err != nil {
		t.Fatalf("DeleteContact returned error: %v", err)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodDelete || req.URL.Path != "/crm/v3/objects/contacts/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestDeleteContact_RequiresContactID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	err := client.DeleteContact(context.Background(), "")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
