package hubspot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Note represents a minimal HubSpot note record.
type Note struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties,omitempty"`
}

// NoteAssociationTarget identifies the record associated with a note.
type NoteAssociationTarget struct {
	ID string `json:"id"`
}

// NoteAssociation defines one record association to apply when creating a note.
type NoteAssociation struct {
	To    NoteAssociationTarget `json:"to"`
	Types []AssociationType     `json:"types,omitempty"`
}

// CreateNoteRequest controls the payload for creating a note.
type CreateNoteRequest struct {
	Body          string
	Timestamp     time.Time
	OwnerID       string
	AttachmentIDs []string
	Associations  []NoteAssociation
}

type noteCreatePayload struct {
	Properties   map[string]string `json:"properties"`
	Associations []NoteAssociation `json:"associations,omitempty"`
}

type noteUpdatePayload struct {
	Properties map[string]string `json:"properties"`
}

// CreateNote creates a HubSpot note and can associate existing file ids as attachments.
func (c *Client) CreateNote(ctx context.Context, req CreateNoteRequest) (Note, error) {
	payload, err := buildCreateNotePayload(req)
	if err != nil {
		return Note{}, err
	}

	resp, err := c.doJSON(ctx, http.MethodPost, "/crm/v3/objects/notes", payload)
	if err != nil {
		return Note{}, fmt.Errorf("create note: %w", err)
	}
	defer resp.Body.Close()

	var parsed Note
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Note{}, fmt.Errorf("decode create note response: %w", err)
	}

	return parsed, nil
}

// SetNoteAttachments replaces the attachment ids on an existing note.
func (c *Client) SetNoteAttachments(ctx context.Context, noteID string, attachmentIDs []string) (Note, error) {
	trimmedNoteID := strings.TrimSpace(noteID)
	if trimmedNoteID == "" {
		return Note{}, errors.New("note id is required")
	}

	resp, err := c.doJSON(ctx, http.MethodPatch, "/crm/v3/objects/notes/"+url.PathEscape(trimmedNoteID), noteUpdatePayload{
		Properties: map[string]string{
			"hs_attachment_ids": joinAttachmentIDs(attachmentIDs),
		},
	})
	if err != nil {
		return Note{}, fmt.Errorf("set note attachments id=%s: %w", trimmedNoteID, err)
	}
	defer resp.Body.Close()

	var parsed Note
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Note{}, fmt.Errorf("decode set note attachments response id=%s: %w", trimmedNoteID, err)
	}

	return parsed, nil
}

// AttachFilesToNote replaces the file ids attached to a note.
func (c *Client) AttachFilesToNote(ctx context.Context, noteID string, fileIDs []string) (Note, error) {
	return c.SetNoteAttachments(ctx, noteID, fileIDs)
}

func buildCreateNotePayload(req CreateNoteRequest) (noteCreatePayload, error) {
	if req.Timestamp.IsZero() {
		return noteCreatePayload{}, errors.New("note timestamp is required")
	}

	if err := validateNoteAssociations(req.Associations); err != nil {
		return noteCreatePayload{}, err
	}

	properties := map[string]string{
		"hs_timestamp": req.Timestamp.UTC().Format(time.RFC3339),
	}

	if trimmedBody := strings.TrimSpace(req.Body); trimmedBody != "" {
		properties["hs_note_body"] = trimmedBody
	}

	if trimmedOwnerID := strings.TrimSpace(req.OwnerID); trimmedOwnerID != "" {
		properties["hubspot_owner_id"] = trimmedOwnerID
	}

	if joinedAttachmentIDs := joinAttachmentIDs(req.AttachmentIDs); joinedAttachmentIDs != "" {
		properties["hs_attachment_ids"] = joinedAttachmentIDs
	}

	return noteCreatePayload{
		Properties:   properties,
		Associations: req.Associations,
	}, nil
}

func validateNoteAssociations(associations []NoteAssociation) error {
	for index, association := range associations {
		if strings.TrimSpace(association.To.ID) == "" {
			return fmt.Errorf("note association %d target id is required", index)
		}

		if len(association.Types) == 0 {
			return fmt.Errorf("note association %d types are required", index)
		}

		for typeIndex, associationType := range association.Types {
			if strings.TrimSpace(associationType.AssociationCategory) == "" {
				return fmt.Errorf("note association %d type %d category is required", index, typeIndex)
			}

			if associationType.AssociationTypeID == 0 {
				return fmt.Errorf("note association %d type %d id is required", index, typeIndex)
			}
		}
	}

	return nil
}

func joinAttachmentIDs(attachmentIDs []string) string {
	filtered := make([]string, 0, len(attachmentIDs))
	for _, attachmentID := range attachmentIDs {
		if trimmedAttachmentID := strings.TrimSpace(attachmentID); trimmedAttachmentID != "" {
			filtered = append(filtered, trimmedAttachmentID)
		}
	}

	return strings.Join(filtered, ";")
}
