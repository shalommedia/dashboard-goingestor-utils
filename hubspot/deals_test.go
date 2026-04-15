package hubspot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListDeals_BuildsQueryAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"results": [{"id": "deal-123", "properties": {"dealname": "Big deal"}}],
				"paging": {"next": {"after": "cursor-2", "link": "https://example.test"}}
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

	resp, err := client.ListDeals(context.Background(), ListDealsRequest{
		After:      "cursor-1",
		Limit:      50,
		Properties: []string{"dealname", "amount"},
	})
	if err != nil {
		t.Fatalf("ListDeals returned error: %v", err)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodGet || req.URL.Path != "/crm/v3/objects/deals" {
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
	if len(properties) != 2 || properties[0] != "dealname" || properties[1] != "amount" {
		t.Fatalf("unexpected properties query: %#v", properties)
	}

	if len(resp.Results) != 1 || resp.Results[0].ID != "deal-123" {
		t.Fatalf("unexpected parsed results: %#v", resp.Results)
	}

	if resp.Paging == nil || resp.Paging.Next == nil || resp.Paging.Next.After != "cursor-2" {
		t.Fatalf("unexpected parsed paging: %#v", resp.Paging)
	}
}

func TestListDeals_ReturnsDecodeError(t *testing.T) {
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
		Retry:      RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.ListDeals(context.Background(), ListDealsRequest{})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}

	if !strings.Contains(err.Error(), "decode deals list response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchDeals_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"total": 1,
				"results": [{"id": "d-1", "properties": {"dealname": "Big deal"}}]
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

	resp, err := client.SearchDeals(context.Background(), DealSearchRequest{
		Query:      "Big",
		Limit:      10,
		Properties: []string{"dealname"},
		Sorts:      []string{"-createdate"},
		FilterGroups: []DealFilterGroup{{
			Filters: []DealFilter{{
				PropertyName: "dealstage",
				Operator:     "EQ",
				Value:        "appointmentscheduled",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("SearchDeals returned error: %v", err)
	}

	if len(resp.Results) != 1 || resp.Results[0].ID != "d-1" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPost {
		t.Fatalf("unexpected method: %s", req.Method)
	}

	if req.URL.Path != "/crm/v3/objects/deals/search" {
		t.Fatalf("unexpected path: %s", req.URL.Path)
	}

	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected json content-type, got %q", got)
	}

	var payload DealSearchRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if payload.Query != "Big" || payload.Limit != 10 {
		t.Fatalf("unexpected search payload: %#v", payload)
	}
}

func TestGetDeal_BuildsPathQueryAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "123",
				"properties": {"dealname": "Big deal"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.GetDeal(context.Background(), "123", GetDealRequest{
		Properties:   []string{"dealname", "amount"},
		Associations: []string{"contacts"},
		Archived:     true,
		IDProperty:   "dealname",
	})
	if err != nil {
		t.Fatalf("GetDeal returned error: %v", err)
	}

	if resp.ID != "123" || resp.Properties["dealname"] != "Big deal" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodGet {
		t.Fatalf("unexpected method: %s", req.Method)
	}

	if req.URL.Path != "/crm/v3/objects/deals/123" {
		t.Fatalf("unexpected path: %s", req.URL.Path)
	}

	query := req.URL.Query()
	if query.Get("idProperty") != "dealname" {
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

func TestGetDeal_RequiresDealID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.GetDeal(context.Background(), "  ", GetDealRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestCreateDeal_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "created-1",
				"properties": {"dealname": "New deal"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.CreateDeal(context.Background(), CreateDealRequest{
		Properties: map[string]string{"dealname": "New deal"},
	})
	if err != nil {
		t.Fatalf("CreateDeal returned error: %v", err)
	}

	if resp.ID != "created-1" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/crm/v3/objects/deals" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	var payload CreateDealRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if payload.Properties["dealname"] != "New deal" {
		t.Fatalf("unexpected create payload: %#v", payload)
	}
}

func TestUpdateDeal_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "123",
				"properties": {"dealname": "Updated"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.UpdateDeal(context.Background(), "123", UpdateDealRequest{
		Properties: map[string]string{"dealname": "Updated"},
	})
	if err != nil {
		t.Fatalf("UpdateDeal returned error: %v", err)
	}

	if resp.ID != "123" || resp.Properties["dealname"] != "Updated" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPatch || req.URL.Path != "/crm/v3/objects/deals/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestUpdateDeal_RequiresDealID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.UpdateDeal(context.Background(), "", UpdateDealRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestEditDeal_DelegatesToUpdateDeal(t *testing.T) {
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

	_, err = client.EditDeal(context.Background(), "123", UpdateDealRequest{Properties: map[string]string{"dealname": "Updated"}})
	if err != nil {
		t.Fatalf("EditDeal returned error: %v", err)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPatch || req.URL.Path != "/crm/v3/objects/deals/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestDeleteDeal_UsesDeleteEndpoint(t *testing.T) {
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

	if err := client.DeleteDeal(context.Background(), "123"); err != nil {
		t.Fatalf("DeleteDeal returned error: %v", err)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodDelete || req.URL.Path != "/crm/v3/objects/deals/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestDeleteDeal_RequiresDealID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	err := client.DeleteDeal(context.Background(), "")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
