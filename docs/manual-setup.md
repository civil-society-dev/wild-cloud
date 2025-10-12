# Manual Setup of Wild Cloud Central

If you want to set up from source (not using the debian package).

```bash
# Prerequisites
sudo apt update
sudo apt install dnsmasq

# Disable systemd-resolved
sudo systemctl disable systemd-resolved
sudo systemctl stop systemd-resolved

# Enable dnsmasq
sudo systemctl enable dnsmasq
sudo systemctl start dnsmasq
```
