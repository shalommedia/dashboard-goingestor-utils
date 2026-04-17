# Project Architecture

## Overview

This repository is a shared Go utilities module for AWS Lambda services. It is organized as small, focused packages that hide AWS SDK setup details, standardize error handling, and provide reusable cross-service helpers.

The current module path is `github.com/shalommedia/dashboard-goingestor-utils` and the workspace targets Go `1.24`.

## Package Boundaries

### logger

- Responsibility: shared structured logging for Lambda-oriented services.
- Key design choices:
  - Uses `log/slog`.
  - Defaults to JSON output for machine-readable Lambda logs.
  - Supports a shared singleton via `Default()` and explicit construction via `New(Config)`.

### s3client

- Responsibility: create and reuse an AWS S3 client and expose narrow helpers for common object operations.
- Key design choices:
  - Uses `sync.Once` to reuse the SDK client across warm Lambda invocations.
  - Wraps AWS calls behind small interfaces for testability.
  - Keeps helper APIs simple and in-memory.

### secretsmanagerclient

- Responsibility: create and reuse an AWS Secrets Manager client and expose helpers for raw secrets, keyed values, and struct decoding.
- Key design choices:
  - Uses the same singleton pattern as `s3client`.
  - Separates raw fetch, map lookup, and JSON unmarshal into distinct helpers.
  - Assumes AWS credentials and region come from the ambient SDK configuration chain.

### pagination

- Responsibility: reusable cursor pagination and retry support for SDK-backed or HTTP-backed integrations.
- Key design choices:
  - Uses generics so the same logic works across item and cursor types.
  - Centralizes retry defaults in `RetryOptions` normalization.
  - Keeps retry behavior context-aware so callers can cancel waits.

### hubspot

- Responsibility: provide a reusable HubSpot HTTP transport foundation with token auth, retries, and rate-limit header parsing.
- Key design choices:
  - Uses a context-first `Do(ctx, method, path, ...)` API for all network operations.
  - Injects auth and default headers in one place to keep domain modules thin.
  - Keeps retry behavior centralized and configurable through `RetryPolicy`.
  - Exposes `ParseRateLimitHeaders` and `AdaptiveThrottle` so callers can observe and adapt to HubSpot quotas.
  - Includes contacts, deals, subscriptions, custom objects, and associations domain helpers built on top of the shared transport.

## Cross-Cutting Patterns

- Context-first APIs: exported functions that perform I/O take `context.Context` as the first parameter.
- Lazy shared clients: package-level `Default()` functions cache heavyweight clients for Lambda reuse.
- Narrow interfaces: internal helpers accept small interfaces instead of concrete SDK clients to make unit tests straightforward.
- Error wrapping: errors should include the operation and relevant resource identifiers using `%w`.
- Small exported surfaces: packages expose a small number of stable helpers instead of broad abstractions.

## Build and Validation

- Build all packages with `go build ./...`.
- Run tests with `go test ./...`.
- Keep module metadata clean with `go mod tidy` after dependency changes.

## Extending the Repository

When adding new packages, prefer the existing utility-package shape:

1. Keep the package focused on one responsibility.
2. Add `New()` for explicit construction when a client or dependency is expensive.
3. Add `Default()` only when reuse across Lambda invocations is useful and safe.
4. Keep internal helpers testable through small interfaces.
5. Do not embed retry, throttling, or logging logic ad hoc in each feature package when it can be centralized.

For future HubSpot SDK work, follow the same package discipline: keep transport and reliability concerns centralized, keep public APIs narrow, and optimize for Lambda reuse rather than per-request reinitialization.

## Streaming APIs (Memory-Efficient For Large Datasets)

Lambda functions have constrained memory. Two streaming APIs enable processing large datasets without buffering entire result sets:

### Pagination Streaming (`FetchPagesStreaming`)

- Fetches pages one-at-a-time from paginated APIs (e.g., HubSpot, Stripe).
- Calls a handler function for each page instead of accumulating all results in memory.
- Ideal for syncing millions of records without OOM.

**Usage**: Call `pagination.FetchPagesStreaming()` with a page handler instead of `FetchAllPages()`.

### S3 Streaming Upload (`PutObjectStream`)

- Uploads objects to S3 via an `io.Reader` stream instead of buffering full payloads.
- Pairs well with `FetchPagesStreaming` to build a pipeline: fetch API pages → transform → stream to S3.
- Supports unknown or large content lengths.

**Usage**: Call `s3client.PutObjectStream()` instead of `PutObject()` for large or streamed data.

**Example**: Stream HubSpot contacts pagination directly to S3 CSV without intermediate buffering:

```go
r, w := io.Pipe()

go func() {
	defer w.Close()
	pagination.FetchPagesStreaming(ctx, "", fetcher, opts,
		func(ctx context.Context, page pagination.PageResult[Contact, string]) error {
			for _, contact := range page.Items {
        fmt.Fprintf(w, "%s,%s\n", contact.ID, contact.Properties["email"])
			}
			return nil
		},
	)
}()

s3client.PutObjectStream(ctx, bucket, "contacts.csv", -1, "text/csv", r)
```

Memory usage remains constant (~1-5MB per page) regardless of total data size, suitable for Lambda environments.

## Current Constraints & Mitigations

- `s3client.GetObject` reads the full object body into memory → use for metadata reads only (sync state, config). For guarded reads, prefer `s3client.GetObjectWithLimit` to enforce a maximum size.
- `secretsmanagerclient.GetSecretValue` assumes a flat JSON object for keyed extraction → use `UnmarshalSecret` for nested structures.
- HubSpot adaptive throttle is process-local state only; cross-process/global quota coordination remains an optional future enhancement.