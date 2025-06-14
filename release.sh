#!/bin/bash

# Build script for creating multi-platform releases

set -e

APP_NAME="duckduckgo-chat-api"
VERSION=${1:-"v1.0.0"}
BUILD_DIR="releases"

echo "🦆 Building $APP_NAME $VERSION"
echo "======================================"

# Clean build directory
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Check Go
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed"
    exit 1
fi

echo "✅ Go $(go version | grep -oP 'go\d+\.\d+\.\d+')"

# Install dependencies
echo "📦 Installing dependencies..."
go mod tidy

echo "🔨 Building binaries..."

# Compilation flags for size optimization
LDFLAGS="-s -w -X main.Version=$VERSION"

# Linux AMD64
echo "  📦 Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/${APP_NAME}_${VERSION}_linux_amd64" .

# Linux ARM64
echo "  📦 Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/${APP_NAME}_${VERSION}_linux_arm64" .

# Windows AMD64
echo "  📦 Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/${APP_NAME}_${VERSION}_windows_amd64.exe" .

# Windows ARM64
echo "  📦 Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/${APP_NAME}_${VERSION}_windows_arm64.exe" .

# macOS AMD64
echo "  📦 macOS AMD64..."
GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/${APP_NAME}_${VERSION}_darwin_amd64" .

# macOS ARM64 (Apple Silicon)
echo "  📦 macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/${APP_NAME}_${VERSION}_darwin_arm64" .

echo ""
echo "✅ Build completed successfully!"
echo ""
echo "📁 Binaries created:"
ls -la "$BUILD_DIR/"

# Create archives
echo ""
echo "📦 Creating archives..."

cd "$BUILD_DIR"

# Archives for Unix systems (tar.gz)
for file in *linux* *darwin*; do
    if [ -f "$file" ]; then
        tar -czf "${file}.tar.gz" "$file"
        echo "  ✅ ${file}.tar.gz"
    fi
done

# Archives for Windows (zip)
if command -v zip &> /dev/null; then
    for file in *windows*.exe; do
        if [ -f "$file" ]; then
            zip "${file%.exe}.zip" "$file"
            echo "  ✅ ${file%.exe}.zip"
        fi
    done
else
    echo "  ⚠️  zip command not found, skipping Windows archives"
    echo "  � Install zip: sudo apt install zip"
fi

cd ..

echo ""
echo "📋 Release files summary:"
ls -la "$BUILD_DIR/"*.{tar.gz,zip} 2>/dev/null || echo "No archives created"

echo ""
echo "🏷️  To create a GitHub release, use:"
echo "   gh release create $VERSION $BUILD_DIR/*.tar.gz $BUILD_DIR/*.zip --title \"Release $VERSION\" --notes \"Release $VERSION\""
