# Snapshot Controller

The snapshot controller is a Kubernetes component that manages the lifecycle of volume snapshots. It works in conjunction with CSI drivers to enable backup and restore functionality for persistent volumes.

## Architecture

The snapshot functionality in Kubernetes is split into two components:
- **Snapshot Controller** (this service): Cluster-wide controller that manages VolumeSnapshot resources
- **CSI Snapshotter Sidecar**: Deployed with each CSI driver (e.g., Longhorn) to perform actual snapshot operations

## Installation

This service is installed automatically by Wild Cloud when setting up cluster services. It deploys:
- The snapshot-controller deployment (2 replicas with leader election)
- Required RBAC permissions
- VolumeSnapshot CRDs (if not already present)

## Dependencies

- Requires a CSI driver with snapshot support (e.g., Longhorn)
- Requires VolumeSnapshot CRDs (included with this service)

## Verification

After installation, verify the controller is running:
```bash
kubectl get pods -n kube-system | grep snapshot-controller
kubectl api-resources | grep snapshot
```

## Usage

Once installed, you can create VolumeSnapshots for any PVC managed by a CSI driver:
```yaml
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: my-snapshot
  namespace: default
spec:
  volumeSnapshotClassName: longhorn-snapshot-class
  source:
    persistentVolumeClaimName: my-pvc
```

## References

- [Kubernetes CSI Snapshot Documentation](https://kubernetes-csi.github.io/docs/snapshot-restore-feature.html)
- [External Snapshotter GitHub](https://github.com/kubernetes-csi/external-snapshotter)