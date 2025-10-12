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
}

export interface OperationListResponse {
  operations: Operation[];
}

export interface OperationResponse {
  operation_id: string;
  message: string;
}
