export enum NodeStatus {
  // Discovery Phase
  DISCOVERED = "discovered",

  // Configuration Phase
  PENDING = "pending",
  CONFIGURING = "configuring",
  CONFIGURED = "configured",

  // Deployment Phase
  APPLYING = "applying",
  PROVISIONING = "provisioning",

  // Operational Phase
  READY = "ready",
  HEALTHY = "healthy",

  // Maintenance States
  MAINTENANCE = "maintenance",
  REPROVISIONING = "reprovisioning",

  // Error States
  UNREACHABLE = "unreachable",
  DEGRADED = "degraded",
  FAILED = "failed",

  // Special States
  UNKNOWN = "unknown",
  ORPHANED = "orphaned"
}

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
