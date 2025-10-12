#!/bin/bash
set -e
set -o pipefail

# Ensure WILD_INSTANCE is set
if [ -z "${WILD_INSTANCE}" ]; then
    echo "❌ ERROR: WILD_INSTANCE is not set"
    exit 1
fi

# Ensure WILD_API_DATA_DIR is set
if [ -z "${WILD_API_DATA_DIR}" ]; then
    echo "❌ ERROR: WILD_API_DATA_DIR is not set"
    exit 1
fi

# Ensure KUBECONFIG is set
if [ -z "${KUBECONFIG}" ]; then
    echo "❌ ERROR: KUBECONFIG is not set"
    exit 1
fi

INSTANCE_DIR="${WILD_API_DATA_DIR}/instances/${WILD_INSTANCE}"
CONFIG_FILE="${INSTANCE_DIR}/config.yaml"
CLUSTER_SETUP_DIR="${INSTANCE_DIR}/setup/cluster-services"
NFS_DIR="${CLUSTER_SETUP_DIR}/nfs"

echo "💾 === Registering NFS Server with Kubernetes Cluster ==="
echo ""

# Templates should already be compiled
echo "📦 Using pre-compiled NFS templates..."
if [ ! -d "${NFS_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${NFS_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

NFS_HOST="$(yq '.cloud.nfs.host' "${CONFIG_FILE}" 2>/dev/null | tr -d '"')"
NFS_MEDIA_PATH="$(yq '.cloud.nfs.mediaPath' "${CONFIG_FILE}" 2>/dev/null | tr -d '"')"
NFS_STORAGE_CAPACITY="$(yq '.cloud.nfs.storageCapacity' "${CONFIG_FILE}" 2>/dev/null | tr -d '"')"

echo "📋 NFS Configuration:"
echo "  Host: ${NFS_HOST}"
echo "  Media path: ${NFS_MEDIA_PATH}"
echo "  Storage capacity: ${NFS_STORAGE_CAPACITY}"
echo ""

# Validate required config values
if [ -z "${NFS_HOST}" ] || [ "${NFS_HOST}" = "null" ]; then
    echo "❌ ERROR: cloud.nfs.host not set in config"
    exit 1
fi
if [ -z "${NFS_MEDIA_PATH}" ] || [ "${NFS_MEDIA_PATH}" = "null" ]; then
    echo "❌ ERROR: cloud.nfs.mediaPath not set in config"
    exit 1
fi
if [ -z "${NFS_STORAGE_CAPACITY}" ] || [ "${NFS_STORAGE_CAPACITY}" = "null" ]; then
    echo "❌ ERROR: cloud.nfs.storageCapacity not set in config"
    exit 1
fi

# Function to resolve NFS host to IP
resolve_nfs_host() {
    echo "🌐 Resolving NFS host: ${NFS_HOST}"
    if [[ "${NFS_HOST}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        # NFS_HOST is already an IP address
        NFS_HOST_IP="${NFS_HOST}"
        echo "  ℹ️  Host is already an IP address"
    else
        # Resolve hostname to IP
        echo "  🔍 Looking up hostname..."
        NFS_HOST_IP=$(getent hosts "${NFS_HOST}" 2>/dev/null | awk '{print $1}' | head -n1 || true)
        echo "  📍 Resolved to: ${NFS_HOST_IP}"
        if [[ -z "${NFS_HOST_IP}" ]]; then
            echo "❌ ERROR: Unable to resolve hostname ${NFS_HOST} to IP address"
            echo "💡 Make sure ${NFS_HOST} is resolvable from this cluster"
            exit 1
        fi

        # Check if resolved IP is localhost - auto-detect network IP instead
        if [[ "${NFS_HOST_IP}" =~ ^127\. ]]; then
            echo "⚠️  Warning: ${NFS_HOST} resolves to localhost (${NFS_HOST_IP})"
            echo "🔍 Auto-detecting network IP for cluster access..."

            # Try to find the primary network interface IP (exclude docker/k8s networks)
            local network_ip=$(ip route get 8.8.8.8 | grep -oP 'src \K\S+' 2>/dev/null)

            if [[ -n "${network_ip}" && ! "${network_ip}" =~ ^127\. ]]; then
                echo "✅ Using detected network IP: ${network_ip}"
                NFS_HOST_IP="${network_ip}"
            else
                echo "❌ Could not auto-detect network IP. Available IPs:"
                ip addr show | grep "inet " | grep -v "127.0.0.1" | grep -v "10.42" | grep -v "172." | awk '{print "  " $2}' | cut -d/ -f1
                echo "💡 Please set NFS_HOST to the correct IP address manually."
                exit 1
            fi
        fi
    fi

    echo "🌐 NFS server IP: ${NFS_HOST_IP}"
    export NFS_HOST_IP
}

# Function to test NFS accessibility
test_nfs_accessibility() {
    echo ""
    echo "🔍 Testing NFS accessibility from cluster..."

    # Check if showmount is available
    if ! command -v showmount >/dev/null 2>&1; then
        echo "📦 Installing NFS client tools..."
        if command -v apt-get >/dev/null 2>&1; then
            sudo apt-get update && sudo apt-get install -y nfs-common
        elif command -v yum >/dev/null 2>&1; then
            sudo yum install -y nfs-utils
        elif command -v dnf >/dev/null 2>&1; then
            sudo dnf install -y nfs-utils
        else
            echo "⚠️ Warning: Unable to install NFS client tools. Skipping accessibility test."
            return 0
        fi
    fi

    # Test if we can reach the NFS server
    echo "🌐 Testing connection to NFS server..."
    if timeout 10 showmount -e "${NFS_HOST_IP}" >/dev/null 2>&1; then
        echo "✅ NFS server is accessible"
        echo "📋 Available exports:"
        showmount -e "${NFS_HOST_IP}"
    else
        echo "❌ Cannot connect to NFS server at ${NFS_HOST_IP}"
        echo "💡 Make sure:"
        echo "  1. NFS server is running on ${NFS_HOST}"
        echo "  2. Network connectivity exists between cluster and NFS host"
        echo "  3. Firewall allows NFS traffic (port 2049)"
        exit 1
    fi

    # Test specific export
    if showmount -e "${NFS_HOST_IP}" | grep -q "${NFS_MEDIA_PATH}"; then
        echo "✅ Media path ${NFS_MEDIA_PATH} is exported"
    else
        echo "❌ Media path ${NFS_MEDIA_PATH} is not found in exports"
        echo "📋 Available exports:"
        showmount -e "${NFS_HOST_IP}"
        echo ""
        echo "💡 Run setup-nfs-host.sh on ${NFS_HOST} to configure the export"
        exit 1
    fi
}

# Function to create test mount
test_nfs_mount() {
    echo ""
    echo "🔧 Testing NFS mount functionality..."

    local test_mount="/tmp/nfs-test-$$"
    mkdir -p "${test_mount}"

    # Try to mount the NFS export
    if timeout 30 sudo mount -t nfs4 "${NFS_HOST_IP}:${NFS_MEDIA_PATH}" "${test_mount}"; then
        echo "✅ NFS mount successful"

        # Test read access
        if ls "${test_mount}" >/dev/null 2>&1; then
            echo "✅ NFS read access working"
        else
            echo "❌ NFS read access failed"
        fi

        # Unmount
        sudo umount "${test_mount}" || echo "⚠️  Warning: Failed to unmount test directory"
    else
        echo "❌ NFS mount failed"
        echo "💡 Check NFS server configuration and network connectivity"
        exit 1
    fi

    # Clean up
    rmdir "${test_mount}" 2>/dev/null || true
}

# Function to create Kubernetes resources
create_k8s_resources() {
    echo ""
    echo "🚀 Creating Kubernetes NFS resources..."

    # Apply the NFS Kubernetes manifests using kustomize (templates already processed)
    echo "📦 Applying NFS manifests..."
    kubectl apply -k "${NFS_DIR}/kustomize"

    echo "✅ NFS PersistentVolume and StorageClass created"

    # Verify resources were created
    echo "🔍 Verifying Kubernetes resources..."
    if kubectl get storageclass nfs >/dev/null 2>&1; then
        echo "✅ StorageClass 'nfs' created"
    else
        echo "❌ StorageClass 'nfs' not found"
        exit 1
    fi

    if kubectl get pv nfs-media-pv >/dev/null 2>&1; then
        echo "✅ PersistentVolume 'nfs-media-pv' created"
        kubectl get pv nfs-media-pv
    else
        echo "❌ PersistentVolume 'nfs-media-pv' not found"
        exit 1
    fi
}

# Function to show usage instructions
show_usage_instructions() {
    echo ""
    echo "✅ === NFS Kubernetes Setup Complete ==="
    echo ""
    echo "💾 NFS server ${NFS_HOST} (${NFS_HOST_IP}) has been registered with the cluster"
    echo ""
    echo "📋 Kubernetes resources created:"
    echo "  - StorageClass: nfs"
    echo "  - PersistentVolume: nfs-media-pv (${NFS_STORAGE_CAPACITY}, ReadWriteMany)"
    echo ""
    echo "💡 To use NFS storage in your applications:"
    echo "  1. Set storageClassName: nfs in your PVC"
    echo "  2. Use accessMode: ReadWriteMany for shared access"
    echo ""
    echo "📝 Example PVC:"
    echo "---"
    echo "apiVersion: v1"
    echo "kind: PersistentVolumeClaim"
    echo "metadata:"
    echo "  name: my-nfs-pvc"
    echo "spec:"
    echo "  accessModes:"
    echo "    - ReadWriteMany"
    echo "  storageClassName: nfs"
    echo "  resources:"
    echo "    requests:"
    echo "      storage: 10Gi"
    echo ""
}

# Main execution
main() {
    resolve_nfs_host
    test_nfs_accessibility
    test_nfs_mount
    create_k8s_resources
    show_usage_instructions
}

# Run main function
echo "🔧 Starting NFS setup process..."
main "$@"