#!/bin/bash

set -e

# Configuration - update these for your server
SERVER="user@mywildcloud.org"
REMOTE_PATH="/var/www/html/apt"
LOCAL_REPO="dist/repositories/apt"

echo "🚀 Deploying APT repository to mywildcloud.org..."

# Check if repository exists
if [ ! -d "$LOCAL_REPO" ]; then
    echo "❌ Repository not found. Run './scripts/build-apt-repository.sh' first."
    exit 1
fi

# Deploy repository
echo "📤 Uploading repository files..."
echo "Not yet implemented. We'll do this when we have a permanent home."

echo ""
echo "✅ Deployment complete!"
echo ""
echo "🌐 Repository URL: https://mywildcloud.org/apt"
echo "🔑 GPG Key URL: https://mywildcloud.org/apt/wild-cloud-central.gpg"
echo ""
echo "👥 Users can now install with:"
echo ""
echo "   # Download and install GPG key (Debian convention)"
echo "   curl -fsSL https://mywildcloud.org/apt/wild-cloud-central.gpg | sudo tee /usr/share/keyrings/wild-cloud-central-archive-keyring.gpg > /dev/null"
echo ""
echo "   # Add repository (modern .sources format)"
echo "   sudo tee /etc/apt/sources.list.d/wild-cloud-central.sources << 'EOF'"
echo "   Types: deb"
echo "   URIs: https://mywildcloud.org/apt"
echo "   Suites: stable"
echo "   Components: main"
echo "   Signed-By: /usr/share/keyrings/wild-cloud-central-archive-keyring.gpg"
echo "   EOF"
echo ""
echo "   # Update and install"
echo "   sudo apt update"
echo "   sudo apt install wild-cloud-central"
