# Developer Guide

This guide covers development, testing, and local building of Wild Cloud Central.

## Development Setup

### Prerequisites

- Go 1.21+
- Docker (for testing)
- make

```bash
sudo apt update
sudo apt install make direnv
echo 'eval "$(direnv hook bash)"' >> $HOME/.bashrc
source $HOME/.bashrc

# Node.js and pnpm setup
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
source $HOME/.bashrc
nvm install --lts

curl -fsSL https://get.pnpm.io/install.sh | sh -
source $HOME/.bashrc
pnpm install -g @anthropic-ai/claude-code

# Golang setup
wget https://go.dev/dl/go1.24.5.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf ./go1.24.5.linux-arm64.tar.gz
echo 'export PATH="$PATH:$HOME/go/bin:/usr/local/go/bin"' >> $HOME/.bashrc
source $HOME/.bashrc
rm ./go1.24.5.linux-arm64.tar.gz
go install -v github.com/go-delve/delve/cmd/dlv@latest

# Python setup
curl -LsSf https://astral.sh/uv/install.sh | sh
source $HOME/.local/bin/env
uv sync

# Runtime dependencies
./scripts/install-wild-cloud-dependencies.sh

# App
cd app && pnpm install && cd ..
# Daemon
cd daemon && go mod tidy && cd ..
# CLI
cd cli && go mod tidy && cd ..
```

### Building Locally

1. **Build the application:**
   ```bash
   make build
   ```

2. **Run locally:**
   ```bash
   make run
   ```

3. **Development with auto-reload:**
   ```bash
   make dev
   ```

### Dependencies
- **gorilla/mux** - HTTP routing
- **gopkg.in/yaml.v3** - YAML configuration parsing

## API Reference

### Endpoints

- `GET /api/v1/health` - Service health check
- `GET /api/v1/config` - Get current configuration
- `PUT /api/v1/config` - Update configuration
- `GET /api/v1/dnsmasq/config` - Generate dnsmasq configuration
- `POST /api/v1/dnsmasq/restart` - Restart dnsmasq service
- `POST /api/v1/pxe/assets` - Download/update PXE boot assets

### Configuration

Edit `config.yaml` to customize your deployment:

```yaml
server:
  port: 5055
  host: "0.0.0.0"

cloud:
  domain: "wildcloud.local"
  dns:
    ip: "192.168.8.50"
  dhcpRange: "192.168.8.100,192.168.8.200"

cluster:
  endpointIp: "192.168.8.60"
  nodes:
    talos:
      version: "v1.8.0"
```

## Testing

> ⚠️ **Note**: These Docker scripts test the installation process only. In production, use `sudo apt install wild-cloud-central` and manage via systemd.

Choose the testing approach that fits your needs:

### 1. Automated Verification - `./tests/integration/test-docker.sh`
- **When to use**: Verify the installation works correctly
- **What it does**: Builds .deb package, installs it, tests all endpoints automatically
- **Best for**: CI/CD, quick verification that everything works

### 2. Background Testing - `./tests/integration/start-background.sh` / `./tests/integration/stop-background.sh`  
- **When to use**: You want to test APIs while doing other work
- **What it does**: Starts services silently in background, gives you your terminal back
- **Example workflow**: Start services, test in another terminal, stop when done
```bash
./tests/integration/start-background.sh           # Services start, terminal returns immediately
curl http://localhost:9081/api/v1/health  # Test in same or different terminal
# Continue working while services run...
./tests/integration/stop-background.sh            # Clean shutdown when finished
```

### 3. Interactive Development - `./tests/integration/start-interactive.sh`
- **When to use**: You want to see what's happening as you test
- **What it does**: Starts services with live logs, takes over your terminal
- **Example workflow**: Start services, watch logs in real-time, Ctrl+C to stop
```bash
./tests/integration/start-interactive.sh          # Services start, shows live logs
# You see all HTTP requests, errors, debug info in real-time
# Press Ctrl+C when done - terminal is "busy" until then
```

### 4. Shell Access - `./tests/integration/debug-container.sh`
- **When to use**: Deep debugging, manual service control, file inspection
- **What it does**: Drops you into the container shell
- **Best for**: Investigating issues, manually starting/stopping services

### Test Access Points
All services bind to localhost (127.0.0.1) on non-standard ports, so they won't interfere with your local services:

- Management UI: http://localhost:9080
- API: http://localhost:9081
- DNS: localhost:9053 (UDP) - test with `dig @localhost -p 9053 wildcloud.local`
- DHCP: localhost:9067 (UDP)
- TFTP: localhost:9069 (UDP)
- Container logs: `docker logs wild-central-bg`

## Architecture

This service replaces the original bash script implementation with:
- Unified configuration management
- Real-time dnsmasq configuration generation
- Integrated Talos factory asset downloading
- Web-based management interface
- Proper systemd service integration

## Make Targets

- `make build` - Build the Go binary
- `make run` - Run the application locally
- `make dev` - Start development server
- `make test` - Run Go tests
- `make clean` - Clean build artifacts
- `make deb` - Create Debian package
- `make repo` - Build APT repository
- `make deploy-repo` - Deploy repository to server