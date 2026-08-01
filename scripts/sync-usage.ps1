<#
.SYNOPSIS
  Manually pushes real Claude usage percentages (read off the Account &
  Usage panel) into claude-status, mirroring to pilab — a stopgap for the
  VS Code extension interface, where Claude Code's statusLine command is
  never invoked, so claude-status never sees real numbers on its own.

.EXAMPLE
  pwsh -File scripts/sync-usage.ps1 -FiveHourPercent 63 -SevenDayPercent 42
#>
param(
    [Parameter(Mandatory = $true)][double]$FiveHourPercent,
    [Parameter(Mandatory = $true)][double]$SevenDayPercent,
    [double]$FiveHourResetHours = 5,
    [double]$SevenDayResetHours = 168,
    [string]$SessionId = "",
    [string]$MirrorHost = "pilab",
    [string]$RemoteBinary = "/home/pi/.local/bin/claude-status",
    [string]$BinaryPath = (Join-Path $env:LOCALAPPDATA "Programs\claude-status\claude-status.exe")
)

$ErrorActionPreference = "Stop"

$latestPath = Join-Path $env:LOCALAPPDATA "claude-status\latest.json"
$existing = $null
if (Test-Path -LiteralPath $latestPath) {
    $existing = Get-Content -Raw -LiteralPath $latestPath | ConvertFrom-Json
}

if ([string]::IsNullOrWhiteSpace($SessionId)) {
    if (-not $existing) {
        throw "No existing snapshot to read a session id from; pass -SessionId explicitly."
    }
    $SessionId = $existing.session.id
}

# Get-Date -Date "...Z" silently re-interprets the parsed instant as local
# time (Kind=Local) instead of keeping it UTC, which threw reset times off
# by exactly this machine's UTC offset. DateTimeOffset.UtcNow avoids that.
$nowEpoch = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$fiveResetEpoch = $nowEpoch + [int64]($FiveHourResetHours * 3600)
$sevenResetEpoch = $nowEpoch + [int64]($SevenDayResetHours * 3600)

$payload = [ordered]@{
    session_id = $SessionId
    rate_limits = [ordered]@{
        five_hour = [ordered]@{ used_percentage = $FiveHourPercent; resets_at = $fiveResetEpoch }
        seven_day = [ordered]@{ used_percentage = $SevenDayPercent; resets_at = $sevenResetEpoch }
    }
}
# Carry the model forward — this snapshot only carries rate limits, and
# ingest would otherwise blank out whatever model was last recorded.
if ($existing -and $existing.model -and $existing.model.id) {
    $payload.model = [ordered]@{ id = $existing.model.id; display_name = $existing.model.display_name }
}

$json = $payload | ConvertTo-Json -Depth 6
Write-Output "$json" | & $BinaryPath ingest --mirror-ssh $MirrorHost --remote-bin $RemoteBinary
