#!/bin/bash

set -e

echo "🐳 Starting wild-cloud-central debug container..."

# Build the Docker image if it doesn't exist
if ! docker images | grep -q wild-cloud-central-test; then
    echo "🔨 Building Docker image..."
    docker build -t wild-cloud-central-test .
fi

echo ""
echo "🔧 Starting container with shell access..."
echo ""
echo "📍 Access points:"
echo "  - Management UI: http://localhost:9080"
echo "  - API directly: http://localhost:9081"
echo ""
echo "💡 Inside the container you can:"
echo "  - Start services manually: /test-installation.sh"
echo "  - Check logs: journalctl or service status"
echo "  - Test APIs: curl http://localhost:5055/api/v1/health"
echo "  - Modify config: nano /etc/wild-cloud-central/config.yaml"
echo "  - View web files: ls /var/www/html/wild-central/"
echo ""

# Run container with shell access
docker run --rm -it \
  -p 127.0.0.1:9081:5055 \
  -p 127.0.0.1:9080:80 \
  -p 127.0.0.1:9053:53/udp \
  -p 127.0.0.1:9067:67/udp \
  -p 127.0.0.1:9069:69/udp \
  --cap-add=NET_ADMIN \
  --cap-add=NET_BIND_SERVICE \
  --name wild-central-debug \
  wild-cloud-central-test \
  /bin/bash