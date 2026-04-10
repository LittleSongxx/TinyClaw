# TinyClaw Node for Windows

This Windows release is available in two forms:

- `TinyClawNodeSetup.exe` as the preferred installer for normal users
- `TinyClawNode-windows-<arch>.zip` as the portable/manual package

After installation, TinyClaw Node creates:

- `TinyClaw Node` desktop shortcut for hidden launch
- `TinyClaw Node Settings` desktop shortcut for the config UI
- `install-node.cmd` for double-click installation
- `configure-node.cmd` for double-click settings
- `configure-node.vbs` for silent settings launch without a console window
- `%ProgramData%\TinyClawNode\config.json` for runtime configuration
- `%ProgramData%\TinyClawNode\logs` for node logs

## What it does

- Registers one Windows desktop node for screenshots, UI automation, window control, browser open, app launch, and Windows-side shell or file tasks.
- Registers one WSL virtual node per enabled distro from the config file.
- Keeps startup optional through a per-user Startup shortcut instead of a Windows Service.

## Install

### Recommended: installer

1. Double-click `TinyClawNodeSetup.exe`.
2. Choose an install directory if you want something other than `%ProgramFiles%\TinyClawNode`.
3. Finish the setup wizard.
4. The installer opens the settings UI automatically.

### Portable/manual package

1. Unzip the package on Windows.
2. Double-click `install-node.cmd`.
3. The installer copies files into `%ProgramFiles%\TinyClawNode`.
4. The installer opens the settings UI after copying the files.

If Windows SmartScreen or UAC prompts for confirmation, allow the installer to continue.

## Configure

The settings UI lets you edit:

- `gateway_ws`
- `workspace_id`
- `device_id`
- `device_token`
- `pairing_code` for initial pairing only
- `node_name`
- `start_at_login`
- `enable_windows_node`
- enabled WSL distros
- each distro `default_cwd`

`private_key` and `public_key` are kept in the config file, but the node generates and preserves them automatically.

The config file shape is:

```json
{
  "gateway_ws": "ws://127.0.0.1:36060/gateway/nodes/ws",
  "workspace_id": "default",
  "device_id": "DESKTOP-1234",
  "device_token": "",
  "private_key": "",
  "public_key": "",
  "pairing_code": "",
  "node_name": "DESKTOP-1234",
  "log_dir": "C:\\ProgramData\\TinyClawNode\\logs",
  "start_at_login": false,
  "enable_windows_node": true,
  "wsl_distros": [
    {
      "name": "Ubuntu-22.04",
      "enabled": true,
      "allow_command_prefixes": [
        "git status",
        "npm test"
      ],
      "allow_write_path_prefixes": [
        "/home/user/workspace"
      ],
      "default_cwd": "/home/user/workspace/project"
    }
  ]
}
```

The GUI preserves the WSL allowlists even though it does not edit them directly.

## Launch

- Double-click `TinyClaw Node` on the desktop.
- Or double-click `TinyClaw Node Settings` on the desktop.
- Or double-click `%ProgramFiles%\TinyClawNode\configure-node.cmd` to reopen settings from the install directory.
- Or run `%ProgramFiles%\TinyClawNode\tinyclaw-node.exe --config %ProgramData%\TinyClawNode\config.json`.
- Or use the login startup shortcut if enabled in settings.

## Build a package from the repo

From the repository root:

```bash
./scripts/package_tinyclaw_node_windows.sh amd64
```

That script builds `tinyclaw-node.exe`, stages the Windows assets from `deploy/windows-node`, writes the portable zip package under `build/release`, and also builds `TinyClawNodeSetup.exe`.
