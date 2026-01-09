#!/bin/bash

# This file to be run on dnsmasq server (Central)

echo "Updating APT repositories."
sudo apt-get update

echo "Installing dnsmasq and nginx."
sudo apt install -y dnsmasq nginx

DNSMASQ_SETUP_DIR="."
# PXE_FTPD_DIR="${DNSMASQ_SETUP_DIR}/pxe-ftpd"
# PXE_WEB_ROOT="${DNSMASQ_SETUP_DIR}/ipxe-web"

# Configure nginx.
# echo "Configuring nginx."
# sudo cp "${DNSMASQ_SETUP_DIR}/nginx.conf" /etc/nginx/sites-available/talos
# sudo chown www-data:www-data /etc/nginx/sites-available/talos
# sudo chmod -R 755 /etc/nginx/sites-available/talos

# Copy assets to nginx web root
# echo "Copying Talos PXE boot assets to nginx web root."
# TALOS_PXE_WEB_ROOT="/var/www/html/talos"
# sudo mkdir -p "${TALOS_PXE_WEB_ROOT}"
# sudo rm -rf ${TALOS_PXE_WEB_ROOT}/* # Clean the web root directory
# sudo cp -r ${PXE_WEB_ROOT}/* "${TALOS_PXE_WEB_ROOT}"
# sudo chown -R www-data:www-data "${TALOS_PXE_WEB_ROOT}"
# sudo chmod -R 755 "${TALOS_PXE_WEB_ROOT}"

# Start nginx service to serve the iPXE script and images
# echo "Starting nginx service."
# sudo ln -s /etc/nginx/sites-available/talos /etc/nginx/sites-enabled/talos > /dev/null 2>&1 || true
# sudo rm -f /etc/nginx/sites-enabled/default
# sudo systemctl reload nginx

# Stop and disable systemd-resolved if it is running
# if systemctl is-active --quiet systemd-resolved; then
#     echo "Stopping and disabling systemd-resolved..."
#     sudo systemctl disable systemd-resolved
#     sudo systemctl stop systemd-resolved
#     # sudo rm -f /etc/resolv.conf
#     echo "systemd-resolved stopped and disabled"
# fi

# Update PXE's iPXE bootloader files.
# echo "Updating iPXE ftpd bootloader files."
# sudo mkdir -p /var/ftpd
# sudo cp ${PXE_FTPD_DIR}/* /var/ftpd/

# Finally, install and configure DNSMasq.
echo "Configuring and starting DNSMasq."
sudo cp "${DNSMASQ_SETUP_DIR}/dnsmasq.conf" /etc/dnsmasq.conf
sudo systemctl restart dnsmasq

echo "DNSMasq installation and configuration completed successfully."
