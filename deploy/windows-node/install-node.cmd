@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
set "PS_SCRIPT=%SCRIPT_DIR%install-node.ps1"

if not exist "%PS_SCRIPT%" (
  echo install-node.ps1 was not found next to install-node.cmd.
  exit /b 1
)

powershell.exe -NoProfile -ExecutionPolicy Bypass -Command ^
  "Start-Process PowerShell -Verb RunAs -ArgumentList '-NoProfile','-ExecutionPolicy','Bypass','-File','\"%PS_SCRIPT%\"'"

exit /b %ERRORLEVEL%
