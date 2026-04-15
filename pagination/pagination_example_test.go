package pagination

import (
	"context"
	"fmt"
)

func ExampleFetchPagesStreaming() {
	type contact struct {
		ID string
	}

	pages := map[string]PageResult[contact, string]{
		"": {
			Items:   []contact{{ID: "1"}, {ID: "2"}},
			Next:    "after-2",
			HasMore: true,
		},
		"after-2": {
			Items:   []contact{{ID: "3"}},
			Next:    "",
			HasMore: false,
		},
	}

	processed := 0

	err := FetchPagesStreaming(context.Background(), "", func(_ context.Context, cursor string) (PageResult[contact, string], error) {
		return pages[cursor], nil
	}, RetryOptions{}, func(_ context.Context, page PageResult[contact, string]) error {
		processed += len(page.Items)
		return nil
	})
	if err != nil {
		fmt.Println("error")
		return
	}

	fmt.Printf("processed=%d\n", processed)
	// Output: processed=3
}
