# Security Review

**Date**: April 7, 2026  
**Scope**: logger, pagination, s3client, secretsmanagerclient, hubspot  
**Result**: ✅ No critical vulnerabilities found

## Dependency Vulnerability Scan

Ran `govulncheck ./...` across all packages using AWS SDK v2 (March 2026 releases).

**Result**: No known CVEs or vulnerabilities detected in any direct or transitive dependencies.

---

## Package-by-Package Security Analysis

### 1. **logger**

**Purpose**: Shared structured logging using `log/slog`

**Security Findings**: ✅ **SECURE**

| Finding | Status | Details |
|---------|--------|---------|
| **Log level parsing** | ✅ Safe | Case-insensitive string match with sane defaults (INFO) |
| **Output handler** | ✅ Safe | Supports JSON (default) and text formats; defaults to stdout |
| **Service tagging** | ✅ Safe | Service name is user-provided but safe (included as context tag, not in format strings) |
| **No hardcoded secrets** | ✅ Safe | No credentials, keys, or sensitive defaults baked in |
| **Concurrency** | ✅ Safe | `sync.Once` ensures single initialization; slog handlers are thread-safe |

**Recommendations**:
- Avoid calling `logger.With()` with secrets or PII as arguments; slog will include them in output.
- For Lambda functions, test log output in CloudWatch to ensure no inadvertent secret leakage.

---

### 2. **s3client**

**Purpose**: S3 object upload/download with lazy client initialization

**Security Findings**: ✅ **IMPROVED: Guarded Read Option Added**

| Finding | Status | Details |
|---------|--------|---------|
| **In-memory download** | ⚠️ Medium | `GetObject()` still reads full object into memory for backward compatibility |
| **Size-limited download option** | ✅ Mitigated | `GetObjectWithLimit()` now enforces max byte reads and fails fast on oversized responses |
| **Client reuse** | ✅ Safe | `sync.Once` prevents concurrent initialization; safe for Lambda reuse |
| **AWS credential chain** | ✅ Safe | Uses ambient `config.LoadDefaultConfig()`, respects IAM roles and env vars |
| **Error handling** | ✅ Safe | Errors wrap operation + resource context using `%w` |
| **No hardcoded endpoint** | ✅ Safe | Uses SDK default/configured region |

**Recommendations**:
1. **Document in-memory limitation**: Add a warning in package docs that objects >100MB are not recommended.
2. **Use streaming for large objects**: ✅ Use `s3client.PutObjectStream()` to upload large payloads without buffering.
3. **Use guarded reads**: ✅ Use `s3client.GetObjectWithLimit()` to enforce max in-memory size for downloads.

---

### 3. **secretsmanagerclient**

**Purpose**: Fetch and unmarshal secrets from AWS Secrets Manager

**Security Findings**: ✅ **IMPROVED: Error Sanitization Implemented**

| Finding | Status | Details |
|---------|--------|---------|
| **Secrets in errors** | ✅ Mitigated | Error messages are sanitized and do not include secret IDs or requested keys |
| **JSON unmarshaling** | ✅ Safe | Uses `json.Unmarshal()` with user-provided struct; no code execution risk |
| **Flat JSON assumption** | ✅ Safe | `GetSecretValue()` only handles flat maps; nested structures must use `UnmarshalSecret()` |
| **Client reuse** | ✅ Safe | `sync.Once` prevents concurrent init; reused safely across Lambda invocations |
| **No static credentials** | ✅ Safe | All auth via ambient AWS SDK chain |
| **Nil pointer checks** | ✅ Safe | Checks `output.SecretString != nil` before dereferencing |

**Recommendations**:
1. **Keep sanitized errors as default**: ✅ implemented.
2. **Validate key names**: If the JSON secret structure is known, validate keys exist before `GetSecretValue()` to avoid 404 disclosures.
3. **Test error logs**: Ensure error messages do not leak secret contents to debug logs.

---

### 5. **hubspot**

**Purpose**: HubSpot HTTP transport with private app token auth, retries, and adaptive throttling scaffolding

**Security Findings**: ✅ **SECURE WITH STANDARD CAVEATS**

| Finding | Status | Details |
|---------|--------|---------|
| **Token auth header injection** | ✅ Safe | Authorization is applied centrally through `Client.Do()` |
| **Retry behavior** | ✅ Safe | Retries are centralized and respect context cancellation and `Retry-After` |
| **Rate-limit adaptation** | ✅ Safe | `AdaptiveThrottle` uses process-local pacing from response headers |
| **Context cancellation** | ✅ Safe | Retry waits and throttle waits exit on context cancellation |

**Recommendations**:
1. Keep private app tokens in Secrets Manager and inject via environment/runtime config.
2. Add domain-level input validation (contacts/deals/search) as modules are added.
3. Consider cross-process rate-limit coordination if many Lambdas share the same HubSpot app at high throughput.

---

### 4. **pagination**

**Purpose**: Generic cursor-based pagination with retry and backoff

**Security Findings**: ✅ **SECURE**

| Finding | Status | Details |
|---------|--------|---------|
| **Context cancellation** | ✅ Safe | Respects context deadlines and cancellation; uses non-blocking timer with `select` |
| **Exponential backoff** | ✅ Safe | Properly implements backoff with configurable multiplier and max delay cap (30s default) |
| **No timing attacks** | ✅ Safe | Backoff uses deterministic jitter calculation; not cryptographic but not exposed to untrusted input |
| **Custom retry logic** | ✅ Safe | Caller-supplied `ShouldRetry()` function is called but not dangerous (no reflection or code eval) |
| **Timeout protection** | ✅ Safe | Max delay cap of 30s protects Lambda execution from unbounded waits |
| **Nil cursor handling** | ✅ Safe | Handles empty/zero cursors (string "", int 0, etc.) correctly |

**Recommendations**:
- For long-running paginated operations, set a context timeout at the Lambda handler level:
  ```go
  ctx, cancel := context.WithTimeout(context.Background(), 59*time.Second)
  defer cancel()
  ```

---

## Cross-Package Security Observations

| Pattern | Status | Details |
|---------|--------|---------|
| **Context-first APIs** | ✅ Best practice | All network operations accept context first; enables cancellation and timeouts |
| **Lazy initialization** | ✅ Best practice | `sync.Once` ensures no concurrent client creation; safe for Lambda warm starts |
| **Error wrapping** | ✅ Best practice | All errors use `%w`; preserves error chains for logging and debugging |
| **No global state abuse** | ✅ Safe | Only package-level clients cached; no mutable shared state |
| **No reflection** | ✅ Safe | No runtime type checking or field manipulation |
| **No external process calls** | ✅ Safe | No `os/exec` or shell invocation |
| **No dynamic code** | ✅ Safe | No `eval()`, plugin loading, or code generation |

---

## Lambda-Specific Security Considerations

Since these utilities target AWS Lambda:

1. **Credential Leakage Risk**: ✅ Low
   - SDK uses IAM roles automatically in Lambda; no credential files needed.
   - Environment variable credentials (if used) should be restricted to Lambda role scope.

2. **Cold Start Overhead**: ✅ Acceptable
   - Client initialization happens once per warm instance via `sync.Once`.
   - Lazy clients reduce cold start time.

3. **Timeout Exposure**: ✅ Protected
   - Context timeouts and backoff caps prevent runaway operations.
   - Test Lambda timeout settings (default 3s is very short; use 60s+ for I/O).

4. **Memory Constraints**: ✅ Mitigated
   - `pagination.FetchPagesStreaming()` processes large result sets one page at a time without accumulation.
   - `s3client.PutObjectStream()` streams uploads without buffering entire payloads.
   - Pattern: fetch API pages → transform → stream to S3 keeps memory constant (~1-5MB) regardless of data size.
   - See [docs/ARCHITECTURE.md](ARCHITECTURE.md#streaming-apis-memory-efficient-for-large-datasets) for examples.

---

## Recommendations Summary

### Critical (Must Fix)
- None identified.

### High Priority
- None identified.

### Medium Priority (Improve Code Quality)
1. **s3client**: ✅ **RESOLVED** — Added `PutObjectStream()` and `GetObjectWithLimit()` safeguards.
2. **secretsmanagerclient**: ✅ **RESOLVED** — Sanitized secret/key values from error messages.

### Low Priority (Best Practices)
- Add test coverage for timeout/cancellation scenarios in pagination and S3 client.
- Consider security-focused unit tests (e.g., verify context cancellation is honored).

---

## Testing Recommendations

Add the following test scenarios to improve security posture:

1. **Logger**: Verify no secrets appear in logs when a structured logger is misused.
2. **S3 Client**: ✅ Covered max-size guardrail and canceled read paths; keep stress testing for very large payloads.
3. **Secrets Manager**: ✅ Covered sanitized error behavior and malformed JSON handling.
4. **Pagination**: Test that context timeout is respected; verify backoff respects max delay cap.

---

## Conclusion

The utility packages follow Go security best practices:
- ✅ No dependency vulnerabilities
- ✅ Safe error handling and context management
- ✅ No hardcoded credentials or secrets
- ✅ Thread-safe client reuse for Lambda
- ⚠️ Residual risk: `GetObject()` remains full in-memory read for backward compatibility; prefer `GetObjectWithLimit()` for guarded reads.

**Overall Risk**: **LOW** — Suitable for production use with noted caveats.
