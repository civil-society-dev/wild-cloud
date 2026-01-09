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
CLUSTER_SETUP_DIR="${INSTANCE_DIR}/setup/cluster-services"
METALLB_DIR="${CLUSTER_SETUP_DIR}/metallb"

echo "🔧 === Setting up MetalLB ==="
echo ""

# Templates should already be compiled
echo "📦 Using pre-compiled MetalLB templates..."
if [ ! -d "${METALLB_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${METALLB_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

echo "🚀 Deploying MetalLB installation..."
kubectl apply -k ${METALLB_DIR}/kustomize/installation

echo "⏳ Waiting for MetalLB controller to be ready..."
kubectl wait --for=condition=Available deployment/controller -n metallb-system --timeout=60s
echo "⏳ Extra buffer for webhook initialization..."
sleep 10

echo "⚙️  Applying MetalLB configuration..."
kubectl apply -k ${METALLB_DIR}/kustomize/configuration

echo ""
echo "✅ MetalLB installed and configured successfully"
echo ""
echo "💡 To verify the installation:"
echo "  kubectl get pods -n metallb-system"
echo "  kubectl get ipaddresspools.metallb.io -n metallb-system"
echo ""
echo "🌐 MetalLB will now provide LoadBalancer IPs for your services"
