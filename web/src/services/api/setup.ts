import { apiClient } from './client';

export interface PhaseCheck {
  phase: string;
  complete: boolean;
  available: boolean;
  prerequisites: string[];
  missingItems: string[];
}

export interface SetupStatus {
  currentPhase: string;
  availablePhases: string[];
  phaseChecks: Record<string, PhaseCheck>;
}

export async function getSetupStatus(instanceName: string): Promise<SetupStatus> {
  return apiClient.get<SetupStatus>(`/api/v1/instances/${instanceName}/setup/status`);
}
