package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultBaseURL   = "https://api.hubapi.com"
	defaultUserAgent = "dashboard-goingestor-utils/hubspot"
)

// HTTPDoer captures the subset of http.Client used by this package.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Config controls how a HubSpot client is constructed.
type Config struct {
	BaseURL    string
	Token      string
	UserAgent  string
	HTTPClient HTTPDoer
	Retry      RetryPolicy
	Throttle   *AdaptiveThrottle
}

// Client provides context-first request execution with auth and retries.
type Client struct {
	baseURL    *url.URL
	token      string
	userAgent  string
	httpClient HTTPDoer
	retry      RetryPolicy
	throttle   *AdaptiveThrottle
}

// New constructs a client with explicit configuration.
func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, errors.New("hubspot token is required")
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse hubspot base url=%q: %w", baseURL, err)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	userAgent := strings.TrimSpace(cfg.UserAgent)
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	return &Client{
		baseURL:    parsedBaseURL,
		token:      cfg.Token,
		userAgent:  userAgent,
		httpClient: httpClient,
		retry:      normalizeRetryPolicy(cfg.Retry),
		throttle:   cfg.Throttle,
	}, nil
}

// Do issues a HubSpot request with retry handling and returns the raw HTTP response.
// Caller owns response body close for successful responses.
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	if c == nil {
		return nil, errors.New("hubspot client is nil")
	}

	if strings.TrimSpace(method) == "" {
		return nil, errors.New("http method is required")
	}

	if strings.TrimSpace(path) == "" {
		return nil, errors.New("request path is required")
	}

	requestURL, err := c.baseURL.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("resolve request path=%q: %w", path, err)
	}

	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("read request body method=%s path=%s: %w", method, path, err)
		}
	}

	attempt := 0
	delay := c.retry.InitialDelay

	for {
		attempt++

		if err := c.throttle.Wait(ctx); err != nil {
			return nil, fmt.Errorf("hubspot throttle wait canceled method=%s path=%s: %w", method, path, err)
		}

		var requestBody io.Reader
		if bodyBytes != nil {
			requestBody = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), requestBody)
		if err != nil {
			return nil, fmt.Errorf("create request method=%s path=%s: %w", method, path, err)
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "application/json")

		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, reqErr := c.httpClient.Do(req)
		if resp != nil {
			c.throttle.Observe(resp.StatusCode, ParseRateLimitHeaders(resp.Header))
		}

		if !c.retry.ShouldRetry(resp, reqErr) || attempt >= c.retry.MaxAttempts {
			if reqErr != nil {
				return nil, fmt.Errorf("hubspot request method=%s path=%s: %w", method, path, reqErr)
			}

			if resp != nil && resp.StatusCode >= 400 {
				return nil, fmt.Errorf("hubspot request failed method=%s path=%s status=%d", method, path, resp.StatusCode)
			}

			return resp, nil
		}

		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}

		wait := retryDelay(resp, delay, c.retry.MaxDelay)
		if err := c.retry.Sleep(ctx, wait); err != nil {
			return nil, fmt.Errorf("hubspot retry wait canceled method=%s path=%s: %w", method, path, err)
		}

		delay = nextRetryDelay(delay, c.retry)
	}
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request method=%s path=%s: %w", method, path, err)
	}

	resp, err := c.Do(ctx, method, path, bytes.NewReader(body), map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("hubspot json request method=%s path=%s: %w", method, path, err)
	}

	return resp, nil
}
