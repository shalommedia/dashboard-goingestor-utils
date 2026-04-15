package pagination

import (
	"context"
	"fmt"
	"time"
)

// PageResult contains a single page of records and the cursor needed to fetch the next page.
type PageResult[T any, C any] struct {
	Items   []T
	Next    C
	HasMore bool
}

// PageFetcher fetches a single page for the given cursor.
type PageFetcher[T any, C any] func(ctx context.Context, cursor C) (PageResult[T, C], error)

// RetryOptions controls retry behavior for transient pagination failures.
type RetryOptions struct {
	MaxAttempts       int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
	ShouldRetry       func(error) bool
}

// FetchWithRetries fetches a single page and retries transient failures using exponential backoff.
func FetchWithRetries[T any, C any](ctx context.Context, cursor C, fetcher PageFetcher[T, C], opts RetryOptions) (PageResult[T, C], error) {
	opts = normalizeRetryOptions(opts)

	var lastErr error
	delay := opts.InitialDelay

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		result, err := fetcher(ctx, cursor)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !opts.ShouldRetry(err) || attempt == opts.MaxAttempts {
			break
		}

		if err := sleep(ctx, delay); err != nil {
			return PageResult[T, C]{}, fmt.Errorf("retry wait canceled: %w", err)
		}

		delay = nextDelay(delay, opts)
	}

	return PageResult[T, C]{}, fmt.Errorf("fetch page with retries: %w", lastErr)
}

// FetchAllPages keeps requesting pages until the provider reports there are no more pages.
func FetchAllPages[T any, C any](ctx context.Context, initialCursor C, fetcher PageFetcher[T, C], opts RetryOptions) ([]T, error) {
	cursor := initialCursor
	items := make([]T, 0)

	for {
		page, err := FetchWithRetries(ctx, cursor, fetcher, opts)
		if err != nil {
			return nil, err
		}

		items = append(items, page.Items...)
		if !page.HasMore {
			return items, nil
		}

		cursor = page.Next
	}
}

// PageHandler is called once per fetched page during streaming pagination.
type PageHandler[T any, C any] func(ctx context.Context, page PageResult[T, C]) error

// FetchPagesStreaming keeps requesting pages until the provider reports there are no more pages.
// Instead of accumulating all items in memory, it calls the handler function for each page,
// allowing callers to process and discard pages incrementally.
// This is ideal for large result sets that would exceed Lambda memory limits.
func FetchPagesStreaming[T any, C any](ctx context.Context, initialCursor C, fetcher PageFetcher[T, C], opts RetryOptions, handler PageHandler[T, C]) error {
	cursor := initialCursor

	for {
		page, err := FetchWithRetries(ctx, cursor, fetcher, opts)
		if err != nil {
			return err
		}

		if err := handler(ctx, page); err != nil {
			return fmt.Errorf("page handler failed: %w", err)
		}

		if !page.HasMore {
			return nil
		}

		cursor = page.Next
	}
}

func normalizeRetryOptions(opts RetryOptions) RetryOptions {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 3
	}

	if opts.InitialDelay <= 0 {
		opts.InitialDelay = time.Second
	}

	if opts.MaxDelay <= 0 {
		opts.MaxDelay = 30 * time.Second
	}

	if opts.BackoffMultiplier <= 1 {
		opts.BackoffMultiplier = 2
	}

	if opts.ShouldRetry == nil {
		opts.ShouldRetry = func(error) bool { return true }
	}

	return opts
}

func nextDelay(current time.Duration, opts RetryOptions) time.Duration {
	next := time.Duration(float64(current) * opts.BackoffMultiplier)
	if next > opts.MaxDelay {
		return opts.MaxDelay
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
