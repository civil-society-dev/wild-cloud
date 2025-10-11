#!/bin/bash

set -e

echo "🧪 Testing wild-cloud-central Docker installation..."

# Change to project root directory
cd "$(dirname "$0")/../.."

# Build the Docker image
echo "🔨 Building Docker image..."
docker build -t wild-cloud-central-test .

# Run the container to test installation
echo "🚀 Running installation test..."
echo "Access points after container starts:"
echo "  - Management UI: http://localhost:9080"
echo "  - API directly: http://localhost:9055"
echo ""
docker run --rm -p 9055:5055 -p 9080:80 wild-cloud-central-test