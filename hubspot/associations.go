package hubspot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// AssociationType describes one HubSpot association label/type.
type AssociationType struct {
	AssociationCategory string `json:"associationCategory"`
	AssociationTypeID   int    `json:"associationTypeId"`
	Label               string `json:"label,omitempty"`
}

// Association represents one association target returned by HubSpot.
type Association struct {
	ToObjectID       string            `json:"toObjectId"`
	AssociationTypes []AssociationType `json:"associationTypes,omitempty"`
}

// AssociationsListResponse is the minimal response model returned by the associations list endpoint.
type AssociationsListResponse struct {
	Results []Association `json:"results"`
}

// ListAssociations fetches the associated records for one object pair.
func (c *Client) ListAssociations(ctx context.Context, fromObjectType, fromObjectID, toObjectType string) (AssociationsListResponse, error) {
	path, trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, err := associationListPath(fromObjectType, fromObjectID, toObjectType)
	if err != nil {
		return AssociationsListResponse{}, fmt.Errorf("list associations fromObjectType=%s fromObjectID=%s toObjectType=%s: %w", strings.TrimSpace(fromObjectType), strings.TrimSpace(fromObjectID), strings.TrimSpace(toObjectType), err)
	}

	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return AssociationsListResponse{}, fmt.Errorf("list associations fromObjectType=%s fromObjectID=%s toObjectType=%s: %w", trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, err)
	}
	defer resp.Body.Close()

	var parsed AssociationsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return AssociationsListResponse{}, fmt.Errorf("decode associations list response fromObjectType=%s fromObjectID=%s toObjectType=%s: %w", trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, err)
	}

	return parsed, nil
}

// CreateDefaultAssociation creates a default association between two records.
func (c *Client) CreateDefaultAssociation(ctx context.Context, fromObjectType, fromObjectID, toObjectType, toObjectID string) error {
	path, trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, trimmedToObjectID, err := associationRecordPath(fromObjectType, fromObjectID, toObjectType, toObjectID)
	if err != nil {
		return fmt.Errorf("create default association fromObjectType=%s fromObjectID=%s toObjectType=%s toObjectID=%s: %w", strings.TrimSpace(fromObjectType), strings.TrimSpace(fromObjectID), strings.TrimSpace(toObjectType), strings.TrimSpace(toObjectID), err)
	}

	defaultPath := strings.Replace(path, "/associations/", "/associations/default/", 1)
	resp, err := c.Do(ctx, http.MethodPut, defaultPath, nil, nil)
	if err != nil {
		return fmt.Errorf("create default association fromObjectType=%s fromObjectID=%s toObjectType=%s toObjectID=%s: %w", trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, trimmedToObjectID, err)
	}
	defer resp.Body.Close()

	return nil
}

// CreateAssociation creates a labeled association between two records.
func (c *Client) CreateAssociation(ctx context.Context, fromObjectType, fromObjectID, toObjectType, toObjectID string, associationTypes []AssociationType) error {
	if len(associationTypes) == 0 {
		return errors.New("association types are required")
	}

	path, trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, trimmedToObjectID, err := associationRecordPath(fromObjectType, fromObjectID, toObjectType, toObjectID)
	if err != nil {
		return fmt.Errorf("create association fromObjectType=%s fromObjectID=%s toObjectType=%s toObjectID=%s: %w", strings.TrimSpace(fromObjectType), strings.TrimSpace(fromObjectID), strings.TrimSpace(toObjectType), strings.TrimSpace(toObjectID), err)
	}

	resp, err := c.doJSON(ctx, http.MethodPut, path, associationTypes)
	if err != nil {
		return fmt.Errorf("create association fromObjectType=%s fromObjectID=%s toObjectType=%s toObjectID=%s: %w", trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, trimmedToObjectID, err)
	}
	defer resp.Body.Close()

	return nil
}

// DeleteAssociation removes an association between two records.
func (c *Client) DeleteAssociation(ctx context.Context, fromObjectType, fromObjectID, toObjectType, toObjectID string) error {
	path, trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, trimmedToObjectID, err := associationRecordPath(fromObjectType, fromObjectID, toObjectType, toObjectID)
	if err != nil {
		return fmt.Errorf("delete association fromObjectType=%s fromObjectID=%s toObjectType=%s toObjectID=%s: %w", strings.TrimSpace(fromObjectType), strings.TrimSpace(fromObjectID), strings.TrimSpace(toObjectType), strings.TrimSpace(toObjectID), err)
	}

	resp, err := c.Do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return fmt.Errorf("delete association fromObjectType=%s fromObjectID=%s toObjectType=%s toObjectID=%s: %w", trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, trimmedToObjectID, err)
	}
	defer resp.Body.Close()

	return nil
}

func associationListPath(fromObjectType, fromObjectID, toObjectType string) (string, string, string, string, error) {
	trimmedFromObjectType := strings.TrimSpace(fromObjectType)
	if trimmedFromObjectType == "" {
		return "", "", "", "", errors.New("from object type is required")
	}

	trimmedFromObjectID := strings.TrimSpace(fromObjectID)
	if trimmedFromObjectID == "" {
		return "", "", "", "", errors.New("from object id is required")
	}

	trimmedToObjectType := strings.TrimSpace(toObjectType)
	if trimmedToObjectType == "" {
		return "", "", "", "", errors.New("to object type is required")
	}

	path := "/crm/v4/objects/" + url.PathEscape(trimmedFromObjectType) + "/" + url.PathEscape(trimmedFromObjectID) + "/associations/" + url.PathEscape(trimmedToObjectType)
	return path, trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, nil
}

func associationRecordPath(fromObjectType, fromObjectID, toObjectType, toObjectID string) (string, string, string, string, string, error) {
	path, trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, err := associationListPath(fromObjectType, fromObjectID, toObjectType)
	if err != nil {
		return "", "", "", "", "", err
	}

	trimmedToObjectID := strings.TrimSpace(toObjectID)
	if trimmedToObjectID == "" {
		return "", "", "", "", "", errors.New("to object id is required")
	}

	return path + "/" + url.PathEscape(trimmedToObjectID), trimmedFromObjectType, trimmedFromObjectID, trimmedToObjectType, trimmedToObjectID, nil
}
