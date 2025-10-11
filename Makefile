# Default target
.DEFAULT_GOAL := help

# Build configuration
BINARY_NAME := wildd
VERSION ?= 0.1.1
BUILD_DIR := build
DIST_DIR := dist
DEB_DIR := debian-package

# Go build configuration
GO_VERSION := $(shell go version | cut -d' ' -f3)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)

.PHONY: help build clean test run install package repo deploy-repo build-arm64 build-amd64 package-arm64 package-amd64 package-all check fmt vet deps-check version

help:
	@echo "🔧 Wild Daemon (wildd) Build System"
	@echo ""
	@echo "📦 Build targets:"
	@echo "  build           - Build for current architecture"
	@echo "  build-arm64     - Build arm64 binary"
	@echo "  build-amd64     - Build amd64 binary"
	@echo "  build-all       - Build all architectures"
	@echo ""
	@echo "📋 Package targets:"
	@echo "  package         - Create .deb package for current arch"
	@echo "  package-arm64   - Create arm64 .deb package"
	@echo "  package-amd64   - Create amd64 .deb package"
	@echo "  package-all     - Create all .deb packages"
	@echo ""
	@echo "🚀 Repository targets:"
	@echo "  repo            - Build APT repository from packages"
	@echo "  deploy-repo     - Deploy repository to server"
	@echo ""
	@echo "🔍 Quality assurance:"
	@echo "  check           - Run all checks (fmt + vet + test)"
	@echo "  fmt             - Format Go code"
	@echo "  vet             - Run go vet"
	@echo "  test            - Run tests"
	@echo ""
	@echo "🛠️  Development:"
	@echo "  run             - Run daemon locally"
	@echo "  dev             - Run daemon in development mode"
	@echo "  clean           - Remove all build artifacts"
	@echo "  deps-check      - Verify and tidy dependencies"
	@echo "  version         - Show build information"
	@echo "  install         - Install to system"

# Function to create debian package for specific architecture
define package_deb
	@echo "📦 Creating .deb package for $(1)..."
	@mkdir -p $(DIST_DIR)/bin $(DIST_DIR)/packages

	@# Copy debian package structure
	@cp -r debian/ $(BUILD_DIR)/$(DEB_DIR)-$(1)/

	@# Copy binary to correct location
	@cp $(BUILD_DIR)/$(2) $(BUILD_DIR)/$(DEB_DIR)-$(1)/usr/bin/$(BINARY_NAME)

	@# Copy static web files if they exist
	@if [ -d "../app/static" ]; then \
		cp -r ../app/static/* $(BUILD_DIR)/$(DEB_DIR)-$(1)/var/www/html/wild-central/; \
	fi
	@chmod 755 $(BUILD_DIR)/$(DEB_DIR)-$(1)/DEBIAN/postinst
	@chmod 755 $(BUILD_DIR)/$(DEB_DIR)-$(1)/DEBIAN/prerm
	@chmod 755 $(BUILD_DIR)/$(DEB_DIR)-$(1)/DEBIAN/postrm

	@# Substitute placeholders in control file
	@sed -i 's/VERSION_PLACEHOLDER/$(VERSION)/g' $(BUILD_DIR)/$(DEB_DIR)-$(1)/DEBIAN/control
	@sed -i 's/ARCH_PLACEHOLDER/$(1)/g' $(BUILD_DIR)/$(DEB_DIR)-$(1)/DEBIAN/control
	@sed -i 's/wild-central/$(BINARY_NAME)/g' $(BUILD_DIR)/$(DEB_DIR)-$(1)/DEBIAN/control

	@# Build package and copy to dist directories
	dpkg-deb --build $(BUILD_DIR)/$(DEB_DIR)-$(1) $(BUILD_DIR)/$(BINARY_NAME)_$(VERSION)_$(1).deb
	@cp $(BUILD_DIR)/$(2) $(DIST_DIR)/bin/$(BINARY_NAME)-$(1)
	@cp $(BUILD_DIR)/$(BINARY_NAME)_$(VERSION)_$(1).deb $(DIST_DIR)/packages/
	@echo "✅ Package created: $(DIST_DIR)/packages/$(BINARY_NAME)_$(VERSION)_$(1).deb"
	@echo "✅ Binary copied: $(DIST_DIR)/bin/$(BINARY_NAME)-$(1)"
endef

build:
	@echo "Building $(BINARY_NAME) for current architecture..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-arm64:
	@echo "Building $(BINARY_NAME) for arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-arm64 .
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)-arm64"

build-amd64:
	@echo "Building $(BINARY_NAME) for amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-amd64 .
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)-amd64"

build-all: build-arm64 build-amd64

clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) $(DEB_DIR)-* $(DEB_DIR)
	@go clean
	@echo "✅ Clean complete"

# Version information
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

install: build
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/bin/
	sudo cp wild-cloud-central.service /etc/systemd/system/wildd.service
	sudo mkdir -p /etc/wild-cloud-central
	sudo systemctl daemon-reload

# Package targets - create .deb packages
package-arm64: build-arm64
	$(call package_deb,arm64,$(BINARY_NAME)-arm64)

package-amd64: build-amd64
	$(call package_deb,amd64,$(BINARY_NAME)-amd64)

package-all: package-arm64 package-amd64

package: package-amd64

repo: package-all
	./scripts/build-apt-repository.sh

deploy-repo: repo
	./scripts/deploy-apt-repository.sh
