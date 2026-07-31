param(
    [string]$TranscriptPath = ""
)

$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true
$RepoDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

if ([string]::IsNullOrWhiteSpace($TranscriptPath)) {
    $TranscriptPath = Join-Path $RepoDir "artifacts\verify-transcript.txt"
}

$TranscriptDir = Split-Path -Parent $TranscriptPath
New-Item -ItemType Directory -Force -Path $TranscriptDir | Out-Null
Start-Transcript -Path $TranscriptPath -Force

try {
    Set-Location $RepoDir

    Write-Host "[1/9] gofmt"
    $GoFiles = rg --files -g "*.go"
    if ($GoFiles) {
        gofmt -w $GoFiles
    }

    Write-Host "[2/9] go mod tidy"
    go mod tidy

    Write-Host "[3/9] go test"
    go test ./...

    Write-Host "[4/9] go vet"
    go vet ./...

    Write-Host "[5/9] Windows build"
    New-Item -ItemType Directory -Force -Path (Join-Path $RepoDir "bin") | Out-Null
    go build -trimpath -o (Join-Path $RepoDir "bin\claude-status.exe") ./cmd/claude-status

    Write-Host "[6/9] Raspberry Pi 4 linux/arm64 build"
    $env:CGO_ENABLED = "0"
    $env:GOOS = "linux"
    $env:GOARCH = "arm64"
    go build -trimpath -o (Join-Path $RepoDir "bin\claude-status-linux-arm64") ./cmd/claude-status

    Write-Host "[7/9] Verify ARM64 build metadata"
    $ArmMetadata = go version -m (Join-Path $RepoDir "bin\claude-status-linux-arm64")
    if (($ArmMetadata -join "`n") -notmatch "GOOS=linux" -or ($ArmMetadata -join "`n") -notmatch "GOARCH=arm64") {
        throw "cross-built binary is not linux/arm64"
    }

    Write-Host "[8/9] Binary ingest smoke test"
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

    Write-Host "[9/9] Linux package and checksum smoke test"
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

    Write-Host "VERIFY_OK"
}
finally {
    Stop-Transcript
}
