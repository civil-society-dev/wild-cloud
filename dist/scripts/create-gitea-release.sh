#!/bin/bash
set -euo pipefail

GITEA_URL="$1"
OWNER="$2"
REPO="$3"
TAG="$4"
VERSION="$5"
TOKEN="$6"

API_URL="${GITEA_URL}/api/v1"
DIST_DIR="dist/packages"

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
    echo "🗑️  Deleting existing release to update..."
    curl -s -X DELETE -H "Authorization: token ${TOKEN}" \
        "${API_URL}/repos/${OWNER}/${REPO}/releases/${RELEASE_ID}"
    echo "✅ Deleted existing release"
fi

# Create new release
echo "📝 Creating new release ${TAG}..."
RELEASE_DATA=$(cat <<EOF
{
  "tag_name": "${TAG}",
  "name": "Wild Cloud Central ${VERSION}",
  "body": "## Wild Cloud Central ${VERSION}\n\n### Installation\n\nDownload the appropriate .deb package for your architecture:\n\n- **arm64**: For Raspberry Pi 4/5, ARM-based servers\n- **amd64**: For x86_64 systems\n\n\`\`\`bash\n# Install package\nsudo dpkg -i wild-cloud-central_${VERSION}_<arch>.deb\nsudo apt-get install -f\n\n# Start service\nsudo systemctl enable wild-cloud-central\nsudo systemctl start wild-cloud-central\n\`\`\`\n\n### What's Included\n\n- Wild Cloud Central API daemon\n- Web-based management interface\n- CLI tools\n- systemd service configuration",
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

# Upload packages
echo ""
echo "📤 Uploading packages..."

for DEB in ${DIST_DIR}/*.deb; do
    if [ ! -f "$DEB" ]; then
        echo "⚠️  No .deb files found in ${DIST_DIR}"
        exit 1
    fi

    FILENAME=$(basename "$DEB")
    echo "   Uploading ${FILENAME}..."

    UPLOAD_RESPONSE=$(curl -s -X POST \
        -H "Authorization: token ${TOKEN}" \
        -F "attachment=@${DEB}" \
        "${API_URL}/repos/${OWNER}/${REPO}/releases/${RELEASE_ID}/assets?name=${FILENAME}")

    ASSET_ID=$(echo "${UPLOAD_RESPONSE}" | jq -r '.id')

    if [ -z "$ASSET_ID" ] || [ "$ASSET_ID" = "null" ]; then
        echo "   ❌ Failed to upload ${FILENAME}"
        echo "${UPLOAD_RESPONSE}" | jq .
    else
        echo "   ✅ Uploaded ${FILENAME} (Asset ID: ${ASSET_ID})"
    fi
done

echo ""
echo "✨ Release complete!"
echo "🔗 View at: ${GITEA_URL}/${OWNER}/${REPO}/releases/tag/${TAG}"
