package pagination

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestFetchPagesStreaming_ProcessesAllPagesInOrder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	type pageDef struct {
		items   []int
		next    string
		hasMore bool
	}

	pages := map[string]pageDef{
		"": {
			items:   []int{1, 2},
			next:    "cursor-1",
			hasMore: true,
		},
		"cursor-1": {
			items:   []int{3, 4},
			next:    "cursor-2",
			hasMore: true,
		},
		"cursor-2": {
			items:   []int{5},
			next:    "",
			hasMore: false,
		},
	}

	var seenCursors []string
	var seenItems []int

	fetcher := func(_ context.Context, cursor string) (PageResult[int, string], error) {
		seenCursors = append(seenCursors, cursor)
		def, ok := pages[cursor]
		if !ok {
			return PageResult[int, string]{}, errors.New("unexpected cursor")
		}

		return PageResult[int, string]{
			Items:   def.items,
			Next:    def.next,
			HasMore: def.hasMore,
		}, nil
	}

	handler := func(_ context.Context, page PageResult[int, string]) error {
		seenItems = append(seenItems, page.Items...)
		return nil
	}

	err := FetchPagesStreaming(ctx, "", fetcher, RetryOptions{}, handler)
	if err != nil {
		t.Fatalf("FetchPagesStreaming returned error: %v", err)
	}

	if !reflect.DeepEqual(seenCursors, []string{"", "cursor-1", "cursor-2"}) {
		t.Fatalf("unexpected cursor sequence: %#v", seenCursors)
	}

	if !reflect.DeepEqual(seenItems, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("unexpected item sequence: %#v", seenItems)
	}
}

func TestFetchPagesStreaming_StopsOnHandlerError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	handlerErr := errors.New("handler failed")

	fetchCalls := 0
	fetcher := func(_ context.Context, _ string) (PageResult[int, string], error) {
		fetchCalls++
		return PageResult[int, string]{
			Items:   []int{1},
			Next:    "cursor-1",
			HasMore: true,
		}, nil
	}

	handler := func(_ context.Context, _ PageResult[int, string]) error {
		return handlerErr
	}

	err := FetchPagesStreaming(ctx, "", fetcher, RetryOptions{}, handler)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, handlerErr) {
		t.Fatalf("expected wrapped handler error, got: %v", err)
	}

	if fetchCalls != 1 {
		t.Fatalf("expected one fetch call, got %d", fetchCalls)
	}
}

func TestFetchPagesStreaming_ReturnsRetryWrappedFetcherError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fetchErr := errors.New("transient failure")

	fetchCalls := 0
	fetcher := func(_ context.Context, _ string) (PageResult[int, string], error) {
		fetchCalls++
		return PageResult[int, string]{}, fetchErr
	}

	err := FetchPagesStreaming(ctx, "", fetcher, RetryOptions{
		MaxAttempts:       2,
		InitialDelay:      time.Millisecond,
		MaxDelay:          time.Millisecond,
		BackoffMultiplier: 2,
		ShouldRetry:       func(error) bool { return true },
	}, func(context.Context, PageResult[int, string]) error {
		return nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, fetchErr) {
		t.Fatalf("expected wrapped fetch error, got: %v", err)
	}

	if fetchCalls != 2 {
		t.Fatalf("expected 2 fetch attempts, got %d", fetchCalls)
	}
}

func TestFetchPagesStreaming_ContextCanceledDuringRetryWait(t *testing.T) {
	t.Parallel()

	fetchErr := errors.New("upstream temporary error")
	fetchCalls := 0
	fetcher := func(_ context.Context, _ string) (PageResult[int, string], error) {
		fetchCalls++
		return PageResult[int, string]{}, fetchErr
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()

	err := FetchPagesStreaming(ctx, "", fetcher, RetryOptions{
		MaxAttempts:       3,
		InitialDelay:      time.Second,
		MaxDelay:          time.Second,
		BackoffMultiplier: 2,
		ShouldRetry:       func(error) bool { return true },
	}, func(context.Context, PageResult[int, string]) error {
		return nil
	})

	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got: %v", err)
	}

	if fetchCalls != 1 {
		t.Fatalf("expected one fetch call before canceled wait, got %d", fetchCalls)
	}
}
