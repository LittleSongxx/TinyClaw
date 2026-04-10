param(
    [string]$ConfigPath = "$env:ProgramData\TinyClawNode\config.json"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

function Get-DefaultLogDir {
    $programData = $env:ProgramData
    if ([string]::IsNullOrWhiteSpace($programData)) {
        $programData = "C:\ProgramData"
    }
    return Join-Path $programData "TinyClawNode\logs"
}

function Get-NodeInstallDir {
    return $PSScriptRoot
}

function Get-PropertyValue {
    param(
        [object]$Object,
        [string]$Name,
        $DefaultValue
    )

    if ($null -eq $Object) {
        return $DefaultValue
    }

    if ($Object -is [System.Collections.IDictionary]) {
        if ($Object.Contains($Name)) {
            $value = $Object[$Name]
            if ($null -ne $value) {
                return $value
            }
        }
        return $DefaultValue
    }

    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property) {
        return $DefaultValue
    }

    $value = $property.Value
    if ($null -eq $value) {
        return $DefaultValue
    }

    return $value
}

function To-StringArray {
    param([object]$Value)

    $items = @()
    foreach ($current in @($Value)) {
        $text = [string]$current
        if (-not [string]::IsNullOrWhiteSpace($text)) {
            $items += $text.Trim()
        }
    }
    return $items
}

function New-DefaultConfig {
    $machineName = [Environment]::MachineName
    return [ordered]@{
        gateway_ws          = "ws://127.0.0.1:36060/gateway/nodes/ws"
        workspace_id        = "default"
        device_id           = $machineName
        device_token        = ""
        private_key         = ""
        public_key          = ""
        pairing_code        = ""
        node_name           = $machineName
        log_dir             = Get-DefaultLogDir
        start_at_login      = $false
        enable_windows_node = $true
        wsl_distros         = @()
    }
}

function Get-WSLConfigMap {
    param([object[]]$Items)

    $map = @{}
    foreach ($item in @($Items)) {
        $name = [string](Get-PropertyValue $item "name" "")
        if ([string]::IsNullOrWhiteSpace($name)) {
            continue
        }
        $map[$name.Trim().ToLowerInvariant()] = [ordered]@{
            name                      = $name.Trim()
            enabled                   = [bool](Get-PropertyValue $item "enabled" $false)
            allow_command_prefixes    = @(To-StringArray (Get-PropertyValue $item "allow_command_prefixes" @()))
            allow_write_path_prefixes = @(To-StringArray (Get-PropertyValue $item "allow_write_path_prefixes" @()))
            default_cwd               = ([string](Get-PropertyValue $item "default_cwd" "")).Trim()
        }
    }
    return $map
}

function Merge-WSLDistroConfigs {
    param(
        [object[]]$ExistingItems,
        [string[]]$DetectedDistros
    )

    $map = Get-WSLConfigMap $ExistingItems
    foreach ($distro in @($DetectedDistros)) {
        $name = ([string]$distro).Trim()
        if ([string]::IsNullOrWhiteSpace($name)) {
            continue
        }
        $key = $name.ToLowerInvariant()
        if (-not $map.ContainsKey($key)) {
            $map[$key] = [ordered]@{
                name                      = $name
                enabled                   = $false
                allow_command_prefixes    = @()
                allow_write_path_prefixes = @()
                default_cwd               = ""
            }
        }
    }

    $result = @()
    foreach ($key in ($map.Keys | Sort-Object)) {
        $result += $map[$key]
    }
    return $result
}

function Get-DetectedWSLDistros {
    $command = Get-Command "wsl.exe" -ErrorAction SilentlyContinue
    if ($null -eq $command) {
        return @()
    }

    $items = @()
    try {
        foreach ($line in (& $command.Source -l -q 2>$null)) {
            $text = ([string]$line).Trim()
            if (-not [string]::IsNullOrWhiteSpace($text)) {
                $items += $text
            }
        }
    } catch {
        return @()
    }
    return $items
}

function Read-Config {
    param([string]$Path)

    $config = New-DefaultConfig
    if (-not (Test-Path -LiteralPath $Path)) {
        $config.wsl_distros = Merge-WSLDistroConfigs @() (Get-DetectedWSLDistros)
        return $config
    }

    $raw = Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json
    $config.gateway_ws = ([string](Get-PropertyValue $raw "gateway_ws" $config.gateway_ws)).Trim()
    $config.workspace_id = ([string](Get-PropertyValue $raw "workspace_id" $config.workspace_id)).Trim()
    $config.device_token = ([string](Get-PropertyValue $raw "device_token" $config.device_token)).Trim()
    $config.private_key = ([string](Get-PropertyValue $raw "private_key" $config.private_key)).Trim()
    $config.public_key = ([string](Get-PropertyValue $raw "public_key" $config.public_key)).Trim()
    $config.pairing_code = ([string](Get-PropertyValue $raw "pairing_code" $config.pairing_code)).Trim()
    $config.device_id = ([string](Get-PropertyValue $raw "device_id" (Get-PropertyValue $raw "node_id" $config.device_id))).Trim()
    $config.node_name = ([string](Get-PropertyValue $raw "node_name" $config.node_name)).Trim()
    $config.log_dir = ([string](Get-PropertyValue $raw "log_dir" $config.log_dir)).Trim()
    $config.start_at_login = [bool](Get-PropertyValue $raw "start_at_login" $config.start_at_login)
    $config.enable_windows_node = [bool](Get-PropertyValue $raw "enable_windows_node" $config.enable_windows_node)
    $config.wsl_distros = Merge-WSLDistroConfigs (Get-PropertyValue $raw "wsl_distros" @()) (Get-DetectedWSLDistros)

    if ([string]::IsNullOrWhiteSpace($config.device_id)) {
        $config.device_id = [Environment]::MachineName
    }
    if ([string]::IsNullOrWhiteSpace($config.node_name)) {
        $config.node_name = $config.device_id
    }
    if ([string]::IsNullOrWhiteSpace($config.log_dir)) {
        $config.log_dir = Get-DefaultLogDir
    }

    return $config
}

function Ensure-Directory {
    param([string]$Path)

    if ([string]::IsNullOrWhiteSpace($Path)) {
        return
    }
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Force -Path $Path | Out-Null
    }
}

function Write-NodeConfig {
    param(
        [System.Collections.IDictionary]$Config,
        [string]$Path
    )

    Ensure-Directory (Split-Path -Parent $Path)
    Ensure-Directory ([string](Get-PropertyValue $Config "log_dir" ""))
    $json = $Config | ConvertTo-Json -Depth 8
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($Path, $json, $utf8NoBom)
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

function Get-StartupShortcutPath {
    $startupDir = [Environment]::GetFolderPath("Startup")
    return Join-Path $startupDir "TinyClaw Node.lnk"
}

function Update-StartupShortcut {
    param(
        [bool]$Enabled,
        [string]$LaunchVbsPath
    )

    $shortcutPath = Get-StartupShortcutPath
    if (-not $Enabled) {
        if (Test-Path -LiteralPath $shortcutPath) {
            Remove-Item -LiteralPath $shortcutPath -Force
        }
        return
    }

    $wscriptPath = (Get-Command "wscript.exe" -ErrorAction Stop).Source
    $arguments = '"' + $LaunchVbsPath + '" "' + $ConfigPath + '"'
    New-Shortcut -Path $shortcutPath -TargetPath $wscriptPath -Arguments $arguments -WorkingDirectory (Split-Path -Parent $LaunchVbsPath) -IconLocation $LaunchVbsPath
}

function Start-NodeHidden {
    param(
        [string]$LaunchVbsPath,
        [string]$NodeExePath
    )

    if (Test-Path -LiteralPath $NodeExePath) {
        Start-Process -WindowStyle Hidden -FilePath $NodeExePath -ArgumentList @("--config", $ConfigPath) | Out-Null
        return
    }

    if (Test-Path -LiteralPath $LaunchVbsPath) {
        $wscriptPath = (Get-Command "wscript.exe" -ErrorAction Stop).Source
        Start-Process -FilePath $wscriptPath -ArgumentList @($LaunchVbsPath, $ConfigPath) | Out-Null
    }
}

function Populate-WSLGrid {
    param(
        [System.Windows.Forms.DataGridView]$Grid,
        [object[]]$Items,
        [string[]]$DetectedDistros
    )

    $detectedMap = @{}
    foreach ($distro in @($DetectedDistros)) {
        $name = ([string]$distro).Trim()
        if (-not [string]::IsNullOrWhiteSpace($name)) {
            $detectedMap[$name.ToLowerInvariant()] = $true
        }
    }

    $Grid.Rows.Clear()
    foreach ($item in @($Items)) {
        $name = ([string](Get-PropertyValue $item "name" "")).Trim()
        if ([string]::IsNullOrWhiteSpace($name)) {
            continue
        }
        $detected = $detectedMap.ContainsKey($name.ToLowerInvariant())
        $rowIndex = $Grid.Rows.Add()
        $row = $Grid.Rows[$rowIndex]
        $row.Cells["Enabled"].Value = [bool](Get-PropertyValue $item "enabled" $false)
        $row.Cells["Distro"].Value = $name
        $row.Cells["DefaultCWD"].Value = ([string](Get-PropertyValue $item "default_cwd" "")).Trim()
        $row.Cells["Status"].Value = $(if ($detected) { "Detected" } else { "Configured only" })
    }
}

$script:InstallDir = Get-NodeInstallDir
$script:NodeExePath = Join-Path $script:InstallDir "tinyclaw-node.exe"
$script:LaunchVbsPath = Join-Path $script:InstallDir "launch-node.vbs"
$script:Config = Read-Config $ConfigPath
$script:DetectedDistros = Get-DetectedWSLDistros
$script:KnownWSLByName = Get-WSLConfigMap $script:Config.wsl_distros

$form = New-Object System.Windows.Forms.Form
$form.Text = "TinyClaw Node Settings"
$form.StartPosition = "CenterScreen"
$form.Size = New-Object System.Drawing.Size(940, 780)
$form.MinimumSize = New-Object System.Drawing.Size(940, 780)

$titleLabel = New-Object System.Windows.Forms.Label
$titleLabel.Location = New-Object System.Drawing.Point(20, 15)
$titleLabel.Size = New-Object System.Drawing.Size(860, 24)
$titleLabel.Font = New-Object System.Drawing.Font("Segoe UI", 13, [System.Drawing.FontStyle]::Bold)
$titleLabel.Text = "TinyClaw Node"
$form.Controls.Add($titleLabel)

$subtitleLabel = New-Object System.Windows.Forms.Label
$subtitleLabel.Location = New-Object System.Drawing.Point(20, 42)
$subtitleLabel.Size = New-Object System.Drawing.Size(880, 36)
$subtitleLabel.Text = "Configure the Windows desktop node and any WSL virtual nodes that should be exposed to TinyClaw."
$form.Controls.Add($subtitleLabel)

$gatewayLabel = New-Object System.Windows.Forms.Label
$gatewayLabel.Location = New-Object System.Drawing.Point(20, 90)
$gatewayLabel.Size = New-Object System.Drawing.Size(130, 20)
$gatewayLabel.Text = "Gateway WS"
$form.Controls.Add($gatewayLabel)

$gatewayTextBox = New-Object System.Windows.Forms.TextBox
$gatewayTextBox.Location = New-Object System.Drawing.Point(160, 88)
$gatewayTextBox.Size = New-Object System.Drawing.Size(720, 24)
$gatewayTextBox.Text = [string]$script:Config.gateway_ws
$form.Controls.Add($gatewayTextBox)

$workspaceLabel = New-Object System.Windows.Forms.Label
$workspaceLabel.Location = New-Object System.Drawing.Point(20, 123)
$workspaceLabel.Size = New-Object System.Drawing.Size(130, 20)
$workspaceLabel.Text = "Workspace ID"
$form.Controls.Add($workspaceLabel)

$workspaceTextBox = New-Object System.Windows.Forms.TextBox
$workspaceTextBox.Location = New-Object System.Drawing.Point(160, 121)
$workspaceTextBox.Size = New-Object System.Drawing.Size(320, 24)
$workspaceTextBox.Text = [string]$script:Config.workspace_id
$form.Controls.Add($workspaceTextBox)

$idLabel = New-Object System.Windows.Forms.Label
$idLabel.Location = New-Object System.Drawing.Point(500, 123)
$idLabel.Size = New-Object System.Drawing.Size(80, 20)
$idLabel.Text = "Device ID"
$form.Controls.Add($idLabel)

$idTextBox = New-Object System.Windows.Forms.TextBox
$idTextBox.Location = New-Object System.Drawing.Point(580, 121)
$idTextBox.Size = New-Object System.Drawing.Size(300, 24)
$idTextBox.Text = [string]$script:Config.device_id
$form.Controls.Add($idTextBox)

$tokenLabel = New-Object System.Windows.Forms.Label
$tokenLabel.Location = New-Object System.Drawing.Point(20, 156)
$tokenLabel.Size = New-Object System.Drawing.Size(130, 20)
$tokenLabel.Text = "Device Token"
$form.Controls.Add($tokenLabel)

$tokenTextBox = New-Object System.Windows.Forms.TextBox
$tokenTextBox.Location = New-Object System.Drawing.Point(160, 154)
$tokenTextBox.Size = New-Object System.Drawing.Size(720, 24)
$tokenTextBox.UseSystemPasswordChar = $true
$tokenTextBox.Text = [string]$script:Config.device_token
$form.Controls.Add($tokenTextBox)

$pairingLabel = New-Object System.Windows.Forms.Label
$pairingLabel.Location = New-Object System.Drawing.Point(20, 189)
$pairingLabel.Size = New-Object System.Drawing.Size(130, 20)
$pairingLabel.Text = "Pairing Code"
$form.Controls.Add($pairingLabel)

$pairingTextBox = New-Object System.Windows.Forms.TextBox
$pairingTextBox.Location = New-Object System.Drawing.Point(160, 187)
$pairingTextBox.Size = New-Object System.Drawing.Size(720, 24)
$pairingTextBox.UseSystemPasswordChar = $true
$pairingTextBox.Text = [string]$script:Config.pairing_code
$form.Controls.Add($pairingTextBox)

$nameLabel = New-Object System.Windows.Forms.Label
$nameLabel.Location = New-Object System.Drawing.Point(20, 222)
$nameLabel.Size = New-Object System.Drawing.Size(130, 20)
$nameLabel.Text = "Node Name"
$form.Controls.Add($nameLabel)

$nameTextBox = New-Object System.Windows.Forms.TextBox
$nameTextBox.Location = New-Object System.Drawing.Point(160, 220)
$nameTextBox.Size = New-Object System.Drawing.Size(720, 24)
$nameTextBox.Text = [string]$script:Config.node_name
$form.Controls.Add($nameTextBox)

$windowsNodeCheckbox = New-Object System.Windows.Forms.CheckBox
$windowsNodeCheckbox.Location = New-Object System.Drawing.Point(160, 252)
$windowsNodeCheckbox.Size = New-Object System.Drawing.Size(240, 24)
$windowsNodeCheckbox.Text = "Enable Windows desktop node"
$windowsNodeCheckbox.Checked = [bool]$script:Config.enable_windows_node
$form.Controls.Add($windowsNodeCheckbox)

$startupCheckbox = New-Object System.Windows.Forms.CheckBox
$startupCheckbox.Location = New-Object System.Drawing.Point(410, 252)
$startupCheckbox.Size = New-Object System.Drawing.Size(220, 24)
$startupCheckbox.Text = "Start at Windows login"
$startupCheckbox.Checked = [bool]$script:Config.start_at_login
$form.Controls.Add($startupCheckbox)

$logLabel = New-Object System.Windows.Forms.Label
$logLabel.Location = New-Object System.Drawing.Point(20, 284)
$logLabel.Size = New-Object System.Drawing.Size(130, 20)
$logLabel.Text = "Log Directory"
$form.Controls.Add($logLabel)

$logTextBox = New-Object System.Windows.Forms.TextBox
$logTextBox.Location = New-Object System.Drawing.Point(160, 282)
$logTextBox.Size = New-Object System.Drawing.Size(720, 24)
$logTextBox.Text = [string]$script:Config.log_dir
$form.Controls.Add($logTextBox)

$wslLabel = New-Object System.Windows.Forms.Label
$wslLabel.Location = New-Object System.Drawing.Point(20, 321)
$wslLabel.Size = New-Object System.Drawing.Size(300, 20)
$wslLabel.Font = New-Object System.Drawing.Font("Segoe UI", 10, [System.Drawing.FontStyle]::Bold)
$wslLabel.Text = "WSL Virtual Nodes"
$form.Controls.Add($wslLabel)

$wslHintLabel = New-Object System.Windows.Forms.Label
$wslHintLabel.Location = New-Object System.Drawing.Point(20, 345)
$wslHintLabel.Size = New-Object System.Drawing.Size(860, 34)
$wslHintLabel.Text = "Each enabled distro becomes a separate virtual node. Set the default Linux working directory for commands when you want the agent to start in a specific repo."
$form.Controls.Add($wslHintLabel)

$wslGrid = New-Object System.Windows.Forms.DataGridView
$wslGrid.Location = New-Object System.Drawing.Point(20, 382)
$wslGrid.Size = New-Object System.Drawing.Size(860, 250)
$wslGrid.AllowUserToAddRows = $false
$wslGrid.AllowUserToDeleteRows = $false
$wslGrid.RowHeadersVisible = $false
$wslGrid.AutoSizeColumnsMode = "Fill"
$wslGrid.SelectionMode = "CellSelect"
$wslGrid.MultiSelect = $false

$enabledColumn = New-Object System.Windows.Forms.DataGridViewCheckBoxColumn
$enabledColumn.Name = "Enabled"
$enabledColumn.HeaderText = "Enabled"
$enabledColumn.FillWeight = 18
[void]$wslGrid.Columns.Add($enabledColumn)

$distroColumn = New-Object System.Windows.Forms.DataGridViewTextBoxColumn
$distroColumn.Name = "Distro"
$distroColumn.HeaderText = "WSL Distro"
$distroColumn.ReadOnly = $true
$distroColumn.FillWeight = 28
[void]$wslGrid.Columns.Add($distroColumn)

$cwdColumn = New-Object System.Windows.Forms.DataGridViewTextBoxColumn
$cwdColumn.Name = "DefaultCWD"
$cwdColumn.HeaderText = "Default Linux Working Dir"
$cwdColumn.FillWeight = 38
[void]$wslGrid.Columns.Add($cwdColumn)

$statusColumn = New-Object System.Windows.Forms.DataGridViewTextBoxColumn
$statusColumn.Name = "Status"
$statusColumn.HeaderText = "Status"
$statusColumn.ReadOnly = $true
$statusColumn.FillWeight = 16
[void]$wslGrid.Columns.Add($statusColumn)

$form.Controls.Add($wslGrid)
Populate-WSLGrid -Grid $wslGrid -Items $script:Config.wsl_distros -DetectedDistros $script:DetectedDistros

$statusLabel = New-Object System.Windows.Forms.Label
$statusLabel.Location = New-Object System.Drawing.Point(20, 644)
$statusLabel.Size = New-Object System.Drawing.Size(860, 22)
$statusLabel.Text = "Config path: $ConfigPath"
$form.Controls.Add($statusLabel)

$refreshButton = New-Object System.Windows.Forms.Button
$refreshButton.Location = New-Object System.Drawing.Point(20, 672)
$refreshButton.Size = New-Object System.Drawing.Size(120, 30)
$refreshButton.Text = "Refresh WSL"
$form.Controls.Add($refreshButton)

$saveButton = New-Object System.Windows.Forms.Button
$saveButton.Location = New-Object System.Drawing.Point(544, 672)
$saveButton.Size = New-Object System.Drawing.Size(110, 30)
$saveButton.Text = "Save"
$form.Controls.Add($saveButton)

$saveLaunchButton = New-Object System.Windows.Forms.Button
$saveLaunchButton.Location = New-Object System.Drawing.Point(662, 672)
$saveLaunchButton.Size = New-Object System.Drawing.Size(128, 30)
$saveLaunchButton.Text = "Save and Launch"
$form.Controls.Add($saveLaunchButton)

$cancelButton = New-Object System.Windows.Forms.Button
$cancelButton.Location = New-Object System.Drawing.Point(798, 672)
$cancelButton.Size = New-Object System.Drawing.Size(82, 30)
$cancelButton.Text = "Cancel"
$form.Controls.Add($cancelButton)

function Build-ConfigFromForm {
    $nodeId = $idTextBox.Text.Trim()
    if ([string]::IsNullOrWhiteSpace($nodeId)) {
        $nodeId = [Environment]::MachineName
    }

    $nodeName = $nameTextBox.Text.Trim()
    if ([string]::IsNullOrWhiteSpace($nodeName)) {
        $nodeName = $nodeId
    }

    $items = @()
    foreach ($row in $wslGrid.Rows) {
        if ($row.IsNewRow) {
            continue
        }

        $name = ([string]$row.Cells["Distro"].Value).Trim()
        if ([string]::IsNullOrWhiteSpace($name)) {
            continue
        }

        $key = $name.ToLowerInvariant()
        $existing = $script:KnownWSLByName[$key]
        if ($null -eq $existing) {
            $existing = [ordered]@{
                allow_command_prefixes    = @()
                allow_write_path_prefixes = @()
            }
        }

        $items += [ordered]@{
            name                      = $name
            enabled                   = [bool]$row.Cells["Enabled"].Value
            allow_command_prefixes    = @(To-StringArray $existing.allow_command_prefixes)
            allow_write_path_prefixes = @(To-StringArray $existing.allow_write_path_prefixes)
            default_cwd               = ([string]$row.Cells["DefaultCWD"].Value).Trim()
        }
    }

    return [ordered]@{
        gateway_ws          = $gatewayTextBox.Text.Trim()
        workspace_id        = $workspaceTextBox.Text.Trim()
        device_id           = $nodeId
        device_token        = $tokenTextBox.Text.Trim()
        private_key         = [string]$script:Config.private_key
        public_key          = [string]$script:Config.public_key
        pairing_code        = $pairingTextBox.Text.Trim()
        node_name           = $nodeName
        log_dir             = $(if ([string]::IsNullOrWhiteSpace($logTextBox.Text)) { Get-DefaultLogDir } else { $logTextBox.Text.Trim() })
        start_at_login      = [bool]$startupCheckbox.Checked
        enable_windows_node = [bool]$windowsNodeCheckbox.Checked
        wsl_distros         = $items
    }
}

function Save-FormConfig {
    param([bool]$LaunchAfterSave)

    $config = Build-ConfigFromForm
    if ([string]::IsNullOrWhiteSpace($config.gateway_ws)) {
        [System.Windows.Forms.MessageBox]::Show("Gateway WS is required.", "TinyClaw Node", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Warning) | Out-Null
        return
    }
    if ([string]::IsNullOrWhiteSpace($config.workspace_id)) {
        [System.Windows.Forms.MessageBox]::Show("Workspace ID is required.", "TinyClaw Node", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Warning) | Out-Null
        return
    }
    if ([string]::IsNullOrWhiteSpace($config.device_token) -and [string]::IsNullOrWhiteSpace($config.pairing_code)) {
        [System.Windows.Forms.MessageBox]::Show("Device token or pairing code is required.", "TinyClaw Node", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Warning) | Out-Null
        return
    }

    Write-NodeConfig -Config $config -Path $ConfigPath
    Update-StartupShortcut -Enabled $config.start_at_login -LaunchVbsPath $script:LaunchVbsPath
    $script:Config = $config
    $script:KnownWSLByName = Get-WSLConfigMap $config.wsl_distros
    $statusLabel.Text = "Saved to $ConfigPath at $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"

    if ($LaunchAfterSave) {
        Start-NodeHidden -LaunchVbsPath $script:LaunchVbsPath -NodeExePath $script:NodeExePath
    }

    [System.Windows.Forms.MessageBox]::Show("Configuration saved.", "TinyClaw Node", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Information) | Out-Null
}

$refreshButton.Add_Click({
    $script:DetectedDistros = Get-DetectedWSLDistros
    $script:Config.wsl_distros = Merge-WSLDistroConfigs $script:Config.wsl_distros $script:DetectedDistros
    $script:KnownWSLByName = Get-WSLConfigMap $script:Config.wsl_distros
    Populate-WSLGrid -Grid $wslGrid -Items $script:Config.wsl_distros -DetectedDistros $script:DetectedDistros
    $statusLabel.Text = "Refreshed WSL distro list at $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
})

$saveButton.Add_Click({
    Save-FormConfig -LaunchAfterSave:$false
})

$saveLaunchButton.Add_Click({
    Save-FormConfig -LaunchAfterSave:$true
})

$cancelButton.Add_Click({
    $form.Close()
})

[void]$form.ShowDialog()
