import { NodeStatus, type StatusDesign } from '../types/nodeStatus';

export const statusDesigns: Record<NodeStatus, StatusDesign> = {
  [NodeStatus.DISCOVERED]: {
    status: NodeStatus.DISCOVERED,
    color: "text-purple-700",
    bgColor: "bg-purple-50",
    icon: "MagnifyingGlassIcon",
    label: "Discovered",
    description: "Node detected on network but not yet configured",
    nextAction: "Configure node settings",
    severity: "info"
  },

  [NodeStatus.PENDING]: {
    status: NodeStatus.PENDING,
    color: "text-gray-700",
    bgColor: "bg-gray-50",
    icon: "ClockIcon",
    label: "Pending",
    description: "Node awaiting configuration",
    nextAction: "Configure and apply settings",
    severity: "neutral"
  },

  [NodeStatus.CONFIGURING]: {
    status: NodeStatus.CONFIGURING,
    color: "text-blue-700",
    bgColor: "bg-blue-50",
    icon: "ArrowPathIcon",
    label: "Configuring",
    description: "Node configuration in progress",
    severity: "info"
  },

  [NodeStatus.CONFIGURED]: {
    status: NodeStatus.CONFIGURED,
    color: "text-indigo-700",
    bgColor: "bg-indigo-50",
    icon: "DocumentCheckIcon",
    label: "Configured",
    description: "Node configured but not yet applied",
    nextAction: "Apply configuration to node",
    severity: "info"
  },

  [NodeStatus.APPLYING]: {
    status: NodeStatus.APPLYING,
    color: "text-blue-700",
    bgColor: "bg-blue-50",
    icon: "ArrowPathIcon",
    label: "Applying",
    description: "Applying configuration to node",
    severity: "info"
  },

  [NodeStatus.PROVISIONING]: {
    status: NodeStatus.PROVISIONING,
    color: "text-blue-700",
    bgColor: "bg-blue-50",
    icon: "ArrowPathIcon",
    label: "Provisioning",
    description: "Node is being provisioned with Talos",
    severity: "info"
  },

  [NodeStatus.READY]: {
    status: NodeStatus.READY,
    color: "text-green-700",
    bgColor: "bg-green-50",
    icon: "CheckCircleIcon",
    label: "Ready",
    description: "Node is ready and operational",
    severity: "success"
  },

  [NodeStatus.HEALTHY]: {
    status: NodeStatus.HEALTHY,
    color: "text-emerald-700",
    bgColor: "bg-emerald-50",
    icon: "HeartIcon",
    label: "Healthy",
    description: "Node is healthy and part of Kubernetes cluster",
    severity: "success"
  },

  [NodeStatus.MAINTENANCE]: {
    status: NodeStatus.MAINTENANCE,
    color: "text-yellow-700",
    bgColor: "bg-yellow-50",
    icon: "WrenchScrewdriverIcon",
    label: "Maintenance",
    description: "Node is in maintenance mode",
    severity: "warning"
  },

  [NodeStatus.REPROVISIONING]: {
    status: NodeStatus.REPROVISIONING,
    color: "text-orange-700",
    bgColor: "bg-orange-50",
    icon: "ArrowPathIcon",
    label: "Reprovisioning",
    description: "Node is being reprovisioned",
    severity: "warning"
  },

  [NodeStatus.UNREACHABLE]: {
    status: NodeStatus.UNREACHABLE,
    color: "text-red-700",
    bgColor: "bg-red-50",
    icon: "ExclamationTriangleIcon",
    label: "Unreachable",
    description: "Node cannot be contacted",
    nextAction: "Check network connectivity",
    severity: "error"
  },

  [NodeStatus.DEGRADED]: {
    status: NodeStatus.DEGRADED,
    color: "text-orange-700",
    bgColor: "bg-orange-50",
    icon: "ExclamationTriangleIcon",
    label: "Degraded",
    description: "Node is experiencing issues",
    nextAction: "Check node health",
    severity: "warning"
  },

  [NodeStatus.FAILED]: {
    status: NodeStatus.FAILED,
    color: "text-red-700",
    bgColor: "bg-red-50",
    icon: "XCircleIcon",
    label: "Failed",
    description: "Node operation failed",
    nextAction: "Review logs and retry",
    severity: "error"
  },

  [NodeStatus.UNKNOWN]: {
    status: NodeStatus.UNKNOWN,
    color: "text-gray-700",
    bgColor: "bg-gray-50",
    icon: "QuestionMarkCircleIcon",
    label: "Unknown",
    description: "Node status cannot be determined",
    nextAction: "Check node connection",
    severity: "neutral"
  },

  [NodeStatus.ORPHANED]: {
    status: NodeStatus.ORPHANED,
    color: "text-purple-700",
    bgColor: "bg-purple-50",
    icon: "ExclamationTriangleIcon",
    label: "Orphaned",
    description: "Node exists in Kubernetes but not in configuration",
    nextAction: "Add to configuration or remove from cluster",
    severity: "warning"
  }
};
