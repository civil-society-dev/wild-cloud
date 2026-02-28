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
SNAPSHOT_CONTROLLER_DIR="${CLUSTER_SETUP_DIR}/snapshot-controller"

echo "🔧 === Setting up Snapshot Controller ==="
echo ""

# Templates should already be compiled
echo "📦 Using pre-compiled snapshot-controller templates..."
if [ ! -d "${SNAPSHOT_CONTROLLER_DIR}/kustomize" ]; then
    echo "❌ ERROR: Compiled templates not found at ${SNAPSHOT_CONTROLLER_DIR}/kustomize"
    echo "Templates should be compiled before deployment."
    exit 1
fi

echo "🚀 Deploying Snapshot Controller..."
kubectl apply -k ${SNAPSHOT_CONTROLLER_DIR}/kustomize/

echo "⏳ Waiting for snapshot-controller to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/snapshot-controller -n kube-system || true

# Check if VolumeSnapshot CRDs are installed
echo "✔️ Checking VolumeSnapshot CRDs..."
kubectl api-resources | grep -q "snapshot.storage.k8s.io" && echo "✅ VolumeSnapshot CRDs found" || echo "⚠️ VolumeSnapshot CRDs not found"

echo ""
echo "✅ Snapshot Controller installed successfully"
echo ""
echo "💡 To verify the installation:"
echo "  kubectl get pods -n kube-system | grep snapshot-controller"
echo "  kubectl get crd | grep snapshot"
echo ""
echo "📘 To create a snapshot:"
echo "  kubectl apply -f - <<EOF"
echo "  apiVersion: snapshot.storage.k8s.io/v1"
echo "  kind: VolumeSnapshot"
echo "  metadata:"
echo "    name: test-snapshot"
echo "    namespace: default"
echo "  spec:"
echo "    volumeSnapshotClassName: longhorn-snapshot-class"
echo "    source:"
echo "      persistentVolumeClaimName: your-pvc"
echo "  EOF"
echo ""