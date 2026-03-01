@echo off
echo Building activity-tracker.exe...

REM Create output directory if it doesn't exist
if not exist "output" mkdir output

REM Build as Windows GUI app (no console window)
go build -ldflags="-H windowsgui" -o output/activity-tracker.exe ./cmd/service

if %errorlevel% equ 0 (
    echo.
    echo Build successful!
    echo Output: output/activity-tracker.exe
    echo.
    echo Copying .env file to output directory...
    if exist ".env" (
        copy /Y .env output\.env >nul
        echo .env copied successfully
    ) else (
        echo Warning: .env file not found, using defaults
    )
    echo.
    echo Usage:
    echo   install-startup.bat              - Add to Windows startup
    echo   uninstall-startup.bat            - Remove from startup
    echo   start output\activity-tracker.exe - Run in background now
    echo.
) else (
    echo Build failed!
)

