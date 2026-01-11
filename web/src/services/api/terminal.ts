import { apiClient } from './client';

export interface TerminalExecResponse {
  stdout: string;
  stderr: string;
  exit_code: number;
}

export const terminalApi = {
  async exec(instanceId: string, command: string): Promise<TerminalExecResponse> {
    return apiClient.post(`/api/v1/instances/${instanceId}/terminal/exec`, { command });
  },
};
