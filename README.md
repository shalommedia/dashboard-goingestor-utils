# dashboard-goingestor-utils

Shared Go utilities for Lambda projects.

## Packages

- `s3client`: reusable AWS S3 client creation helpers
- `secretsmanagerclient`: reusable AWS Secrets Manager read helpers
- `pagination`: generic pagination and retry helpers for SDK or HTTP-based APIs
- `logger`: shared structured logging helpers for Lambda services
- `hubspot`: HubSpot transport client with auth headers, retries, rate-limit parsing, and CRM helpers for contacts, deals, custom objects, and associations

## Usage

```go
import "github.com/shalommedia/dashboard-goingestor-utils/s3client"
```

```go
client, err := s3client.Default(ctx)
if err != nil {
    log.Printf("failed to create s3 client: %v", err)
}
```

```go
apiKey, err := secretsmanagerclient.GetSecretValue(ctx, "my-app-secret", "api_key")
if err != nil {
    log.Printf("failed to read api key: %v", err)
}
```

```go
type DatabaseSecret struct {
    Host     string `json:"host"`
    Port     string `json:"port"`
    Username string `json:"username"`
    Password string `json:"password"`
    DBName   string `json:"dbname"`
}

var dbSecret DatabaseSecret
err = secretsmanagerclient.UnmarshalSecret(ctx, "my-db-secret", &dbSecret)
if err != nil {
    log.Printf("failed to read db secret: %v", err)
}
```

```go
type HubSpotContact struct {
	ID string `json:"id"`
}

contacts, err := pagination.FetchAllPages(ctx, "", func(ctx context.Context, cursor string) (pagination.PageResult[HubSpotContact, string], error) {
	resp, err := hubspotClient.FetchContacts(ctx, cursor)
	if err != nil {
		return pagination.PageResult[HubSpotContact, string]{}, err
	}

	return pagination.PageResult[HubSpotContact, string]{
		Items:   resp.Results,
		Next:    resp.Paging.Next.After,
		HasMore: resp.Paging != nil && resp.Paging.Next.After != "",
	}, nil
}, pagination.RetryOptions{})
```

```go
log := logger.New(logger.Config{
	Service: "hubspot-sync",
	Level:   "info",
})

log.Info("sync started", "job_id", jobID)
log.Error("sync failed", "error", err)
```

```go
hubspotClient, err := hubspot.New(hubspot.Config{
	Token: os.Getenv("HUBSPOT_PRIVATE_APP_TOKEN"),
})
if err != nil {
	log.Printf("failed to create hubspot client: %v", err)
}

resp, err := hubspotClient.Do(ctx, http.MethodGet, "/crm/v3/objects/contacts", nil, nil)
if err != nil {
	log.Printf("hubspot request failed: %v", err)
}
defer resp.Body.Close()

limitInfo := hubspot.ParseRateLimitHeaders(resp.Header)
log.Printf("remaining quota: %d", limitInfo.Remaining)
```

### HubSpot Reliability (Retry + Adaptive Throttle)

Use `RetryPolicy` and `AdaptiveThrottle` together to improve stability under rate limits:

```go
hubspotClient, err := hubspot.New(hubspot.Config{
	Token:    os.Getenv("HUBSPOT_PRIVATE_APP_TOKEN"),
	Throttle: hubspot.NewAdaptiveThrottle(),
	Retry: hubspot.RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: 250 * time.Millisecond,
		MaxDelay:     5 * time.Second,
	},
})
if err != nil {
	log.Printf("failed to create hubspot client: %v", err)
}

contactsPage, err := hubspotClient.ListContacts(ctx, hubspot.ListContactsRequest{
	Limit:      100,
	Properties: []string{"email", "firstname", "lastname"},
})
if err != nil {
	log.Printf("failed to list contacts: %v", err)
}

_ = contactsPage

searchPage, err := hubspotClient.SearchContacts(ctx, hubspot.ContactSearchRequest{
	Limit:      10,
	Properties: []string{"email", "firstname", "lastname"},
	FilterGroups: []hubspot.ContactFilterGroup{{
		Filters: []hubspot.ContactFilter{{
			PropertyName: "email",
			Operator:     "CONTAINS_TOKEN",
			Value:        "example.com",
		}},
	}},
})
if err != nil {
	log.Printf("failed to search contacts: %v", err)
}

created, err := hubspotClient.CreateContact(ctx, hubspot.CreateContactRequest{
	Properties: map[string]string{
		"email":     "new-contact@example.com",
		"firstname": "New",
		"lastname":  "Contact",
	},
})
if err != nil {
	log.Printf("failed to create contact: %v", err)
}

fetched, err := hubspotClient.GetContact(ctx, created.ID, hubspot.GetContactRequest{
	Properties: []string{"email", "firstname", "lastname"},
})
if err != nil {
	log.Printf("failed to fetch contact: %v", err)
}

updated, err := hubspotClient.UpdateContact(ctx, fetched.ID, hubspot.UpdateContactRequest{
	Properties: map[string]string{"firstname": "Updated"},
})
if err != nil {
	log.Printf("failed to update contact: %v", err)
}

if err := hubspotClient.DeleteContact(ctx, updated.ID); err != nil {
	log.Printf("failed to delete contact: %v", err)
}

_ = searchPage
```

## Streaming APIs (For Large Datasets)

### Pagination Streaming

For large result sets, use `FetchPagesStreaming` to process pages one-at-a-time without accumulating all results in memory:

```go
import (
	"io"

	"github.com/shalommedia/dashboard-goingestor-utils/hubspot"
	"github.com/shalommedia/dashboard-goingestor-utils/pagination"
)

err := pagination.FetchPagesStreaming(ctx, "", 
	func(ctx context.Context, cursor string) (pagination.PageResult[hubspot.Contact, string], error) {
		resp, err := hubspotClient.ListContacts(ctx, hubspot.ListContactsRequest{
			After:      cursor,
			Limit:      100,
			Properties: []string{"email"},
		})
		if err != nil {
			return pagination.PageResult[hubspot.Contact, string]{}, err
		}

		nextCursor := ""
		hasMore := resp.Paging != nil && resp.Paging.Next != nil && resp.Paging.Next.After != ""
		if hasMore {
			nextCursor = resp.Paging.Next.After
		}

		return pagination.PageResult[hubspot.Contact, string]{
			Items:   resp.Results,
			Next:    nextCursor,
			HasMore: hasMore,
		}, nil
	},
	pagination.RetryOptions{},
	func(ctx context.Context, page pagination.PageResult[hubspot.Contact, string]) error {
		// Process each page incrementally
		for _, contact := range page.Items {
			log.Info("processing contact", "id", contact.ID)
		}
		return nil
	},
)
```

### S3 Streaming Upload

For streaming large payloads to S3 without buffering, use `PutObjectStream`:

```go
import "io"

// Example: stream directly from HubSpot pagination to S3
r, w := io.Pipe()

go func() {
	defer w.Close()
	// Write streaming data to pipe as pages arrive
	pagination.FetchPagesStreaming(ctx, "", fetcher, opts,
		func(ctx context.Context, page pagination.PageResult[hubspot.Contact, string]) error {
			for _, contact := range page.Items {
				fmt.Fprintf(w, "%s,%s\n", contact.ID, contact.Properties["email"])
			}
			return nil
		},
	)
}()

// Upload the stream to S3 without buffering
err := s3client.PutObjectStream(ctx, "my-bucket", "contacts.csv", -1, "text/csv", r)
if err != nil {
	log.Printf("failed to upload: %v", err)
}
```

### S3 Size-Limited Download

For safer in-memory reads, use `GetObjectWithLimit` to enforce a byte cap:

```go
data, err := s3client.GetObjectWithLimit(ctx, "my-bucket", "state.json", 1024*1024)
if err != nil {
	log.Printf("failed to read object: %v", err)
}

_ = data
```

## Note

The current module path is `github.com/shalommedia/dashboard-goingestor-utils`.
