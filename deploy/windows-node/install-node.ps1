#Requires -RunAsAdministrator

param(
    [string]$PackageRoot = $(Split-Path -Parent $MyInvocation.MyCommand.Path),
    [string]$InstallDir = "$env:ProgramFiles\TinyClawNode",
    [string]$ConfigPath = "$env:ProgramData\TinyClawNode\config.json"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Ensure-Directory {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Force -Path $Path | Out-Null
    }
}

function New-Shortcut {
    param(
        [string]$Path,
        [string]$TargetPath,
        [string]$Arguments,
        [string]$WorkingDirectory,
        [string]$IconLocation
    )

    $shell = New-Object -ComObject WScript.Shell
    $shortcut = $shell.CreateShortcut($Path)
    $shortcut.TargetPath = $TargetPath
    if (-not [string]::IsNullOrWhiteSpace($Arguments)) {
        $shortcut.Arguments = $Arguments
    }
    if (-not [string]::IsNullOrWhiteSpace($WorkingDirectory)) {
        $shortcut.WorkingDirectory = $WorkingDirectory
    }
    if (-not [string]::IsNullOrWhiteSpace($IconLocation)) {
        $shortcut.IconLocation = $IconLocation
    }
    $shortcut.Save()
}

function Get-DetectedWSLDistros {
    $command = Get-Command "wsl.exe" -ErrorAction SilentlyContinue
    if ($null -eq $command) {
        return @()
    }

    $items = @()
    foreach ($line in (& $command.Source -l -q 2>$null)) {
        $text = ([string]$line).Trim()
        if (-not [string]::IsNullOrWhiteSpace($text)) {
            $items += $text
        }
    }
    return $items
}

function Initialize-Config {
    param(
        [string]$TemplatePath,
        [string]$TargetPath
    )

    if (Test-Path -LiteralPath $TargetPath) {
        return
    }

    $config = Get-Content -LiteralPath $TemplatePath -Raw -Encoding UTF8 | ConvertFrom-Json
    $machineName = [Environment]::MachineName
    $config.workspace_id = "default"
    $config.device_id = $machineName
    $config.node_name = $machineName
    $config.log_dir = "$env:ProgramData\TinyClawNode\logs"
    $config.wsl_distros = @()

    foreach ($distro in Get-DetectedWSLDistros) {
        $config.wsl_distros += [ordered]@{
            name                      = $distro
            enabled                   = $false
            allow_command_prefixes    = @()
            allow_write_path_prefixes = @()
            default_cwd               = ""
        }
    }

    Ensure-Directory (Split-Path -Parent $TargetPath)
    $json = $config | ConvertTo-Json -Depth 8
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($TargetPath, $json, $utf8NoBom)
}

function Update-StartupShortcutIfConfigured {
    param(
        [string]$CurrentConfigPath,
        [string]$LaunchVbsPath,
        [string]$NodeExePath
    )

    if (-not (Test-Path -LiteralPath $CurrentConfigPath)) {
        return
    }

    $config = Get-Content -LiteralPath $CurrentConfigPath -Raw -Encoding UTF8 | ConvertFrom-Json
    $enabled = $false
    if ($null -ne $config.PSObject.Properties["start_at_login"]) {
        $enabled = [bool]$config.start_at_login
    }

    $startupShortcut = Join-Path ([Environment]::GetFolderPath("Startup")) "TinyClaw Node.lnk"
    if (-not $enabled) {
        if (Test-Path -LiteralPath $startupShortcut) {
            Remove-Item -LiteralPath $startupShortcut -Force
        }
        return
    }

    $wscriptPath = (Get-Command "wscript.exe" -ErrorAction Stop).Source
    $arguments = '"' + $LaunchVbsPath + '" "' + $CurrentConfigPath + '"'
    New-Shortcut -Path $startupShortcut -TargetPath $wscriptPath -Arguments $arguments -WorkingDirectory $InstallDir -IconLocation $NodeExePath
}

$expectedFiles = @(
    "tinyclaw-node.exe",
    "configure-node.cmd",
    "configure-node.ps1",
    "configure-node.vbs",
    "install-node.cmd",
    "install-node.ps1",
    "launch-node.vbs",
    "config.template.json",
    "README.md"
)

foreach ($file in $expectedFiles) {
    $sourcePath = Join-Path $PackageRoot $file
    if (-not (Test-Path -LiteralPath $sourcePath)) {
        throw "Missing package file: $sourcePath"
    }
}

Ensure-Directory $InstallDir
Ensure-Directory (Split-Path -Parent $ConfigPath)
Ensure-Directory "$env:ProgramData\TinyClawNode\logs"

foreach ($file in $expectedFiles) {
    Copy-Item -LiteralPath (Join-Path $PackageRoot $file) -Destination (Join-Path $InstallDir $file) -Force
}

$templatePath = Join-Path $InstallDir "config.template.json"
$configureCmdPath = Join-Path $InstallDir "configure-node.cmd"
$configureVbsPath = Join-Path $InstallDir "configure-node.vbs"
$launchVbsPath = Join-Path $InstallDir "launch-node.vbs"
$nodeExePath = Join-Path $InstallDir "tinyclaw-node.exe"

Initialize-Config -TemplatePath $templatePath -TargetPath $ConfigPath
Update-StartupShortcutIfConfigured -CurrentConfigPath $ConfigPath -LaunchVbsPath $launchVbsPath -NodeExePath $nodeExePath

$desktopDir = [Environment]::GetFolderPath("Desktop")
$settingsShortcut = Join-Path $desktopDir "TinyClaw Node Settings.lnk"
$launchShortcut = Join-Path $desktopDir "TinyClaw Node.lnk"

$wscriptPath = (Get-Command "wscript.exe" -ErrorAction Stop).Source

New-Shortcut -Path $launchShortcut -TargetPath $wscriptPath -Arguments ('"' + $launchVbsPath + '" "' + $ConfigPath + '"') -WorkingDirectory $InstallDir -IconLocation $nodeExePath
New-Shortcut -Path $settingsShortcut -TargetPath $wscriptPath -Arguments ('"' + $configureVbsPath + '" "' + $ConfigPath + '"') -WorkingDirectory $InstallDir -IconLocation $nodeExePath

Write-Host "TinyClaw Node installed to $InstallDir"
Write-Host "Configuration file: $ConfigPath"
Write-Host "Tip: next time you can just double-click install-node.cmd or TinyClaw Node Settings."

Start-Process -FilePath $wscriptPath -ArgumentList @($configureVbsPath, $ConfigPath) -WorkingDirectory $InstallDir | Out-Null
