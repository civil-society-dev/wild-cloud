# Bare Metal Talos Administration Guide

This guide covers bare metal specific operations, configurations, and best practices for Talos Linux clusters.

## META-Based Network Configuration

Talos supports META-based network configuration for bare metal deployments where configuration is embedded in the disk image.

### Basic META Configuration
```yaml
# META configuration for bare metal networking
machine:
  network:
    interfaces:
      - interface: eth0
        addresses:
          - 192.168.1.100/24
        routes:
          - network: 0.0.0.0/0
            gateway: 192.168.1.1
        mtu: 1500
    nameservers:
      - 8.8.8.8
      - 1.1.1.1
```

### Advanced Network Configurations

#### VLAN Configuration
```yaml
machine:
  network:
    interfaces:
      - interface: eth0.100  # VLAN 100
        vlan:
          parentDevice: eth0
          vid: 100
        addresses:
          - 192.168.100.10/24
        routes:
          - network: 192.168.100.0/24
```

#### Interface Bonding
```yaml
machine:
  network:
    interfaces:
      - interface: bond0
        bond:
          mode: 802.3ad
          lacpRate: fast
          xmitHashPolicy: layer3+4
          miimon: 100
          updelay: 200
          downdelay: 200
          interfaces:
            - eth0
            - eth1
        addresses:
          - 192.168.1.100/24
        routes:
          - network: 0.0.0.0/0
            gateway: 192.168.1.1
```

#### Bridge Configuration
```yaml
machine:
  network:
    interfaces:
      - interface: br0
        bridge:
          stp:
            enabled: false
          interfaces:
            - eth0
            - eth1
        addresses:
          - 192.168.1.100/24
        routes:
          - network: 0.0.0.0/0
            gateway: 192.168.1.1
```

### Network Troubleshooting Commands
```bash
# Check interface configuration
talosctl -n <IP> get addresses
talosctl -n <IP> get routes
talosctl -n <IP> get links

# Check network configuration
talosctl -n <IP> get networkconfig -o yaml

# Test network connectivity
talosctl -n <IP> list /sys/class/net
talosctl -n <IP> read /proc/net/dev
```

## Disk Encryption for Bare Metal

### LUKS2 Encryption Configuration
```yaml
machine:
  systemDiskEncryption:
    state:
      provider: luks2
      keys:
        - slot: 0
          static:
            passphrase: "your-secure-passphrase"
    ephemeral:
      provider: luks2
      keys:
        - slot: 0
          nodeID: {}
```

### TPM-Based Encryption
```yaml
machine:
  systemDiskEncryption:
    state:
      provider: luks2
      keys:
        - slot: 0
          tpm: {}
    ephemeral:
      provider: luks2
      keys:
        - slot: 0
          tpm: {}
```

### Key Management Operations
```bash
# Check encryption status
talosctl -n <IP> get encryptionconfig -o yaml

# Rotate encryption keys
talosctl -n <IP> apply-config --file updated-config.yaml --mode staged
```

## SecureBoot Implementation

### UKI (Unified Kernel Image) Setup
SecureBoot requires UKI format images with embedded signatures.

#### Generate SecureBoot Keys
```bash
# Generate platform key (PK)
talosctl gen secureboot uki --platform-key-path platform.key --platform-cert-path platform.crt

# Generate PCR signing key
talosctl gen secureboot pcr --pcr-key-path pcr.key --pcr-cert-path pcr.crt

# Generate database entries
talosctl gen secureboot database --enrolled-certificate platform.crt
```

#### Machine Configuration for SecureBoot
```yaml
machine:
  secureboot:
    enabled: true
    uklPath: /boot/vmlinuz
  systemDiskEncryption:
    state:
      provider: luks2
      keys:
        - slot: 0
          tpm:
            pcrTargets:
              - 0
              - 1
              - 7
```

### UEFI Configuration
- Enable SecureBoot in UEFI firmware
- Enroll platform keys and certificates
- Configure TPM 2.0 for PCR measurements
- Set boot order for UKI images

## Hardware-Specific Configurations

### Performance Tuning for Bare Metal

#### CPU Governor Configuration
```yaml
machine:
  sysfs:
    "devices.system.cpu.cpu0.cpufreq.scaling_governor": "performance"
    "devices.system.cpu.cpu1.cpufreq.scaling_governor": "performance"
```

#### Hardware Vulnerability Mitigations
```yaml
machine:
  kernel:
    args:
      - mitigations=off  # For maximum performance (less secure)
      # or
      - mitigations=auto  # Default balanced approach
```

#### IOMMU Configuration
```yaml
machine:
  kernel:
    args:
      - intel_iommu=on
      - iommu=pt
```

### Memory Management
```yaml
machine:
  kernel:
    args:
      - hugepages=1024  # 1GB hugepages
      - transparent_hugepage=never
```

## Ingress Firewall for Bare Metal

### Basic Firewall Configuration
```yaml
machine:
  network:
    firewall:
      defaultAction: block
      rules:
        - name: allow-talos-api
          portSelector:
            ports:
              - 50000
              - 50001
          ingress:
            - subnet: 192.168.1.0/24
        - name: allow-kubernetes-api
          portSelector:
            ports:
              - 6443
          ingress:
            - subnet: 0.0.0.0/0
        - name: allow-etcd
          portSelector:
            ports:
              - 2379
              - 2380
          ingress:
            - subnet: 192.168.1.0/24
```

### Advanced Firewall Rules
```yaml
machine:
  network:
    firewall:
      defaultAction: block
      rules:
        - name: allow-ssh-management
          portSelector:
            ports:
              - 22
          ingress:
            - subnet: 10.0.1.0/24  # Management network only
        - name: allow-monitoring
          portSelector:
            ports:
              - 9100  # Node exporter
              - 10250 # kubelet metrics
          ingress:
            - subnet: 192.168.1.0/24
```

## System Extensions for Bare Metal

### Common Bare Metal Extensions
```yaml
machine:
  install:
    extensions:
      - image: ghcr.io/siderolabs/iscsi-tools:latest
      - image: ghcr.io/siderolabs/util-linux-tools:latest
      - image: ghcr.io/siderolabs/drbd:latest
```

### Storage Extensions
```yaml
machine:
  install:
    extensions:
      - image: ghcr.io/siderolabs/zfs:latest
      - image: ghcr.io/siderolabs/nut-client:latest
      - image: ghcr.io/siderolabs/smartmontools:latest
```

### Checking Extension Status
```bash
# List installed extensions
talosctl -n <IP> get extensions

# Check extension services
talosctl -n <IP> get extensionserviceconfigs
```

## Static Pod Configuration for Bare Metal

### Local Storage Static Pods
```yaml
machine:
  pods:
    - name: local-storage-provisioner
      namespace: kube-system
      image: rancher/local-path-provisioner:v0.0.24
      args:
        - --config-path=/etc/config/config.json
      env:
        - name: POD_NAMESPACE
          value: kube-system
      volumeMounts:
        - name: config
          mountPath: /etc/config
        - name: local-storage
          mountPath: /opt/local-path-provisioner
      volumes:
        - name: config
          hostPath:
            path: /etc/local-storage
        - name: local-storage
          hostPath:
            path: /var/lib/local-storage
```

### Hardware Monitoring Static Pods
```yaml
machine:
  pods:
    - name: node-exporter
      namespace: monitoring
      image: prom/node-exporter:latest
      args:
        - --path.rootfs=/host
        - --collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($$|/)
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
      volumeMounts:
        - name: proc
          mountPath: /host/proc
          readOnly: true
        - name: sys
          mountPath: /host/sys
          readOnly: true
        - name: rootfs
          mountPath: /host
          readOnly: true
      volumes:
        - name: proc
          hostPath:
            path: /proc
        - name: sys
          hostPath:
            path: /sys
        - name: rootfs
          hostPath:
            path: /
```

## Bare Metal Boot Asset Management

### PXE Boot Configuration
For network booting, configure DHCP/TFTP with appropriate boot assets:

```bash
# Download kernel and initramfs for PXE
curl -LO https://github.com/siderolabs/talos/releases/download/v1.11.0/vmlinuz-amd64
curl -LO https://github.com/siderolabs/talos/releases/download/v1.11.0/initramfs-amd64.xz
```

### USB Boot Asset Creation
```bash
# Write installer image to USB
sudo dd if=metal-amd64.iso of=/dev/sdX bs=4M status=progress
```

### Image Factory Integration
For custom bare metal images:
```bash
# Generate schematic for bare metal with extensions
curl -X POST --data-binary @schematic.yaml \
  https://factory.talos.dev/schematics

# Download custom installer
curl -LO https://factory.talos.dev/image/<schematic-id>/v1.11.0/metal-amd64.iso
```

## Hardware Compatibility and Drivers

### Check Hardware Support
```bash
# Check PCI devices
talosctl -n <IP> read /proc/bus/pci/devices

# Check USB devices
talosctl -n <IP> read /proc/bus/usb/devices

# Check loaded kernel modules
talosctl -n <IP> read /proc/modules

# Check hardware information
talosctl -n <IP> read /proc/cpuinfo
talosctl -n <IP> read /proc/meminfo
```

### Common Hardware Issues

#### Network Interface Issues
```bash
# Check interface status
talosctl -n <IP> list /sys/class/net/

# Check driver information
talosctl -n <IP> read /sys/class/net/eth0/device/driver

# Check firmware loading
talosctl -n <IP> dmesg | grep firmware
```

#### Storage Controller Issues
```bash
# Check block devices
talosctl -n <IP> disks

# Check SMART status (if smartmontools extension installed)
talosctl -n <IP> list /dev/disk/by-id/
```

## Bare Metal Monitoring and Maintenance

### Hardware Health Monitoring
```bash
# Check system temperatures (if available)
talosctl -n <IP> read /sys/class/thermal/thermal_zone0/temp

# Check power supply status
talosctl -n <IP> read /sys/class/power_supply/*/status

# Monitor system events for hardware issues
talosctl -n <IP> dmesg | grep -i error
talosctl -n <IP> dmesg | grep -i "machine check"
```

### Performance Monitoring
```bash
# Check CPU performance
talosctl -n <IP> read /proc/cpuinfo | grep MHz
talosctl -n <IP> cgroups --preset cpu

# Check memory performance
talosctl -n <IP> memory
talosctl -n <IP> cgroups --preset memory

# Check I/O performance
talosctl -n <IP> read /proc/diskstats
```

## Security Hardening for Bare Metal

### BIOS/UEFI Security
- Enable SecureBoot
- Disable unused boot devices
- Set administrator passwords
- Enable TPM 2.0
- Disable legacy boot modes

### Physical Security
- Secure physical access to servers
- Use chassis intrusion detection
- Implement network port security
- Consider hardware-based attestation

### Network Security
```yaml
machine:
  network:
    firewall:
      defaultAction: block
      rules:
        # Only allow necessary services
        - name: allow-cluster-traffic
          portSelector:
            ports:
              - 6443   # Kubernetes API
              - 2379   # etcd client
              - 2380   # etcd peer
              - 10250  # kubelet API
              - 50000  # Talos API
          ingress:
            - subnet: 192.168.1.0/24
```

This bare metal guide provides comprehensive coverage of hardware-specific configurations, performance optimization, security hardening, and operational practices for Talos Linux on physical servers.