#!/bin/bash

# Script de build pour créer des releases multi-plateformes

set -e

APP_NAME="duckduckgo-chat-api"
VERSION=${1:-"v1.0.0"}
BUILD_DIR="releases"

echo "🦆 Construction de $APP_NAME $VERSION"
echo "======================================"

# Nettoyer le répertoire de build
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Vérifier Go
if ! command -v go &> /dev/null; then
    echo "❌ Go n'est pas installé"
    exit 1
fi

echo "✅ Go $(go version | grep -oP 'go\d+\.\d+\.\d+')"

# Installation des dépendances
echo "📦 Installation des dépendances..."
go mod tidy

echo "🔨 Construction des binaires..."

# Flags de compilation pour optimiser la taille
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
echo "✅ Construction terminée avec succès !"
echo ""
echo "📁 Binaires créés:"
ls -la "$BUILD_DIR/"

# Créer des archives
echo ""
echo "📦 Création des archives..."

cd "$BUILD_DIR"

# Archives pour les systèmes Unix (tar.gz)
for file in *linux* *darwin*; do
    if [ -f "$file" ]; then
        tar -czf "${file}.tar.gz" "$file"
        echo "  ✅ ${file}.tar.gz"
    fi
done

# Archives pour Windows (zip)
for file in *windows*.exe; do
    if [ -f "$file" ]; then
        zip "${file%.exe}.zip" "$file"
        echo "  ✅ ${file%.exe}.zip"
    fi
done

cd ..

echo ""
echo "📋 Résumé des fichiers de release:"
ls -la "$BUILD_DIR/"*.{tar.gz,zip} 2>/dev/null || echo "Aucune archive créée"

echo ""
echo "🏷️  Pour créer une release GitHub, utilisez:"
echo "   gh release create $VERSION $BUILD_DIR/*.tar.gz $BUILD_DIR/*.zip --title \"Release $VERSION\" --notes \"Release $VERSION\""
