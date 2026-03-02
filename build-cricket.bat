@echo off
echo Building Cricket Tracker...
echo.

REM Check if Tesseract is installed
where tesseract >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo WARNING: Tesseract OCR not found!
    echo Cricket Tracker will not work without Tesseract.
    echo.
    echo Download from: https://github.com/UB-Mannheim/tesseract/wiki
    echo.
    pause
)

echo Tesseract found: 
tesseract --version | findstr "tesseract"
echo.

echo Building...
go build -o output\service.exe .\cmd\service

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo Build successful!
    echo ========================================
    echo.
    echo Executable: output\service.exe
    echo.
    echo To run cricket tracker:
    echo   .\output\service.exe --type cricket-tracker
    echo.
    echo To run activity tracker:
    echo   .\output\service.exe
    echo.
) else (
    echo.
    echo ========================================
    echo Build failed!
    echo ========================================
    echo.
    echo Check the error messages above.
    echo.
)

pause
