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

// ListPropertiesRequest controls query options for listing an object's properties.
type ListPropertiesRequest struct {
	DataSensitivity string
}

// PropertyOption describes one selectable value for enumeration-backed properties.
type PropertyOption struct {
	Label        string `json:"label,omitempty"`
	Value        string `json:"value,omitempty"`
	Description  string `json:"description,omitempty"`
	DisplayOrder int    `json:"displayOrder,omitempty"`
	Hidden       bool   `json:"hidden,omitempty"`
	ReadOnly     bool   `json:"readOnly,omitempty"`
}

// Property describes a HubSpot CRM property definition.
type Property struct {
	Name           string           `json:"name,omitempty"`
	Label          string           `json:"label,omitempty"`
	Type           string           `json:"type,omitempty"`
	FieldType      string           `json:"fieldType,omitempty"`
	Description    string           `json:"description,omitempty"`
	GroupName      string           `json:"groupName,omitempty"`
	HasUniqueValue bool             `json:"hasUniqueValue,omitempty"`
	Hidden         bool             `json:"hidden,omitempty"`
	FormField      bool             `json:"formField,omitempty"`
	Archived       bool             `json:"archived,omitempty"`
	Options        []PropertyOption `json:"options,omitempty"`
}

// ListProperties fetches all property definitions for a HubSpot object type.
func (c *Client) ListProperties(ctx context.Context, objectType string, req ListPropertiesRequest) ([]Property, error) {
	trimmedObjectType := strings.TrimSpace(objectType)
	if trimmedObjectType == "" {
		return nil, errors.New("object type is required")
	}

	path := "/crm/v3/properties/" + url.PathEscape(trimmedObjectType)
	query := url.Values{}
	if strings.TrimSpace(req.DataSensitivity) != "" {
		query.Set("dataSensitivity", req.DataSensitivity)
	}

	if encoded := query.Encode(); encoded != "" {
		path = fmt.Sprintf("%s?%s", path, encoded)
	}

	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list properties objectType=%s: %w", trimmedObjectType, err)
	}
	defer resp.Body.Close()

	var parsed []Property
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode list properties response objectType=%s: %w", trimmedObjectType, err)
	}

	return parsed, nil
}

// GetProperty fetches one property definition for a HubSpot object type.
func (c *Client) GetProperty(ctx context.Context, objectType, propertyName string) (Property, error) {
	trimmedObjectType := strings.TrimSpace(objectType)
	if trimmedObjectType == "" {
		return Property{}, errors.New("object type is required")
	}

	trimmedPropertyName := strings.TrimSpace(propertyName)
	if trimmedPropertyName == "" {
		return Property{}, errors.New("property name is required")
	}

	path := fmt.Sprintf("/crm/v3/properties/%s/%s", url.PathEscape(trimmedObjectType), url.PathEscape(trimmedPropertyName))

	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return Property{}, fmt.Errorf("get property objectType=%s property=%s: %w", trimmedObjectType, trimmedPropertyName, err)
	}
	defer resp.Body.Close()

	var parsed Property
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Property{}, fmt.Errorf("decode get property response objectType=%s property=%s: %w", trimmedObjectType, trimmedPropertyName, err)
	}

	return parsed, nil
}
