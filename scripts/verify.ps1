param(
    [string]$TranscriptPath = ""
)

$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true
$OutputEncoding = [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
$RepoDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

if ([string]::IsNullOrWhiteSpace($TranscriptPath)) {
    $TranscriptPath = Join-Path $RepoDir "artifacts\verify-transcript.txt"
}

$TranscriptDir = Split-Path -Parent $TranscriptPath
New-Item -ItemType Directory -Force -Path $TranscriptDir | Out-Null
Start-Transcript -Path $TranscriptPath -Force

try {
    Set-Location $RepoDir

    Write-Host "[1/12] gofmt"
    $GoFiles = rg --files -g "*.go"
    if ($GoFiles) {
        gofmt -w $GoFiles
    }

    Write-Host "[2/12] PowerShell syntax"
    $PowerShellScripts = rg --files -g "*.ps1"
    foreach ($PowerShellScript in $PowerShellScripts) {
        $Tokens = $null
        $ParseErrors = $null
        [System.Management.Automation.Language.Parser]::ParseFile(
            (Join-Path $RepoDir $PowerShellScript),
            [ref]$Tokens,
            [ref]$ParseErrors
        ) | Out-Null
        if ($ParseErrors.Count -gt 0) {
            throw "PowerShell syntax error in $PowerShellScript`: $($ParseErrors[0].Message)"
        }
    }

    Write-Host "[3/12] go mod tidy"
    go mod tidy

    Write-Host "[4/12] repeated shuffled tests"
    go test -shuffle=on -count=3 ./...

    Write-Host "[5/12] coverage floor"
    $CoveragePath = Join-Path $RepoDir "artifacts\coverage.out"
    go test "-coverprofile=$CoveragePath" ./...
    $CoverageSummary = go tool cover "-func=$CoveragePath"
    $CoverageSummary | Write-Host
    $TotalCoverage = $CoverageSummary | Select-String '^total:' | Select-Object -Last 1
    if (-not $TotalCoverage -or $TotalCoverage.Line -notmatch '([0-9]+(?:\.[0-9]+)?)%') {
        throw "unable to parse total Go test coverage"
    }
    if ([double]$Matches[1] -lt 80) {
        throw "total Go test coverage $($Matches[1])% is below the 80% floor"
    }

    Write-Host "[6/12] go vet"
    go vet ./...

    Write-Host "[7/12] Windows build"
    New-Item -ItemType Directory -Force -Path (Join-Path $RepoDir "bin") | Out-Null
    go build -trimpath -o (Join-Path $RepoDir "bin\claude-status.exe") ./cmd/claude-status

    Write-Host "[8/12] Raspberry Pi 4 linux/arm64 build"
    $env:CGO_ENABLED = "0"
    $env:GOOS = "linux"
    $env:GOARCH = "arm64"
    go build -trimpath -o (Join-Path $RepoDir "bin\claude-status-linux-arm64") ./cmd/claude-status

    Write-Host "[9/12] Verify ARM64 build metadata"
    $ArmMetadata = go version -m (Join-Path $RepoDir "bin\claude-status-linux-arm64")
    if (($ArmMetadata -join "`n") -notmatch "GOOS=linux" -or ($ArmMetadata -join "`n") -notmatch "GOARCH=arm64") {
        throw "cross-built binary is not linux/arm64"
    }

    Write-Host "[10/12] Binary ingest smoke test"
    $SmokeDir = Join-Path $RepoDir ("artifacts\smoke-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Force -Path $SmokeDir | Out-Null
    try {
        $StatusLine = Get-Content -Raw (Join-Path $RepoDir "examples\statusline-input.json") |
            & (Join-Path $RepoDir "bin\claude-status.exe") ingest --state-dir $SmokeDir
        Write-Host $StatusLine
        if ($StatusLine -notmatch "\[Opus\] 5h 51% .* 7d 34% .* ctx 72%") {
            throw "unexpected status line: $StatusLine"
        }
        $SnapshotPath = Join-Path $SmokeDir "latest.json"
        if (-not (Test-Path -LiteralPath $SnapshotPath)) {
            throw "ingest did not create latest.json"
        }
        $RawSnapshot = Get-Content -Raw -LiteralPath $SnapshotPath
        $Snapshot = $RawSnapshot | ConvertFrom-Json
        if ($Snapshot.session.id -ne "demo-session-abc123") {
            throw "unexpected persisted session ID"
        }
        if ($RawSnapshot -match "transcript_path|this/field/is/intentionally/not/persisted") {
            throw "sanitized snapshot contains transcript data"
        }
    }
    finally {
        if (Test-Path -LiteralPath $SmokeDir) {
            Remove-Item -LiteralPath $SmokeDir -Recurse -Force
        }
    }

    Write-Host "[11/12] Linux package smoke test"
    $GitCommand = Get-Command git
    $GitRoot = (Resolve-Path (Join-Path (Split-Path -Parent $GitCommand.Source) "..")).Path
    $GitBash = Join-Path $GitRoot "bin\bash.exe"
    if (-not (Test-Path -LiteralPath $GitBash)) {
        throw "Git Bash not found at $GitBash"
    }
    & $GitBash -n scripts/install.sh scripts/uninstall.sh scripts/package.sh
    & $GitBash scripts/package.sh v0.0.0-verify
    $Packages = Get-ChildItem -LiteralPath (Join-Path $RepoDir "dist") -Filter "claude-status_v0.0.0-verify_linux_*.tar.gz"
    if ($Packages.Count -ne 2) {
        throw "expected 2 Linux packages, found $($Packages.Count)"
    }
    $Checksums = Get-Content -LiteralPath (Join-Path $RepoDir "dist\SHA256SUMS")
    if (($Checksums | Where-Object { $_ -match "claude-status_v0.0.0-verify_linux_" }).Count -ne 2) {
        throw "SHA256SUMS does not contain both verification packages"
    }

    Write-Host "[12/12] SHA256 verification"
    foreach ($Checksum in $Checksums) {
        if ($Checksum -notmatch '^([0-9a-fA-F]{64})\s+\*?\.?/?(.+)$') {
            throw "invalid checksum line: $Checksum"
        }
        $ExpectedHash = $Matches[1]
        $PackageName = $Matches[2]
        $ActualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $RepoDir "dist\$PackageName")).Hash
        if ($ActualHash -ne $ExpectedHash) {
            throw "SHA256 mismatch for $PackageName"
        }
        Write-Host "SHA256_OK $PackageName"
    }

    Write-Host "VERIFY_OK"
}
finally {
    Stop-Transcript
}
