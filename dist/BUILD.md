# Building Wild Cloud Central

This repository contains the packaging and distribution infrastructure for Wild Cloud Central.

## Prerequisites

- Go 1.21+ (for building the daemon)
- Node.js 20+ and pnpm (for building the web app)
- dpkg-deb (for creating .deb packages)
- git

## Quick Start

```bash
# Build for current architecture (amd64)
make package

# Build for all architectures
make package-all

# Build APT repository
make repo
```

## Build Process

The build process:

1. **Uses source from monorepo:**
   - `../api` - Go daemon
   - `../web` - React web interface

2. **Builds artifacts:**
   - Compiles Go daemon binary
   - Builds React production bundle

3. **Creates .deb packages:**
   - Packages binary and web app
   - Includes systemd service files
   - Adds postinst/prerm/postrm scripts

## Build Targets

- `make build` - Build for amd64
- `make build-arm64` - Build for arm64
- `make build-amd64` - Build for amd64
- `make build-all` - Build for both architectures
- `make package` - Create .deb package (amd64)
- `make package-arm64` - Create arm64 .deb package
- `make package-amd64` - Create amd64 .deb package
- `make package-all` - Create all .deb packages
- `make repo` - Build APT repository from packages
- `make clean` - Remove all build artifacts

## Version Control

Set the version when building:

```bash
VERSION=0.2.0 make package-all
```

## Build from Specific Commits

```bash
# Build from specific git refs
API_REF=v0.2.0 WEBAPP_REF=v0.2.0 make build-all
```

## Output

Built packages are placed in:
- `dist/packages/` - .deb package files
- `dist/bin/` - Binary executables
- `build/src/` - Cloned source repositories (gitignored)

## Testing Packages Locally

```bash
# Install the package
sudo dpkg -i dist/packages/wild-cloud-central_0.1.1_amd64.deb

# Fix dependencies if needed
sudo apt-get install -f

# Start the service
sudo systemctl start wild-cloud-central

# Check status
sudo systemctl status wild-cloud-central
```

## Troubleshooting

**pnpm not found:**
```bash
curl -fsSL https://get.pnpm.io/install.sh | sh -
```

**Go not installed:**
```bash
sudo apt install golang-go
```

**Build fails to clone repos:**
- Check internet connectivity
- Verify git repository URLs are accessible
