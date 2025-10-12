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
DOCKER_REGISTRY_DIR="${CLUSTER_SETUP_DIR}/docker-registry"

echo "🔧 === Setting up Docker Registry ==="
echo ""

# Templates should already be compiled
echo "📦 Using pre-compiled Docker Registry templates..."
if [ ! -d "${DOCKER_REGISTRY_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${DOCKER_REGISTRY_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

echo "🚀 Deploying Docker Registry..."
kubectl apply -k "${DOCKER_REGISTRY_DIR}/kustomize"

echo "⏳ Waiting for Docker Registry to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/docker-registry -n docker-registry

echo ""
echo "✅ Docker Registry installed successfully"
echo ""
echo "📊 Deployment status:"
kubectl get pods -n docker-registry
kubectl get services -n docker-registry
echo ""
echo "💡 To use the registry:"
echo "  docker tag myimage registry.local/myimage"
echo "  docker push registry.local/myimage"