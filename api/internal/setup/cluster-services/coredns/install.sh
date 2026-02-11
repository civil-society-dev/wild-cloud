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
COREDNS_DIR="${CLUSTER_SETUP_DIR}/coredns"

echo "🔧 === Setting up CoreDNS ==="
echo ""

# Templates should already be compiled
echo "📦 Using pre-compiled CoreDNS templates..."
if [ ! -d "${COREDNS_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${COREDNS_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

# Apply the custom DNS override using kustomize
# TODO: Is this needed now that we are no longer on k3s?
echo "🚀 Applying CoreDNS custom override configuration..."
kubectl apply -k "${COREDNS_DIR}/kustomize/"

echo "🔄 Restarting CoreDNS pods to apply changes..."
kubectl rollout restart deployment/coredns -n kube-system
echo "⏳ Waiting for CoreDNS rollout to complete..."
kubectl rollout status deployment/coredns -n kube-system

echo ""
echo "✅ CoreDNS configured successfully"
echo ""
echo "💡 To verify the installation:"
echo "  kubectl get pods -n kube-system -l k8s-app=kube-dns"
echo "  kubectl get svc -n kube-system coredns"
echo "  kubectl describe svc -n kube-system coredns"
echo ""
echo "📋 To view CoreDNS logs:"
echo "  kubectl logs -n kube-system -l k8s-app=kube-dns -f"
