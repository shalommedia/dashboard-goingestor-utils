package hubspot

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type sequenceHTTPClient struct {
	responses []*http.Response
	errors    []error
	requests  []*http.Request
	index     int
}

func (c *sequenceHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.requests = append(c.requests, req)

	if c.index >= len(c.responses) && c.index >= len(c.errors) {
		return nil, errors.New("unexpected extra request")
	}

	var resp *http.Response
	if c.index < len(c.responses) {
		resp = c.responses[c.index]
	}

	var err error
	if c.index < len(c.errors) {
		err = c.errors[c.index]
	}

	c.index++
	return resp, err
}

func TestNew_RequiresToken(t *testing.T) {
	t.Parallel()

	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClientDo_AddsHeadersAndUsesRelativePath(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}},
	}

	client, err := New(Config{
		Token:      "token-123",
		BaseURL:    "https://api.hubapi.com",
		UserAgent:  "test-agent",
		HTTPClient: clientImpl,
		Retry: RetryPolicy{
			MaxAttempts: 1,
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.Do(context.Background(), http.MethodGet, "/crm/v3/objects/contacts", nil, map[string]string{
		"X-Trace-Id": "abc",
	})
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	defer resp.Body.Close()

	if len(clientImpl.requests) != 1 {
		t.Fatalf("expected one request, got %d", len(clientImpl.requests))
	}

	req := clientImpl.requests[0]
	if req.URL.String() != "https://api.hubapi.com/crm/v3/objects/contacts" {
		t.Fatalf("unexpected request url: %s", req.URL.String())
	}

	if got := req.Header.Get("Authorization"); got != "Bearer token-123" {
		t.Fatalf("unexpected authorization header: %s", got)
	}

	if got := req.Header.Get("User-Agent"); got != "test-agent" {
		t.Fatalf("unexpected user-agent header: %s", got)
	}

	if got := req.Header.Get("X-Trace-Id"); got != "abc" {
		t.Fatalf("unexpected trace header: %s", got)
	}
}

func TestClientDo_RetriesOnHTTP500(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{
			{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("err")),
				Header:     make(http.Header),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			},
		},
	}

	sleepCalls := 0
	client, err := New(Config{
		Token:      "token-123",
		HTTPClient: clientImpl,
		Retry: RetryPolicy{
			MaxAttempts:       2,
			InitialDelay:      1,
			MaxDelay:          1,
			BackoffMultiplier: 2,
			Sleep: func(context.Context, time.Duration) error {
				sleepCalls++
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.Do(context.Background(), http.MethodGet, "/crm/v3/objects/deals", nil, nil)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if len(clientImpl.requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(clientImpl.requests))
	}

	if sleepCalls != 1 {
		t.Fatalf("expected one retry sleep, got %d", sleepCalls)
	}
}

func TestClientDo_ReturnsErrorOnLastFailure(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader("bad gateway")),
			Header:     make(http.Header),
		}},
	}

	client, err := New(Config{
		Token:      "token-123",
		HTTPClient: clientImpl,
		Retry: RetryPolicy{
			MaxAttempts: 1,
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.Do(context.Background(), http.MethodGet, "/crm/v3/objects/companies", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "status=502") {
		t.Fatalf("expected status code in error, got: %v", err)
	}
}

func TestClientDo_HonorsRetryAfterHeader(t *testing.T) {
	t.Parallel()

	firstHeaders := make(http.Header)
	firstHeaders.Set("Retry-After", "2")

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{
			{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader("rate limited")),
				Header:     firstHeaders,
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			},
		},
	}

	var observedSleep []time.Duration
	client, err := New(Config{
		Token:      "token-123",
		HTTPClient: clientImpl,
		Retry: RetryPolicy{
			MaxAttempts:       2,
			InitialDelay:      50 * time.Millisecond,
			MaxDelay:          5 * time.Second,
			BackoffMultiplier: 2,
			Sleep: func(_ context.Context, d time.Duration) error {
				observedSleep = append(observedSleep, d)
				return nil
			},
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	resp, err := client.Do(context.Background(), http.MethodGet, "/crm/v3/objects/contacts", nil, nil)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	defer resp.Body.Close()

	if len(observedSleep) != 1 || observedSleep[0] != 2*time.Second {
		t.Fatalf("expected Retry-After sleep of 2s, got %#v", observedSleep)
	}
}

func TestClientDo_ReturnsCancellationWhenRetryWaitCanceled(t *testing.T) {
	t.Parallel()

	clientImpl := &sequenceHTTPClient{
		responses: []*http.Response{{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("err")),
			Header:     make(http.Header),
		}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client, err := New(Config{
		Token:      "token-123",
		HTTPClient: clientImpl,
		Retry: RetryPolicy{
			MaxAttempts:       2,
			InitialDelay:      time.Second,
			MaxDelay:          time.Second,
			BackoffMultiplier: 2,
			Sleep:             sleep,
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, err = client.Do(ctx, http.MethodGet, "/crm/v3/objects/contacts", nil, nil)
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "retry wait canceled") {
		t.Fatalf("unexpected error: %v", err)
	}
}
