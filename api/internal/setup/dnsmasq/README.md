# Central dnsmasq setup

## Overview

dnsmasq solves two problems for us. It provides:

- LAN DNS w/ forwarding of internal and external cloud domains to the cluster.
- PXE for setting up cluster nodes.

### PXE Bootloading

A "PXE client" is any machine that is booting using PXE. This is a great way to set up a new cluster node.

- PXE client broadcasts a request for help across the LAN.
- Dnsmasq's DHCP service responds with an IP address and TFTP server info.
- PXE client downloads PXE's iPEXE bootloader files:
  - `ipxe.efi`
  - `undionly.kpxe`
  - `ipxe-arm64.efi`
    (`pxelinux.0`) via TFTP.
- PXE client reads the bootloader config for the correct web address and fetches the boot files:
  - The kernel, `vmlinuz`.
  - The initial RAM disk, `initrd`.
  - The Talos image,

## Setup

- Install a Linux machine on your LAN. Record it's IP address in your `config:cloud.dns.ip`.
- Ensure it is accessible with ssh.
- From your wild-cloud directory, run `wild-central-generate-setup`.
- Run `cluster/dnsmasq/bin/create-setup-bundle.sh`
- Run `cluster/dnsmasq/bin/transfer-setup-bundle.sh`

Now ssh into your dnsmasq machine and do the following:

```bash
sudo -i
cd /root/dnsmasq-setup
./setup.sh
```

## Future setup

To provide a better user experience, we have been creating a debian apt package that also does this while providing a UI.

Development repo: https://github.com/civil-society-dev/wild-central

The setup will look something like this:

```bash
# Download and install GPG key
curl -fsSL https://mywildcloud.org/apt/wild-cloud-central.gpg | sudo tee /usr/share/keyrings/wild-cloud-central-archive-keyring.gpg > /dev/null

# Add repository (modern .sources format)
sudo tee /etc/apt/sources.list.d/wild-cloud-central.sources << 'EOF'
Types: deb
URIs: https://mywildcloud.org/apt
Suites: stable
Components: main
Signed-By: /usr/share/keyrings/wild-cloud-central-archive-keyring.gpg
EOF

# Update and install
sudo apt update
sudo apt install wild-cloud-central
```

browse to `http://localhost:5050`!
