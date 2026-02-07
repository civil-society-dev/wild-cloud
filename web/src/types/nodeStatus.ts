export const NodeStatus = {
  DISCOVERED: "discovered",
  PENDING: "pending",
  CONFIGURING: "configuring",
  CONFIGURED: "configured",
  APPLYING: "applying",
  PROVISIONING: "provisioning",
  READY: "ready",
  HEALTHY: "healthy",
  MAINTENANCE: "maintenance",
  REPROVISIONING: "reprovisioning",
  UNREACHABLE: "unreachable",
  DEGRADED: "degraded",
  FAILED: "failed",
  UNKNOWN: "unknown",
  ORPHANED: "orphaned",
} as const;

export type NodeStatus = typeof NodeStatus[keyof typeof NodeStatus];

export interface StatusDesign {
  status: NodeStatus;
  color: string;
  bgColor: string;
  icon: string;
  label: string;
  description: string;
  nextAction?: string;
  severity: "info" | "warning" | "error" | "success" | "neutral";
}
