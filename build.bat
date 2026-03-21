@echo off
setlocal enabledelayedexpansion

set BINARY_NAME=btk-sorgu
set BUILD_DIR=bin
set DIST_DIR=dist

:: Get version info
for /f "tokens=*" %%i in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%i
if not defined VERSION set VERSION=0.0.0

for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set COMMIT=%%i
if not defined COMMIT set COMMIT=unknown

set LDFLAGS=-s -w -X 'main.Version=%VERSION%'

if "%~1"=="" goto help
goto %~1

:build
    if not exist %BUILD_DIR% mkdir %BUILD_DIR%
    go build -ldflags "%LDFLAGS%" -o %BUILD_DIR%\%BINARY_NAME%.exe .
    goto end

:build-all
    if not exist %DIST_DIR% mkdir %DIST_DIR%
    echo Building for all platforms...
    set GOOS=windows& set GOARCH=amd64& go build -ldflags "%LDFLAGS%" -o %DIST_DIR%\%BINARY_NAME%-windows-amd64.exe .
    set GOOS=windows& set GOARCH=arm64& go build -ldflags "%LDFLAGS%" -o %DIST_DIR%\%BINARY_NAME%-windows-arm64.exe .
    set GOOS=linux& set GOARCH=amd64& go build -ldflags "%LDFLAGS%" -o %DIST_DIR%\%BINARY_NAME%-linux-amd64 .
    set GOOS=linux& set GOARCH=arm64& go build -ldflags "%LDFLAGS%" -o %DIST_DIR%\%BINARY_NAME%-linux-arm64 .
    set GOOS=darwin& set GOARCH=amd64& go build -ldflags "%LDFLAGS%" -o %DIST_DIR%\%BINARY_NAME%-darwin-amd64 .
    set GOOS=darwin& set GOARCH=arm64& go build -ldflags "%LDFLAGS%" -o %DIST_DIR%\%BINARY_NAME%-darwin-arm64 .
    echo Done. Output: %DIST_DIR%\
    goto end

:clean
    if exist %BUILD_DIR% rmdir /s /q %BUILD_DIR%
    if exist %DIST_DIR% rmdir /s /q %DIST_DIR%
    go clean
    goto end

:test
    go test ./...
    goto end

:test-race
    go test -race ./...
    goto end

:test-cover
    go test -cover ./...
    goto end

:test-verbose
    go test -v ./...
    goto end

:bench
    go test -bench=. -benchmem ./...
    goto end

:run
    call :build
    %BUILD_DIR%\%BINARY_NAME%.exe
    goto end

:fmt
    go fmt ./...
    goto end

:vet
    go vet ./...
    goto end

:lint
    call :fmt
    call :vet
    goto end

:help
    echo Available commands:
    echo   build        - Build the binary to bin\
    echo   build-all    - Cross-compile for all platforms to dist\
    echo   clean        - Remove build artifacts
    echo   test         - Run all tests
    echo   test-race    - Run tests with race detector
    echo   test-cover   - Run tests with coverage
    echo   test-verbose - Run tests with verbose output
    echo   bench        - Run benchmarks
    echo   run          - Build and run (TUI mode)
    echo   fmt          - Format code
    echo   vet          - Run go vet
    echo   lint         - Run fmt and vet
    goto end

:end
    endlocal
