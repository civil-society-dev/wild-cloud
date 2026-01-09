import { RefreshCw, Activity, AlertCircle } from 'lucide-react';
import { useStatus, useHealth, useMessages } from '../hooks';
import { formatTimestamp } from '../utils/formatters';
import { Message } from './Message';
import { Card, CardHeader, CardTitle, CardContent, Button, Badge } from './ui';

export const SystemStatus = () => {
  const { data: status, isLoading: statusLoading, error: statusError, refetch } = useStatus();
  const { mutate: checkHealth, isPending: healthLoading, error: healthError, data: healthData } = useHealth();
  const { messages, setMessage } = useMessages();

  // Handle health check messaging
  if (healthError) {
    setMessage('health', `Health check failed: ${healthError.message}`, 'error');
  } else if (healthData) {
    setMessage('health', `Service: ${healthData.service} - Status: ${healthData.status}`, 'success');
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>System Status</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex gap-2">
          <Button onClick={() => refetch()} disabled={statusLoading} variant="outline">
            <RefreshCw className={`mr-2 h-4 w-4 ${statusLoading ? 'animate-spin' : ''}`} />
            {statusLoading ? 'Checking...' : 'Refresh Status'}
          </Button>
          <Button onClick={() => checkHealth()} disabled={healthLoading} variant="outline">
            <Activity className="mr-2 h-4 w-4" />
            {healthLoading ? 'Checking...' : 'Check Health'}
          </Button>
        </div>
        
        {statusError && (
          <div className="p-3 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-md flex items-start gap-2">
            <AlertCircle className="h-4 w-4 text-red-600 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-red-800 dark:text-red-200">Status Error</p>
              <p className="text-sm text-red-700 dark:text-red-300">{statusError.message}</p>
            </div>
          </div>
        )}

        {healthError && (
          <div className="p-3 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-md flex items-start gap-2">
            <AlertCircle className="h-4 w-4 text-red-600 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-red-800 dark:text-red-200">Health Check Error</p>
              <p className="text-sm text-red-700 dark:text-red-300">{healthError.message}</p>
            </div>
          </div>
        )}

        {healthData && (
          <div className="p-3 bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800 rounded-md">
            <p className="text-sm text-green-800 dark:text-green-200">
              ✓ Service: {healthData.service} - Status: {healthData.status}
            </p>
          </div>
        )}
        
        <Message message={messages.health} />
        
        {status && (
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">Status</p>
              <Badge 
                variant={status.status === 'running' ? 'default' : 'destructive'} 
                className={`text-xs font-medium ${status.status === 'running' ? 'bg-emerald-600 hover:bg-emerald-700' : ''}`}
              >
                <div className={`w-2 h-2 rounded-full mr-2 ${status.status === 'running' ? 'bg-emerald-200' : 'bg-red-200'}`} />
                {status.status}
              </Badge>
            </div>
            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">Version</p>
              <p className="text-sm">{status.version}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">Uptime</p>
              <p className="text-sm">{status.uptime}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm font-medium text-muted-foreground">Last Updated</p>
              <p className="text-sm">{formatTimestamp(status.timestamp)}</p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
};