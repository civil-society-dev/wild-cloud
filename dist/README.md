# Packaging Wild Central

## Installation

Download the latest `.deb` package from the [releases page](https://git.civilsociety.dev/wild-cloud/wild-cloud/releases) and install:

```bash
sudo dpkg -i wild-cloud-central_*.deb
sudo apt-get install -f  # Fix any dependency issues
```

## Quick Start

1. **Configure the service** (optional):

   ```bash
   sudo cp /etc/wild-cloud-central/config.yaml.example /etc/wild-cloud-central/config.yaml
   sudo nano /etc/wild-cloud-central/config.yaml
   ```

2. **Start the service**:

   ```bash
   sudo systemctl enable wild-cloud-central
   sudo systemctl start wild-cloud-central
   ```

3. **Access the web interface**:
   Open http://your-server-ip in your browser

## Version Management

Wild Cloud uses a single VERSION file at the monorepo root (`wild-cloud/VERSION`) that tracks the unified version for all components (API, CLI, Web, and the package itself).

### Current Version
```bash
cat ../VERSION  # Shows current version
```

### Development Workflow (Testing Phase)

During development, iterate on the same version:

```bash
# Make changes to code
make package-all

# Update existing release (no version bump)
make release  # Updates assets in current release
```

### Creating a New Release

When ready for a new release:

```bash
# 1. Bump version
echo "0.1.2" > ../VERSION

# 2. Build packages
make package-all

# 3. Create new release
make release  # Creates v0.1.2 tag and release

# 4. Commit and tag
cd ..
git add VERSION
git commit -m "Bump version to 0.1.2"
git tag -a v0.1.2 -m "Wild Cloud Central v0.1.2"
git push origin main v0.1.2
```

### Release Behavior

The `make release` command automatically:
- **If release exists**: Updates release assets (deletes old files, uploads new ones)
- **If release doesn't exist**: Creates new release with all assets

This allows iterating during testing without creating multiple releases.

### Release Artifacts

Each release includes:
- **Debian Packages** (`.deb`)
  - `wild-cloud-central_VERSION_amd64.deb` - Full installation for x86_64
  - `wild-cloud-central_VERSION_arm64.deb` - Full installation for ARM64
- **Standalone Binaries**
  - `wild-cloud-central-amd64` - API daemon only (for Docker/K8s)
  - `wild-cloud-central-arm64` - API daemon only (for Docker/K8s)
- **Checksums**
  - `SHA256SUMS` - Verify all artifacts

### Future: Independent Component Versioning

For future enhancement when components need independent versions, see `future/independent-versioning.md`.

## Developer tooling

Makefile commands for packaging:

Package targets (create .deb packages):

make package         - Create .deb package for current arch
make package-arm64   - Create arm64 .deb package
make package-amd64   - Create amd64 .deb package
make package-all     - Create all .deb packages

Repository targets:

make repo            - Build APT repository from packages
make deploy-repo     - Deploy repository to server

Directory structure:

build/          - Intermediate build artifacts
dist/bin/       - Final binaries for distribution
dist/packages/  - OS packages (.deb files)
dist/repositories/ - APT repository for deployment

Example workflows:
make clean && make repo      - Full release build

## Future packaging

We'll be putting the packages in a proper repository in the future for installing Wild Cloud Central on a fresh Debian/Ubuntu system:

### APT Repository (TBD)

```bash
# Download and install GPG key
curl -fsSL https://mywildcloud.org/apt/wild-cloud-central.gpg | sudo tee /usr/share/keyrings/wild-cloud-central-archive-keyring.gpg > /dev/null

# Add repository (modern .sources format)
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
