$ErrorActionPreference = 'Stop'

function Write-Json($value) {
    $value | ConvertTo-Json -Depth 10 -Compress
}

function Add-GoPath {
    param(
        [string]$Path,
        [ref]$Paths
    )

    if ([string]::IsNullOrWhiteSpace($Path)) {
        return
    }

    if ($Path.ToLowerInvariant().EndsWith('.go')) {
        $Paths.Value += $Path
    }
}

function Get-GoPathsFromPatch {
    param([string]$PatchText)

    $paths = @()
    if ([string]::IsNullOrWhiteSpace($PatchText)) {
        return $paths
    }

    foreach ($line in ($PatchText -split "`r?`n")) {
        if (-not ($line.StartsWith('*** Add File: ') -or $line.StartsWith('*** Update File: '))) {
            continue
        }

        $path = $line.Substring($line.IndexOf(':') + 1).Trim()
        if ($path.Contains(' -> ')) {
            $path = $path.Split(' -> ')[0].Trim()
        }

        if ($path.ToLowerInvariant().EndsWith('.go')) {
            $paths += $path
        }
    }

    return $paths
}

$rawInput = [Console]::In.ReadToEnd()
if ([string]::IsNullOrWhiteSpace($rawInput)) {
    Write-Json @{}
    exit 0
}

try {
    $payload = $rawInput | ConvertFrom-Json -Depth 20
}
catch {
    Write-Json @{}
    exit 0
}

$toolName = [string]$payload.tool_name
$editTools = @(
    'create_file',
    'replace_string_in_file',
    'multi_replace_string_in_file',
    'apply_patch',
    'editFiles'
)

if ($editTools -notcontains $toolName) {
    Write-Json @{}
    exit 0
}

$goPaths = @()
$toolInput = $payload.tool_input

if ($null -ne $toolInput) {
    $propertyNames = @($toolInput.PSObject.Properties.Name)

    if ($propertyNames -contains 'filePath') {
        Add-GoPath -Path ([string]$toolInput.filePath) -Paths ([ref]$goPaths)
    }

    if ($propertyNames -contains 'files') {
        foreach ($fileEntry in @($toolInput.files)) {
            if ($fileEntry -is [string]) {
                Add-GoPath -Path $fileEntry -Paths ([ref]$goPaths)
                continue
            }

            if ($null -ne $fileEntry -and ($fileEntry.PSObject.Properties.Name -contains 'filePath')) {
                Add-GoPath -Path ([string]$fileEntry.filePath) -Paths ([ref]$goPaths)
            }
        }
    }

    if ($propertyNames -contains 'input') {
        $goPaths += Get-GoPathsFromPatch -PatchText ([string]$toolInput.input)
    }
}

if ($goPaths.Count -eq 0) {
    Write-Json @{}
    exit 0
}

$workspace = [string]$payload.cwd
if ([string]::IsNullOrWhiteSpace($workspace)) {
    $workspace = (Get-Location).Path
}

Push-Location $workspace
try {
    $buildOutput = & go build ./... 2>&1 | Out-String
    $exitCode = $LASTEXITCODE
}
finally {
    Pop-Location
}

if ($exitCode -eq 0) {
    Write-Json @{}
    exit 0
}

$summary = $buildOutput.Trim()
if ($summary.Length -gt 4000) {
    $summary = $summary.Substring(0, 4000)
}

Write-Json @{
    decision           = 'block'
    reason             = 'Go type check failed after editing Go files.'
    hookSpecificOutput = @{
        hookEventName     = 'PostToolUse'
        additionalContext = "go build ./... failed after editing Go files:`n$summary"
    }
}
exit 0
