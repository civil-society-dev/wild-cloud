# Longhorn Storage

See: [Longhorn Docs v 1.8.1](https://longhorn.io/docs/1.8.1/deploy/install/install-with-kubectl/)

## Installation Notes

- Manifest copied from https://raw.githubusercontent.com/longhorn/longhorn/v1.8.1/deploy/longhorn.yaml
- Using kustomize to apply custom configuration (see `kustomization.yaml`)

## Important Settings

- **Number of Replicas**: Set to 1 (default is 3) to accommodate smaller clusters
  - This avoids "degraded" volumes when fewer than 3 nodes are available
  - For production with 3+ nodes, consider changing back to 3 for better availability

## Common Operations

- View volumes: `kubectl get volumes.longhorn.io -n longhorn-system`
- Check volume status: `kubectl describe volumes.longhorn.io <volume-name> -n longhorn-system`
- Access Longhorn UI: Set up port-forwarding with `kubectl -n longhorn-system port-forward service/longhorn-frontend 8080:80`
