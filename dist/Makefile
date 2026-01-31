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

# Source paths
API_SOURCE := ../api
WEB_SOURCE := ../web

# Output files
ARM64_BINARY := $(BUILD_DIR)/wild-cloud-central-arm64
AMD64_BINARY := $(BUILD_DIR)/wild-cloud-central-amd64
WEB_DIST_MARKER := $(BUILD_DIR)/.web-built
ARM64_DEB := $(DIST_DIR)/packages/wild-cloud-central_$(VERSION)_arm64.deb
AMD64_DEB := $(DIST_DIR)/packages/wild-cloud-central_$(VERSION)_amd64.deb

# Gitea release configuration
GITEA_URL := https://git.civilsociety.dev
GITEA_OWNER := wild-cloud
GITEA_REPO := wild-cloud
RELEASE_TAG := v$(VERSION)

.PHONY: help clean version build build-all package package-all repo deploy-repo release

help:
	@echo "📦 Wild Cloud Central Package Builder"
	@echo ""
	@echo "This repository builds .deb packages from the monorepo."
	@echo "For development, use the individual component directories:"
	@echo "  - api (Go daemon)"
	@echo "  - web (React frontend)"
	@echo ""
	@echo "📦 Build targets:"
	@echo "  build           - Build for current architecture (amd64)"
	@echo "  build-all       - Build all architectures"
	@echo ""
	@echo "📋 Package targets:"
	@echo "  package         - Create .deb package (amd64)"
	@echo "  package-all     - Create all .deb packages"
	@echo ""
	@echo "🚀 Repository targets:"
	@echo "  repo            - Build APT repository from packages"
	@echo "  deploy-repo     - Deploy repository to server"
	@echo ""
	@echo "🎯 Release targets:"
	@echo "  release         - Create/update Gitea release with packages"
	@echo ""
	@echo "🛠️  Utilities:"
	@echo "  clean           - Remove all build artifacts"
	@echo "  version         - Show build information"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION         - Package version (default: 0.1.1)"
	@echo "  GITEA_TOKEN     - Gitea API token (required for release)"
	@echo "                   Run: source .envrc"

clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) $(DEB_DIR)-* $(DEB_DIR)
	@echo "✅ Clean complete"

version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# Web app build - track with marker file
$(WEB_DIST_MARKER): $(shell find $(WEB_SOURCE)/src -type f 2>/dev/null | head -100) $(WEB_SOURCE)/package.json
	@echo "🔧 Building web application..."
	@ARCH=amd64 VERSION=$(VERSION) ./scripts/build-package.sh
	@touch $(WEB_DIST_MARKER)
	@echo "✅ Web app built"

# ARM64 binary
$(ARM64_BINARY): $(shell find $(API_SOURCE) -name '*.go' 2>/dev/null | head -100)
	@echo "🔧 Building arm64 binary..."
	@mkdir -p $(BUILD_DIR)/bin
	@cd $(API_SOURCE) && GOOS=linux GOARCH=arm64 go build -ldflags="-X main.Version=$(VERSION)" -o ../dist/$(ARM64_BINARY) .
	@echo "✅ Build complete: $(ARM64_BINARY)"

# AMD64 binary
$(AMD64_BINARY): $(shell find $(API_SOURCE) -name '*.go' 2>/dev/null | head -100)
	@echo "🔧 Building amd64 binary..."
	@mkdir -p $(BUILD_DIR)/bin
	@cd $(API_SOURCE) && GOOS=linux GOARCH=amd64 go build -ldflags="-X main.Version=$(VERSION)" -o ../dist/$(AMD64_BINARY) .
	@echo "✅ Build complete: $(AMD64_BINARY)"

# ARM64 package
$(ARM64_DEB): $(ARM64_BINARY) $(WEB_DIST_MARKER)
	@echo "📦 Creating arm64 .deb package..."
	@./scripts/package-deb.sh arm64 $(ARM64_BINARY) $(VERSION)
	@echo "✅ Package created: $(ARM64_DEB)"

# AMD64 package
$(AMD64_DEB): $(AMD64_BINARY) $(WEB_DIST_MARKER)
	@echo "📦 Creating amd64 .deb package..."
	@./scripts/package-deb.sh amd64 $(AMD64_BINARY) $(VERSION)
	@echo "✅ Package created: $(AMD64_DEB)"

# Convenience targets
build: $(AMD64_BINARY) $(WEB_DIST_MARKER)

build-all: $(ARM64_BINARY) $(AMD64_BINARY) $(WEB_DIST_MARKER)

package: $(AMD64_DEB)

package-all: $(ARM64_DEB) $(AMD64_DEB)

repo: $(ARM64_DEB) $(AMD64_DEB)
	@echo "🏗️  Building APT repository..."
	@./scripts/build-apt-repository.sh

deploy-repo: repo
	@echo "🚀 Deploying APT repository..."
	@./scripts/deploy-apt-repository.sh

release: $(ARM64_DEB) $(AMD64_DEB)
	@echo "🎯 Creating release $(RELEASE_TAG)..."
	@if [ -z "$(GITEA_TOKEN)" ]; then \
		echo "❌ Error: GITEA_TOKEN environment variable not set"; \
		echo "   Get a token from $(GITEA_URL)/user/settings/applications"; \
		echo "   Or run: source .envrc"; \
		exit 1; \
	fi
	@./scripts/create-gitea-release.sh "$(GITEA_URL)" "$(GITEA_OWNER)" "$(GITEA_REPO)" "$(RELEASE_TAG)" "$(VERSION)" "$(GITEA_TOKEN)"
