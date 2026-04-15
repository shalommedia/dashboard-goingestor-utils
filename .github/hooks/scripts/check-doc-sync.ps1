$ErrorActionPreference = 'Stop'

function Write-Json($value) {
    $value | ConvertTo-Json -Depth 10 -Compress
}

$rawInput = [Console]::In.ReadToEnd()
if ([string]::IsNullOrWhiteSpace($rawInput)) {
    Write-Json @{}
    exit 0
}

try {
    $payload = $rawInput | ConvertFrom-Json -Depth 20
} catch {
    Write-Json @{}
    exit 0
}

if ($payload.stop_hook_active -eq $true) {
    Write-Json @{}
    exit 0
}

$workspace = [string]$payload.cwd
if ([string]::IsNullOrWhiteSpace($workspace)) {
    $workspace = (Get-Location).Path
}

Push-Location $workspace
try {
    $tracked = @(& git diff --name-only --diff-filter=ACMR HEAD 2>$null)
    $untracked = @(& git ls-files --others --exclude-standard 2>$null)
} finally {
    Pop-Location
}

$changed = @($tracked + $untracked | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Sort-Object -Unique)
if ($changed.Count -eq 0) {
    Write-Json @{}
    exit 0
}

$docFiles = @(
    'README.md',
    'docs/ARCHITECTURE.md',
    '.github/copilot-instructions.md'
)

$docPrefixes = @(
    '.github/skills/'
)

$codePatterns = @(
    '*.go',
    'go.mod',
    'go.sum'
)

$hasCodeChange = $false
foreach ($path in $changed) {
    $normalized = $path.Replace('\', '/')
    foreach ($pattern in $codePatterns) {
        if ($normalized -like $pattern) {
            $hasCodeChange = $true
            break
        }
    }

    if ($hasCodeChange) {
        break
    }
}

if (-not $hasCodeChange) {
    Write-Json @{}
    exit 0
}

$hasDocChange = $false
foreach ($path in $changed) {
    $normalized = $path.Replace('\', '/')
    if ($docFiles -contains $normalized) {
        $hasDocChange = $true
        break
    }

    foreach ($prefix in $docPrefixes) {
        if ($normalized.StartsWith($prefix) -and $normalized.EndsWith('SKILL.md')) {
            $hasDocChange = $true
            break
        }
    }

    if ($hasDocChange) {
        break
    }
}

if ($hasDocChange) {
    Write-Json @{}
    exit 0
}

$touchesModuleFiles = $false
$touchesPackageCode = $false
$touchesRepoConventions = $false
$touchesHubspotSdk = $false

foreach ($path in $changed) {
    $normalized = $path.Replace('\', '/')

    if ($normalized -eq 'go.mod' -or $normalized -eq 'go.sum') {
        $touchesModuleFiles = $true
    }

    if ($normalized.StartsWith('logger/') -or
        $normalized.StartsWith('pagination/') -or
        $normalized.StartsWith('s3client/') -or
        $normalized.StartsWith('secretsmanagerclient/')) {
        $touchesPackageCode = $true
    }

    if ($normalized.StartsWith('.github/hooks/') -or
        $normalized.StartsWith('.github/copilot-instructions.md')) {
        $touchesRepoConventions = $true
    }

    if ($normalized.StartsWith('hubspot/')) {
        $touchesHubspotSdk = $true
    }
}

$recommendedDocs = @()
if ($touchesModuleFiles -or $touchesPackageCode -or $touchesHubspotSdk) {
    $recommendedDocs += 'README.md'
}

if ($touchesModuleFiles -or $touchesPackageCode -or $touchesHubspotSdk) {
    $recommendedDocs += 'docs/ARCHITECTURE.md'
}

if ($touchesRepoConventions -or $touchesHubspotSdk) {
    $recommendedDocs += '.github/copilot-instructions.md'
}

if ($touchesHubspotSdk) {
    $recommendedDocs += '.github/skills/hubspot-sdk-go/SKILL.md'
}

$recommendedDocs = @($recommendedDocs | Sort-Object -Unique)
if ($recommendedDocs.Count -eq 0) {
    $recommendedDocs = @('README.md', 'docs/ARCHITECTURE.md', '.github/copilot-instructions.md')
}

$changedPreview = ($changed | Select-Object -First 8) -join "`n- "
if (-not [string]::IsNullOrWhiteSpace($changedPreview)) {
    $changedPreview = '- ' + $changedPreview
}

$docPreview = ($recommendedDocs | ForEach-Object { "- $_" }) -join "`n"

Write-Json @{
    hookSpecificOutput = @{
        hookEventName = 'Stop'
    }
    systemMessage = "Doc sync reminder: code changed but no docs/customization files were updated. Consider updating:`n$docPreview`nChanged files:`n$changedPreview"
}
exit 0
