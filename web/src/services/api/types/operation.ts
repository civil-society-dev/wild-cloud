export interface BootstrapProgress {
  current_step: number;
  step_name: string;
  attempt: number;
  max_attempts: number;
  step_description: string;
}

export interface OperationDetails {
  bootstrap?: BootstrapProgress;
}

export interface Operation {
  id: string;
  instance_name: string;
  type: string;
  target: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  message: string;
  progress: number;
  started: string;
  completed?: string;
  error?: string;
  details?: OperationDetails;
}

export interface OperationListResponse {
  operations: Operation[];
}

export interface OperationResponse {
  operation_id: string;
  message: string;
}
