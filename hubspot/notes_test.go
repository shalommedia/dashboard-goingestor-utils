package hubspot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCreateNote_BuildsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "note-123",
				"properties": {
					"hs_note_body": "Proposal attached",
					"hs_attachment_ids": "24332474034"
				}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.CreateNote(context.Background(), CreateNoteRequest{
		Body:          "Proposal attached",
		Timestamp:     time.Date(2026, time.April, 15, 10, 30, 0, 0, time.UTC),
		OwnerID:       "14240720",
		AttachmentIDs: []string{"24332474034"},
		Associations: []NoteAssociation{{
			To: NoteAssociationTarget{ID: "581751"},
			Types: []AssociationType{{
				AssociationCategory: "HUBSPOT_DEFINED",
				AssociationTypeID:   202,
			}},
		}},
	})
	if err != nil {
		t.Fatalf("CreateNote returned error: %v", err)
	}

	if resp.ID != "note-123" || resp.Properties["hs_attachment_ids"] != "24332474034" {
		t.Fatalf("unexpected parsed note: %#v", resp)
	}

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPost || req.URL.Path != "/crm/v3/objects/notes" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected json content-type, got %q", got)
	}

	var payload noteCreatePayload
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if payload.Properties["hs_note_body"] != "Proposal attached" {
		t.Fatalf("unexpected note body: %#v", payload.Properties)
	}

	if payload.Properties["hs_timestamp"] != "2026-04-15T10:30:00Z" {
		t.Fatalf("unexpected note timestamp: %s", payload.Properties["hs_timestamp"])
	}

	if payload.Properties["hubspot_owner_id"] != "14240720" {
		t.Fatalf("unexpected owner id: %s", payload.Properties["hubspot_owner_id"])
	}

	if payload.Properties["hs_attachment_ids"] != "24332474034" {
		t.Fatalf("unexpected attachment ids: %s", payload.Properties["hs_attachment_ids"])
	}

	if len(payload.Associations) != 1 || payload.Associations[0].To.ID != "581751" {
		t.Fatalf("unexpected associations payload: %#v", payload.Associations)
	}

	if len(payload.Associations[0].Types) != 1 || payload.Associations[0].Types[0].AssociationTypeID != 202 {
		t.Fatalf("unexpected association types payload: %#v", payload.Associations[0].Types)
	}
}

func TestCreateNote_RequiresTimestamp(t *testing.T) {
	t.Parallel()

	client, err := New(Config{Token: "token-123", Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.CreateNote(context.Background(), CreateNoteRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "note timestamp is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetNoteAttachments_BuildsPatchRequest(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": "note-123",
				"properties": {
					"hs_attachment_ids": "24332474034;24332474044"
				}
			}`)),
			Header: make(http.Header),
		}},
	}

	client, err := New(Config{Token: "token-123", HTTPClient: clientImpl, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.SetNoteAttachments(context.Background(), "note-123", []string{"24332474034", "24332474044"})
	if err != nil {
		t.Fatalf("SetNoteAttachments returned error: %v", err)
	}

	if resp.Properties["hs_attachment_ids"] != "24332474034;24332474044" {
		t.Fatalf("unexpected parsed note attachments: %#v", resp)
	}

	req := clientImpl.requests[0]
	if req.Method != http.MethodPatch || req.URL.Path != "/crm/v3/objects/notes/note-123" {
		t.Fatalf("unexpected request: method=%s path=%s", req.Method, req.URL.Path)
	}

	var payload noteUpdatePayload
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}

	if payload.Properties["hs_attachment_ids"] != "24332474034;24332474044" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestSetNoteAttachments_RequiresNoteID(t *testing.T) {
	t.Parallel()

	client, err := New(Config{Token: "token-123", Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.SetNoteAttachments(context.Background(), " ", []string{"24332474034"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "note id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
