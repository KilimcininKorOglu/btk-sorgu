@echo off
setlocal enabledelayedexpansion

echo ========================================
echo   BTK Sorgu - Build Script (Windows)
echo ========================================
echo.

:: Get version from main.go
for /f "tokens=4 delims= " %%a in ('findstr /C:"const Version" main.go') do (
    set VERSION=%%~a
)
set VERSION=%VERSION:"=%
echo Version: %VERSION%
echo.

:: Create dist directory
if not exist dist mkdir dist

echo Building for all platforms...
echo.

:: Windows AMD64
echo [1/6] Windows AMD64...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o dist\btk-sorgu-windows-amd64.exe .
if %errorlevel% neq 0 goto :error

:: Windows ARM64
echo [2/6] Windows ARM64...
set GOOS=windows
set GOARCH=arm64
go build -ldflags="-s -w" -o dist\btk-sorgu-windows-arm64.exe .
if %errorlevel% neq 0 goto :error

:: Linux AMD64
echo [3/6] Linux AMD64...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o dist\btk-sorgu-linux-amd64 .
if %errorlevel% neq 0 goto :error

:: Linux ARM64
echo [4/6] Linux ARM64...
set GOOS=linux
set GOARCH=arm64
go build -ldflags="-s -w" -o dist\btk-sorgu-linux-arm64 .
if %errorlevel% neq 0 goto :error

:: macOS AMD64
echo [5/6] macOS AMD64 (Intel)...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="-s -w" -o dist\btk-sorgu-darwin-amd64 .
if %errorlevel% neq 0 goto :error

:: macOS ARM64
echo [6/6] macOS ARM64 (Apple Silicon)...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags="-s -w" -o dist\btk-sorgu-darwin-arm64 .
if %errorlevel% neq 0 goto :error

:: Copy .env file if exists
if exist .env (
    echo.
    echo Copying .env file...
    copy .env dist\.env >nul
)

:: Copy .env.example if exists
if exist .env.example (
    echo Copying .env.example file...
    copy .env.example dist\.env.example >nul
)

echo.
echo ========================================
echo   Build completed successfully!
echo ========================================
echo.
echo Output files:
dir /b dist
echo.
goto :end

:error
echo.
echo Build failed with error %errorlevel%
exit /b %errorlevel%

:end
endlocal
