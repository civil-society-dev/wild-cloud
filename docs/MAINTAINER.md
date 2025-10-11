# Maintainer Guide

This guide covers the complete build pipeline, package creation, repository management, and deployment for Wild Cloud Central.

## Build System Overview

Wild Cloud Central uses a modern, multi-stage build system with clear separation of concerns:

1. **Build** - Compile binaries with version information
2. **Package** - Create .deb packages for distribution  
3. **Repository** - Build APT repository with GPG signing
4. **Deploy** - Upload to production server

### Quick Reference

```bash
make help          # Show all available targets
make version       # Show build information
make check         # Run quality checks (fmt + vet + test)
make clean         # Remove all build artifacts
```

## Development Workflow

### Code Quality Pipeline

Before building, always run quality checks:

```bash
make check
```

This runs:
- `go fmt` - Code formatting
- `go vet` - Static analysis  
- `go test` - Unit tests

### Building Binaries

```bash
# Build for current architecture
make build

# Build for specific architecture
make build-amd64
make build-arm64

# Build all architectures
make build-all
```

Binaries include version information from Git and build metadata.

## Package Management

### Creating Debian Packages

```bash
# Create package for current architecture
make package

# Create packages for specific architectures
make package-amd64
make package-arm64  

# Create all packages
make package-all

# Legacy alias (deprecated)
make deb
```

This creates `build/wild-cloud-central_0.1.0_amd64.deb` with:

- Binary installed to `/usr/bin/wild-cloud-central`
- Systemd service file
- Configuration template
- Web interface files
- Nginx configuration

### Package Structure

The .deb package includes:

- `/usr/bin/wild-cloud-central` - Main binary
- `/etc/systemd/system/wild-cloud-central.service` - Systemd service
- `/etc/wild-cloud-central/config.yaml.example` - Configuration template
- `/var/www/html/wild-central/` - Web interface files
- `/etc/nginx/sites-available/wild-central` - Nginx configuration

### Post-installation Setup

The package automatically:

- Creates `wildcloud` system user
- Creates required directories with proper permissions
- Configures nginx
- Enables systemd service
- Sets up file ownership

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
   VERSION := 0.2.0
   ```

2. **Quality assurance and build**:

   ```bash
   make clean           # Clean previous builds
   make check           # Run quality checks  
   make build-all       # Build all architectures
   ./tests/integration/test-docker.sh     # Integration tests
   ```

3. **Create packages and repository**:

   ```bash
   make package-all     # Create .deb packages
   make repo           # Build APT repository  
   ```

4. **Deploy**:

   ```bash
   make deploy-repo    # Upload to server
   ```

### Quick Development Release

For amd64-only development releases:

```bash
make clean && make check && make repo && make deploy-repo
```

### Multi-architecture Release

For production releases with full architecture support:

```bash
make clean && make check && make package-all && make repo && make deploy-repo
```

5. **Verify deployment**:

   ```bash
   curl -I https://mywildcloud.org/apt/dists/stable/Release
   curl -I https://mywildcloud.org/apt/wild-cloud-central.gpg
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
