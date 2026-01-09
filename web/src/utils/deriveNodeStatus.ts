import type { Node } from '../services/api/types';
import { NodeStatus } from '../types/nodeStatus';

export function deriveNodeStatus(node: Node): NodeStatus {
  // Priority 1: Active operations
  if (node.applyInProgress) {
    return NodeStatus.APPLYING;
  }

  if (node.configureInProgress) {
    return NodeStatus.CONFIGURING;
  }

  // Priority 2: Maintenance states
  if (node.maintenance) {
    if (node.applied) {
      return NodeStatus.MAINTENANCE;
    } else {
      return NodeStatus.REPROVISIONING;
    }
  }

  // Priority 3: Error states
  if (node.isReachable === false) {
    return NodeStatus.UNREACHABLE;
  }

  // Priority 4: Lifecycle progression
  if (!node.configured) {
    return NodeStatus.PENDING;
  }

  if (node.configured && !node.applied) {
    return NodeStatus.CONFIGURED;
  }

  if (node.applied) {
    // Check Kubernetes membership and readiness
    if (node.inKubernetes === true && node.kubernetesReady === true) {
      return NodeStatus.HEALTHY;
    }

    // In Kubernetes but not Ready
    if (node.inKubernetes === true && node.kubernetesReady === false) {
      return NodeStatus.DEGRADED;
    }

    // Applied and reachable but not yet in Kubernetes
    if (node.isReachable === true && node.inKubernetes !== true) {
      return NodeStatus.READY;
    }

    // Applied but status unknown (no cluster status data yet)
    if (node.isReachable === undefined && node.inKubernetes === undefined) {
      return NodeStatus.READY;
    }

    // Applied but not reachable at all
    if (node.isReachable === false) {
      return NodeStatus.UNREACHABLE;
    }
  }

  // Fallback
  return NodeStatus.UNKNOWN;
}
