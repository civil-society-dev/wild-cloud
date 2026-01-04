import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useAppEnhanced, useAppEvents } from '@/hooks/useApps';
import {
  RefreshCw,
  AlertCircle,
  CheckCircle,
} from 'lucide-react';

interface AppStatusDialogProps {
  instanceName: string;
  appName: string;
  open: boolean;
  onClose: () => void;
}

export function AppStatusDialog({
  instanceName,
  appName,
  open,
  onClose,
}: AppStatusDialogProps) {
  const { data: appDetails, isLoading, refetch } = useAppEnhanced(instanceName, appName);
  const { data: eventsData } = useAppEvents(instanceName, appName, 20);

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

  const getPodStatusColor = (status: string | undefined) => {
    if (!status) return 'text-muted-foreground';
    const lowerStatus = status.toLowerCase();
    if (lowerStatus.includes('running')) return 'text-green-600 dark:text-green-400';
    if (lowerStatus.includes('pending')) return 'text-yellow-600 dark:text-yellow-400';
    if (lowerStatus.includes('failed')) return 'text-red-600 dark:text-red-400';
    return 'text-muted-foreground';
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
            Application runtime status
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex justify-end">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCw className="h-4 w-4 mr-2" />
              Refresh
            </Button>
          </div>

          {isLoading ? (
            <div className="space-y-4">
              <Skeleton className="h-32 w-full" />
              <Skeleton className="h-48 w-full" />
            </div>
          ) : appDetails?.runtime ? (
            <>
              {appDetails.runtime.replicas && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">Replicas</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-4 gap-2 text-sm">
                      <div className="bg-muted rounded p-2">
                        <p className="text-xs text-muted-foreground">Desired</p>
                        <p className="font-semibold">{appDetails.runtime.replicas.desired}</p>
                      </div>
                      <div className="bg-muted rounded p-2">
                        <p className="text-xs text-muted-foreground">Current</p>
                        <p className="font-semibold">{appDetails.runtime.replicas.current}</p>
                      </div>
                      <div className="bg-muted rounded p-2">
                        <p className="text-xs text-muted-foreground">Ready</p>
                        <p className="font-semibold">{appDetails.runtime.replicas.ready}</p>
                      </div>
                      <div className="bg-muted rounded p-2">
                        <p className="text-xs text-muted-foreground">Available</p>
                        <p className="font-semibold">{appDetails.runtime.replicas.available}</p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              )}

              {appDetails.runtime.pods && appDetails.runtime.pods.length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">Pods</CardTitle>
                    <CardDescription>{appDetails.runtime.pods.length} pod(s)</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {appDetails.runtime.pods.map((pod) => (
                        <div
                          key={pod.name}
                          className="border rounded-lg p-3 hover:bg-muted/50 transition-colors"
                        >
                          <div className="flex items-start justify-between mb-2">
                            <div className="flex-1 min-w-0">
                              <p className="text-sm font-medium truncate">{pod.name}</p>
                              {pod.node && (
                                <p className="text-xs text-muted-foreground">Node: {pod.node}</p>
                              )}
                            </div>
                            <div className="flex gap-2 ml-2">
                              <Badge variant="outline" className={getPodStatusColor(pod.status)}>
                                {pod.status}
                              </Badge>
                            </div>
                          </div>
                          <div className="grid grid-cols-3 gap-2 text-xs">
                            <div>
                              <span className="text-muted-foreground">Ready:</span>{' '}
                              <span className="font-medium">{pod.ready}</span>
                            </div>
                            <div>
                              <span className="text-muted-foreground">Restarts:</span>{' '}
                              <span className="font-medium">{pod.restarts}</span>
                            </div>
                            <div>
                              <span className="text-muted-foreground">Age:</span>{' '}
                              <span className="font-medium">{pod.age}</span>
                            </div>
                          </div>
                          {pod.ip && (
                            <div className="text-xs mt-1">
                              <span className="text-muted-foreground">IP:</span>{' '}
                              <span className="font-mono">{pod.ip}</span>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              )}

              {appDetails.runtime.resources && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">Resource Usage</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {appDetails.runtime.resources.cpu && (
                      <div>
                        <div className="flex justify-between text-sm mb-1">
                          <span>CPU</span>
                          <span className="font-mono text-xs">
                            {appDetails.runtime.resources.cpu.used} / {appDetails.runtime.resources.cpu.limit}
                          </span>
                        </div>
                        <div className="w-full bg-muted rounded-full h-2">
                          <div
                            className="bg-primary rounded-full h-2 transition-all"
                            style={{ width: `${Math.min(appDetails.runtime.resources.cpu.percentage, 100)}%` }}
                          />
                        </div>
                        <p className="text-xs text-muted-foreground mt-1">
                          {appDetails.runtime.resources.cpu.percentage.toFixed(1)}% used
                        </p>
                      </div>
                    )}

                    {appDetails.runtime.resources.memory && (
                      <div>
                        <div className="flex justify-between text-sm mb-1">
                          <span>Memory</span>
                          <span className="font-mono text-xs">
                            {appDetails.runtime.resources.memory.used} / {appDetails.runtime.resources.memory.limit}
                          </span>
                        </div>
                        <div className="w-full bg-muted rounded-full h-2">
                          <div
                            className="bg-primary rounded-full h-2 transition-all"
                            style={{ width: `${Math.min(appDetails.runtime.resources.memory.percentage, 100)}%` }}
                          />
                        </div>
                        <p className="text-xs text-muted-foreground mt-1">
                          {appDetails.runtime.resources.memory.percentage.toFixed(1)}% used
                        </p>
                      </div>
                    )}

                    {appDetails.runtime.resources.storage && (
                      <div>
                        <div className="flex justify-between text-sm mb-1">
                          <span>Storage</span>
                          <span className="font-mono text-xs">
                            {appDetails.runtime.resources.storage.used} / {appDetails.runtime.resources.storage.limit}
                          </span>
                        </div>
                        <div className="w-full bg-muted rounded-full h-2">
                          <div
                            className="bg-primary rounded-full h-2 transition-all"
                            style={{ width: `${Math.min(appDetails.runtime.resources.storage.percentage, 100)}%` }}
                          />
                        </div>
                        <p className="text-xs text-muted-foreground mt-1">
                          {appDetails.runtime.resources.storage.percentage.toFixed(1)}% used
                        </p>
                      </div>
                    )}
                  </CardContent>
                </Card>
              )}

              {eventsData?.events && eventsData.events.length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">Recent Events</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {eventsData.events.map((event, idx) => (
                        <div key={idx} className="flex items-start gap-2 text-sm border-b pb-2">
                          {event.type === 'Warning' ? (
                            <AlertCircle className="h-4 w-4 text-yellow-500 mt-0.5" />
                          ) : (
                            <CheckCircle className="h-4 w-4 text-green-500 mt-0.5" />
                          )}
                          <div className="flex-1 min-w-0">
                            <p className="font-medium">{event.reason}</p>
                            <p className="text-muted-foreground text-xs">{event.message}</p>
                            <p className="text-muted-foreground text-xs mt-1">
                              {event.timestamp} {event.count > 1 && `(${event.count}x)`}
                            </p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              )}
            </>
          ) : (
            <p className="text-center text-muted-foreground py-8">No status information available</p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
