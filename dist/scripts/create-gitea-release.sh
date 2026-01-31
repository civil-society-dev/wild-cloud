#!/bin/bash
set -euo pipefail

GITEA_URL="$1"
OWNER="$2"
REPO="$3"
TAG="$4"
VERSION="$5"
TOKEN="$6"

API_URL="${GITEA_URL}/api/v1"
PACKAGES_DIR="dist/packages"
BINARIES_DIR="dist/bin"
SKIP_RELEASE_CREATION=false

echo "📦 Release configuration:"
echo "   Repository: ${OWNER}/${REPO}"
echo "   Tag: ${TAG}"
echo "   Version: ${VERSION}"
echo ""

# Check if release already exists
echo "🔍 Checking if release ${TAG} exists..."
RELEASE_ID=$(curl -s -H "Authorization: token ${TOKEN}" \
    "${API_URL}/repos/${OWNER}/${REPO}/releases/tags/${TAG}" \
    | jq -r '.id // empty')

if [ -n "$RELEASE_ID" ]; then
    echo "✅ Release ${TAG} exists (ID: ${RELEASE_ID})"
    echo "📦 Updating existing release assets..."

    # Get existing assets
    EXISTING_ASSETS=$(curl -s -H "Authorization: token ${TOKEN}" \
        "${API_URL}/repos/${OWNER}/${REPO}/releases/${RELEASE_ID}/assets" \
        | jq -r '.[].id')

    # Delete old .deb assets
    for ASSET_ID in $EXISTING_ASSETS; do
        ASSET_NAME=$(curl -s -H "Authorization: token ${TOKEN}" \
            "${API_URL}/repos/${OWNER}/${REPO}/releases/assets/${ASSET_ID}" \
            | jq -r '.name')

        if [[ "$ASSET_NAME" == *.deb ]] || [[ "$ASSET_NAME" == wild-cloud-central-* ]] || [[ "$ASSET_NAME" == "SHA256SUMS" ]]; then
            echo "   Deleting old asset: ${ASSET_NAME}"
            curl -s -X DELETE -H "Authorization: token ${TOKEN}" \
                "${API_URL}/repos/${OWNER}/${REPO}/releases/assets/${ASSET_ID}"
        fi
    done

    echo "✅ Cleaned up old package assets"
    SKIP_RELEASE_CREATION=true
fi

# Create new release if it doesn't exist
if [ "$SKIP_RELEASE_CREATION" != "true" ]; then
    echo "📝 Creating new release ${TAG}..."
    RELEASE_DATA=$(cat <<EOF
{
  "tag_name": "${TAG}",
  "name": "Wild Cloud Central ${VERSION}",
  "body": "## Wild Cloud Central ${VERSION}\n\n### Installation Options\n\n#### Full Installation (.deb package)\n\nDownload the appropriate .deb package for your architecture:\n\n- **arm64**: \`wild-cloud-central_${VERSION}_arm64.deb\` - For Raspberry Pi 4/5, ARM-based servers\n- **amd64**: \`wild-cloud-central_${VERSION}_amd64.deb\` - For x86_64 systems\n\n\`\`\`bash\n# Install package\nwget https://git.civilsociety.dev/wild-cloud/wild-cloud/releases/download/v${VERSION}/wild-cloud-central_${VERSION}_amd64.deb\nsudo dpkg -i wild-cloud-central_${VERSION}_amd64.deb\nsudo apt-get install -f\n\n# Start service\nsudo systemctl enable wild-cloud-central\nsudo systemctl start wild-cloud-central\n\`\`\`\n\n#### Standalone Daemon Binary\n\nFor Docker, Kubernetes, or custom deployments:\n\n- **arm64**: \`wild-cloud-central-arm64\`\n- **amd64**: \`wild-cloud-central-amd64\`\n\n\`\`\`bash\n# Download and run\nwget https://git.civilsociety.dev/wild-cloud/wild-cloud/releases/download/v${VERSION}/wild-cloud-central-amd64\nchmod +x wild-cloud-central-amd64\n./wild-cloud-central-amd64\n\`\`\`\n\n### Package Contents\n\n- Wild Cloud Central API daemon\n- Web-based management interface\n- CLI tools\n- systemd service configuration\n- nginx configuration\n- dnsmasq integration\n\n### Verification\n\nVerify downloads with SHA256 checksums:\n\n\`\`\`bash\nwget https://git.civilsociety.dev/wild-cloud/wild-cloud/releases/download/v${VERSION}/SHA256SUMS\nsha256sum -c SHA256SUMS\n\`\`\`",
  "draft": false,
  "prerelease": false
}
EOF
)

    RELEASE_RESPONSE=$(curl -s -X POST -H "Authorization: token ${TOKEN}" \
        -H "Content-Type: application/json" \
        -d "${RELEASE_DATA}" \
        "${API_URL}/repos/${OWNER}/${REPO}/releases")

    RELEASE_ID=$(echo "${RELEASE_RESPONSE}" | jq -r '.id')

    if [ -z "$RELEASE_ID" ] || [ "$RELEASE_ID" = "null" ]; then
        echo "❌ Failed to create release"
        echo "${RELEASE_RESPONSE}" | jq .
        exit 1
    fi

    echo "✅ Created release ${TAG} (ID: ${RELEASE_ID})"
fi

# Generate checksums
echo ""
echo "🔐 Generating checksums..."
cd dist
sha256sum packages/*.deb bin/wild-cloud-central-* > SHA256SUMS
echo "✅ Created SHA256SUMS"
cd ..

# Upload artifacts
echo ""
echo "📤 Uploading release artifacts..."

# Function to upload a file
upload_file() {
    local FILE="$1"
    local FILENAME=$(basename "$FILE")

    echo "   Uploading ${FILENAME}..."

    UPLOAD_RESPONSE=$(curl -s -X POST \
        -H "Authorization: token ${TOKEN}" \
        -F "attachment=@${FILE}" \
        "${API_URL}/repos/${OWNER}/${REPO}/releases/${RELEASE_ID}/assets?name=${FILENAME}")

    ASSET_ID=$(echo "${UPLOAD_RESPONSE}" | jq -r '.id')

    if [ -z "$ASSET_ID" ] || [ "$ASSET_ID" = "null" ]; then
        echo "   ❌ Failed to upload ${FILENAME}"
        echo "${UPLOAD_RESPONSE}" | jq .
        return 1
    else
        echo "   ✅ Uploaded ${FILENAME}"
        return 0
    fi
}

# Upload .deb packages (only current version)
for DEB in ${PACKAGES_DIR}/wild-cloud-central_${VERSION}_*.deb; do
    if [ ! -f "$DEB" ]; then
        echo "⚠️  No .deb files found for version ${VERSION} in ${PACKAGES_DIR}"
        exit 1
    fi
    upload_file "$DEB"
done

# Upload standalone binaries
for BINARY in ${BINARIES_DIR}/wild-cloud-central-*; do
    if [ -f "$BINARY" ]; then
        upload_file "$BINARY"
    fi
done

# Upload checksums
upload_file "dist/SHA256SUMS"

echo ""
echo "✨ Release complete!"
echo "🔗 View at: ${GITEA_URL}/${OWNER}/${REPO}/releases/tag/${TAG}"
