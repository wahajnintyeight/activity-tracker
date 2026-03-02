@echo off
echo Starting Cricket Tracker...
echo.
echo Make sure:
echo 1. RabbitMQ is running
echo 2. Tesseract OCR is installed
echo 3. Cricket 24 game is running
echo 4. .env file is configured with correct scoreboard coordinates
echo.
pause

cd /d "%~dp0"
if exist "output\service.exe" (
    output\service.exe --type cricket-tracker
) else (
    echo Error: service.exe not found in output folder
    echo Please run build.bat first
    pause
)
