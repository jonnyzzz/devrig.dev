@echo off
REM
REM see https://devrig.dev for more details
REM
REM devrig.bat - Windows batch wrapper for devrig.ps1
REM This calls the PowerShell script with all arguments

setlocal enabledelayedexpansion

REM Get the directory where this script is located
set "SCRIPT_DIR=%~dp0"

REM Remove trailing backslash
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

REM Call PowerShell script with all arguments
powershell.exe -ExecutionPolicy Bypass -NoProfile -File "%SCRIPT_DIR%\devrig.ps1" %*

REM Exit with the same exit code as PowerShell
exit /b %ERRORLEVEL%
