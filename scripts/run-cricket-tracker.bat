@echo off
cd /d "%~dp0.."
echo Starting Cricket Tracker...

if exist "output\service.exe" (
    output\service.exe --type cricket-tracker
) else (
    echo Error: service.exe not found in output folder
    echo Please run build.bat first
    pause
)
