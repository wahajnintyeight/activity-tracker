@echo off
echo Starting Cricket Tracker Service...
echo.

set "APP_DIR=%~dp0"
set "EXE_PATH=%APP_DIR%output\service.exe"

if not exist "%EXE_PATH%" (
    echo Error: service.exe not found in output directory
    echo Please run build-cricket.bat first
    pause
    exit /b 1
)

REM Start the service
net start CricketTracker

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Cricket Tracker service started successfully!
    echo.
    echo To check status: sc query CricketTracker
    echo To stop: net stop CricketTracker
) else (
    echo.
    echo Failed to start service
    echo Make sure the service is installed first:
    echo   install-service.bat
)

echo.
pause
