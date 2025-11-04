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
    // Check Kubernetes membership for healthy state
    if (node.inKubernetes === true) {
      return NodeStatus.HEALTHY;
    }

    // Applied but not yet in Kubernetes (could be provisioning or ready)
    if (node.isReachable === true) {
      return NodeStatus.READY;
    }

    // Applied but status unknown
    if (node.isReachable === undefined && node.inKubernetes === undefined) {
      return NodeStatus.READY;
    }

    // Applied but having issues
    if (node.inKubernetes === false) {
      return NodeStatus.DEGRADED;
    }
  }

  // Fallback
  return NodeStatus.UNKNOWN;
}
