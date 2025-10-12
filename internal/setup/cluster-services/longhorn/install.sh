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
LONGHORN_DIR="${CLUSTER_SETUP_DIR}/longhorn"

echo "🔧 === Setting up Longhorn ==="
echo ""

# Templates should already be compiled
echo "📦 Using pre-compiled Longhorn templates..."
if [ ! -d "${LONGHORN_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${LONGHORN_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

echo "🚀 Deploying Longhorn..."
kubectl apply -k ${LONGHORN_DIR}/kustomize/

echo "⏳ Waiting for Longhorn to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/longhorn-driver-deployer -n longhorn-system || true

echo ""
echo "✅ Longhorn installed successfully"
echo ""
echo "💡 To verify the installation:"
echo "  kubectl get pods -n longhorn-system"
echo "  kubectl get storageclass"
echo ""
echo "🌐 To access the Longhorn UI:"
echo "  kubectl port-forward -n longhorn-system svc/longhorn-frontend 8080:80"
