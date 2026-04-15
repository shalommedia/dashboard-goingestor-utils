package hubspot

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RateLimitInfo contains parsed HubSpot rate-limit headers when present.
type RateLimitInfo struct {
	IntervalMilliseconds int
	Max                  int
	Remaining            int
	DailyRemaining       int
	RetryAfter           time.Duration
}

// ParseRateLimitHeaders extracts common HubSpot rate-limit metadata from response headers.
func ParseRateLimitHeaders(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{
		IntervalMilliseconds: parseInt(headers.Get("X-HubSpot-RateLimit-Interval-Milliseconds")),
		Max:                  parseInt(headers.Get("X-HubSpot-RateLimit-Max")),
		Remaining:            parseInt(headers.Get("X-HubSpot-RateLimit-Remaining")),
		DailyRemaining:       parseInt(headers.Get("X-HubSpot-RateLimit-Daily-Remaining")),
	}

	retryAfter := strings.TrimSpace(headers.Get("Retry-After"))
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
			info.RetryAfter = time.Duration(seconds) * time.Second
		}
	}

	return info
}

func parseInt(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}

	return value
}
