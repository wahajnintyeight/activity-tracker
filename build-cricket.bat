@echo off
echo Building Cricket Tracker...
echo.

echo Building...
go build -o output\service.exe .\cmd\service

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo ========================================
    echo Build failed!
    echo ========================================
    echo.
    echo Check the error messages above.
    echo.
    pause
    exit /b 1
)

echo.
echo ========================================
echo Build successful!
echo ========================================
echo.
echo Where is the team score panel on screen?
echo   [L] Left
echo   [M] Middle
echo.
choice /c LM /n /m "Select position (L/M): "

if %ERRORLEVEL% EQU 1 set TEAM_SCORE_POSITION=left
if %ERRORLEVEL% EQU 2 set TEAM_SCORE_POSITION=middle

echo.
echo Starting Cricket Tracker (team-score-position=%TEAM_SCORE_POSITION%)...
echo.
start "" "output\service.exe" --type cricket-tracker --team-score-position %TEAM_SCORE_POSITION%

pause
