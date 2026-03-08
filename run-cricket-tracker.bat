@echo off
echo Starting Cricket Tracker...

cd /d "%~dp0"
if exist "output\service.exe" (
    output\service.exe --type cricket-tracker
) else (
    echo Error: service.exe not found in output folder
    echo Please run build.bat first
    pause
)
