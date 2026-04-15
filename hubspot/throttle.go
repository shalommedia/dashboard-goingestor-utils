package hubspot

import (
	"context"
	"sync"
	"time"
)

// AdaptiveThrottle applies lightweight pacing based on HubSpot rate-limit headers.
// It is process-local and safe for concurrent use.
type AdaptiveThrottle struct {
	mu          sync.Mutex
	nextAllowed time.Time
	now         func() time.Time
}

// NewAdaptiveThrottle creates a new process-local adaptive throttle state.
func NewAdaptiveThrottle() *AdaptiveThrottle {
	return &AdaptiveThrottle{now: time.Now}
}

// Wait blocks until the next allowed send time when throttling is active.
func (a *AdaptiveThrottle) Wait(ctx context.Context) error {
	if a == nil {
		return nil
	}

	a.mu.Lock()
	next := a.nextAllowed
	now := a.now
	a.mu.Unlock()

	delay := time.Until(next)
	if now != nil {
		delay = next.Sub(now())
	}

	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Observe updates the throttle state from response metadata.
func (a *AdaptiveThrottle) Observe(statusCode int, info RateLimitInfo) {
	if a == nil {
		return
	}

	nowFn := a.now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn()

	var delay time.Duration
	if info.RetryAfter > 0 {
		delay = info.RetryAfter
	} else if statusCode == 429 {
		if info.IntervalMilliseconds > 0 {
			delay = time.Duration(info.IntervalMilliseconds) * time.Millisecond
		} else {
			delay = time.Second
		}
	} else if info.IntervalMilliseconds > 0 && info.Max > 0 {
		lowWatermark := info.Max / 10
		if lowWatermark < 1 {
			lowWatermark = 1
		}

		if info.Remaining >= 0 && info.Remaining <= lowWatermark {
			// Pace requests when quota is low so we can cross the reset window safely.
			step := time.Duration(info.IntervalMilliseconds) * time.Millisecond / time.Duration(info.Remaining+1)
			if step > 0 {
				delay = step
			}
		}
	}

	if delay <= 0 {
		return
	}

	candidate := now.Add(delay)
	a.mu.Lock()
	if candidate.After(a.nextAllowed) {
		a.nextAllowed = candidate
	}
	a.mu.Unlock()
}
