@echo off
cd /d "%~dp0.."
echo Stopping Cricket Tracker Service...
echo.

net stop CricketTracker

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Cricket Tracker service stopped successfully!
) else (
    echo.
    echo Failed to stop service
    echo Make sure the service is running first
)

echo.
pause
