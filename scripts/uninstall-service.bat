@echo off
cd /d "%~dp0.."
echo Uninstalling Cricket Tracker Windows Service...
echo.

set "APP_DIR=%CD%\"
set "EXE_PATH=%APP_DIR%output\service.exe"

if not exist "%EXE_PATH%" (
    echo Error: service.exe not found in output directory
    pause
    exit /b 1
)

REM Stop service if running
net stop CricketTracker >nul 2>&1

REM Uninstall service
"%EXE_PATH%" uninstall

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Success! Cricket Tracker service uninstalled.
) else (
    echo.
    echo Failed to uninstall service
)

echo.
pause
