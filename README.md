# Packaging Wild Central

## Desired Experience

This is the desired experience for installing Wild Cloud Central on a fresh Debian/Ubuntu system:

### APT Repository (Recommended)

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

### Manual Installation

Download the latest `.deb` package from the [releases page](https://github.com/wildcloud/wild-central/releases) and install:

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
