#!/bin/bash

set -e

echo "🚀 Starting wild-cloud-central in background..."

# Build the Docker image if it doesn't exist
if ! docker images | grep -q wild-cloud-central-test; then
    echo "🔨 Building Docker image..."
    docker build -t wild-cloud-central-test .
fi

# Stop any existing container
docker rm -f wild-central-bg 2>/dev/null || true

echo "🌐 Starting services in background..."

# Start container in background 
docker run -d \
  --name wild-central-bg \
  -p 127.0.0.1:9081:5055 \
  -p 127.0.0.1:9080:80 \
  -p 127.0.0.1:9053:53/udp \
  -p 127.0.0.1:9067:67/udp \
  -p 127.0.0.1:9069:69/udp \
  --cap-add=NET_ADMIN \
  --cap-add=NET_BIND_SERVICE \
  wild-cloud-central-test \
  /bin/bash -c '
    # Start nginx
    nginx &
    
    # Start dnsmasq 
    dnsmasq --keep-in-foreground --log-facility=- &
    
    # Start wild-cloud-central
    /usr/bin/wild-cloud-central &
    
    # Wait indefinitely
    tail -f /dev/null
  '

echo "⏳ Waiting for services to start..."
sleep 5

# Test if services are running
if curl -s http://localhost:9081/api/v1/health > /dev/null 2>&1; then
    echo "✅ Services started successfully!"
    echo ""
    echo "📍 Access points (localhost only):"
    echo "  - Management UI: http://localhost:9080"
    echo "  - API: http://localhost:9081/api/v1/health"
    echo "  - DNS: localhost:9053 (for testing)"
    echo "  - DHCP: localhost:9067 (for testing)" 
    echo "  - TFTP: localhost:9069 (for testing)"
    echo ""
    echo "🔧 Container management:"
    echo "  - View logs: docker logs wild-central-bg"
    echo "  - Stop services: docker stop wild-central-bg"
    echo "  - Remove container: docker rm wild-central-bg"
    echo ""
    echo "💡 Test commands:"
    echo "  curl http://localhost:9081/api/v1/health"
    echo "  dig @localhost -p 9053 wildcloud.local"
    echo "  curl http://localhost:9081/api/v1/dnsmasq/config"
else
    echo "❌ Services failed to start. Check logs with: docker logs wild-central-bg"
    exit 1
fi