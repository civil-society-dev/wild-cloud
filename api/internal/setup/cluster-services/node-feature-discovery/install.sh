#!/bin/bash
set -e
set -o pipefail

# Ensure WILD_INSTANCE is set
if [ -z "${WILD_INSTANCE}" ]; then
    echo "ERROR: WILD_INSTANCE is not set"
    exit 1
fi

# Ensure WILD_API_DATA_DIR is set
if [ -z "${WILD_API_DATA_DIR}" ]; then
    echo "ERROR: WILD_API_DATA_DIR is not set"
    exit 1
fi

# Ensure KUBECONFIG is set
if [ -z "${KUBECONFIG}" ]; then
    echo "ERROR: KUBECONFIG is not set"
    exit 1
fi

INSTANCE_DIR="${WILD_API_DATA_DIR}/instances/${WILD_INSTANCE}"
CLUSTER_SETUP_DIR="${INSTANCE_DIR}/setup/cluster-services"
NFD_DIR="${CLUSTER_SETUP_DIR}/node-feature-discovery"

echo "🔧 === Setting up Node Feature Discovery ==="
echo ""

# Templates should already be compiled
echo "📦 Using pre-compiled Node Feature Discovery templates..."
if [ ! -d "${NFD_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${NFD_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

echo "🚀 Deploying Node Feature Discovery..."
kubectl apply -k "${NFD_DIR}/kustomize"

echo "⏳ Waiting for Node Feature Discovery DaemonSet to be ready..."
kubectl rollout status daemonset/node-feature-discovery-worker -n node-feature-discovery --timeout=300s

echo ""
echo "✅ Node Feature Discovery installed successfully"
echo ""
echo "💡 To verify the installation:"
echo "  kubectl get pods -n node-feature-discovery"
echo "  kubectl get nodes --show-labels | grep feature.node.kubernetes.io"
echo ""
echo "🎮 GPU nodes should now be labeled with GPU device information:"
echo "  kubectl get nodes --show-labels | grep pci-10de"