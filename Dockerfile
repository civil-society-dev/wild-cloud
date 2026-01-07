# Test Dockerfile for wild-cloud-central package installation
FROM debian:bookworm-slim

# Install only runtime dependencies (no build tools needed)
RUN apt-get update && apt-get install -y \
    curl \
    dnsmasq \
    nginx \
    ca-certificates \
    policykit-1 \
    systemd \
    && rm -rf /var/lib/apt/lists/*

# Copy pre-built .deb package from build directory
# This must be built on the host first with: make package-arm64 or make package-amd64
COPY build/wild-cloud-central_*.deb /tmp/

# Install the .deb package (simulating what a user would do)
RUN dpkg -i /tmp/wild-cloud-central_*.deb || true && \
    apt-get update && apt-get install -f -y && \
    rm /tmp/*.deb

# Copy example config to the installed location
RUN cp /etc/wild-cloud-central/config.yaml.example /etc/wild-cloud-central/config.yaml

# Create required directories with proper permissions
RUN mkdir -p /var/www/html/talos /var/ftpd /var/lib/wild-cloud-central /var/log/wild-cloud-central
RUN chown -R www-data:www-data /var/www/html
RUN chmod 755 /var/ftpd

# Create a simple test script
COPY tests/test-installation.sh /test-installation.sh
RUN chmod +x /test-installation.sh

# Expose required ports
EXPOSE 5055 53/udp 67/udp 69/udp 80

# Health check to verify service is working
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:5055/api/v1/health || exit 1

# Test the installation
CMD ["/test-installation.sh"]