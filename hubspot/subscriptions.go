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

// ListSubscriptionsRequest controls query options for listing subscriptions.
type ListSubscriptionsRequest struct {
	After      string
	Limit      int
	Properties []string
}

// Subscription represents a minimal HubSpot subscription record.
type Subscription struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties,omitempty"`
}

// SubscriptionFilter defines one search filter clause.
type SubscriptionFilter struct {
	PropertyName string   `json:"propertyName"`
	Operator     string   `json:"operator"`
	Value        string   `json:"value,omitempty"`
	HighValue    string   `json:"highValue,omitempty"`
	Values       []string `json:"values,omitempty"`
}

// SubscriptionFilterGroup defines an OR group of AND filters.
type SubscriptionFilterGroup struct {
	Filters []SubscriptionFilter `json:"filters"`
}

// SubscriptionSearchRequest controls query options for the subscriptions search endpoint.
type SubscriptionSearchRequest struct {
	Query        string                    `json:"query,omitempty"`
	Limit        int                       `json:"limit,omitempty"`
	After        string                    `json:"after,omitempty"`
	Sorts        []string                  `json:"sorts,omitempty"`
	Properties   []string                  `json:"properties,omitempty"`
	FilterGroups []SubscriptionFilterGroup `json:"filterGroups,omitempty"`
}

// SubscriptionsSearchResponse is the minimal response model from subscriptions search endpoint.
type SubscriptionsSearchResponse struct {
	Total   int            `json:"total,omitempty"`
	Results []Subscription `json:"results"`
	Paging  *PagingDetails `json:"paging,omitempty"`
}

// GetSubscriptionRequest controls query options for reading a single subscription.
type GetSubscriptionRequest struct {
	Properties   []string
	Associations []string
	Archived     bool
	IDProperty   string
}

// CreateSubscriptionRequest is the minimal payload for creating a subscription.
type CreateSubscriptionRequest struct {
	Properties map[string]string `json:"properties"`
}

// UpdateSubscriptionRequest is the minimal payload for updating a subscription.
type UpdateSubscriptionRequest struct {
	Properties map[string]string `json:"properties"`
}

// SubscriptionsListResponse is the minimal response model returned by the subscriptions list endpoint.
type SubscriptionsListResponse struct {
	Results []Subscription `json:"results"`
	Paging  *PagingDetails `json:"paging,omitempty"`
}

// ListSubscriptions fetches a page of subscriptions from HubSpot.
func (c *Client) ListSubscriptions(ctx context.Context, req ListSubscriptionsRequest) (SubscriptionsListResponse, error) {
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

	path := "/crm/v3/objects/subscriptions"
	if encoded := query.Encode(); encoded != "" {
		path = fmt.Sprintf("%s?%s", path, encoded)
	}

	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return SubscriptionsListResponse{}, fmt.Errorf("list subscriptions: %w", err)
	}
	defer resp.Body.Close()

	var parsed SubscriptionsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return SubscriptionsListResponse{}, fmt.Errorf("decode subscriptions list response: %w", err)
	}

	return parsed, nil
}

// SearchSubscriptions executes a subscriptions search request.
func (c *Client) SearchSubscriptions(ctx context.Context, req SubscriptionSearchRequest) (SubscriptionsSearchResponse, error) {
	resp, err := c.doJSON(ctx, http.MethodPost, "/crm/v3/objects/subscriptions/search", req)
	if err != nil {
		return SubscriptionsSearchResponse{}, fmt.Errorf("search subscriptions: %w", err)
	}
	defer resp.Body.Close()

	var parsed SubscriptionsSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return SubscriptionsSearchResponse{}, fmt.Errorf("decode subscriptions search response: %w", err)
	}

	return parsed, nil
}

// GetSubscription reads a single subscription by id.
func (c *Client) GetSubscription(ctx context.Context, subscriptionID string, req GetSubscriptionRequest) (Subscription, error) {
	trimmedID := strings.TrimSpace(subscriptionID)
	if trimmedID == "" {
		return Subscription{}, errors.New("subscription id is required")
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

	path := "/crm/v3/objects/subscriptions/" + url.PathEscape(trimmedID)
	if encoded := query.Encode(); encoded != "" {
		path = fmt.Sprintf("%s?%s", path, encoded)
	}

	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return Subscription{}, fmt.Errorf("get subscription id=%s: %w", trimmedID, err)
	}
	defer resp.Body.Close()

	var parsed Subscription
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Subscription{}, fmt.Errorf("decode get subscription response id=%s: %w", trimmedID, err)
	}

	return parsed, nil
}

// CreateSubscription creates a single subscription.
func (c *Client) CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (Subscription, error) {
	resp, err := c.doJSON(ctx, http.MethodPost, "/crm/v3/objects/subscriptions", req)
	if err != nil {
		return Subscription{}, fmt.Errorf("create subscription: %w", err)
	}
	defer resp.Body.Close()

	var parsed Subscription
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Subscription{}, fmt.Errorf("decode create subscription response: %w", err)
	}

	return parsed, nil
}

// UpdateSubscription updates a single subscription by id.
func (c *Client) UpdateSubscription(ctx context.Context, subscriptionID string, req UpdateSubscriptionRequest) (Subscription, error) {
	trimmedID := strings.TrimSpace(subscriptionID)
	if trimmedID == "" {
		return Subscription{}, errors.New("subscription id is required")
	}

	resp, err := c.doJSON(ctx, http.MethodPatch, "/crm/v3/objects/subscriptions/"+url.PathEscape(trimmedID), req)
	if err != nil {
		return Subscription{}, fmt.Errorf("update subscription id=%s: %w", trimmedID, err)
	}
	defer resp.Body.Close()

	var parsed Subscription
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Subscription{}, fmt.Errorf("decode update subscription response id=%s: %w", trimmedID, err)
	}

	return parsed, nil
}

// EditSubscription updates a single subscription by id.
func (c *Client) EditSubscription(ctx context.Context, subscriptionID string, req UpdateSubscriptionRequest) (Subscription, error) {
	return c.UpdateSubscription(ctx, subscriptionID, req)
}

// DeleteSubscription deletes a single subscription by id.
func (c *Client) DeleteSubscription(ctx context.Context, subscriptionID string) error {
	trimmedID := strings.TrimSpace(subscriptionID)
	if trimmedID == "" {
		return errors.New("subscription id is required")
	}

	resp, err := c.Do(ctx, http.MethodDelete, "/crm/v3/objects/subscriptions/"+url.PathEscape(trimmedID), nil, nil)
	if err != nil {
		return fmt.Errorf("delete subscription id=%s: %w", trimmedID, err)
	}
	defer resp.Body.Close()

	return nil
}