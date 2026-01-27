# Wild Cloud

Build your own private cloud for the price of a smartphone and an Internet connection.

Wild Cloud is an open-source platform that lets you run your own data center on your local network. Host your applications, own your data, and stop renting from big tech.

**[Get Started →](https://mywildcloud.org/get-started/)**

## What You Get

- **On your own domain** — Access your apps at `photos.yourcloud.com`, `notes.yourcloud.com`, etc.
- **One-click app deployment** — Install self-hosted apps from the Wild Directory with minimal configuration
- **Web-based management** — Configure and monitor everything through your browser
- **Infrastructure-as-code** — All configuration stored in version-controlled YAML files for the technically inclined

## How It Works

Wild Cloud runs on a small management device called **Wild Central** (typically a Raspberry Pi) connected to your local network. Wild Central orchestrates a Kubernetes cluster running on additional compute nodes, handles DNS and networking, and provides the web interface for managing everything.

```
┌─────────────────────────────────────────────────────────────┐
│                      Your Network                           │
│                                                             │
│   ┌──────────────┐     ┌─────────────────────────────────┐  │
│   │ Wild Central │────▶│    Kubernetes Cluster           │  │
│   │  (Raspberry  │     │  ┌─────────┐ ┌─────────┐        │  │
│   │     Pi)      │     │  │ Control │ │ Worker  │ ...    │  │
│   └──────────────┘     │  │  Nodes  │ │  Nodes  │        │  │
│         │              │  └─────────┘ └─────────┘        │  │
│         ▼              └─────────────────────────────────┘  │
│   Web UI / API                                              │
└─────────────────────────────────────────────────────────────┘
```

## Requirements

- A **domain name** with DNS managed by Cloudflare
- A **Wild Central device** — Raspberry Pi or similar (~$100)
- **Compute nodes** — At least 3 control + 3 worker machines for a production cluster
- Basic **networking gear** — Router, switch, and ideally a UPS

See the [full hardware guide](https://mywildcloud.org/get-started/) for detailed recommendations.

## Installation

### On Debian/Ubuntu (Recommended)

```bash
# Add the Wild Cloud repository
curl -fsSL https://mywildcloud.org/apt/wild-cloud-central.gpg | sudo tee /usr/share/keyrings/wild-cloud-central-archive-keyring.gpg > /dev/null

sudo tee /etc/apt/sources.list.d/wild-cloud-central.sources << 'EOF'
Types: deb
URIs: https://mywildcloud.org/apt
Suites: stable
Components: main
Signed-By: /usr/share/keyrings/wild-cloud-central-archive-keyring.gpg
EOF

# Install
sudo apt update
sudo apt install wild-cloud-central

# Start the service
sudo systemctl enable --now wild-cloud-central
```

Then open `http://<your-server-ip>` in your browser to begin setup.

## Project Structure

This monorepo contains:

| Component | Description |
|-----------|-------------|
| [api/](api/) | Wild Central API — the backend service that manages your cloud |
| [cli/](cli/) | Command-line interface for terminal-based management |
| [web/](web/) | Web application for browser-based management |
| [dist/](dist/) | Packaging and distribution tooling |
| [docs/](docs/) | Guides for maintenance, troubleshooting, and upgrades |

## Documentation

- **[Getting Started](https://mywildcloud.org/get-started/)** — Hardware setup and initial configuration
- **[Developer Guide](docs/DEVELOPER.md)** — Contributing to Wild Cloud
- **[Maintainer Guide](docs/MAINTAINER.md)** — Building and releasing packages

## Community

Wild Cloud is a project of the [Civil Society Technology Foundation](https://civilsociety.dev), built to empower communities with self-hosted infrastructure.

- **Website**: [mywildcloud.org](https://mywildcloud.org)
- **Issues**: Report bugs and request features on GitHub

## License

GNU AGPLv3 — see [LICENSE](LICENSE) for details.
