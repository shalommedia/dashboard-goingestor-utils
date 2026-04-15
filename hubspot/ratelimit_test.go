package hubspot

import (
	"net/http"
	"testing"
	"time"
)

func TestParseRateLimitHeaders(t *testing.T) {
	t.Parallel()

	headers := make(http.Header)
	headers.Set("X-HubSpot-RateLimit-Interval-Milliseconds", "10000")
	headers.Set("X-HubSpot-RateLimit-Max", "100")
	headers.Set("X-HubSpot-RateLimit-Remaining", "42")
	headers.Set("X-HubSpot-RateLimit-Daily-Remaining", "9999")
	headers.Set("Retry-After", "3")

	info := ParseRateLimitHeaders(headers)

	if info.IntervalMilliseconds != 10000 {
		t.Fatalf("unexpected interval: %d", info.IntervalMilliseconds)
	}

	if info.Max != 100 {
		t.Fatalf("unexpected max: %d", info.Max)
	}

	if info.Remaining != 42 {
		t.Fatalf("unexpected remaining: %d", info.Remaining)
	}

	if info.DailyRemaining != 9999 {
		t.Fatalf("unexpected daily remaining: %d", info.DailyRemaining)
	}

	if info.RetryAfter != 3*time.Second {
		t.Fatalf("unexpected retry-after: %s", info.RetryAfter)
	}
}

func TestParseRateLimitHeaders_InvalidValuesFallbackToZero(t *testing.T) {
	t.Parallel()

	headers := make(http.Header)
	headers.Set("X-HubSpot-RateLimit-Interval-Milliseconds", "bad")
	headers.Set("Retry-After", "NaN")

	info := ParseRateLimitHeaders(headers)

	if info.IntervalMilliseconds != 0 {
		t.Fatalf("expected interval 0, got %d", info.IntervalMilliseconds)
	}

	if info.RetryAfter != 0 {
		t.Fatalf("expected retry-after 0, got %s", info.RetryAfter)
	}
}
