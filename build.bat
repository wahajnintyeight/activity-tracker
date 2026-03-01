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
    echo Copying .env file to output directory...
    if exist ".env" (
        copy /Y .env output\.env >nul
        echo .env copied successfully
    ) else (
        echo Warning: .env file not found, using defaults
    )
    echo.
    echo Usage:
    echo   output\activity-tracker.exe run          - Run manually
    echo   output\activity-tracker.exe install      - Install as service (Admin)
    echo   output\activity-tracker.exe start        - Start service (Admin)
    echo   output\activity-tracker.exe stop         - Stop service (Admin)
    echo   output\activity-tracker.exe uninstall    - Remove service (Admin)
    echo.
    echo Note: Run from activity-tracker root directory or ensure .env is in output/
) else (
    echo Build failed!
)
