# Project Guidelines

## Architecture

- This repository is a shared Go utilities module for AWS Lambda services.
- Package responsibilities and extension guidance are documented in [docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md).
- Prefer small, single-purpose packages over large shared utility grab-bags.

## Code Style

- Follow the existing package shape: `New()` for explicit construction, `Default()` for lazy shared reuse only when it is safe.
- Put `context.Context` first in exported functions that perform I/O or blocking work.
- Keep AWS or external SDK interactions behind narrow interfaces when that improves testability.
- Wrap errors with operation context and resource identifiers using `%w`.
- Preserve the current style of small public APIs and helper-focused packages.

## Readability Rules (Anti-Spaghetti)

- Prefer straightforward control flow over cleverness. Use early returns to reduce nesting.
- Keep functions short and single-purpose. Split logic into named helpers when a function starts mixing concerns.
- Avoid complex or deeply nested loops. Prefer simple linear passes and small helper functions.
- Avoid dense one-liners that hide intent. Optimize for code that is easy to review by humans.
- Keep conditionals explicit and readable; avoid chained boolean expressions when they reduce clarity.
- Preserve performance optimizations that matter for Lambda workloads (client reuse, bounded allocations, centralized retry/throttle).
- Do not add abstractions unless they remove repetition or improve testability with clear value.
- When touching performance-sensitive paths, keep the same Big-O behavior unless a change is explicitly required.

## Build And Test

- Use `go build ./...` to validate compilation across the module.
- Use `go test ./...` for test validation.
- Use `go mod tidy` after dependency changes.

## Conventions

- Assume Lambda-style runtime behavior: reuse heavyweight clients and loggers across warm invocations instead of rebuilding them per call.
- Prefer structured logging with the `logger` package for new service-facing packages.
- Keep retry and pagination behavior centralized rather than reimplementing backoff logic in multiple packages.
- If adding HubSpot SDK packages, keep transport, retry, and throttling concerns centralized and keep the exported API stable.
- Update [README.md](../README.md) when adding a new reusable package or changing import or usage patterns.