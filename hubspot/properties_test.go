package hubspot

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListProperties_BuildsQueryAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`[
				{
					"name": "country",
					"label": "Country",
					"type": "enumeration",
					"fieldType": "select",
					"options": [
						{"label": "United States", "value": "US"},
						{"label": "Canada", "value": "CA"}
					]
				}
			]`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.ListProperties(context.Background(), "contacts", ListPropertiesRequest{DataSensitivity: "sensitive"})
	if err != nil {
		t.Fatalf("ListProperties returned error: %v", err)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodGet {
		t.Fatalf("unexpected method: %s", req.Method)
	}

	if req.URL.Path != "/crm/v3/properties/contacts" {
		t.Fatalf("unexpected path: %s", req.URL.Path)
	}

	if req.URL.Query().Get("dataSensitivity") != "sensitive" {
		t.Fatalf("unexpected dataSensitivity query: %s", req.URL.Query().Get("dataSensitivity"))
	}

	if len(resp) != 1 || resp[0].Name != "country" {
		t.Fatalf("unexpected parsed properties: %#v", resp)
	}

	if len(resp[0].Options) != 2 || resp[0].Options[0].Value != "US" {
		t.Fatalf("unexpected property options: %#v", resp[0].Options)
	}
}

func TestListProperties_RequiresObjectType(t *testing.T) {
	t.Parallel()

	client, err := New(Config{Token: "token-123", Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.ListProperties(context.Background(), " ", ListPropertiesRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "object type is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetProperty_BuildsPathAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"name": "country",
				"label": "Country",
				"type": "enumeration",
				"fieldType": "select",
				"groupName": "contactinformation",
				"options": [
					{"label": "United States", "value": "US", "displayOrder": 0},
					{"label": "Canada", "value": "CA", "displayOrder": 1}
				]
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.GetProperty(context.Background(), "contacts", "country")
	if err != nil {
		t.Fatalf("GetProperty returned error: %v", err)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodGet {
		t.Fatalf("unexpected method: %s", req.Method)
	}

	if req.URL.Path != "/crm/v3/properties/contacts/country" {
		t.Fatalf("unexpected path: %s", req.URL.Path)
	}

	if resp.Name != "country" || resp.FieldType != "select" {
		t.Fatalf("unexpected parsed property: %#v", resp)
	}

	if len(resp.Options) != 2 || resp.Options[1].Label != "Canada" {
		t.Fatalf("unexpected parsed property options: %#v", resp.Options)
	}
}

func TestGetProperty_RequiresPropertyName(t *testing.T) {
	t.Parallel()

	client, err := New(Config{Token: "token-123", Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.GetProperty(context.Background(), "contacts", " ")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "property name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
