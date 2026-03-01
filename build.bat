@echo off
echo Building activity-tracker.exe...

REM Create output directory if it doesn't exist
if not exist "output" mkdir output

REM Build the executable to output directory
go build -o output/activity-tracker.exe ./cmd/service

if %errorlevel% equ 0 (
    echo.
    echo Build successful!
    echo Output: output/activity-tracker.exe
    echo.
    echo Usage:
    echo   output\activity-tracker.exe run          - Run manually
    echo   output\activity-tracker.exe install      - Install as service (Admin)
    echo   output\activity-tracker.exe start        - Start service (Admin)
    echo   output\activity-tracker.exe stop         - Stop service (Admin)
    echo   output\activity-tracker.exe uninstall    - Remove service (Admin)
) else (
    echo Build failed!
)
