import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { useAppEnhanced, useAppLogs } from '@/hooks/useApps';
import { RefreshCw } from 'lucide-react';

interface AppLogsDialogProps {
  instanceName: string;
  appName: string;
  open: boolean;
  onClose: () => void;
}

export function AppLogsDialog({
  instanceName,
  appName,
  open,
  onClose,
}: AppLogsDialogProps) {
  const [logParams, setLogParams] = useState({ tail: 100, sinceSeconds: 3600 });

  const { data: appDetails } = useAppEnhanced(instanceName, appName);
  const { data: logs, refetch: refetchLogs } = useAppLogs(
    instanceName,
    appName,
    open ? logParams : undefined
  );

  const getStatusBadge = (status: string) => {
    const variants: Record<string, 'success' | 'destructive' | 'warning' | 'outline'> = {
      running: 'success',
      error: 'destructive',
      deploying: 'outline',
      stopped: 'warning',
      added: 'outline',
      deployed: 'outline',
    };

    return (
      <Badge variant={variants[status] || 'outline'}>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </Badge>
    );
  };

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-5xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            {appName}
            {appDetails && getStatusBadge(appDetails.status)}
          </DialogTitle>
          <DialogDescription>
            Application logs
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <div className="flex gap-2">
              <select
                value={logParams.tail}
                onChange={(e) => setLogParams({ ...logParams, tail: parseInt(e.target.value) })}
                className="px-3 py-1 border rounded text-sm"
              >
                <option value={50}>Last 50 lines</option>
                <option value={100}>Last 100 lines</option>
                <option value={200}>Last 200 lines</option>
                <option value={500}>Last 500 lines</option>
              </select>
            </div>
            <Button variant="outline" size="sm" onClick={() => refetchLogs()}>
              <RefreshCw className="h-4 w-4 mr-2" />
              Refresh
            </Button>
          </div>

          <Card>
            <CardContent className="p-4">
              <div className="bg-black text-green-400 font-mono text-xs p-4 rounded-lg max-h-96 overflow-y-auto">
                {logs && logs.logs && Array.isArray(logs.logs) && logs.logs.length > 0 ? (
                  logs.logs.map((line, idx) => {
                    if (typeof line === 'string') {
                      return (
                        <div key={idx} className="whitespace-pre-wrap break-all">
                          {line}
                        </div>
                      );
                    } else if (line && typeof line === 'object' && 'message' in line) {
                      const timestamp = line.timestamp ? new Date(line.timestamp).toLocaleTimeString() : '';
                      return (
                        <div key={idx} className="whitespace-pre-wrap break-all">
                          {timestamp && <span className="text-gray-500">[{timestamp}] </span>}
                          {line.message}
                        </div>
                      );
                    } else {
                      return (
                        <div key={idx} className="whitespace-pre-wrap break-all">
                          {JSON.stringify(line)}
                        </div>
                      );
                    }
                  })
                ) : logs && typeof logs === 'object' && !Array.isArray(logs) ? (
                  <div className="whitespace-pre-wrap break-all">
                    {JSON.stringify(logs, null, 2)}
                  </div>
                ) : (
                  <p className="text-gray-500">No logs available</p>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </DialogContent>
    </Dialog>
  );
}
