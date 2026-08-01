param(
    [string]$BinaryPath = "",
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA "Programs\claude-status"),
    [string]$MirrorHost = "pilab",
    [string]$RemoteBinary = "/home/pi/.local/bin/claude-status"
)

$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true
$RepoDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

if ([string]::IsNullOrWhiteSpace($BinaryPath)) {
    $BinaryPath = Join-Path $RepoDir "bin\claude-status.exe"
}
$BinaryPath = (Resolve-Path -LiteralPath $BinaryPath).Path
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$InstalledBinary = Join-Path $InstallDir "claude-status.exe"
Copy-Item -LiteralPath $BinaryPath -Destination $InstalledBinary -Force

$Timestamp = Get-Date -Format "yyyyMMdd-HHmmss"

function Backup-ConfigFile {
    param([string]$Path)
    if (Test-Path -LiteralPath $Path) {
        $BackupPath = "$Path.claude-status-backup-$Timestamp"
        Copy-Item -LiteralPath $Path -Destination $BackupPath
        return $BackupPath
    }
    return $null
}

function Write-Utf8Atomic {
    param(
        [string]$Path,
        [string]$Content
    )
    $Directory = Split-Path -Parent $Path
    New-Item -ItemType Directory -Force -Path $Directory | Out-Null
    $Temporary = Join-Path $Directory ("." + [IO.Path]::GetFileName($Path) + "." + [guid]::NewGuid().ToString("N") + ".tmp")
    try {
        [IO.File]::WriteAllText($Temporary, $Content, [Text.UTF8Encoding]::new($false))
        Move-Item -LiteralPath $Temporary -Destination $Path -Force
    }
    finally {
        if (Test-Path -LiteralPath $Temporary) {
            Remove-Item -LiteralPath $Temporary -Force
        }
    }
}

$ClaudeSettingsPath = Join-Path $env:USERPROFILE ".claude\settings.json"
$ClaudeBackup = Backup-ConfigFile -Path $ClaudeSettingsPath
if (Test-Path -LiteralPath $ClaudeSettingsPath) {
    $ClaudeSettings = Get-Content -Raw -LiteralPath $ClaudeSettingsPath | ConvertFrom-Json -AsHashtable
}
else {
    $ClaudeSettings = [ordered]@{}
}
$ClaudeCommand = '"' + $InstalledBinary + '" ingest --mirror-ssh ' + $MirrorHost + ' --remote-bin ' + $RemoteBinary
$ClaudeSettings["statusLine"] = [ordered]@{
    type = "command"
    command = $ClaudeCommand
    padding = 1
    refreshInterval = 5
}
$ClaudeJson = $ClaudeSettings | ConvertTo-Json -Depth 100
Write-Utf8Atomic -Path $ClaudeSettingsPath -Content ($ClaudeJson + [Environment]::NewLine)

$CodexConfigPath = Join-Path $env:USERPROFILE ".codex\config.toml"
if (-not (Test-Path -LiteralPath $CodexConfigPath)) {
    throw "Codex config was not found at $CodexConfigPath"
}
$CodexBackup = Backup-ConfigFile -Path $CodexConfigPath
$CodexConfig = Get-Content -Raw -LiteralPath $CodexConfigPath
$NotifyMatch = [regex]::Match($CodexConfig, '(?m)^\s*notify\s*=\s*(?<array>\[[^\r\n]*\])\s*$')
$ExistingNotify = @()
if ($NotifyMatch.Success) {
    $ExistingNotify = @($NotifyMatch.Groups["array"].Value | ConvertFrom-Json)
}

$ForwardProgram = ""
$ForwardArguments = @()
if ($ExistingNotify.Count -ge 2 -and $ExistingNotify[0] -eq $InstalledBinary -and $ExistingNotify[1] -eq "codex-notify") {
    for ($Index = 2; $Index -lt $ExistingNotify.Count; $Index++) {
        if ($ExistingNotify[$Index] -eq "--forward" -and $Index + 1 -lt $ExistingNotify.Count) {
            $ForwardProgram = $ExistingNotify[++$Index]
        }
        elseif ($ExistingNotify[$Index] -eq "--forward-arg" -and $Index + 1 -lt $ExistingNotify.Count) {
            $ForwardArguments += $ExistingNotify[++$Index]
        }
    }
}
elseif ($ExistingNotify.Count -gt 0) {
    $ForwardProgram = $ExistingNotify[0]
    if ($ExistingNotify.Count -gt 1) {
        $ForwardArguments = @($ExistingNotify[1..($ExistingNotify.Count - 1)])
    }
}

$NewNotify = @(
    $InstalledBinary,
    "codex-notify",
    "--mirror-ssh", $MirrorHost,
    "--remote-bin", $RemoteBinary
)
if (-not [string]::IsNullOrWhiteSpace($ForwardProgram)) {
    $NewNotify += @("--forward", $ForwardProgram)
    foreach ($ForwardArgument in $ForwardArguments) {
        $NewNotify += @("--forward-arg", [string]$ForwardArgument)
    }
}
$NotifyLine = "notify = " + ($NewNotify | ConvertTo-Json -Compress -AsArray)
if ($NotifyMatch.Success) {
    $CodexConfig = $CodexConfig.Remove($NotifyMatch.Index, $NotifyMatch.Length).Insert($NotifyMatch.Index, $NotifyLine)
}
else {
    $CodexConfig = $NotifyLine + [Environment]::NewLine + $CodexConfig
}
Write-Utf8Atomic -Path $CodexConfigPath -Content $CodexConfig

Write-Host "installed binary: $InstalledBinary"
Write-Host "configured Claude statusLine: $ClaudeSettingsPath"
Write-Host "configured Codex notify: $CodexConfigPath"
if ($ClaudeBackup) { Write-Host "Claude backup: $ClaudeBackup" }
if ($CodexBackup) { Write-Host "Codex backup: $CodexBackup" }
