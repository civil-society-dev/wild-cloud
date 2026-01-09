# Maintainer Guide

This guide covers the complete build pipeline, package creation, repository management, and deployment for Wild Cloud Central.

## Build System Overview

Wild Cloud Central uses a **self-contained packaging approach** that clones source repositories and builds packages independently:

1. **Clone** - Fetch wild-central-api (Go) and wild-web-app (React) from git
2. **Build** - Compile binaries and build web assets with version information
3. **Package** - Create .deb packages for distribution
4. **Repository** - Build APT repository with GPG signing
5. **Deploy** - Upload to production server

**Important**: This repository is for **packaging only**. Development happens in the individual component repositories (wild-central-api, wild-web-app).

### Quick Reference

```bash
make help          # Show all available targets
make version       # Show build information
make clean         # Remove all build artifacts
```

## Repository Philosophy

This repository **does not contain source code**. Instead, it:
- Clones source repos during package build
- Compiles binaries and web assets
- Creates distribution packages
- Manages APT repository

For development work, use the individual repositories:
- `wild-central-api` - Go daemon development
- `wild-web-app` - React frontend development

### Building Packages

Requirements:

```bash
sudo apt-get update && sudo apt-get install -y aptly
```

The build process automatically:
1. Clones source repositories (wild-central-api, wild-web-app)
2. Builds Go daemon binary for target architecture
3. Builds React web application with production optimizations
4. Creates .deb package with all components

```bash
# Build for current architecture (auto-detected)
make build           # Builds binary only
make package         # Builds and packages (recommended)

# Build for specific architecture
make build-amd64     # Build amd64 binary
make build-arm64     # Build arm64 binary
make package-amd64   # Build and package for amd64
make package-arm64   # Build and package for arm64

# Build all architectures
make build-all       # Build binaries for all architectures
make package-all     # Build and package for all architectures
```

**Environment Variables**:
- `VERSION` - Package version (default: from Makefile)
- `API_REF` - wild-central-api git ref (default: main)
- `WEBAPP_REF` - wild-web-app git ref (default: main)

**Example - Build from specific branches**:
```bash
API_REF=v0.2.0 WEBAPP_REF=v0.2.0 VERSION=0.2.0 make package-all
```

## Package Management

### Package Build Output

Packages are created in two locations:
- `build/wild-cloud-central_VERSION_ARCH.deb` - Build artifact
- `dist/packages/wild-cloud-central_VERSION_ARCH.deb` - Distribution copy
- `dist/bin/wild-cloud-central-ARCH` - Standalone binary copy

### Testing Packages

Before deploying, always test packages using Docker:

```bash
# Build and test (recommended)
./tests/build-and-test.sh

# Or manually
make package-arm64  # or package-amd64
docker build -t wild-cloud-central-test .
docker run --rm wild-cloud-central-test
```

The test suite validates:
- Package installation
- File placement
- User/group creation
- DNS configuration
- Polkit rules
- API health endpoint
- Web interface accessibility

### Package Structure

The .deb package includes:

- `/usr/bin/wild-cloud-central` - Main binary
- `/etc/systemd/system/wild-cloud-central.service` - Systemd service
- `/etc/wild-cloud-central/config.yaml.example` - Configuration template
- `/var/www/html/wild-central/` - Web interface files
- `/etc/nginx/sites-available/wild-central` - Nginx configuration

### Post-installation Setup

The package automatically:

- Creates `wildcloud` system user and group
- Creates required directories with proper permissions:
  - `/etc/wild-cloud-central/` - Configuration directory
  - `/var/lib/wild-cloud-central/` - Data directory
  - `/var/log/wild-cloud-central/` - Log directory
  - `/var/www/html/talos/` - Talos boot assets
  - `/var/ftpd/` - TFTP directory
- Sets up DNS configuration:
  - Creates `/etc/systemd/resolved.conf.d/wild-cloud.conf` (owned by wildcloud)
  - Configures `/etc/resolv.conf` symlink (if not in container)
- Installs polkit rules for service management
- Configures nginx reverse proxy
- Enables systemd service (if systemd present)
- Sets up proper file ownership and permissions

**Container-Friendly**: Installation script detects container environments and gracefully handles:
- Read-only `/etc/resolv.conf`
- Absence of systemd
- Bind-mounted files

## APT Repository Management

### Building Repository

```bash
make repo
```

This uses `./scripts/build-apt-repository.sh` with **aptly** to create a professional APT repository in `dist/repositories/apt/`:

- Complete repository metadata with all hash types (MD5, SHA1, SHA256, SHA512)
- Contents files for enhanced package discovery
- Multiple compression formats (.gz, .bz2) for compatibility
- Proper GPG signing with modern InRelease format
- Industry-standard repository structure following Debian conventions

The repository includes:
- `pool/main/w/wild-cloud-central/` - Package files
- `dists/stable/main/binary-amd64/` - Metadata and package lists  
- `dists/stable/main/binary-arm64/` - ARM64 package metadata
- `dists/stable/InRelease` - Modern GPG signature (preferred)
- `dists/stable/Release.asc` - Traditional GPG signature compatibility
- `wild-cloud-central.gpg` - GPG public key for users

### Aptly Configuration

The build system automatically configures aptly to:
- Use strong RSA 4096-bit GPG keys
- Generate complete security metadata to prevent "weak security information" warnings
- Create Contents files for better package discovery
- Support multiple architectures (amd64, arm64)

### GPG Key Management

#### First-time Setup

```bash
./scripts/setup-gpg.sh
```

This creates:

- 4096-bit RSA GPG key pair
- Public key exported as `dist/wild-cloud-central.gpg` (binary format for APT)
- Key configured for 2-year expiration
- Automatic aptly configuration for repository signing

#### Key Renewal

When the key expires, regenerate with:

```bash
gpg --delete-secret-keys "Wild Cloud Central"
gpg --delete-keys "Wild Cloud Central" 
make clean  # Remove old GPG key and aptly state
./scripts/setup-gpg.sh
```

### Repository Deployment

1. **Configure server details** in `scripts/deploy-apt-repository.sh`:

   ```bash
   SERVER="user@mywildcloud.org"
   REMOTE_PATH="/var/www/html/apt"
   ```

2. **Deploy repository**:

   ```bash
   make deploy-repo
   ```

This uploads the aptly-generated repository with complete security metadata, eliminating "weak security information" warnings and ensuring compatibility with modern APT security standards.

This uploads:

- Complete repository structure to server
- GPG public key for user verification
- Proper file permissions and structure

### Server Requirements

The target server needs:

- Web server (nginx/apache) serving `/var/www/html/apt`
- HTTPS support for `https://mywildcloud.org/apt`
- SSH access for deployment

### Repository Structure

```
/var/www/html/apt/
├── dists/
│   └── stable/
│       ├── InRelease (modern GPG signature)
│       ├── Release
│       ├── Release.asc
│       └── main/
│           ├── binary-amd64/
│           │   ├── Packages
│           │   ├── Packages.gz
│           │   └── Release
│           └── binary-arm64/
│               ├── Packages
│               ├── Packages.gz
│               └── Release
├── pool/
│   └── main/
│       └── w/
│           └── wild-cloud-central/
│               ├── wild-cloud-central_0.1.0_amd64.deb
│               └── wild-cloud-central_0.1.0_arm64.deb
├── Contents-amd64 (enhanced package discovery)
├── Contents-amd64.gz
└── wild-cloud-central.gpg (binary format for APT)
```

## Release Process

### Standard Release

1. **Update version** in `Makefile`:

   ```makefile
   VERSION ?= 0.2.0
   ```

2. **Build and test**:

   ```bash
   make clean              # Clean previous builds
   make package-all        # Build all architectures
   ./tests/build-and-test.sh  # Test package installation
   ```

3. **Create repository**:

   ```bash
   make repo              # Build APT repository
   ```

4. **Deploy**:

   ```bash
   make deploy-repo       # Upload to server
   ```

5. **Verify deployment**:

   ```bash
   curl -I https://mywildcloud.org/apt/dists/stable/Release
   curl -I https://mywildcloud.org/apt/wild-cloud-central.gpg
   ```

### Quick Single-Architecture Release

For rapid testing on one architecture:

```bash
make clean && make package-arm64 && ./tests/build-and-test.sh
```

### Production Release

For full production releases with testing:

```bash
# Clean and build all
make clean
make package-all

# Test each architecture
./tests/build-and-test.sh  # Tests native architecture

# Build repository and deploy
make repo
make deploy-repo
```

### Building from Specific Git References

To build from specific commits or tags:

```bash
API_REF=v1.2.3 WEBAPP_REF=v1.2.3 VERSION=1.2.3 make package-all
```

## User Installation

Users install packages using the modern APT `.sources` format:

```bash
# Download and install GPG key (binary format)
curl -fsSL https://mywildcloud.org/apt/wild-cloud-central.gpg | \
  sudo tee /usr/share/keyrings/wild-cloud-central-archive-keyring.gpg > /dev/null

# Add repository using modern .sources format
sudo tee /etc/apt/sources.list.d/wild-cloud-central.sources << 'EOF'
Types: deb
URIs: https://mywildcloud.org/apt
Suites: stable  
Components: main
Signed-By: /usr/share/keyrings/wild-cloud-central-archive-keyring.gpg
EOF

# Update and install
sudo apt update
sudo apt install wild-cloud-central
```

### Legacy Installation (Deprecated)

The old `.list` format still works but generates warnings:

```bash
# Download GPG key (requires conversion)
curl -fsSL https://mywildcloud.org/apt/wild-cloud-central.gpg | \
  sudo gpg --dearmor -o /usr/share/keyrings/wild-cloud-central.gpg

# Add repository using legacy format (deprecated)
echo 'deb [signed-by=/usr/share/keyrings/wild-cloud-central.gpg] https://mywildcloud.org/apt stable main' | \
  sudo tee /etc/apt/sources.list.d/wild-cloud-central.list
```

## Troubleshooting

### Build Issues

- **"pnpm not found"**: Install pnpm: `curl -fsSL https://get.pnpm.io/install.sh | sh -`
- **"go: command not found"**: Install Go 1.21+
- **TypeScript errors**: Build script automatically falls back to skipping type checking
- **Clone failures**: Check internet connectivity and git repository URLs
- **Architecture mismatch**: Ensure building for correct architecture (arm64 vs amd64)

### Package Issues

- **"dpkg-deb: error"**: Check that `debian/` structure is complete
- **Wrong architecture**: Use `make package-arm64` or `make package-amd64` explicitly
- **Missing files in package**: Verify `build/src/wild-web-app/dist/` exists after build

### Docker Test Issues

- **"No .deb package found"**: Run `make package-arm64` (or appropriate arch) first
- **"permission denied" (Docker)**: Add user to docker group: `sudo usermod -aG docker $USER`
- **Container fails to start**: Check Docker daemon is running
- **Test timeout**: Services may need more time to start on slower systems

### GPG Issues

- **"no default secret key"**: Run `./scripts/setup-gpg.sh`
- **Key conflicts**: Delete existing keys before recreating
- **Permission errors**: Ensure `~/.gnupg` has correct permissions (700)

### Repository Issues

- **Package not found**: Verify `dpkg-scanpackages` output
- **Signature verification failed**: Regenerate GPG key and re-sign
- **404 errors**: Check web server configuration and file permissions
- **Legacy format warnings**: Use modern `.sources` format instead of `.list`
- **GPG key mismatch**: Ensure deployed key matches signing key

### Deployment Issues

- **SSH failures**: Verify server credentials in `deploy-repo.sh`
- **Permission denied**: Ensure target directory is writable
- **rsync errors**: Check network connectivity and paths

## Monitoring

### Service Health

```bash
curl https://mywildcloud.org/apt/dists/stable/Release
curl https://mywildcloud.org/apt/wild-cloud-central.gpg
```

### Package Statistics

Monitor download statistics through web server logs:

```bash
grep "wild-cloud-central.*\.deb" /var/log/nginx/access.log | wc -l
```

### Repository Integrity

Verify signatures regularly:

```bash
gpg --verify Release.asc Release
```

## Docker Testing Infrastructure

### Overview

The `tests/` directory provides comprehensive Docker-based testing to validate package installation without affecting your system.

### Test Structure

```
tests/
├── build-and-test.sh           # Main test runner (auto-detects architecture)
├── test-installation.sh        # Installation verification script
└── integration/
    ├── test-docker.sh          # Simple test runner
    ├── start-interactive.sh    # Interactive testing with live services
    ├── start-background.sh     # Background daemon for testing
    ├── debug-container.sh      # Shell access for debugging
    └── stop-background.sh      # Stop background services
```

### Running Tests

**Recommended - Automated Testing**:
```bash
./tests/build-and-test.sh
```

This script:
1. Detects your architecture (arm64/amd64)
2. Builds package if needed (or asks to reuse existing)
3. Builds Docker test image
4. Runs full installation test suite

**Manual Testing**:
```bash
# Build package first
make package-arm64  # or package-amd64

# Run tests
./tests/integration/test-docker.sh
```

**Interactive Testing**:
```bash
./tests/integration/start-interactive.sh
# Access at http://localhost:9080 (web) and http://localhost:9081 (API)
# Press Ctrl+C to stop
```

**Background Testing**:
```bash
./tests/integration/start-background.sh
# Services run in background
# Test with: curl http://localhost:9081/api/v1/health
./tests/integration/stop-background.sh  # When done
```

**Debug Mode**:
```bash
./tests/integration/debug-container.sh
# Opens shell inside container with package installed
# Start services manually: /test-installation.sh
```

### What Tests Validate

The test suite checks:

**Installation**:
- ✅ Binary placement (`/usr/bin/wild-cloud-central`)
- ✅ Configuration files
- ✅ Systemd service file
- ✅ Nginx configuration
- ✅ Web assets deployment
- ✅ Polkit rules installation
- ✅ systemd-resolved configuration
- ✅ DNS config file ownership
- ✅ wildcloud user and group creation

**Runtime**:
- ✅ API health endpoint (`/api/v1/health`)
- ✅ Configuration endpoint (`/api/v1/config`)
- ✅ Dnsmasq config endpoint (`/api/v1/dnsmasq/config`)
- ✅ Web interface accessibility through nginx

### Dockerfile Architecture

The `Dockerfile` uses a **pre-built package approach**:
- Does NOT build from source
- Copies pre-built `.deb` package from `build/` directory
- Tests actual package users will install
- Container-friendly (handles read-only files, missing systemd)

**Key Features**:
- Auto-detects architecture
- Gracefully handles container limitations
- Validates all package components
- Tests service startup and API functionality

### Test Output Example

```
🚀 Testing wild-cloud-central installation...
✅ Checking binary installation...
✅ Checking configuration...
✅ Checking systemd service...
✅ Checking nginx configuration...
✅ Checking web assets...
✅ Checking polkit rules...
✅ Checking systemd-resolved configuration...
✅ Checking wildcloud user...
🔧 Starting services...
✅ Health check passed
✅ Configuration endpoint working
✅ Dnsmasq config endpoint working
✅ Web interface accessible
🎉 All installation tests passed!
```

### Continuous Integration

Tests should be run:
- Before every release
- After modifying package structure
- After changing postinst/prerm/postrm scripts
- When updating dependencies

### Adding New Tests

To add tests to `test-installation.sh`:

1. Add verification check after existing checks
2. Follow the pattern: echo message, test condition, handle failure
3. Update this documentation

Example:
```bash
echo "✅ Checking new feature..."
if [ -f "/path/to/feature" ]; then
    echo "   Feature installed"
else
    echo "❌ Feature not found"
    exit 1
fi
```
