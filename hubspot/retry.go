package hubspot

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RetryPolicy configures request retry behavior.
type RetryPolicy struct {
	MaxAttempts       int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
	ShouldRetry       func(resp *http.Response, err error) bool
	Sleep             func(ctx context.Context, delay time.Duration) error
}

func normalizeRetryPolicy(policy RetryPolicy) RetryPolicy {
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 3
	}

	if policy.InitialDelay <= 0 {
		policy.InitialDelay = 250 * time.Millisecond
	}

	if policy.MaxDelay <= 0 {
		policy.MaxDelay = 5 * time.Second
	}

	if policy.BackoffMultiplier <= 1 {
		policy.BackoffMultiplier = 2
	}

	if policy.ShouldRetry == nil {
		policy.ShouldRetry = defaultShouldRetry
	}

	if policy.Sleep == nil {
		policy.Sleep = sleep
	}

	return policy
}

func defaultShouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
	}

	if resp == nil {
		return false
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}

	return resp.StatusCode >= http.StatusInternalServerError
}

func retryDelay(resp *http.Response, fallback, maxDelay time.Duration) time.Duration {
	if resp != nil {
		retryAfter := strings.TrimSpace(resp.Header.Get("Retry-After"))
		if retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				delay := time.Duration(seconds) * time.Second
				if delay > maxDelay {
					return maxDelay
				}

				return delay
			}
		}
	}

	if fallback > maxDelay {
		return maxDelay
	}

	return fallback
}

func nextRetryDelay(current time.Duration, policy RetryPolicy) time.Duration {
	next := time.Duration(float64(current) * policy.BackoffMultiplier)
	if next > policy.MaxDelay {
		return policy.MaxDelay
	}

	if next <= 0 || math.IsInf(float64(next), 0) {
		return policy.MaxDelay
	}

	return next
}

func sleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
