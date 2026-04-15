package hubspot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListAssociations_BuildsPathAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"results": [{
					"toObjectId": "456",
					"associationTypes": [{"associationCategory": "HUBSPOT_DEFINED", "associationTypeId": 1, "label": "Primary"}]
				}]
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.ListAssociations(context.Background(), "deals", "123", "contacts")
	if err != nil {
		t.Fatalf("ListAssociations returned error: %v", err)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodGet || req.URL.Path != "/crm/v4/objects/deals/123/associations/contacts" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	if len(resp.Results) != 1 || resp.Results[0].ToObjectID != "456" {
		t.Fatalf("unexpected parsed response: %#v", resp)
	}

	if len(resp.Results[0].AssociationTypes) != 1 || resp.Results[0].AssociationTypes[0].AssociationTypeID != 1 {
		t.Fatalf("unexpected parsed association types: %#v", resp.Results[0].AssociationTypes)
	}
}

func TestListAssociations_RequiresFromObjectType(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := client.ListAssociations(context.Background(), "", "123", "contacts")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestListAssociations_ReturnsDecodeError(t *testing.T) {
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

	_, err = client.ListAssociations(context.Background(), "deals", "123", "contacts")
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}

	if !strings.Contains(err.Error(), "decode associations list response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateDefaultAssociation_UsesDefaultEndpoint(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := client.CreateDefaultAssociation(context.Background(), "deals", "123", "contacts", "456"); err != nil {
		t.Fatalf("CreateDefaultAssociation returned error: %v", err)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPut || req.URL.Path != "/crm/v4/objects/deals/123/associations/default/contacts/456" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestCreateAssociation_BuildsRequestBody(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	associationTypes := []AssociationType{{
		AssociationCategory: "USER_DEFINED",
		AssociationTypeID:   37,
	}}

	if err := client.CreateAssociation(context.Background(), "deals", "123", "contacts", "456", associationTypes); err != nil {
		t.Fatalf("CreateAssociation returned error: %v", err)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPut || req.URL.Path != "/crm/v4/objects/deals/123/associations/contacts/456" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected json content-type, got %q", got)
	}

	var payload []AssociationType
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if len(payload) != 1 || payload[0].AssociationTypeID != 37 || payload[0].AssociationCategory != "USER_DEFINED" {
		t.Fatalf("unexpected create association payload: %#v", payload)
	}
}

func TestCreateAssociation_RequiresAssociationTypes(t *testing.T) {
	t.Parallel()

	client := &Client{}
	err := client.CreateAssociation(context.Background(), "deals", "123", "contacts", "456", nil)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestDeleteAssociation_UsesDeleteEndpoint(t *testing.T) {
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

	if err := client.DeleteAssociation(context.Background(), "deals", "123", "contacts", "456"); err != nil {
		t.Fatalf("DeleteAssociation returned error: %v", err)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodDelete || req.URL.Path != "/crm/v4/objects/deals/123/associations/contacts/456" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}
}

func TestDeleteAssociation_RequiresToObjectID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	err := client.DeleteAssociation(context.Background(), "deals", "123", "contacts", "")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
