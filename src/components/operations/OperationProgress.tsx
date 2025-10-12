import { Card } from '../ui/card';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Loader2, CheckCircle, AlertCircle, XCircle, Clock } from 'lucide-react';
import { useOperation } from '../../hooks/useOperations';

interface OperationProgressProps {
  operationId: string;
  onComplete?: () => void;
  onError?: (error: string) => void;
  showDetails?: boolean;
}

export function OperationProgress({
  operationId,
  onComplete,
  onError,
  showDetails = true
}: OperationProgressProps) {
  const { operation, error, isLoading, cancel, isCancelling } = useOperation(operationId);

  // Handle operation completion
  if (operation?.status === 'completed' && onComplete) {
    setTimeout(onComplete, 100); // Delay slightly to ensure state updates
  }

  // Handle operation error
  if (operation?.status === 'failed' && onError && operation.error) {
    setTimeout(() => onError(operation.error!), 100);
  }

  const getStatusIcon = () => {
    if (isLoading) {
      return <Loader2 className="h-5 w-5 animate-spin text-blue-500" />;
    }

    switch (operation?.status) {
      case 'pending':
        return <Clock className="h-5 w-5 text-gray-500" />;
      case 'running':
        return <Loader2 className="h-5 w-5 animate-spin text-blue-500" />;
      case 'completed':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'failed':
        return <AlertCircle className="h-5 w-5 text-red-500" />;
      case 'cancelled':
        return <XCircle className="h-5 w-5 text-orange-500" />;
      default:
        return null;
    }
  };

  const getStatusBadge = () => {
    if (isLoading) {
      return <Badge variant="default">Loading...</Badge>;
    }

    const variants: Record<string, 'secondary' | 'default' | 'success' | 'destructive' | 'warning'> = {
      pending: 'secondary',
      running: 'default',
      completed: 'success',
      failed: 'destructive',
      cancelled: 'warning',
    };

    const labels: Record<string, string> = {
      pending: 'Pending',
      running: 'Running',
      completed: 'Completed',
      failed: 'Failed',
      cancelled: 'Cancelled',
    };

    const status = operation?.status || 'pending';

    return (
      <Badge variant={variants[status]}>
        {labels[status] || status}
      </Badge>
    );
  };

  const getProgressPercentage = () => {
    if (!operation) return 0;
    if (operation.status === 'completed') return 100;
    if (operation.status === 'failed' || operation.status === 'cancelled') return 0;
    return operation.progress || 0;
  };

  if (error) {
    return (
      <Card className="p-4 border-red-200 bg-red-50 dark:bg-red-950/20">
        <div className="flex items-center gap-3">
          <AlertCircle className="h-5 w-5 text-red-500" />
          <div className="flex-1">
            <p className="text-sm font-medium text-red-900 dark:text-red-100">
              Error loading operation
            </p>
            <p className="text-xs text-red-700 dark:text-red-300 mt-1">
              {error.message}
            </p>
          </div>
        </div>
      </Card>
    );
  }

  if (isLoading) {
    return (
      <Card className="p-4">
        <div className="flex items-center gap-3">
          <Loader2 className="h-5 w-5 animate-spin text-primary" />
          <span className="text-sm">Loading operation status...</span>
        </div>
      </Card>
    );
  }

  const progressPercentage = getProgressPercentage();
  const canCancel = operation?.status === 'pending' || operation?.status === 'running';

  return (
    <Card className="p-4">
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {getStatusIcon()}
            <div>
              <p className="text-sm font-medium">
                {operation?.type || 'Operation'}
              </p>
              {operation?.message && (
                <p className="text-xs text-muted-foreground mt-0.5">
                  {operation.message}
                </p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            {getStatusBadge()}
            {canCancel && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => cancel()}
                disabled={isCancelling}
              >
                {isCancelling ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  'Cancel'
                )}
              </Button>
            )}
          </div>
        </div>

        {(operation?.status === 'running' || operation?.status === 'pending') && (
          <div className="space-y-1">
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>Progress</span>
              <span>{progressPercentage}%</span>
            </div>
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div
                className="bg-primary h-2 rounded-full transition-all duration-300"
                style={{ width: `${progressPercentage}%` }}
              />
            </div>
          </div>
        )}

        {operation?.error && (
          <div className="p-2 bg-red-50 dark:bg-red-950/20 rounded border border-red-200 dark:border-red-800">
            <p className="text-xs text-red-700 dark:text-red-300">
              Error: {operation.error}
            </p>
          </div>
        )}

        {showDetails && operation && (
          <div className="pt-2 border-t text-xs text-muted-foreground space-y-1">
            <div className="flex justify-between">
              <span>Operation ID:</span>
              <span className="font-mono">{operation.id}</span>
            </div>
            {operation.started && (
              <div className="flex justify-between">
                <span>Started:</span>
                <span>{new Date(operation.started).toLocaleString()}</span>
              </div>
            )}
            {operation.completed && (
              <div className="flex justify-between">
                <span>Completed:</span>
                <span>{new Date(operation.completed).toLocaleString()}</span>
              </div>
            )}
          </div>
        )}
      </div>
    </Card>
  );
}
