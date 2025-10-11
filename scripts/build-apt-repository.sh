#!/bin/bash

set -e

# Configuration
REPO_NAME="wild-cloud-central"
DIST="stable"
COMPONENT="main"
REPO_DIR="dist/repositories/apt"
PACKAGE_DIR="dist/packages"

echo "🏗️  Building APT repository with aptly..."

# Initialize aptly database if needed (aptly auto-creates on first use)
echo "📦 Ensuring aptly is ready..."

# Check if repository is already published and drop it first
if aptly publish list 2>/dev/null | grep -q "filesystem:dist/repositories/apt:./stable"; then
    echo "🗑️  Dropping existing published repository..."
    aptly publish drop -force-drop stable filesystem:dist/repositories/apt: || echo "⚠️  Could not drop existing published repository"
fi

# Check if repository already exists and drop it to start fresh
if aptly repo list 2>/dev/null | grep -q "\[$REPO_NAME\]"; then
    echo "🗑️  Removing existing repository '$REPO_NAME'..."
    aptly repo drop -force "$REPO_NAME" || echo "⚠️  Could not drop existing repository"
fi

# Create local repository
echo "📦 Creating local repository '$REPO_NAME'..."
aptly repo create -distribution="$DIST" -component="$COMPONENT" "$REPO_NAME"

# Add packages to repository
if [ -d "$PACKAGE_DIR" ] && [ "$(ls -A $PACKAGE_DIR/*.deb 2>/dev/null)" ]; then
    echo "📦 Adding packages to repository..."
    aptly repo add "$REPO_NAME" "$PACKAGE_DIR"/*.deb
else
    echo "⚠️  No .deb files found in $PACKAGE_DIR directory"
    exit 1
fi

# Create output directory
mkdir -p "$REPO_DIR"

# Get GPG signing key
SIGNING_KEY=$(gpg --list-secret-keys --with-colons | grep '^sec' | head -1 | cut -d: -f5)
if [ -z "$SIGNING_KEY" ]; then
    echo "❌ No GPG signing key found"
    exit 1
fi

echo "🔐 Using GPG key: $SIGNING_KEY"

# Publish repository with GPG signing
echo "🚀 Publishing repository..."
aptly publish repo -distribution="$DIST" -component="$COMPONENT" -architectures="amd64,arm64" -gpg-key="$SIGNING_KEY" "$REPO_NAME" "filesystem:$REPO_DIR:"

# Export GPG public key for users
echo "🔑 Exporting GPG public key..."
gpg --export "$SIGNING_KEY" > "$REPO_DIR/wild-cloud-central.gpg"

echo "✅ APT repository built successfully!"
echo ""
echo "📦 Repository location: $REPO_DIR/"
echo "🔑 GPG key location: $REPO_DIR/wild-cloud-central.gpg"
echo ""
echo "📋 Users can install with:"
echo ""
echo "   # Download and install GPG key"
echo "   curl -fsSL https://mywildcloud.org/apt/wild-cloud-central.gpg | sudo tee /usr/share/keyrings/wild-cloud-central-archive-keyring.gpg > /dev/null"
echo ""
echo "   # Add repository"
echo "   sudo tee /etc/apt/sources.list.d/wild-cloud-central.sources << 'EOF'"
echo "   Types: deb"
echo "   URIs: https://mywildcloud.org/apt"
echo "   Suites: $DIST"
echo "   Components: $COMPONENT"
echo "   Signed-By: /usr/share/keyrings/wild-cloud-central-archive-keyring.gpg"
echo "   EOF"
echo ""
echo "   # Update and install"
echo "   sudo apt update"
echo "   sudo apt install wild-cloud-central"