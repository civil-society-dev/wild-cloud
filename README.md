# Wild Cloud

## Installing Wild Central

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

## Features

- **Web Management Interface** - Browser-based configuration and monitoring
- **REST API** - JSON API for programmatic management
- **DNS/DHCP Services** - Integrated dnsmasq configuration management
- **PXE Boot Support** - Automatic Talos Linux asset downloading and serving

## Basic Configuration

The service uses `/etc/wild-cloud-central/config.yaml` for configuration:

```yaml
cloud:
  domain: "wildcloud.local"
  dns:
    ip: "192.168.8.50" # Your server's IP
  dhcpRange: "192.168.8.100,192.168.8.200"

cluster:
  endpointIp: "192.168.8.60" # Talos cluster endpoint
  nodes:
    talos:
      version: "v1.8.0" # Talos version to use
```

## Service Management

```bash
# Check status
sudo systemctl status wild-cloud-central

# View logs
sudo journalctl -u wild-cloud-central -f

# Restart service
sudo systemctl restart wild-cloud-central

# Stop service
sudo systemctl stop wild-cloud-central
```

## Support

- **Documentation**: See `docs/` directory for detailed guides
- **Issues**: Report problems on the project issue tracker
- **API Reference**: Available at `/api/v1/` endpoints when service is running

## Documentation

- [Developer Guide](docs/DEVELOPER.md) - Development setup, testing, and API reference
- [Maintainer Guide](docs/MAINTAINER.md) - Package management and repository deployment
