package hubspot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListSubscriptions_BuildsQueryAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"results": [{"id": "sub-123", "properties": {"name": "Starter plan"}}],
				"paging": {"next": {"after": "cursor-2", "link": "https://example.test"}}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.ListSubscriptions(context.Background(), ListSubscriptionsRequest{
		After:      "cursor-1",
		Limit:      50,
		Properties: []string{"name", "status"},
	})
	if err != nil {
		t.Fatalf("ListSubscriptions returned error: %v", err)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodGet || req.URL.Path != "/crm/v3/objects/subscriptions" {
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
	if len(properties) != 2 || properties[0] != "name" || properties[1] != "status" {
		t.Fatalf("unexpected properties query: %#v", properties)
	}

	if len(resp.Results) != 1 || resp.Results[0].ID != "sub-123" {
		t.Fatalf("unexpected parsed results: %#v", resp.Results)
	}

	if resp.Paging == nil || resp.Paging.Next == nil || resp.Paging.Next.After != "cursor-2" {
		t.Fatalf("unexpected parsed paging: %#v", resp.Paging)
	}
}

func TestListSubscriptions_ReturnsDecodeError(t *testing.T) {
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

	_, err = client.ListSubscriptions(context.Background(), ListSubscriptionsRequest{})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}

	if !strings.Contains(err.Error(), "decode subscriptions list response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchSubscriptions_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"total": 1,
				"results": [{"id": "sub-1", "properties": {"name": "Starter plan"}}]
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.SearchSubscriptions(context.Background(), SubscriptionSearchRequest{
		Query:      "Starter",
		Limit:      10,
		Properties: []string{"name"},
		Sorts:      []string{"-createdate"},
		FilterGroups: []SubscriptionFilterGroup{{
			Filters: []SubscriptionFilter{{
				PropertyName: "status",
				Operator:     "EQ",
				Value:        "active",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("SearchSubscriptions returned error: %v", err)
	}

	if len(resp.Results) != 1 || resp.Results[0].ID != "sub-1" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPost {
		t.Fatalf("unexpected method: %s", req.Method)
	}

	if req.URL.Path != "/crm/v3/objects/subscriptions/search" {
		t.Fatalf("unexpected path: %s", req.URL.Path)
	}

	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected json content-type, got %q", got)
	}

	var payload SubscriptionSearchRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if payload.Query != "Starter" || payload.Limit != 10 {
		t.Fatalf("unexpected search payload: %#v", payload)
	}
}

func TestGetSubscription_BuildsPathQueryAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "123",
				"properties": {"name": "Starter plan"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.GetSubscription(context.Background(), "123", GetSubscriptionRequest{
		Properties:   []string{"name", "status"},
		Associations: []string{"contacts"},
		Archived:     true,
		IDProperty:   "external_id",
	})
	if err != nil {
		t.Fatalf("GetSubscription returned error: %v", err)
	}

	if resp.ID != "123" || resp.Properties["name"] != "Starter plan" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodGet {
		t.Fatalf("unexpected method: %s", req.Method)
	}

	if req.URL.Path != "/crm/v3/objects/subscriptions/123" {
		t.Fatalf("unexpected path: %s", req.URL.Path)
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

func TestGetSubscription_RequiresSubscriptionID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.GetSubscription(context.Background(), "  ", GetSubscriptionRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestCreateSubscription_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "created-1",
				"properties": {"name": "New plan"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.CreateSubscription(context.Background(), CreateSubscriptionRequest{
		Properties: map[string]string{"name": "New plan"},
	})
	if err != nil {
		t.Fatalf("CreateSubscription returned error: %v", err)
	}

	if resp.ID != "created-1" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/crm/v3/objects/subscriptions" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	var payload CreateSubscriptionRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if payload.Properties["name"] != "New plan" {
		t.Fatalf("unexpected create payload: %#v", payload)
	}
}

func TestUpdateSubscription_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "123",
				"properties": {"name": "Updated plan"}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.UpdateSubscription(context.Background(), "123", UpdateSubscriptionRequest{
		Properties: map[string]string{"name": "Updated plan"},
	})
	if err != nil {
		t.Fatalf("UpdateSubscription returned error: %v", err)
	}

	if resp.ID != "123" || resp.Properties["name"] != "Updated plan" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPatch || req.URL.Path != "/crm/v3/objects/subscriptions/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestUpdateSubscription_RequiresSubscriptionID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.UpdateSubscription(context.Background(), "", UpdateSubscriptionRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestEditSubscription_DelegatesToUpdateSubscription(t *testing.T) {
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

	_, err = client.EditSubscription(context.Background(), "123", UpdateSubscriptionRequest{Properties: map[string]string{"name": "Updated plan"}})
	if err != nil {
		t.Fatalf("EditSubscription returned error: %v", err)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPatch || req.URL.Path != "/crm/v3/objects/subscriptions/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestDeleteSubscription_UsesDeleteEndpoint(t *testing.T) {
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

	if err := client.DeleteSubscription(context.Background(), "123"); err != nil {
		t.Fatalf("DeleteSubscription returned error: %v", err)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodDelete || req.URL.Path != "/crm/v3/objects/subscriptions/123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestDeleteSubscription_RequiresSubscriptionID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	err := client.DeleteSubscription(context.Background(), "")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
