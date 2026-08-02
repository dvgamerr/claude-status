param(
    [string]$BinaryPath = "",
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA "Programs\claude-status"),
    [string]$MirrorHost = "pilab",
    [string]$RemoteBinary = "/home/pi/.local/bin/claude-status"
)

$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true
$RepoDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path

# The relay itself now runs as a real Windows Service (`claude-status
# service install`, internal/service/manager_windows.go) instead of a
# Scheduled Task — the same command also works unchanged on Linux
# (systemd --user) and macOS (launchd). `service install` stops and
# restarts an already-running instance on its own when re-run, so there's
# no separate "stop before replacing the binary" step needed here anymore.

if ([string]::IsNullOrWhiteSpace($BinaryPath)) {
    $BinaryPath = Join-Path $RepoDir "bin\claude-status.exe"
}
$BinaryPath = (Resolve-Path -LiteralPath $BinaryPath).Path
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$InstalledBinary = Join-Path $InstallDir "claude-status.exe"
$StagedBinary = Join-Path $InstallDir ("claude-status." + [guid]::NewGuid().ToString("N") + ".exe")
Copy-Item -LiteralPath $BinaryPath -Destination $StagedBinary
try {
    $BinaryInstalled = $false
    for ($Attempt = 0; $Attempt -lt 100; $Attempt++) {
        try {
            Move-Item -LiteralPath $StagedBinary -Destination $InstalledBinary -Force
            $BinaryInstalled = $true
            break
        }
        catch {
            if ($Attempt -eq 99) {
                throw
            }
            Start-Sleep -Milliseconds 100
        }
    }
    if (-not $BinaryInstalled) {
        throw "failed to install $InstalledBinary"
    }
}
finally {
    if (Test-Path -LiteralPath $StagedBinary) {
        Remove-Item -LiteralPath $StagedBinary -Force
    }
}

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
$ClaudeCommand = '"' + $InstalledBinary + '" ingest'
$ClaudeSettings["statusLine"] = [ordered]@{
    type = "command"
    command = $ClaudeCommand
    padding = 1
    refreshInterval = 5
}

# The activity command turns hook events into the dashboard's working / idle /
# needs-approval mascot state. It is registered on four hooks and is additive:
# any other hook already configured on the same event is left in place.
$ActivityCommand = '"' + $InstalledBinary + '" activity'

function Set-ClaudeStatusHook {
    param($Settings, [string]$EventName, [string]$Command, [string]$Matcher)

    if (-not $Settings.Contains("hooks")) {
        $Settings["hooks"] = [ordered]@{}
    }
    $Hooks = $Settings["hooks"]
    $ExistingGroups = @()
    if ($Hooks.Contains($EventName)) {
        $ExistingGroups = @($Hooks[$EventName])
    }

    # Drop any group this script previously installed for this event, then
    # re-add it fresh; leave every other tool's hook group untouched.
    $KeptGroups = @($ExistingGroups | Where-Object {
        $GroupHooks = @($_.hooks)
        -not ($GroupHooks | Where-Object { $_.command -like '*claude-status*" activity*' })
    })

    $NewGroup = [ordered]@{
        hooks = @([ordered]@{ type = "command"; command = $Command })
    }
    if (-not [string]::IsNullOrWhiteSpace($Matcher)) {
        $NewGroup = [ordered]@{ matcher = $Matcher; hooks = $NewGroup.hooks }
    }

    $Hooks[$EventName] = @($KeptGroups) + @($NewGroup)
    $Settings["hooks"] = $Hooks
}

Set-ClaudeStatusHook -Settings $ClaudeSettings -EventName "UserPromptSubmit" -Command $ActivityCommand
Set-ClaudeStatusHook -Settings $ClaudeSettings -EventName "PreToolUse" -Command $ActivityCommand -Matcher "*"
Set-ClaudeStatusHook -Settings $ClaudeSettings -EventName "Stop" -Command $ActivityCommand
Set-ClaudeStatusHook -Settings $ClaudeSettings -EventName "Notification" -Command $ActivityCommand

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
    "codex-notify"
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

# Registering a Windows Service requires an elevated (Administrator) shell,
# unlike the old Scheduled Task registration this replaces.
& $InstalledBinary service install --mirror-ssh $MirrorHost --remote-bin $RemoteBinary --refresh 1s

Write-Host "installed binary: $InstalledBinary"
Write-Host "configured Claude statusLine: $ClaudeSettingsPath"
Write-Host "configured Codex notify: $CodexConfigPath"
if ($ClaudeBackup) { Write-Host "Claude backup: $ClaudeBackup" }
if ($CodexBackup) { Write-Host "Codex backup: $CodexBackup" }
