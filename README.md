# dashboard-goingestor-utils

Shared Go utilities for Lambda projects.

## Packages

- `s3client`: reusable AWS S3 client creation helpers
- `secretsmanagerclient`: reusable AWS Secrets Manager read helpers
- `pagination`: generic pagination and retry helpers for SDK or HTTP-based APIs
- `logger`: shared structured logging helpers for Lambda services

## Usage

```go
import "dashboard-goingestor-utils/s3client"
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

## Note

Update the `module` value in `go.mod` after you create the GitHub repository so other Lambda repos can import it using the real GitHub path.
