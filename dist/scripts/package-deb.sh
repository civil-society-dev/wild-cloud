#!/bin/bash
set -euo pipefail

ARCH="$1"
BINARY_PATH="$2"
VERSION="$3"

BUILD_DIR="build"
DIST_DIR="dist"
DEB_DIR="debian-package"
WEB_DIST="${BUILD_DIR}/src/web/dist"

echo "📦 Creating .deb package for ${ARCH}..."

# Create directories
mkdir -p "${DIST_DIR}/bin" "${DIST_DIR}/packages"

# Copy debian package structure
cp -r debian/ "${BUILD_DIR}/${DEB_DIR}-${ARCH}/"

# Copy binary to correct location
mkdir -p "${BUILD_DIR}/${DEB_DIR}-${ARCH}/usr/bin"
cp "${BINARY_PATH}" "${BUILD_DIR}/${DEB_DIR}-${ARCH}/usr/bin/wild-cloud-central"

# Copy static web files from built web app
mkdir -p "${BUILD_DIR}/${DEB_DIR}-${ARCH}/var/www/html/wild-central"
if [ -d "${WEB_DIST}" ]; then
    cp -r "${WEB_DIST}"/* "${BUILD_DIR}/${DEB_DIR}-${ARCH}/var/www/html/wild-central/"
    echo "✅ Copied web app files"
else
    echo "⚠️  Warning: Web app dist not found at ${WEB_DIST}"
fi

# Set script permissions
chmod 755 "${BUILD_DIR}/${DEB_DIR}-${ARCH}/DEBIAN/postinst"
chmod 755 "${BUILD_DIR}/${DEB_DIR}-${ARCH}/DEBIAN/prerm"
chmod 755 "${BUILD_DIR}/${DEB_DIR}-${ARCH}/DEBIAN/postrm"

# Substitute placeholders in control file
sed -i "s/VERSION_PLACEHOLDER/${VERSION}/g" "${BUILD_DIR}/${DEB_DIR}-${ARCH}/DEBIAN/control"
sed -i "s/ARCH_PLACEHOLDER/${ARCH}/g" "${BUILD_DIR}/${DEB_DIR}-${ARCH}/DEBIAN/control"

# Build package and copy to dist directories
dpkg-deb --build "${BUILD_DIR}/${DEB_DIR}-${ARCH}" "${BUILD_DIR}/wild-cloud-central_${VERSION}_${ARCH}.deb"
cp "${BINARY_PATH}" "${DIST_DIR}/bin/wild-cloud-central-${ARCH}"
cp "${BUILD_DIR}/wild-cloud-central_${VERSION}_${ARCH}.deb" "${DIST_DIR}/packages/"

echo "✅ Package created: ${DIST_DIR}/packages/wild-cloud-central_${VERSION}_${ARCH}.deb"
echo "✅ Binary copied: ${DIST_DIR}/bin/wild-cloud-central-${ARCH}"
