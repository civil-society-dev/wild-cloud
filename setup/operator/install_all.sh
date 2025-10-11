#!/bin/bash
set -e

# Navigate to script directory
SCRIPT_PATH="$(realpath "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$(dirname "$SCRIPT_PATH")"
cd "$SCRIPT_DIR"

echo "Setting up your operator tooling..."
echo

./backup/install.sh

echo "Operator tooling setup complete!"
