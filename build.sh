#!/bin/bash

# Script de construction et de démarrage de l'API DuckDuckGo Chat

set -e

API_DIR="api"
API_NAME="duckduckgo-chat-api"

echo "🦆 Construction de l'API DuckDuckGo Chat"
echo "======================================"

# Vérifier si nous sommes dans le bon répertoire
if [ ! -d "$API_DIR" ]; then
    echo "❌ Répertoire api/ non trouvé"
    exit 1
fi

cd "$API_DIR"

# Vérifier la version de Go
echo "📋 Vérification de Go..."
if ! command -v go &> /dev/null; then
    echo "❌ Go n'est pas installé"
    exit 1
fi

GO_VERSION=$(go version | grep -oP 'go\d+\.\d+' | sed 's/go//')
REQUIRED_VERSION="1.21"

if ! printf '%s\n%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
    echo "❌ Go $REQUIRED_VERSION+ requis (version actuelle: $GO_VERSION)"
    exit 1
fi

echo "✅ Go $GO_VERSION détecté"

# Installation des dépendances
echo "📦 Installation des dépendances..."
go mod tidy

# Vérification des dépendances
echo "🔍 Vérification des dépendances..."
go mod verify

# Construction de l'API
echo "🔨 Construction de l'API..."

# Construction pour Linux
echo "  📦 Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "build/${API_NAME}_linux_amd64" .

# Construction pour Windows
echo "  📦 Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "build/${API_NAME}_windows_amd64.exe" .

# Construction pour macOS
echo "  📦 macOS AMD64..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "build/${API_NAME}_darwin_amd64" .

# Construction pour macOS ARM64 (Apple Silicon)
echo "  📦 macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "build/${API_NAME}_darwin_arm64" .

echo ""
echo "✅ Construction terminée avec succès !"
echo ""
echo "📁 Binaires disponibles dans build/:"
ls -la build/

echo ""
echo "🚀 Démarrage de l'API..."
echo ""

# Démarrer l'API
if [ -f "build/${API_NAME}_linux_amd64" ]; then
    echo "🌟 API DuckDuckGo Chat démarrée sur http://localhost:8080"
    echo "📋 Documentation: http://localhost:8080/"
    echo "❓ Santé de l'API: http://localhost:8080/api/v1/health"
    echo ""
    echo "🛑 Appuyez sur Ctrl+C pour arrêter"
    ./build/${API_NAME}_linux_amd64
else
    echo "❌ Erreur: binaire non trouvé"
    exit 1
fi
