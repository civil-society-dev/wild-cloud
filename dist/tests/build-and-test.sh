#!/bin/bash

set -e

echo "🏗️  Wild Cloud Central - Build and Test"
echo "========================================"
echo ""

# Change to project root
cd "$(dirname "$0")/.."

# Detect architecture
ARCH=$(dpkg --print-architecture)
echo "📋 Detected architecture: $ARCH"
echo ""

# Check if we need to build
if [ -f "build/wild-cloud-central_*_${ARCH}.deb" ]; then
    echo "📦 Found existing .deb package for $ARCH"
    echo ""
    read -p "Use existing package? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "🔨 Building new package for $ARCH..."
        make clean
        if [ "$ARCH" = "arm64" ]; then
            make package-arm64
        else
            make package-amd64
        fi
    fi
else
    echo "🔨 Building package for $ARCH..."
    if [ "$ARCH" = "arm64" ]; then
        make package-arm64
    else
        make package-amd64
    fi
fi

echo ""
echo "🐳 Building Docker test image..."
docker build -t wild-cloud-central-test .

echo ""
echo "🧪 Running installation tests..."
docker run --rm wild-cloud-central-test

echo ""
echo "✅ All tests passed!"
echo ""
echo "💡 To run interactive testing:"
echo "   ./tests/integration/start-interactive.sh"
echo ""
echo "💡 To run in background:"
echo "   ./tests/integration/start-background.sh"
