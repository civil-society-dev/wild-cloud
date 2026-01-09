#!/bin/bash

set -e

echo "🚀 Starting wild-cloud-central for interactive testing..."

# Change to project root directory
cd "$(dirname "$0")/../.."

# Check if package exists
if [ ! -f build/wild-cloud-central_*_amd64.deb ]; then
    echo "❌ No .deb package found in build/ directory"
    echo "   Run 'make package' first to build the package"
    exit 1
fi

# Build the Docker image if it doesn't exist
if ! docker images | grep -q wild-cloud-central-test; then
    echo "🔨 Building Docker image..."
    docker build -t wild-cloud-central-test .
fi

echo ""
echo "🌐 Starting services... This will take a few seconds."
echo ""
echo "📍 Access points:"
echo "  - Management UI: http://localhost:9080"
echo "  - API directly: http://localhost:9081"
echo "  - Health check: http://localhost:9081/api/v1/health"
echo ""
echo "🔧 Available API endpoints:"
echo "  - GET  /api/v1/health"
echo "  - GET  /api/v1/config"
echo "  - PUT  /api/v1/config"
echo "  - GET  /api/v1/dnsmasq/config"
echo "  - POST /api/v1/dnsmasq/restart"
echo "  - POST /api/v1/pxe/assets"
echo ""
echo "💡 Example commands to try:"
echo "  curl http://localhost:9081/api/v1/health"
echo "  curl http://localhost:9081/api/v1/config"
echo "  curl http://localhost:9081/api/v1/dnsmasq/config"
echo "  curl -X POST http://localhost:9081/api/v1/pxe/assets"
echo ""
echo "🛑 Press Ctrl+C to stop all services"
echo ""

# Create a custom startup script that keeps services running
docker run --rm -it \
  -p 127.0.0.1:9081:5055 \
  -p 127.0.0.1:9080:80 \
  -p 127.0.0.1:9053:53/udp \
  -p 127.0.0.1:9067:67/udp \
  -p 127.0.0.1:9069:69/udp \
  --cap-add=NET_ADMIN \
  --cap-add=NET_BIND_SERVICE \
  --name wild-central-interactive \
  wild-cloud-central-test \
  /bin/bash -c '
    echo "🔧 Starting all services..."
    
    # Start nginx
    nginx &
    NGINX_PID=$!
    
    # Start dnsmasq 
    dnsmasq --keep-in-foreground --log-facility=- &
    DNSMASQ_PID=$!
    
    # Start wild-cloud-central
    /usr/bin/wild-cloud-central &
    SERVICE_PID=$!
    
    # Wait for services to start
    sleep 3
    
    echo "✅ All services started!"
    echo "   - nginx (PID: $NGINX_PID)"
    echo "   - dnsmasq (PID: $DNSMASQ_PID)" 
    echo "   - wild-cloud-central (PID: $SERVICE_PID)"
    echo ""
    echo "🌐 Services are now available:"
    echo "   - Web UI: http://localhost:9080"
    echo "   - API: http://localhost:9081"
    echo ""
    
    # Function to handle shutdown
    shutdown() {
        echo ""
        echo "🛑 Shutting down services..."
        kill $SERVICE_PID $DNSMASQ_PID $NGINX_PID 2>/dev/null || true
        echo "✅ Shutdown complete."
        exit 0
    }
    
    # Set up signal handlers
    trap shutdown SIGTERM SIGINT
    
    # Keep container running and wait for signals
    echo "✨ Container is ready! Press Ctrl+C to stop."
    wait
  '