@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
set "NODE_EXE=%SCRIPT_DIR%tinyclaw-node.exe"
set "CONFIG_PATH=%ProgramData%\TinyClawNode\config.json"

if not exist "%NODE_EXE%" (
  echo tinyclaw-node.exe was not found next to configure-node.cmd.
  exit /b 1
)

"%NODE_EXE%" --configure --config "%CONFIG_PATH%"
exit /b %ERRORLEVEL%
