#!/bin/bash
set -e
set -o pipefail

# Ensure WILD_INSTANCE is set
if [ -z "${WILD_INSTANCE}" ]; then
    echo "❌ ERROR: WILD_INSTANCE is not set"
    exit 1
fi

# Ensure WILD_CENTRAL_DATA is set
if [ -z "${WILD_CENTRAL_DATA}" ]; then
    echo "❌ ERROR: WILD_CENTRAL_DATA is not set"
    exit 1
fi

# Ensure KUBECONFIG is set
if [ -z "${KUBECONFIG}" ]; then
    echo "❌ ERROR: KUBECONFIG is not set"
    exit 1
fi

INSTANCE_DIR="${WILD_CENTRAL_DATA}/instances/${WILD_INSTANCE}"
CLUSTER_SETUP_DIR="${INSTANCE_DIR}/setup/cluster-services"
NVIDIA_PLUGIN_DIR="${CLUSTER_SETUP_DIR}/nvidia-device-plugin"

echo "🎮 === Setting up NVIDIA Device Plugin ==="
echo ""

# Check if we have NVIDIA GPUs in the cluster
echo "🔍 Checking for worker nodes in the cluster..."

# Check if any worker nodes exist (device plugin only runs on worker nodes)
WORKER_NODES=$(kubectl get nodes --selector='!node-role.kubernetes.io/control-plane' -o name | wc -l)
if [ "$WORKER_NODES" -eq 0 ]; then
    echo "❌ ERROR: No worker nodes found in cluster. NVIDIA Device Plugin requires worker nodes."
    exit 1
fi

echo "✅ Found $WORKER_NODES worker node(s)"
echo ""

# Templates should already be compiled
echo "📦 Using pre-compiled NVIDIA Device Plugin templates..."
if [ ! -d "${NVIDIA_PLUGIN_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${NVIDIA_PLUGIN_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

echo "🚀 Deploying NVIDIA Device Plugin..."
kubectl apply -k ${NVIDIA_PLUGIN_DIR}/kustomize

echo "⏳ Waiting for NVIDIA Device Plugin DaemonSet to be ready..."
kubectl rollout status daemonset/nvidia-device-plugin-daemonset -n kube-system --timeout=120s

echo ""
echo "✅ NVIDIA Device Plugin installed successfully"
echo ""
echo "💡 To verify the installation:"
echo "  kubectl get pods -n kube-system | grep nvidia"
echo "  kubectl get nodes -o json | jq '.items[].status.capacity | select(has(\"nvidia.com/gpu\"))'"
echo ""
echo "🎮 GPU nodes should now be labeled with GPU product information:"
echo "  kubectl get nodes --show-labels | grep nvidia"
echo ""