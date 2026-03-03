@echo off
echo Installing Cricket Tracker as Windows Service...
echo.

REM Get current directory
set "APP_DIR=%~dp0"
set "EXE_PATH=%APP_DIR%output\service.exe"

REM Check if executable exists
if not exist "%EXE_PATH%" (
    echo Error: service.exe not found in output directory
    echo Please run build-cricket.bat first
    pause
    exit /b 1
)

REM Install as Windows service
"%EXE_PATH%" install

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Success! Cricket Tracker installed as Windows service.
    echo.
    echo To start the service:
    echo   net start CricketTracker
    echo.
    echo To stop the service:
    echo   net stop CricketTracker
    echo.
    echo To uninstall the service:
    echo   "%EXE_PATH%" uninstall
    echo.
) else (
    echo.
    echo Failed to install service
    echo Make sure you're running as Administrator
)

echo.
pause
