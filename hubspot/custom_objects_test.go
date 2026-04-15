package hubspot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListCustomObjects_BuildsQueryAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"results": [{"id": "custom-123", "properties": {"name": "Custom row"}}],
				"paging": {"next": {"after": "cursor-2", "link": "https://example.test"}}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.ListCustomObjects(context.Background(), "2-123456", ListCustomObjectsRequest{
		After:      "cursor-1",
		Limit:      50,
		Properties: []string{"name", "external_id"},
	})
	if err != nil {
		t.Fatalf("ListCustomObjects returned error: %v", err)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodGet || req.URL.Path != "/crm/v3/objects/2-123456" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	query := req.URL.Query()
	if query.Get("after") != "cursor-1" {
		t.Fatalf("unexpected after query: %s", query.Get("after"))
	}

	if query.Get("limit") != "50" {
		t.Fatalf("unexpected limit query: %s", query.Get("limit"))
	}

	properties := query["properties"]
	if len(properties) != 2 || properties[0] != "name" || properties[1] != "external_id" {
		t.Fatalf("unexpected properties query: %#v", properties)
	}

	if len(resp.Results) != 1 || resp.Results[0].ID != "custom-123" {
		t.Fatalf("unexpected parsed results: %#v", resp.Results)
	}

	if resp.Paging == nil || resp.Paging.Next == nil || resp.Paging.Next.After != "cursor-2" {
		t.Fatalf("unexpected parsed paging: %#v", resp.Paging)
	}
}

func TestListCustomObjects_RequiresObjectTypeID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.ListCustomObjects(context.Background(), "  ", ListCustomObjectsRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestListCustomObjects_ReturnsDecodeError(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("{")),
			Header:     make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.ListCustomObjects(context.Background(), "2-123456", ListCustomObjectsRequest{})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}

	if !strings.Contains(err.Error(), "decode custom objects list response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchCustomObjects_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"total": 1,
				"results": [{"id": "custom-1", "properties": {"name": "Custom row"}}]
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.SearchCustomObjects(context.Background(), "2-123456", CustomObjectSearchRequest{
		Query:      "Custom",
		Limit:      10,
		Properties: []string{"name"},
		Sorts:      []string{"-createdate"},
		FilterGroups: []CustomObjectFilterGroup{{
			Filters: []CustomObjectFilter{{
				PropertyName: "status",
				Operator:     "EQ",
				Value:        "active",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("SearchCustomObjects returned error: %v", err)
	}

	if len(resp.Results) != 1 || resp.Results[0].ID != "custom-1" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/crm/v3/objects/2-123456/search" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected json content-type, got %q", got)
	}

	var payload CustomObjectSearchRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if payload.Query != "Custom" || payload.Limit != 10 {
		t.Fatalf("unexpected search payload: %#v", payload)
	}
}

func TestSearchCustomObjects_RequiresObjectTypeID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.SearchCustomObjects(context.Background(), "", CustomObjectSearchRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestGetCustomObject_BuildsPathQueryAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "123",
				"properties": {"name": "Custom row"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.GetCustomObject(context.Background(), "2-123456", "123", GetCustomObjectRequest{
		Properties:   []string{"name", "external_id"},
		Associations: []string{"contacts"},
		Archived:     true,
		IDProperty:   "external_id",
	})
	if err != nil {
		t.Fatalf("GetCustomObject returned error: %v", err)
	}

	if resp.ID != "123" || resp.Properties["name"] != "Custom row" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodGet || req.URL.Path != "/crm/v3/objects/2-123456/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	query := req.URL.Query()
	if query.Get("idProperty") != "external_id" {
		t.Fatalf("unexpected idProperty: %s", query.Get("idProperty"))
	}

	if query.Get("archived") != "true" {
		t.Fatalf("unexpected archived: %s", query.Get("archived"))
	}

	if len(query["properties"]) != 2 {
		t.Fatalf("unexpected properties query: %#v", query["properties"])
	}

	if len(query["associations"]) != 1 || query["associations"][0] != "contacts" {
		t.Fatalf("unexpected associations query: %#v", query["associations"])
	}
}

func TestGetCustomObject_RequiresObjectID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.GetCustomObject(context.Background(), "2-123456", "", GetCustomObjectRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestCreateCustomObject_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "created-1",
				"properties": {"name": "New row"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.CreateCustomObject(context.Background(), "2-123456", CreateCustomObjectRequest{
		Properties: map[string]string{"name": "New row"},
	})
	if err != nil {
		t.Fatalf("CreateCustomObject returned error: %v", err)
	}

	if resp.ID != "created-1" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/crm/v3/objects/2-123456" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	var payload CreateCustomObjectRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if payload.Properties["name"] != "New row" {
		t.Fatalf("unexpected create payload: %#v", payload)
	}
}

func TestCreateCustomObject_RequiresObjectTypeID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.CreateCustomObject(context.Background(), "", CreateCustomObjectRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestUpdateCustomObject_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "123",
				"properties": {"name": "Updated"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.UpdateCustomObject(context.Background(), "2-123456", "123", UpdateCustomObjectRequest{
		Properties: map[string]string{"name": "Updated"},
	})
	if err != nil {
		t.Fatalf("UpdateCustomObject returned error: %v", err)
	}

	if resp.ID != "123" || resp.Properties["name"] != "Updated" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPatch || req.URL.Path != "/crm/v3/objects/2-123456/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestUpdateCustomObject_RequiresObjectTypeID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.UpdateCustomObject(context.Background(), "", "123", UpdateCustomObjectRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestUpdateCustomObject_RequiresObjectID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.UpdateCustomObject(context.Background(), "2-123456", "", UpdateCustomObjectRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestEditCustomObject_DelegatesToUpdateCustomObject(t *testing.T) {
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

	_, err = client.EditCustomObject(context.Background(), "2-123456", "123", UpdateCustomObjectRequest{Properties: map[string]string{"name": "Updated"}})
	if err != nil {
		t.Fatalf("EditCustomObject returned error: %v", err)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPatch || req.URL.Path != "/crm/v3/objects/2-123456/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestDeleteCustomObject_UsesDeleteEndpoint(t *testing.T) {
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

	if err := client.DeleteCustomObject(context.Background(), "2-123456", "123"); err != nil {
		t.Fatalf("DeleteCustomObject returned error: %v", err)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodDelete || req.URL.Path != "/crm/v3/objects/2-123456/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestDeleteCustomObject_RequiresObjectTypeID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	err := client.DeleteCustomObject(context.Background(), "", "123")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestDeleteCustomObject_RequiresObjectID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	err := client.DeleteCustomObject(context.Background(), "2-123456", "")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
