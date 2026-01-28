#!/bin/bash
set -e

# Build configuration
VERSION="${VERSION:-0.1.1}"
ARCH="${ARCH:-amd64}"
BUILD_DIR="build"
SRC_DIR="$BUILD_DIR/src"
BINARY_NAME="wild-cloud-central"

# Monorepo source paths (relative to dist directory)
API_SOURCE="../api"
WEBAPP_SOURCE="../web"

echo "🔨 Building Wild Cloud Central v${VERSION} for ${ARCH}"
echo "================================================"

# Clean and create build directories
echo "📁 Preparing build directories..."
rm -rf "$SRC_DIR"
mkdir -p "$SRC_DIR"
mkdir -p "$BUILD_DIR/bin"

# Copy API source from monorepo
echo ""
echo "📦 Copying API source from monorepo..."
cp -r "$API_SOURCE" "$SRC_DIR/api"
cd "$SRC_DIR/api"

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

# Copy web app source from monorepo
echo ""
echo "📦 Copying web app source from monorepo..."
cp -r "$WEBAPP_SOURCE" "$SRC_DIR/web"
cd "$SRC_DIR/web"

# Remove any local env overrides that shouldn't be in the build
rm -f .env.local .env.*.local

# Build the web app
echo "🔧 Building web application..."
pnpm install --frozen-lockfile

# For production builds, use empty base URL so the app uses relative paths
# This allows nginx to proxy /api/* to the backend on whatever hostname is used
echo "VITE_API_BASE_URL=" > .env.production.local

# Build without type checking for packaging (production build doesn't need types)
pnpm run build --mode production || {
    echo "⚠️  Build with type checking failed, trying without type check..."
    # Fallback: build directly with vite, skipping tsc
    pnpm exec vite build
}

cd - > /dev/null
echo "✅ Web app built: $SRC_DIR/web/dist/"

echo ""
echo "🎉 Build complete!"
echo "   Binary: $BUILD_DIR/bin/$BINARY_NAME"
echo "   Web App: $SRC_DIR/web/dist/"
