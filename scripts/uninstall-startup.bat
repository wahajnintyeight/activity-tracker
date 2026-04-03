@echo off
cd /d "%~dp0.."
echo Removing Activity Tracker from Windows Startup...
echo.

set "STARTUP_FOLDER=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"
set "SHORTCUT_PATH=%STARTUP_FOLDER%\ActivityTracker.lnk"

if exist "%SHORTCUT_PATH%" (
    del "%SHORTCUT_PATH%"
    echo Removed from startup
) else (
    echo Not found in startup
)

echo.
echo Done
echo.
pause
