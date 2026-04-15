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

// ListDealsRequest controls query options for listing deals.
type ListDealsRequest struct {
	After      string
	Limit      int
	Properties []string
}

// Deal represents a minimal HubSpot deal record.
type Deal struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties,omitempty"`
}

// DealFilter defines one search filter clause.
type DealFilter struct {
	PropertyName string   `json:"propertyName"`
	Operator     string   `json:"operator"`
	Value        string   `json:"value,omitempty"`
	HighValue    string   `json:"highValue,omitempty"`
	Values       []string `json:"values,omitempty"`
}

// DealFilterGroup defines an OR group of AND filters.
type DealFilterGroup struct {
	Filters []DealFilter `json:"filters"`
}

// DealSearchRequest controls query options for the deals search endpoint.
type DealSearchRequest struct {
	Query        string            `json:"query,omitempty"`
	Limit        int               `json:"limit,omitempty"`
	After        string            `json:"after,omitempty"`
	Sorts        []string          `json:"sorts,omitempty"`
	Properties   []string          `json:"properties,omitempty"`
	FilterGroups []DealFilterGroup `json:"filterGroups,omitempty"`
}

// DealsSearchResponse is the minimal response model from deals search endpoint.
type DealsSearchResponse struct {
	Total   int            `json:"total,omitempty"`
	Results []Deal         `json:"results"`
	Paging  *PagingDetails `json:"paging,omitempty"`
}

// GetDealRequest controls query options for reading a single deal.
type GetDealRequest struct {
	Properties   []string
	Associations []string
	Archived     bool
	IDProperty   string
}

// CreateDealRequest is the minimal payload for creating a deal.
type CreateDealRequest struct {
	Properties map[string]string `json:"properties"`
}

// UpdateDealRequest is the minimal payload for updating a deal.
type UpdateDealRequest struct {
	Properties map[string]string `json:"properties"`
}

// DealsListResponse is the minimal response model returned by the deals list endpoint.
type DealsListResponse struct {
	Results []Deal         `json:"results"`
	Paging  *PagingDetails `json:"paging,omitempty"`
}

// ListDeals fetches a page of deals from HubSpot.
func (c *Client) ListDeals(ctx context.Context, req ListDealsRequest) (DealsListResponse, error) {
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

	path := "/crm/v3/objects/deals"
	if encoded := query.Encode(); encoded != "" {
		path = fmt.Sprintf("%s?%s", path, encoded)
	}

	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return DealsListResponse{}, fmt.Errorf("list deals: %w", err)
	}
	defer resp.Body.Close()

	var parsed DealsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return DealsListResponse{}, fmt.Errorf("decode deals list response: %w", err)
	}

	return parsed, nil
}

// SearchDeals executes a deals search request.
func (c *Client) SearchDeals(ctx context.Context, req DealSearchRequest) (DealsSearchResponse, error) {
	resp, err := c.doJSON(ctx, http.MethodPost, "/crm/v3/objects/deals/search", req)
	if err != nil {
		return DealsSearchResponse{}, fmt.Errorf("search deals: %w", err)
	}
	defer resp.Body.Close()

	var parsed DealsSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return DealsSearchResponse{}, fmt.Errorf("decode deals search response: %w", err)
	}

	return parsed, nil
}

// GetDeal reads a single deal by id.
func (c *Client) GetDeal(ctx context.Context, dealID string, req GetDealRequest) (Deal, error) {
	trimmedID := strings.TrimSpace(dealID)
	if trimmedID == "" {
		return Deal{}, errors.New("deal id is required")
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

	path := "/crm/v3/objects/deals/" + url.PathEscape(trimmedID)
	if encoded := query.Encode(); encoded != "" {
		path = fmt.Sprintf("%s?%s", path, encoded)
	}

	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return Deal{}, fmt.Errorf("get deal id=%s: %w", trimmedID, err)
	}
	defer resp.Body.Close()

	var parsed Deal
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Deal{}, fmt.Errorf("decode get deal response id=%s: %w", trimmedID, err)
	}

	return parsed, nil
}

// CreateDeal creates a single deal.
func (c *Client) CreateDeal(ctx context.Context, req CreateDealRequest) (Deal, error) {
	resp, err := c.doJSON(ctx, http.MethodPost, "/crm/v3/objects/deals", req)
	if err != nil {
		return Deal{}, fmt.Errorf("create deal: %w", err)
	}
	defer resp.Body.Close()

	var parsed Deal
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Deal{}, fmt.Errorf("decode create deal response: %w", err)
	}

	return parsed, nil
}

// UpdateDeal updates a single deal by id.
func (c *Client) UpdateDeal(ctx context.Context, dealID string, req UpdateDealRequest) (Deal, error) {
	trimmedID := strings.TrimSpace(dealID)
	if trimmedID == "" {
		return Deal{}, errors.New("deal id is required")
	}

	resp, err := c.doJSON(ctx, http.MethodPatch, "/crm/v3/objects/deals/"+url.PathEscape(trimmedID), req)
	if err != nil {
		return Deal{}, fmt.Errorf("update deal id=%s: %w", trimmedID, err)
	}
	defer resp.Body.Close()

	var parsed Deal
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Deal{}, fmt.Errorf("decode update deal response id=%s: %w", trimmedID, err)
	}

	return parsed, nil
}

// EditDeal updates a single deal by id.
func (c *Client) EditDeal(ctx context.Context, dealID string, req UpdateDealRequest) (Deal, error) {
	return c.UpdateDeal(ctx, dealID, req)
}

// DeleteDeal deletes a single deal by id.
func (c *Client) DeleteDeal(ctx context.Context, dealID string) error {
	trimmedID := strings.TrimSpace(dealID)
	if trimmedID == "" {
		return errors.New("deal id is required")
	}

	resp, err := c.Do(ctx, http.MethodDelete, "/crm/v3/objects/deals/"+url.PathEscape(trimmedID), nil, nil)
	if err != nil {
		return fmt.Errorf("delete deal id=%s: %w", trimmedID, err)
	}
	defer resp.Body.Close()

	return nil
}
