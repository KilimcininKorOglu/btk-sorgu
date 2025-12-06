#!/bin/bash

echo "========================================"
echo "  BTK Sorgu - Build Script"
echo "========================================"
echo

# Get version from main.go
VERSION=$(grep -oP 'const Version = "\K[^"]+' main.go)
echo "Version: $VERSION"
echo

# Create dist directory
mkdir -p dist

echo "Building for all platforms..."
echo

# Windows AMD64
echo "[1/6] Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/btk-sorgu-windows-amd64.exe . || exit 1

# Windows ARM64
echo "[2/6] Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o dist/btk-sorgu-windows-arm64.exe . || exit 1

# Linux AMD64
echo "[3/6] Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/btk-sorgu-linux-amd64 . || exit 1

# Linux ARM64
echo "[4/6] Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/btk-sorgu-linux-arm64 . || exit 1

# macOS AMD64
echo "[5/6] macOS AMD64 (Intel)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/btk-sorgu-darwin-amd64 . || exit 1

# macOS ARM64
echo "[6/6] macOS ARM64 (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/btk-sorgu-darwin-arm64 . || exit 1

# Copy .env file if exists
if [ -f .env ]; then
    echo
    echo "Copying .env file..."
    cp .env dist/.env
fi

# Copy .env.example if exists
if [ -f .env.example ]; then
    echo "Copying .env.example file..."
    cp .env.example dist/.env.example
fi

# Create checksums
echo
echo "Creating checksums..."
cd dist
sha256sum * > checksums.txt
cat checksums.txt
cd ..

echo
echo "========================================"
echo "  Build completed successfully!"
echo "========================================"
echo
echo "Output files:"
ls -lh dist/
