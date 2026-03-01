@echo off
echo Installing Activity Tracker to Windows Startup...
echo.

REM Get the current directory
set "APP_DIR=%~dp0"
set "EXE_PATH=%APP_DIR%output\activity-tracker.exe"

REM Check if executable exists
if not exist "%EXE_PATH%" (
    echo Error: activity-tracker.exe not found in output directory
    echo Please run build.bat first
    pause
    exit /b 1
)

REM Add to startup folder
set "STARTUP_FOLDER=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"
set "SHORTCUT_PATH=%STARTUP_FOLDER%\ActivityTracker.lnk"

REM Create shortcut using PowerShell
powershell -Command "$WshShell = New-Object -ComObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%SHORTCUT_PATH%'); $Shortcut.TargetPath = '%EXE_PATH%'; $Shortcut.WorkingDirectory = '%APP_DIR%output'; $Shortcut.Save()"

if exist "%SHORTCUT_PATH%" (
    echo.
    echo Success! Activity Tracker installed to startup.
    echo.
    echo The app will start automatically when you log in.
    echo It runs in the background without showing a window.
    echo.
    echo To start now: start output\activity-tracker.exe
    echo To uninstall: uninstall-startup.bat
) else (
    echo.
    echo Failed to create startup shortcut
)

echo.
pause
