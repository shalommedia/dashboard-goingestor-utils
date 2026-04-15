$ErrorActionPreference = 'Stop'

function Write-Json($value) {
    $value | ConvertTo-Json -Depth 10 -Compress
}

$context = @'
Repository: dashboard-goingestor-utils
Module: github.com/shalommedia/dashboard-goingestor-utils
Build commands: go build ./..., go test ./..., go mod tidy
Architecture: shared Go utilities for AWS Lambda with focused packages.
Packages:
- logger: structured slog logging with reusable defaults
- pagination: generic retry and cursor pagination helpers
- s3client: reusable S3 client with in-memory helpers and streaming upload support
- secretsmanagerclient: reusable Secrets Manager helpers for raw, keyed, and unmarshaled secrets
Key conventions:
- Put context.Context first for I/O and blocking work
- Prefer New() for explicit construction and Default() only for safe shared reuse
- Keep SDK interactions behind narrow interfaces when useful for testability
- Wrap errors with operation and resource context using %w
- Update README.md, docs/ARCHITECTURE.md, .github/copilot-instructions.md, or relevant SKILL.md files when repository behavior or public usage changes
'@

Write-Json @{
    hookSpecificOutput = @{
        hookEventName     = 'SessionStart'
        additionalContext = $context.Trim()
    }
}
exit 0
