#!/bin/bash

# Talos cluster initialization script
# This script performs one-time cluster setup: generates secrets, base configs, and sets up talosctl

set -euo pipefail

# Check if WC_HOME is set
if [ -z "${WC_HOME:-}" ]; then
    echo "Error: WC_HOME environment variable not set. Run \`source ./env.sh\`."
    exit 1
fi

NODE_SETUP_DIR="${WC_HOME}/setup/cluster-nodes"

# Get cluster configuration from config.yaml
CLUSTER_NAME=$(wild-config cluster.name)
VIP=$(wild-config cluster.nodes.control.vip)
TALOS_VERSION=$(wild-config cluster.nodes.talos.version)

echo "Initializing Talos cluster: $CLUSTER_NAME"
echo "VIP: $VIP"
echo "Talos version: $TALOS_VERSION"

# Create directories
mkdir -p generated final patch

# Check if cluster secrets already exist
if [ -f "generated/secrets.yaml" ]; then
    echo ""
    echo "⚠️  Cluster secrets already exist!"
    echo "This will regenerate ALL cluster certificates and invalidate existing nodes."
    echo ""
    read -p "Do you want to continue? (y/N): " -r
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cancelled."
        exit 0
    fi
    echo ""
fi

# Generate fresh cluster secrets
echo "Generating cluster secrets..."
cd generated
talosctl gen secrets -o secrets.yaml --force

echo "Generating base machine configs..."
talosctl gen config --with-secrets secrets.yaml "$CLUSTER_NAME" "https://$VIP:6443" --force
cd ..

# Setup talosctl context
echo "Setting up talosctl context..."

# Remove existing context if it exists
talosctl config context "$CLUSTER_NAME" --remove 2>/dev/null || true

# Merge new configuration
talosctl config merge ./generated/talosconfig
talosctl config endpoint "$VIP"

echo ""
echo "✅ Cluster initialization complete!"
echo ""
echo "Cluster details:"
echo "  - Name: $CLUSTER_NAME"
echo "  - VIP: $VIP"
echo "  - Secrets: generated/secrets.yaml"
echo "  - Base configs: generated/controlplane.yaml, generated/worker.yaml"
echo ""
echo "Talosctl context configured:"
talosctl config info
echo ""
echo "Next steps:"
echo "1. Register nodes with hardware detection:"
echo "   ./detect-node-hardware.sh <maintenance-ip> <node-number>"
echo ""
echo "2. Generate machine configurations:"
echo "   ./generate-machine-configs.sh"
echo ""
echo "3. Apply configurations to nodes"