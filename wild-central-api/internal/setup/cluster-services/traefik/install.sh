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
TRAEFIK_DIR="${CLUSTER_SETUP_DIR}/traefik"

echo "🌐 === Setting up Traefik Ingress Controller ==="
echo ""

# Check MetalLB dependency
echo "🔍 Verifying MetalLB is ready (required for Traefik LoadBalancer service)..."
kubectl wait --for=condition=Ready pod -l component=controller -n metallb-system --timeout=60s 2>/dev/null || {
    echo "⚠️  MetalLB controller not ready, but continuing with Traefik installation"
    echo "💡 Note: Traefik LoadBalancer service may not get external IP without MetalLB"
}

# Install required CRDs first
echo "📦 Installing Gateway API CRDs..."
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml

echo "📦 Installing Traefik CRDs..."
kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v3.4/docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml

echo "⏳ Waiting for CRDs to be established..."
kubectl wait --for condition=established crd/gateways.gateway.networking.k8s.io --timeout=60s
kubectl wait --for condition=established crd/gatewayclasses.gateway.networking.k8s.io --timeout=60s
kubectl wait --for condition=established crd/ingressroutes.traefik.io --timeout=60s
kubectl wait --for condition=established crd/middlewares.traefik.io --timeout=60s

# Templates should already be compiled
echo "📦 Using pre-compiled Traefik templates..."
if [ ! -d "${TRAEFIK_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${TRAEFIK_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

# Apply Traefik using kustomize
echo "🚀 Deploying Traefik..."
kubectl apply -k ${TRAEFIK_DIR}/kustomize

# Wait for Traefik to be ready
echo "⏳ Waiting for Traefik to be ready..."
kubectl wait --for=condition=Available deployment/traefik -n traefik --timeout=120s

echo ""
echo "✅ Traefik installed successfully"
echo ""
echo "💡 To verify the installation:"
echo "  kubectl get pods -n traefik"
echo "  kubectl get svc -n traefik"
echo ""
