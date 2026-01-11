import { useState, useMemo, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import { Card, CardContent } from '@/components/ui/card';
import { useAppEnhanced, useAppLogs } from '@/hooks/useApps';
import { RefreshCw } from 'lucide-react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

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
  const [logParams, setLogParams] = useState<{
    tail: number;
    sinceSeconds: number;
    pod?: string;
    container?: string;
  }>({ tail: 100, sinceSeconds: 3600 });

  const { data: appDetails } = useAppEnhanced(instanceName, appName);
  const { data: logs, refetch: refetchLogs } = useAppLogs(
    instanceName,
    appName,
    open ? logParams : undefined
  );

  // Build a list of selectable "components" - each is a pod/container combination
  // For apps like Mastodon where each pod has a different container, this gives us
  // { label: "sidekiq", podName: "mastodon-sidekiq-xxx", containerName: "sidekiq" }
  const components = useMemo(() => {
    if (!appDetails?.runtime?.pods) return [];
    const result: { label: string; podName: string; containerName: string }[] = [];
    appDetails.runtime.pods.forEach((pod) => {
      pod.containers?.forEach((c) => {
        result.push({
          label: c.name,
          podName: pod.name,
          containerName: c.name,
        });
      });
    });
    return result;
  }, [appDetails?.runtime?.pods]);

  // Track selected component index
  const [selectedIndex, setSelectedIndex] = useState<number>(0);

  // Update logParams when selected component changes
  useEffect(() => {
    if (components.length > 0) {
      const comp = components[selectedIndex] || components[0];
      setLogParams((prev) => ({
        ...prev,
        pod: comp.podName,
        container: comp.containerName,
      }));
    }
  }, [components, selectedIndex]);

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
          <div className="flex flex-wrap gap-4 items-center">
            <div className="flex items-center gap-2">
              <Label htmlFor="tail-select">Lines:</Label>
              <Select
                value={logParams.tail.toString()}
                onValueChange={(v) => setLogParams({ ...logParams, tail: parseInt(v) })}
              >
                <SelectTrigger id="tail-select" className="w-24">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="50">50</SelectItem>
                  <SelectItem value="100">100</SelectItem>
                  <SelectItem value="200">200</SelectItem>
                  <SelectItem value="500">500</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {components.length > 1 && (
              <div className="flex items-center gap-2">
                <Label htmlFor="component-select">Component:</Label>
                <Select
                  value={selectedIndex.toString()}
                  onValueChange={(v) => setSelectedIndex(parseInt(v))}
                >
                  <SelectTrigger id="component-select" className="w-40">
                    <SelectValue placeholder="Select component" />
                  </SelectTrigger>
                  <SelectContent>
                    {components.map((comp, idx) => (
                      <SelectItem key={`${comp.podName}-${comp.containerName}`} value={idx.toString()}>
                        {comp.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            <div className="ml-auto">
              <Button variant="outline" size="sm" onClick={() => refetchLogs()}>
                <RefreshCw className="h-4 w-4 mr-2" />
                Refresh
              </Button>
            </div>
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
