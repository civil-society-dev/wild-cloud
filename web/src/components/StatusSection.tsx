import { Server, RefreshCw } from 'lucide-react';
import { useStatus } from '../hooks';
import { Card, CardHeader, CardTitle, CardContent, Button } from './ui';

export const StatusSection = () => {
  const { data: status, isLoading, error, refetch } = useStatus();

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Server className="h-5 w-5" />
          Server Status
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center justify-between">
          <span>Current Status</span>
          <Button 
            onClick={() => refetch()} 
            disabled={isLoading} 
            variant="outline" 
            size="sm"
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
            {isLoading ? 'Refreshing...' : 'Refresh'}
          </Button>
        </div>

        {error && (
          <div className="p-3 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded-md">
            <p className="text-sm text-red-800 dark:text-red-200">
              Failed to fetch status: {error.message}
            </p>
          </div>
        )}

        {status && (
          <div className="space-y-2">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <span className="text-sm font-medium">Status</span>
                <p className="text-muted-foreground">{status.status}</p>
              </div>
              <div>
                <span className="text-sm font-medium">Version</span>
                <p className="text-muted-foreground">{status.version}</p>
              </div>
            </div>
            
            {status.uptime && (
              <div>
                <span className="text-sm font-medium">Uptime</span>
                <p className="text-muted-foreground">{status.uptime}</p>
              </div>
            )}

            <pre className="p-4 bg-muted rounded-md text-sm overflow-auto max-h-48">
              {JSON.stringify(status, null, 2)}
            </pre>
          </div>
        )}

        {isLoading && !status && (
          <div className="flex items-center justify-center p-8">
            <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        )}
      </CardContent>
    </Card>
  );
};