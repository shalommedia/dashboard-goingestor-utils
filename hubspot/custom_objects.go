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

// ListCustomObjectsRequest controls query options for listing custom objects.
type ListCustomObjectsRequest struct {
	After      string
	Limit      int
	Properties []string
}

// CustomObject represents a minimal HubSpot custom object record.
type CustomObject struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties,omitempty"`
}

// CustomObjectFilter defines one search filter clause.
type CustomObjectFilter struct {
	PropertyName string   `json:"propertyName"`
	Operator     string   `json:"operator"`
	Value        string   `json:"value,omitempty"`
	HighValue    string   `json:"highValue,omitempty"`
	Values       []string `json:"values,omitempty"`
}

// CustomObjectFilterGroup defines an OR group of AND filters.
type CustomObjectFilterGroup struct {
	Filters []CustomObjectFilter `json:"filters"`
}

// CustomObjectSearchRequest controls query options for a custom object search endpoint.
type CustomObjectSearchRequest struct {
	Query        string                    `json:"query,omitempty"`
	Limit        int                       `json:"limit,omitempty"`
	After        string                    `json:"after,omitempty"`
	Sorts        []string                  `json:"sorts,omitempty"`
	Properties   []string                  `json:"properties,omitempty"`
	FilterGroups []CustomObjectFilterGroup `json:"filterGroups,omitempty"`
}

// CustomObjectsSearchResponse is the minimal response model from a custom object search endpoint.
type CustomObjectsSearchResponse struct {
	Total   int            `json:"total,omitempty"`
	Results []CustomObject `json:"results"`
	Paging  *PagingDetails `json:"paging,omitempty"`
}

// GetCustomObjectRequest controls query options for reading a single custom object.
type GetCustomObjectRequest struct {
	Properties   []string
	Associations []string
	Archived     bool
	IDProperty   string
}

// CreateCustomObjectRequest is the minimal payload for creating a custom object.
type CreateCustomObjectRequest struct {
	Properties map[string]string `json:"properties"`
}

// UpdateCustomObjectRequest is the minimal payload for updating a custom object.
type UpdateCustomObjectRequest struct {
	Properties map[string]string `json:"properties"`
}

// CustomObjectsListResponse is the minimal response model returned by a custom object list endpoint.
type CustomObjectsListResponse struct {
	Results []CustomObject `json:"results"`
	Paging  *PagingDetails `json:"paging,omitempty"`
}

// ListCustomObjects fetches a page of custom objects by object type id.
func (c *Client) ListCustomObjects(ctx context.Context, objectTypeID string, req ListCustomObjectsRequest) (CustomObjectsListResponse, error) {
	trimmedObjectTypeID := strings.TrimSpace(objectTypeID)

	basePath, err := customObjectBasePath(objectTypeID)
	if err != nil {
		return CustomObjectsListResponse{}, fmt.Errorf("list custom objects objectTypeID=%s: %w", trimmedObjectTypeID, err)
	}

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

	path := basePath
	if encoded := query.Encode(); encoded != "" {
		path = fmt.Sprintf("%s?%s", path, encoded)
	}

	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return CustomObjectsListResponse{}, fmt.Errorf("list custom objects objectTypeID=%s: %w", trimmedObjectTypeID, err)
	}
	defer resp.Body.Close()

	var parsed CustomObjectsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return CustomObjectsListResponse{}, fmt.Errorf("decode custom objects list response objectTypeID=%s: %w", trimmedObjectTypeID, err)
	}

	return parsed, nil
}

// SearchCustomObjects executes a custom object search request by object type id.
func (c *Client) SearchCustomObjects(ctx context.Context, objectTypeID string, req CustomObjectSearchRequest) (CustomObjectsSearchResponse, error) {
	trimmedObjectTypeID := strings.TrimSpace(objectTypeID)

	basePath, err := customObjectBasePath(objectTypeID)
	if err != nil {
		return CustomObjectsSearchResponse{}, fmt.Errorf("search custom objects objectTypeID=%s: %w", trimmedObjectTypeID, err)
	}

	resp, err := c.doJSON(ctx, http.MethodPost, basePath+"/search", req)
	if err != nil {
		return CustomObjectsSearchResponse{}, fmt.Errorf("search custom objects objectTypeID=%s: %w", trimmedObjectTypeID, err)
	}
	defer resp.Body.Close()

	var parsed CustomObjectsSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return CustomObjectsSearchResponse{}, fmt.Errorf("decode custom objects search response objectTypeID=%s: %w", trimmedObjectTypeID, err)
	}

	return parsed, nil
}

// GetCustomObject reads a single custom object by object type id and object id.
func (c *Client) GetCustomObject(ctx context.Context, objectTypeID, objectID string, req GetCustomObjectRequest) (CustomObject, error) {
	trimmedObjectTypeID := strings.TrimSpace(objectTypeID)

	path, trimmedObjectID, err := customObjectRecordPath(objectTypeID, objectID)
	if err != nil {
		return CustomObject{}, fmt.Errorf("get custom object objectTypeID=%s id=%s: %w", trimmedObjectTypeID, strings.TrimSpace(objectID), err)
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

	if encoded := query.Encode(); encoded != "" {
		path = fmt.Sprintf("%s?%s", path, encoded)
	}

	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return CustomObject{}, fmt.Errorf("get custom object objectTypeID=%s id=%s: %w", trimmedObjectTypeID, trimmedObjectID, err)
	}
	defer resp.Body.Close()

	var parsed CustomObject
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return CustomObject{}, fmt.Errorf("decode get custom object response objectTypeID=%s id=%s: %w", trimmedObjectTypeID, trimmedObjectID, err)
	}

	return parsed, nil
}

// CreateCustomObject creates a single custom object by object type id.
func (c *Client) CreateCustomObject(ctx context.Context, objectTypeID string, req CreateCustomObjectRequest) (CustomObject, error) {
	trimmedObjectTypeID := strings.TrimSpace(objectTypeID)

	basePath, err := customObjectBasePath(objectTypeID)
	if err != nil {
		return CustomObject{}, fmt.Errorf("create custom object objectTypeID=%s: %w", trimmedObjectTypeID, err)
	}

	resp, err := c.doJSON(ctx, http.MethodPost, basePath, req)
	if err != nil {
		return CustomObject{}, fmt.Errorf("create custom object objectTypeID=%s: %w", trimmedObjectTypeID, err)
	}
	defer resp.Body.Close()

	var parsed CustomObject
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return CustomObject{}, fmt.Errorf("decode create custom object response objectTypeID=%s: %w", trimmedObjectTypeID, err)
	}

	return parsed, nil
}

// UpdateCustomObject updates a single custom object by object type id and object id.
func (c *Client) UpdateCustomObject(ctx context.Context, objectTypeID, objectID string, req UpdateCustomObjectRequest) (CustomObject, error) {
	trimmedObjectTypeID := strings.TrimSpace(objectTypeID)

	path, trimmedObjectID, err := customObjectRecordPath(objectTypeID, objectID)
	if err != nil {
		return CustomObject{}, fmt.Errorf("update custom object objectTypeID=%s id=%s: %w", trimmedObjectTypeID, strings.TrimSpace(objectID), err)
	}

	resp, err := c.doJSON(ctx, http.MethodPatch, path, req)
	if err != nil {
		return CustomObject{}, fmt.Errorf("update custom object objectTypeID=%s id=%s: %w", trimmedObjectTypeID, trimmedObjectID, err)
	}
	defer resp.Body.Close()

	var parsed CustomObject
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return CustomObject{}, fmt.Errorf("decode update custom object response objectTypeID=%s id=%s: %w", trimmedObjectTypeID, trimmedObjectID, err)
	}

	return parsed, nil
}

// EditCustomObject updates a single custom object by object type id and object id.
func (c *Client) EditCustomObject(ctx context.Context, objectTypeID, objectID string, req UpdateCustomObjectRequest) (CustomObject, error) {
	return c.UpdateCustomObject(ctx, objectTypeID, objectID, req)
}

// DeleteCustomObject deletes a single custom object by object type id and object id.
func (c *Client) DeleteCustomObject(ctx context.Context, objectTypeID, objectID string) error {
	trimmedObjectTypeID := strings.TrimSpace(objectTypeID)

	path, trimmedObjectID, err := customObjectRecordPath(objectTypeID, objectID)
	if err != nil {
		return fmt.Errorf("delete custom object objectTypeID=%s id=%s: %w", trimmedObjectTypeID, strings.TrimSpace(objectID), err)
	}

	resp, err := c.Do(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return fmt.Errorf("delete custom object objectTypeID=%s id=%s: %w", trimmedObjectTypeID, trimmedObjectID, err)
	}
	defer resp.Body.Close()

	return nil
}

func customObjectBasePath(objectTypeID string) (string, error) {
	trimmedObjectTypeID := strings.TrimSpace(objectTypeID)
	if trimmedObjectTypeID == "" {
		return "", errors.New("custom object type id is required")
	}

	return "/crm/v3/objects/" + url.PathEscape(trimmedObjectTypeID), nil
}

func customObjectRecordPath(objectTypeID, objectID string) (string, string, error) {
	basePath, err := customObjectBasePath(objectTypeID)
	if err != nil {
		return "", "", err
	}

	trimmedObjectID := strings.TrimSpace(objectID)
	if trimmedObjectID == "" {
		return "", "", errors.New("custom object id is required")
	}

	return basePath + "/" + url.PathEscape(trimmedObjectID), trimmedObjectID, nil
}
