---
name: hubspot-sdk-go
description: 'Design and implement a public Go HubSpot SDK for AWS Lambda with adaptive rate limiting, retries with jitter, optimized API clients, and shared helper package patterns (S3 and Secrets Manager). Use when creating new HubSpot API modules, hardening reliability, or preparing OSS-quality releases.'
argument-hint: 'Goal or module to implement (for example: contacts list API with throttling and retries)'
user-invocable: true
---

# HubSpot SDK in Go (Public, Lambda-Optimized)

## What This Skill Produces
- A reusable public Go module named `hubspot-sdk-go`.
- A consistent SDK architecture for HubSpot APIs with:
  - Adaptive throttling based on HubSpot rate-limit response headers.
  - Retry behavior for 429, 5xx, and transient network failures.
  - Lambda-friendly performance (connection reuse, minimal allocations, context-aware calls).
- Shared package quality standards that also apply to helper packages like S3 and Secrets Manager clients.

## When to Use
Use this skill when you need to:
- Create or expand HubSpot API support (phase 1 modules: Contacts, Deals, Search, Custom Objects).
- Standardize SDK behavior for throttling and retries.
- Improve performance for AWS Lambda workloads.
- Prepare the package for public release and team-wide reuse.

## Inputs to Confirm Before Coding
1. Module scope and package naming are final (`hubspot-sdk-go`).
2. Go baseline is `1.23` for broad compatibility.
3. Required API scope is defined (contacts, deals, search, custom objects).
4. Authentication mode is fixed for phase 1 (private app token), with OAuth scheduled for phase 2.
5. Public release policy is strict semver (no breaking changes outside major versions), with changelog and docs examples.

## Architecture Blueprint
1. Build a layered structure:
   - `client`: core HTTP transport, auth injection, retry/throttle middleware.
   - `hubspot`: domain modules (`contacts`, `deals`, `search`, `customobjects`).
   - `internal`: non-exported internals (rate limiter state, backoff, response parsing).
   - `helpers`: utility packages with same quality rules (S3/Secrets Manager style consistency).
2. Keep module boundaries explicit:
   - Public API in stable exported interfaces and DTOs.
   - Internal logic hidden behind unexported types.
3. Ensure context-first signatures:
   - Every network-bound function starts with `ctx context.Context`.

## Required Reliability Behavior

### Adaptive Throttling (All HubSpot API Calls)
1. Parse relevant HubSpot rate-limit headers on every response.
2. Maintain limiter state across warm Lambda invocations when possible (per-process scope).
3. Apply adaptive pacing before sending the next request:
   - If remaining quota is low, slow request rate proportionally.
   - If reset window is near, delay until reset with a safety margin.
4. For `429` responses:
   - Honor `Retry-After` when present.
   - If absent, apply exponential backoff with jitter.
5. Expose throttle metrics/events for observability (wait time, throttled count).

### Retries (All HubSpot API Calls)
1. Default retry attempts: `3` total attempts.
2. Retryable categories:
   - `429` (rate limited)
   - `5xx` (server errors)
   - transient transport failures (timeouts, temporary network issues)
3. Backoff policy:
   - Exponential backoff with jitter.
   - Cap maximum wait to protect Lambda timeout budget.
4. Do not retry non-idempotent operations unless caller opts in explicitly.
5. Keep retry logic centralized in shared transport/middleware.

## Lambda Optimization Rules
1. Reuse HTTP clients and limiter state across invocations.
2. Avoid rebuilding heavy objects per call.
3. Minimize allocations in hot paths:
   - Reuse buffers where safe.
   - Avoid reflection-heavy generic maps for common DTO paths.
4. Enforce request deadlines from context to avoid runaway execution.
5. Keep logs structured and low-noise by default.

## Streaming for Large Datasets

For Lambda workloads syncing large HubSpot datasets (millions of contacts, deals, etc.) to data lakes or data warehouses:

- **Pagination Streaming**: Use the repository's `pagination.FetchPagesStreaming()` API to fetch pages one-at-a-time instead of accumulating all results in memory.
- **S3 Streaming Upload**: Use `s3client.PutObjectStream()` to stream transformed data to S3 without intermediate buffering.
- **Pattern**: Build a pipeline—fetch API pages → transform incrementally → stream to destination—keeping memory constant (~1-5MB per page) regardless of total data size.
- **Reference**: See examples in [README.md](../../README.md#streaming-apis-for-large-datasets) and [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md#streaming-apis-memory-efficient-for-large-datasets).

## Implementation Workflow
1. Define module contract first:
   - Public interfaces, request/response structs, error model.
2. Implement core transport:
   - Auth, headers, user-agent, request builder, response decoder.
3. Add reliability middleware:
   - Retry classifier, backoff strategy, adaptive throttle policy.
4. Implement API module incrementally:
   - Start with Contacts and Deals.
   - Add Search patterns.
   - Add Custom Objects abstractions.
5. Add helper package alignment:
   - Ensure S3 and Secrets helpers follow same context/error/reuse conventions.
6. Add tests for each layer:
   - Unit tests for retry/throttle decisions.
   - Integration-like tests with mocked HTTP server behavior.
7. Validate public package readiness:
   - Godoc examples, README usage, versioning notes.

## Decision Points and Branching
1. Auth branch:
   - If private app token is sufficient, ship token auth first.
   - If multi-tenant usage is required, design OAuth token provider abstraction.
2. API expansion branch:
   - If domain-specific DTO churn is high, keep generic internal mapper and stable public DTOs.
   - If endpoints are stable, publish strongly typed request/response models.
3. Throughput branch:
   - Use per-process adaptive throttling as the default and required behavior.
   - If cross-process coordination is needed later, treat it as an optional enhancement outside phase 1.
4. Retry safety branch:
   - If operation is idempotent, apply standard retries.
   - If not idempotent, disable retries by default and require explicit opt-in.

## Quality Gates (Must Pass)
1. Correctness:
   - Retries trigger only for approved transient failures.
   - `Retry-After` is honored for `429`.
   - Pagination returns complete sets without duplication.
2. Performance:
   - No unnecessary client reinitialization.
   - Hot-path allocs reviewed and reduced.
3. API design:
   - Context-first signatures.
   - Stable exported surface and meaningful error wrapping.
4. Operability:
   - Structured logs for retry/throttle outcomes.
   - Optional counters/hooks for metrics.
5. Public package standards:
   - Clear README examples.
   - Versioning and changelog discipline.
   - Strict semver compatibility discipline (no breaking changes outside major versions).

## Completion Checklist
1. Core SDK client implemented with auth, transport, error handling.
2. Adaptive throttling active for all requests.
3. Retries with jitter implemented and configurable.
4. Contacts, Deals, Search, and Custom Objects modules available.
5. Lambda optimization checks completed.
6. Unit tests and behavior tests in place.
7. Public docs and examples ready.
8. Helper package conventions aligned with SDK standards.

## Suggested Invocation Prompts
- `/hubspot-sdk-go Build the contacts module with adaptive throttling and retry middleware.`
- `/hubspot-sdk-go Add deals search endpoints and tests for 429 plus Retry-After behavior.`
- `/hubspot-sdk-go Review this module for Lambda performance and public API stability.`
