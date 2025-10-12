import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Badge } from '../ui/badge';
import { Button } from '../ui/button';
import { Loader2, CheckCircle, AlertCircle, XCircle, Clock, ChevronDown, ChevronUp } from 'lucide-react';
import { useCancelOperation, type Operation } from '../../services/api';
import { useState } from 'react';

interface OperationCardProps {
  operation: Operation;
  expandable?: boolean;
}

export function OperationCard({ operation, expandable = false }: OperationCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const { mutate: cancelOperation, isPending: isCancelling } = useCancelOperation();

  const getStatusIcon = () => {
    switch (operation.status) {
      case 'pending':
        return <Clock className="h-4 w-4 text-gray-500" />;
      case 'running':
        return <Loader2 className="h-4 w-4 animate-spin text-blue-500" />;
      case 'completed':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'failed':
        return <AlertCircle className="h-4 w-4 text-red-500" />;
      case 'cancelled':
        return <XCircle className="h-4 w-4 text-orange-500" />;
      default:
        return null;
    }
  };

  const getStatusBadge = () => {
    const variants: Record<string, 'secondary' | 'default' | 'destructive' | 'outline'> = {
      pending: 'secondary',
      running: 'default',
      completed: 'outline',
      failed: 'destructive',
      cancelled: 'secondary',
    };

    return (
      <Badge variant={variants[operation.status]}>
        {operation.status.charAt(0).toUpperCase() + operation.status.slice(1)}
      </Badge>
    );
  };

  const canCancel = operation.status === 'pending' || operation.status === 'running';

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3 flex-1 min-w-0">
            {getStatusIcon()}
            <div className="flex-1 min-w-0">
              <CardTitle className="text-base">
                {operation.type}
              </CardTitle>
              {operation.target && (
                <p className="text-sm text-muted-foreground mt-1">
                  Target: {operation.target}
                </p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {getStatusBadge()}
            {canCancel && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => cancelOperation({ operationId: operation.id, instanceName: operation.instance_name })}
                disabled={isCancelling}
              >
                {isCancelling ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  'Cancel'
                )}
              </Button>
            )}
            {expandable && (
              <Button
                size="sm"
                variant="ghost"
                onClick={() => setIsExpanded(!isExpanded)}
              >
                {isExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
              </Button>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-3">
        {operation.message && (
          <p className="text-sm text-muted-foreground">
            {operation.message}
          </p>
        )}

        {(operation.status === 'running' || operation.status === 'pending') && (
          <div className="space-y-1">
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>Progress</span>
              <span>{operation.progress}%</span>
            </div>
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div
                className="bg-primary h-2 rounded-full transition-all duration-300"
                style={{ width: `${operation.progress}%` }}
              />
            </div>
          </div>
        )}

        {operation.error && (
          <div className="p-2 bg-red-50 dark:bg-red-950/20 rounded border border-red-200 dark:border-red-800">
            <p className="text-xs text-red-700 dark:text-red-300">
              {operation.error}
            </p>
          </div>
        )}

        {isExpanded && (
          <div className="pt-3 border-t text-xs text-muted-foreground space-y-2">
            <div className="flex justify-between">
              <span>Operation ID:</span>
              <span className="font-mono">{operation.id}</span>
            </div>
            <div className="flex justify-between">
              <span>Started:</span>
              <span>{new Date(operation.started).toLocaleString()}</span>
            </div>
            {operation.completed && (
              <div className="flex justify-between">
                <span>Completed:</span>
                <span>{new Date(operation.completed).toLocaleString()}</span>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
