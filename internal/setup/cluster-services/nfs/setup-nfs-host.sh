#!/bin/bash
set -e
set -o pipefail

# Navigate to script directory
SCRIPT_PATH="$(realpath "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$(dirname "$SCRIPT_PATH")"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

usage() {
    echo "Usage: setup-nfs-host.sh [server] [media-path] [options]"
    echo ""
    echo "Set up NFS server on the specified host."
    echo ""
    echo "Examples:"
    echo "  setup-nfs-host.sh box-01 /data/media"
    echo ""
    echo "Options:"
    echo "  -h, --help  Show this help message"
    echo "  -e, --export-options  Set the NFS export options"

}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -e|--export-options)
            if [[ -z "$2" ]]; then
                echo "Error: --export-options requires a value"
                exit 1
            else
                NFS_EXPORT_OPTIONS="$2"
            fi
            shift 2
            ;;
        -*)
            echo "Unknown option $1"
            usage
            exit 1
            ;;
        *)
            # First non-option argument is server
            if [[ -z "$NFS_HOST" ]]; then
                export NFS_HOST="$1"
            # Second non-option argument is media path
            elif [[ -z "$NFS_MEDIA_PATH" ]]; then
                export NFS_MEDIA_PATH="$1"
            else
                echo "Too many arguments"
                usage
                exit 1
            fi
            shift
            ;;
    esac
done

echo "Setting up NFS server on this host..."

# Check if required NFS variables are configured
if [[ -z "${NFS_HOST}" ]]; then
    echo "NFS_HOST not set. Please set NFS_HOST=<hostname> in your environment"
    echo "Example: export NFS_HOST=box-01"
    exit 1
fi

# Ensure NFS_MEDIA_PATH is explicitly set
if [[ -z "${NFS_MEDIA_PATH}" ]]; then
    echo "Error: NFS_MEDIA_PATH not set. Please set it in your environment"
    echo "Example: export NFS_MEDIA_PATH=/data/media"
    exit 1
fi

# Set default for NFS_EXPORT_OPTIONS if not already set
if [[ -z "${NFS_EXPORT_OPTIONS}" ]]; then
    export NFS_EXPORT_OPTIONS="*(rw,sync,no_subtree_check,no_root_squash)"
    echo "Using default NFS_EXPORT_OPTIONS: ${NFS_EXPORT_OPTIONS}"
fi

echo "Target NFS host: ${NFS_HOST}"
echo "Media path: ${NFS_MEDIA_PATH}"
echo "Export options: ${NFS_EXPORT_OPTIONS}"

# Function to check if we're running on the correct host
check_host() {
    local current_hostname=$(hostname)
    if [[ "${current_hostname}" != "${NFS_HOST}" ]]; then
        echo "Warning: Current host (${current_hostname}) differs from NFS_HOST (${NFS_HOST})"
        echo "This script should be run on ${NFS_HOST}"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

# Function to install NFS server and SMB/CIFS
install_nfs_server() {
    echo "Installing NFS server and SMB/CIFS packages..."
    
    # Detect package manager and install NFS server + Samba
    if command -v apt-get >/dev/null 2>&1; then
        # Debian/Ubuntu
        sudo apt-get update
        sudo apt-get install -y nfs-kernel-server nfs-common samba samba-common-bin
    elif command -v yum >/dev/null 2>&1; then
        # RHEL/CentOS
        sudo yum install -y nfs-utils samba samba-client
    elif command -v dnf >/dev/null 2>&1; then
        # Fedora
        sudo dnf install -y nfs-utils samba samba-client
    else
        echo "Error: Unable to detect package manager. Please install NFS server and Samba manually."
        exit 1
    fi
}

# Function to create media directory
create_media_directory() {
    echo "Creating media directory: ${NFS_MEDIA_PATH}"
    
    # Create directory if it doesn't exist
    sudo mkdir -p "${NFS_MEDIA_PATH}"
    
    # Set appropriate permissions
    # Using 755 for directory, allowing read/execute for all, write for owner
    sudo chmod 755 "${NFS_MEDIA_PATH}"
    
    echo "Media directory created with appropriate permissions"
    echo "Directory info:"
    ls -la "${NFS_MEDIA_PATH}/"
}

# Function to configure NFS exports
configure_nfs_exports() {
    echo "Configuring NFS exports..."
    
    local export_line="${NFS_MEDIA_PATH} ${NFS_EXPORT_OPTIONS}"
    local exports_file="/etc/exports"
    
    # Backup existing exports file
    sudo cp "${exports_file}" "${exports_file}.backup.$(date +%Y%m%d-%H%M%S)" 2>/dev/null || true
    
    # Check if export already exists
    if sudo grep -q "^${NFS_MEDIA_PATH}" "${exports_file}" 2>/dev/null; then
        echo "Export for ${NFS_MEDIA_PATH} already exists, updating..."
        sudo sed -i "s|^${NFS_MEDIA_PATH}.*|${export_line}|" "${exports_file}"
    else
        echo "Adding new export for ${NFS_MEDIA_PATH}..."
        echo "${export_line}" | sudo tee -a "${exports_file}"
    fi
    
    # Export the filesystems
    sudo exportfs -rav
    
    echo "NFS exports configured:"
    sudo exportfs -v
}

# Function to start and enable NFS services
start_nfs_services() {
    echo "Starting NFS services..."
    
    # Start and enable NFS server
    sudo systemctl enable nfs-server
    sudo systemctl start nfs-server
    
    # Also enable related services
    sudo systemctl enable rpcbind
    sudo systemctl start rpcbind
    
    echo "NFS services started and enabled"
    
    # Show service status
    sudo systemctl status nfs-server --no-pager --lines=5
}

# Function to configure SMB/CIFS sharing
configure_smb_sharing() {
    echo "Configuring SMB/CIFS sharing..."
    
    local smb_config="/etc/samba/smb.conf"
    local share_name="media"
    
    # Backup existing config
    sudo cp "${smb_config}" "${smb_config}.backup.$(date +%Y%m%d-%H%M%S)" 2>/dev/null || true
    
    # Check if share already exists
    if sudo grep -q "^\[${share_name}\]" "${smb_config}" 2>/dev/null; then
        echo "SMB share '${share_name}' already exists, updating..."
        # Remove existing share section
        sudo sed -i "/^\[${share_name}\]/,/^\[/{ /^\[${share_name}\]/d; /^\[/!d; }" "${smb_config}"
    fi
    
    # Add media share configuration
    cat << EOF | sudo tee -a "${smb_config}"

[${share_name}]
    comment = Media files for Wild Cloud
    path = ${NFS_MEDIA_PATH}
    browseable = yes
    read only = no
    guest ok = yes
    create mask = 0664
    directory mask = 0775
    force user = $(whoami)
    force group = $(whoami)
EOF
    
    echo "SMB share configuration added"
    
    # Test configuration
    if sudo testparm -s >/dev/null 2>&1; then
        echo "✓ SMB configuration is valid"
    else
        echo "✗ SMB configuration has errors"
        sudo testparm
        exit 1
    fi
}

# Function to start SMB services
start_smb_services() {
    echo "Starting SMB services..."
    
    # Enable and start Samba services
    sudo systemctl enable smbd
    sudo systemctl start smbd
    sudo systemctl enable nmbd
    sudo systemctl start nmbd
    
    echo "SMB services started and enabled"
    
    # Show service status
    sudo systemctl status smbd --no-pager --lines=3
}

# Function to test NFS setup
test_nfs_setup() {
    echo "Testing NFS setup..."
    
    # Test if NFS is responding
    if command -v showmount >/dev/null 2>&1; then
        echo "Available NFS exports:"
        showmount -e localhost || echo "Warning: showmount failed, but NFS may still be working"
    fi
    
    # Check if the export directory is accessible
    if [[ -d "${NFS_MEDIA_PATH}" ]]; then
        echo "✓ Media directory exists and is accessible"
    else
        echo "✗ Media directory not accessible"
        exit 1
    fi
}

# Function to show usage instructions
show_usage_instructions() {
    echo
    echo "=== NFS/SMB Host Setup Complete ==="
    echo
    echo "NFS and SMB servers are now running on this host with media directory: ${NFS_MEDIA_PATH}"
    echo
    echo "Access methods:"
    echo "1. NFS (for Kubernetes): Use setup-nfs-k8s.sh to register with cluster"
    echo "2. SMB/CIFS (for Windows): \\\\${NFS_HOST}\\media"
    echo
    echo "To add media files:"
    echo "- Copy directly to: ${NFS_MEDIA_PATH}"
    echo "- Or mount SMB share from Windows and copy there"
    echo
    echo "Windows SMB mount:"
    echo "- Open File Explorer"
    echo "- Map network drive to: \\\\${NFS_HOST}\\media"
    echo "- Or use: \\\\$(hostname -I | awk '{print $1}')\\media"
    echo
    echo "To verify services:"
    echo "- NFS: showmount -e ${NFS_HOST}"
    echo "- SMB: smbclient -L ${NFS_HOST} -N"
    echo "- Status: systemctl status nfs-server smbd"
    echo
    echo "Current NFS exports:"
    sudo exportfs -v
    echo
}

# Main execution
main() {
    check_host
    install_nfs_server
    create_media_directory
    configure_nfs_exports
    start_nfs_services
    configure_smb_sharing
    start_smb_services
    test_nfs_setup
    show_usage_instructions
}

# Run main function
main "$@"