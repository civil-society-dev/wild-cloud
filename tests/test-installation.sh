#!/bin/bash

set -e

echo "🚀 Testing wild-cloud-central installation..."

# Verify the binary was installed
echo "✅ Checking binary installation..."
if [ -f "/usr/bin/wild-cloud-central" ]; then
    echo "   Binary installed at /usr/bin/wild-cloud-central"
else
    echo "❌ Binary not found at /usr/bin/wild-cloud-central"
    exit 1
fi

# Verify config was installed
echo "✅ Checking configuration..."
if [ -f "/etc/wild-cloud-central/config.yaml" ]; then
    echo "   Config installed at /etc/wild-cloud-central/config.yaml"
else
    echo "❌ Config not found at /etc/wild-cloud-central/config.yaml"
    exit 1
fi

# Verify systemd service file was installed
echo "✅ Checking systemd service..."
if [ -f "/etc/systemd/system/wild-cloud-central.service" ]; then
    echo "   Service file installed at /etc/systemd/system/wild-cloud-central.service"
else
    echo "❌ Service file not found"
    exit 1
fi

# Verify nginx config was installed
echo "✅ Checking nginx configuration..."
if [ -f "/etc/nginx/sites-available/wild-central" ]; then
    echo "   Nginx config installed at /etc/nginx/sites-available/wild-central"
    # Enable the site for testing
    ln -sf /etc/nginx/sites-available/wild-central /etc/nginx/sites-enabled/
    rm -f /etc/nginx/sites-enabled/default
else
    echo "❌ Nginx config not found"
    exit 1
fi

# Verify web assets were installed
echo "✅ Checking web assets..."
if [ -f "/var/www/html/wild-central/index.html" ]; then
    echo "   Web assets installed at /var/www/html/wild-central/"
else
    echo "❌ Web assets not found"
    exit 1
fi

# Verify polkit rules were installed
echo "✅ Checking polkit rules..."
if [ -f "/etc/polkit-1/rules.d/50-wild-cloud-central.rules" ]; then
    echo "   Polkit rules installed"
else
    echo "❌ Polkit rules not found"
    exit 1
fi

# Verify systemd-resolved configuration directory exists
echo "✅ Checking systemd-resolved configuration..."
if [ -d "/etc/systemd/resolved.conf.d" ]; then
    echo "   systemd-resolved config directory exists"
else
    echo "❌ systemd-resolved config directory not found"
    exit 1
fi

# Verify wild-cloud DNS config file was created
if [ -f "/etc/systemd/resolved.conf.d/wild-cloud.conf" ]; then
    echo "   wild-cloud DNS config file exists"
    # Check ownership
    if stat -c '%U:%G' /etc/systemd/resolved.conf.d/wild-cloud.conf | grep -q "wildcloud:wildcloud"; then
        echo "   wild-cloud DNS config owned by wildcloud user"
    else
        echo "⚠️  Warning: wild-cloud DNS config not owned by wildcloud user"
    fi
else
    echo "❌ wild-cloud DNS config file not found"
    exit 1
fi

# Verify /etc/resolv.conf is symlink to stub resolver
echo "✅ Checking resolv.conf configuration..."
if [ -L "/etc/resolv.conf" ]; then
    LINK_TARGET=$(readlink /etc/resolv.conf)
    if [ "$LINK_TARGET" = "/run/systemd/resolve/stub-resolv.conf" ]; then
        echo "   /etc/resolv.conf correctly symlinked to stub resolver"
    else
        echo "⚠️  Warning: /etc/resolv.conf symlinked to $LINK_TARGET (expected stub resolver)"
    fi
else
    echo "⚠️  Warning: /etc/resolv.conf is not a symlink"
fi

# Verify wildcloud user exists
echo "✅ Checking wildcloud user..."
if id wildcloud >/dev/null 2>&1; then
    echo "   wildcloud user exists"
    USER_GROUPS=$(groups wildcloud)
    echo "   wildcloud groups: $USER_GROUPS"
else
    echo "❌ wildcloud user not found"
    exit 1
fi

# Start nginx (simulating systemd)
echo "🔧 Starting nginx..."
nginx &
NGINX_PID=$!

# Start dnsmasq (simulating systemd)
echo "🔧 Starting dnsmasq..."
dnsmasq --keep-in-foreground --log-facility=- &
DNSMASQ_PID=$!

# Start wild-cloud-central service (simulating systemd)
echo "🔧 Starting wild-cloud-central service..."
/usr/bin/wild-cloud-central &
SERVICE_PID=$!

# Wait for service to start
echo "⏳ Waiting for services to start..."
sleep 8

# Give the API a moment to bind to the port
echo "⏳ Waiting for API to be ready..."
for i in {1..10}; do
    if curl -s http://localhost:5055/api/v1/health >/dev/null 2>&1; then
        echo "   API is ready!"
        break
    fi
    sleep 1
done

# Test health endpoint
echo "🩺 Testing health endpoint..."
if curl -s http://localhost:5055/api/v1/health | grep -q "ok"; then
    echo "   ✅ Health check passed"
else
    echo "   ❌ Health check failed"
    exit 1
fi

# Test configuration endpoint
echo "🔧 Testing configuration endpoint..."
CONFIG_RESPONSE=$(curl -s http://localhost:5055/api/v1/config)
if echo "$CONFIG_RESPONSE" | grep -q "configured"; then
    echo "   ✅ Configuration endpoint working"
else
    echo "   ❌ Configuration endpoint failed"
    echo "   Response: $CONFIG_RESPONSE"
    echo "   Checking if service is still running..."
    if kill -0 $SERVICE_PID 2>/dev/null; then
        echo "   Service is running"
    else
        echo "   Service has died"
    fi
    exit 1
fi

# Test dnsmasq config endpoint
echo "🔧 Testing dnsmasq config endpoint..."
if curl -s http://localhost:5055/api/v1/dnsmasq/config | grep -q "config_file"; then
    echo "   ✅ Dnsmasq config endpoint working"
else
    echo "   ❌ Dnsmasq config endpoint failed"
    exit 1
fi

# Test web interface accessibility (through nginx)
echo "🌐 Testing web interface..."
if curl -s http://localhost:80/ | grep -q "Wild Cloud Central"; then
    echo "   ✅ Web interface accessible through nginx"
else
    echo "   ❌ Web interface not accessible"
    exit 1
fi

echo ""
echo "🎉 All installation tests passed!"
echo ""
echo "Services running:"
echo "  - wild-cloud-central: http://localhost:5055"
echo "  - Web interface: http://localhost:80"
echo "  - API health: http://localhost:5055/api/v1/health"
echo ""
echo "Installation simulation successful! 🚀"

# Keep services running for manual testing
echo "Services will continue running. Press Ctrl+C to stop."

# Function to handle shutdown
shutdown() {
    echo ""
    echo "🛑 Shutting down services..."
    kill $SERVICE_PID 2>/dev/null || true
    kill $DNSMASQ_PID 2>/dev/null || true
    kill $NGINX_PID 2>/dev/null || true
    echo "Shutdown complete."
    exit 0
}

# Set up signal handlers
trap shutdown SIGTERM SIGINT

# Wait for signals
wait