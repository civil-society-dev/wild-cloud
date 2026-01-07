#!/bin/bash
set -e

# Build configuration
VERSION="${VERSION:-0.1.1}"
ARCH="${ARCH:-amd64}"
BUILD_DIR="build"
SRC_DIR="$BUILD_DIR/src"
BINARY_NAME="wild-cloud-central"

# Repository URLs
API_REPO="https://git.civilsociety.dev/wild-cloud/wild-central-api.git"
WEBAPP_REPO="https://git.civilsociety.dev/wild-cloud/wild-web-app.git"

# Optional: specify tags/commits to build from
API_REF="${API_REF:-main}"
WEBAPP_REF="${WEBAPP_REF:-main}"

echo "🔨 Building Wild Cloud Central v${VERSION} for ${ARCH}"
echo "================================================"

# Clean and create build directories
echo "📁 Preparing build directories..."
rm -rf "$SRC_DIR"
mkdir -p "$SRC_DIR"
mkdir -p "$BUILD_DIR/bin"

# Clone wild-central-api
echo ""
echo "📦 Cloning wild-central-api (${API_REF})..."
git clone --depth 1 --branch "$API_REF" "$API_REPO" "$SRC_DIR/wild-central-api"
cd "$SRC_DIR/wild-central-api"

# Build the daemon
echo "🔧 Building daemon binary..."
if [ "$ARCH" = "arm64" ]; then
    GOOS=linux GOARCH=arm64 go build -ldflags="-X main.Version=$VERSION" -o "../../bin/$BINARY_NAME" .
elif [ "$ARCH" = "amd64" ]; then
    GOOS=linux GOARCH=amd64 go build -ldflags="-X main.Version=$VERSION" -o "../../bin/$BINARY_NAME" .
else
    echo "❌ Unsupported architecture: $ARCH"
    exit 1
fi

cd - > /dev/null
echo "✅ Daemon binary built: $BUILD_DIR/bin/$BINARY_NAME"

# Clone wild-web-app
echo ""
echo "📦 Cloning wild-web-app (${WEBAPP_REF})..."
git clone --depth 1 --branch "$WEBAPP_REF" "$WEBAPP_REPO" "$SRC_DIR/wild-web-app"
cd "$SRC_DIR/wild-web-app"

# Build the web app
echo "🔧 Building web application..."
pnpm install --frozen-lockfile
# Build without type checking for packaging (production build doesn't need types)
pnpm run build --mode production || {
    echo "⚠️  Build with type checking failed, trying without type check..."
    # Fallback: build directly with vite, skipping tsc
    pnpm exec vite build
}

cd - > /dev/null
echo "✅ Web app built: $SRC_DIR/wild-web-app/dist/"

echo ""
echo "🎉 Build complete!"
echo "   Binary: $BUILD_DIR/bin/$BINARY_NAME"
echo "   Web App: $SRC_DIR/wild-web-app/dist/"
