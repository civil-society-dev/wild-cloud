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
EXTERNALDNS_DIR="${CLUSTER_SETUP_DIR}/externaldns"

echo "🌐 === Setting up ExternalDNS ==="
echo ""

# Check cert-manager dependency
echo "🔍 Verifying cert-manager is ready (required for ExternalDNS)..."
kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=60s 2>/dev/null && \
kubectl wait --for=condition=Available deployment/cert-manager-webhook -n cert-manager --timeout=60s 2>/dev/null || {
    echo "⚠️  cert-manager not ready, but continuing with ExternalDNS installation"
    echo "💡 Note: ExternalDNS may not work properly without cert-manager"
}

# Templates should already be compiled
echo "📦 Using pre-compiled ExternalDNS templates..."
if [ ! -d "${EXTERNALDNS_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${EXTERNALDNS_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

# Apply ExternalDNS manifests using kustomize
echo "🚀 Deploying ExternalDNS..."
kubectl apply -k ${EXTERNALDNS_DIR}/kustomize

# Setup Cloudflare API token secret
echo "🔐 Creating Cloudflare API token secret..."
SECRETS_FILE="${WILD_API_DATA_DIR}/instances/${WILD_INSTANCE}/secrets.yaml"
CLOUDFLARE_API_TOKEN=$(yq '.cloudflare.token' "$SECRETS_FILE" 2>/dev/null | tr -d '"')

if [ -z "$CLOUDFLARE_API_TOKEN" ] || [ "$CLOUDFLARE_API_TOKEN" = "null" ]; then
    echo "❌ ERROR: Cloudflare API token not found."
    echo "💡 Please set: wild secret set cloudflare.token YOUR_TOKEN"
    exit 1
fi
kubectl create secret generic cloudflare-api-token \
  --namespace externaldns \
  --from-literal=api-token="${CLOUDFLARE_API_TOKEN}" \
  --dry-run=client -o yaml | kubectl apply -f -

# Wait for ExternalDNS to be ready
echo "⏳ Waiting for Cloudflare ExternalDNS to be ready..."
kubectl rollout status deployment/external-dns -n externaldns --timeout=60s

# echo "⏳ Waiting for CoreDNS ExternalDNS to be ready..."
# kubectl rollout status deployment/external-dns-coredns -n externaldns --timeout=60s

echo ""
echo "✅ ExternalDNS installed successfully"
echo ""
echo "💡 To verify the installation:"
echo "  kubectl get pods -n externaldns"
echo "  kubectl logs -n externaldns -l app=external-dns -f"
echo "  kubectl logs -n externaldns -l app=external-dns-coredns -f"
echo ""
