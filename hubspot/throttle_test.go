package hubspot

import (
	"context"
	"testing"
	"time"
)

func TestAdaptiveThrottle_ObserveRetryAfterSetsNextAllowed(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	throttle := &AdaptiveThrottle{now: func() time.Time { return base }}

	throttle.Observe(429, RateLimitInfo{RetryAfter: 3 * time.Second})

	throttle.mu.Lock()
	next := throttle.nextAllowed
	throttle.mu.Unlock()

	if !next.Equal(base.Add(3 * time.Second)) {
		t.Fatalf("unexpected next allowed: %s", next)
	}
}

func TestAdaptiveThrottle_ObserveLowRemainingAppliesPacing(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	throttle := &AdaptiveThrottle{now: func() time.Time { return base }}

	throttle.Observe(200, RateLimitInfo{
		IntervalMilliseconds: 10000,
		Max:                  100,
		Remaining:            5,
	})

	throttle.mu.Lock()
	next := throttle.nextAllowed
	throttle.mu.Unlock()

	if !next.After(base) {
		t.Fatalf("expected pacing delay, got next=%s base=%s", next, base)
	}
}

func TestAdaptiveThrottle_WaitReturnsContextCancellation(t *testing.T) {
	t.Parallel()

	throttle := &AdaptiveThrottle{}
	throttle.mu.Lock()
	throttle.nextAllowed = time.Now().Add(time.Second)
	throttle.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := throttle.Wait(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err != context.Canceled {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
