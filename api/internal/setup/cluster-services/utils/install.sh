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
UTILS_DIR="${CLUSTER_SETUP_DIR}/utils"

echo "🔧 === Setting up Cluster Utilities ==="
echo ""

# Templates should already be compiled
echo "📦 Using pre-compiled utils templates..."
if [ ! -d "${UTILS_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${UTILS_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

echo "🚀 Applying utility manifests..."
kubectl apply -f ${UTILS_DIR}/kustomize/

echo ""
echo "✅ Cluster utilities installed successfully"
echo ""
echo "💡 Utility resources have been deployed to the cluster"
