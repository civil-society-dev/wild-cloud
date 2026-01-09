# Default target
.DEFAULT_GOAL := help

# Build configuration
VERSION ?= 0.1.1
BUILD_DIR := build
DIST_DIR := dist
DEB_DIR := debian-package

# Git information
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

.PHONY: help build clean package repo deploy-repo build-arm64 build-amd64 package-arm64 package-amd64 package-all version

help:
	@echo "📦 Wild Cloud Central Package Builder"
	@echo ""
	@echo "This repository builds .deb packages from the monorepo."
	@echo "For development, use the individual component directories:"
	@echo "  - api (Go daemon)"
	@echo "  - wild-web-app (React frontend)"
	@echo ""
	@echo "📦 Build targets:"
	@echo "  build           - Build for current architecture (amd64)"
	@echo "  build-arm64     - Build arm64 binary and web app"
	@echo "  build-amd64     - Build amd64 binary and web app"
	@echo "  build-all       - Build all architectures"
	@echo ""
	@echo "📋 Package targets:"
	@echo "  package         - Create .deb package (amd64)"
	@echo "  package-arm64   - Create arm64 .deb package"
	@echo "  package-amd64   - Create amd64 .deb package"
	@echo "  package-all     - Create all .deb packages"
	@echo ""
	@echo "🚀 Repository targets:"
	@echo "  repo            - Build APT repository from packages"
	@echo "  deploy-repo     - Deploy repository to server"
	@echo ""
	@echo "🛠️  Utilities:"
	@echo "  clean           - Remove all build artifacts"
	@echo "  version         - Show build information"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION         - Package version (default: 0.1.1)"

# Function to create debian package for specific architecture
define package_deb
	@echo "📦 Creating .deb package for $(1)..."
	@mkdir -p $(DIST_DIR)/bin $(DIST_DIR)/packages

	@# Copy debian package structure
	@cp -r debian/ $(BUILD_DIR)/$(DEB_DIR)-$(1)/

	@# Copy binary to correct location
	@mkdir -p $(BUILD_DIR)/$(DEB_DIR)-$(1)/usr/bin
	@cp $(BUILD_DIR)/$(2) $(BUILD_DIR)/$(DEB_DIR)-$(1)/usr/bin/wild-cloud-central

	@# Copy static web files from built web app
	@mkdir -p $(BUILD_DIR)/$(DEB_DIR)-$(1)/var/www/html/wild-central
	@if [ -d "$(BUILD_DIR)/src/wild-web-app/dist" ]; then \
		cp -r $(BUILD_DIR)/src/wild-web-app/dist/* $(BUILD_DIR)/$(DEB_DIR)-$(1)/var/www/html/wild-central/; \
		echo "✅ Copied web app files"; \
	else \
		echo "⚠️  Warning: Web app dist not found at $(BUILD_DIR)/src/wild-web-app/dist"; \
	fi

	@# Set script permissions
	@chmod 755 $(BUILD_DIR)/$(DEB_DIR)-$(1)/DEBIAN/postinst
	@chmod 755 $(BUILD_DIR)/$(DEB_DIR)-$(1)/DEBIAN/prerm
	@chmod 755 $(BUILD_DIR)/$(DEB_DIR)-$(1)/DEBIAN/postrm

	@# Substitute placeholders in control file
	@sed -i 's/VERSION_PLACEHOLDER/$(VERSION)/g' $(BUILD_DIR)/$(DEB_DIR)-$(1)/DEBIAN/control
	@sed -i 's/ARCH_PLACEHOLDER/$(1)/g' $(BUILD_DIR)/$(DEB_DIR)-$(1)/DEBIAN/control

	@# Build package and copy to dist directories
	dpkg-deb --build $(BUILD_DIR)/$(DEB_DIR)-$(1) $(BUILD_DIR)/wild-cloud-central_$(VERSION)_$(1).deb
	@cp $(BUILD_DIR)/$(2) $(DIST_DIR)/bin/wild-cloud-central-$(1)
	@cp $(BUILD_DIR)/wild-cloud-central_$(VERSION)_$(1).deb $(DIST_DIR)/packages/
	@echo "✅ Package created: $(DIST_DIR)/packages/wild-cloud-central_$(VERSION)_$(1).deb"
	@echo "✅ Binary copied: $(DIST_DIR)/bin/wild-cloud-central-$(1)"
endef

build: build-amd64

build-arm64:
	@echo "Building for arm64..."
	@ARCH=arm64 VERSION=$(VERSION) ./scripts/build-package.sh
	@mv $(BUILD_DIR)/bin/wild-cloud-central $(BUILD_DIR)/wild-cloud-central-arm64
	@echo "✅ Build complete: $(BUILD_DIR)/wild-cloud-central-arm64"

build-amd64:
	@echo "Building for amd64..."
	@ARCH=amd64 VERSION=$(VERSION) ./scripts/build-package.sh
	@mv $(BUILD_DIR)/bin/wild-cloud-central $(BUILD_DIR)/wild-cloud-central-amd64
	@echo "✅ Build complete: $(BUILD_DIR)/wild-cloud-central-amd64"

build-all: build-arm64 build-amd64

clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) $(DEB_DIR)-* $(DEB_DIR)
	@echo "✅ Clean complete"

# Version information
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# Package targets - create .deb packages
package-arm64: build-arm64
	$(call package_deb,arm64,wild-cloud-central-arm64)

package-amd64: build-amd64
	$(call package_deb,amd64,wild-cloud-central-amd64)

package-all: package-arm64 package-amd64

package: package-amd64

repo: package-all
	./scripts/build-apt-repository.sh

deploy-repo: repo
	./scripts/deploy-apt-repository.sh
