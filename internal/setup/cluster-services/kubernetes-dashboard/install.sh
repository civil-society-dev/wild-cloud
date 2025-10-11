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
KUBERNETES_DASHBOARD_DIR="${CLUSTER_SETUP_DIR}/kubernetes-dashboard"

echo "🎮 === Setting up Kubernetes Dashboard ==="
echo ""

# Templates should already be compiled
echo "📦 Using pre-compiled Dashboard templates..."
if [ ! -d "${KUBERNETES_DASHBOARD_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${KUBERNETES_DASHBOARD_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

NAMESPACE="kubernetes-dashboard"

# Apply the official dashboard installation
echo "🚀 Installing Kubernetes Dashboard core components..."
kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.7.0/aio/deploy/recommended.yaml

# Wait for cert-manager certificates to be ready
echo "🔐 Waiting for cert-manager certificates to be ready..."
kubectl wait --for=condition=Ready certificate wildcard-internal-wild-cloud -n cert-manager --timeout=300s || echo "⚠️  Warning: Internal wildcard certificate not ready yet"
kubectl wait --for=condition=Ready certificate wildcard-wild-cloud -n cert-manager --timeout=300s || echo "⚠️  Warning: Wildcard certificate not ready yet"

# Copying cert-manager secrets to the dashboard namespace (if available)
echo "📋 Copying cert-manager secrets to dashboard namespace..."
if kubectl get secret wildcard-internal-wild-cloud-tls -n cert-manager >/dev/null 2>&1; then
    kubectl get secret wildcard-internal-wild-cloud-tls -n cert-manager -o yaml | \
        sed "s/namespace: cert-manager/namespace: ${NAMESPACE}/" | \
        kubectl apply -f -
else
    echo "⚠️  Warning: wildcard-internal-wild-cloud-tls secret not yet available"
fi

if kubectl get secret wildcard-wild-cloud-tls -n cert-manager >/dev/null 2>&1; then
    kubectl get secret wildcard-wild-cloud-tls -n cert-manager -o yaml | \
        sed "s/namespace: cert-manager/namespace: ${NAMESPACE}/" | \
        kubectl apply -f -
else
    echo "⚠️  Warning: wildcard-wild-cloud-tls secret not yet available"
fi

# Apply dashboard customizations using kustomize
echo "🔧 Applying dashboard customizations..."
kubectl apply -k "${KUBERNETES_DASHBOARD_DIR}/kustomize"

# Restart CoreDNS to pick up the changes
# echo "🔄 Restarting CoreDNS to pick up DNS changes..."
# kubectl delete pods -n kube-system -l k8s-app=kube-dns

# Wait for dashboard to be ready
echo "⏳ Waiting for Kubernetes Dashboard to be ready..."
kubectl rollout status deployment/kubernetes-dashboard -n $NAMESPACE --timeout=60s

echo ""
echo "✅ Kubernetes Dashboard installed successfully"
echo ""
# INTERNAL_DOMAIN should be available in environment (set from config before deployment)
if [ -n "${INTERNAL_DOMAIN}" ]; then
    echo "🌐 Access the dashboard at: https://dashboard.${INTERNAL_DOMAIN}"
else
    echo "💡 Access the dashboard via the configured internal domain"
fi
echo ""
echo "💡 To get the authentication token:"
echo "  kubectl create token admin-user -n kubernetes-dashboard"
echo ""
