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
import { ServiceStatusBadge } from './ServiceStatusBadge';
import { useServiceStatus } from '@/hooks/useServices';
import { RefreshCw } from 'lucide-react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';

interface ServiceStatusDialogProps {
  instanceName: string;
  serviceName: string;
  open: boolean;
  onClose: () => void;
}

export function ServiceStatusDialog({
  instanceName,
  serviceName,
  open,
  onClose,
}: ServiceStatusDialogProps) {
  const { data: status, isLoading, refetch } = useServiceStatus(instanceName, serviceName);

  const getPodStatusColor = (status: string) => {
    if (status.toLowerCase().includes('running')) return 'text-green-600 dark:text-green-400';
    if (status.toLowerCase().includes('pending')) return 'text-yellow-600 dark:text-yellow-400';
    if (status.toLowerCase().includes('failed')) return 'text-red-600 dark:text-red-400';
    return 'text-muted-foreground';
  };

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-5xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            {serviceName}
            {status && <ServiceStatusBadge status={status.deploymentStatus} />}
          </DialogTitle>
          <DialogDescription>Service status and details</DialogDescription>
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
          ) : status ? (
            <>
              {/* Status Section */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">Status Information</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">Service Name</p>
                      <p className="text-sm">{status.name}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-muted-foreground">Namespace</p>
                      <p className="text-sm">{status.namespace}</p>
                    </div>
                  </div>

                  {status.replicas && (
                    <div>
                      <p className="text-sm font-medium text-muted-foreground mb-2">Replicas</p>
                      <div className="grid grid-cols-4 gap-2 text-sm">
                        <div className="bg-muted rounded p-2">
                          <p className="text-xs text-muted-foreground">Desired</p>
                          <p className="font-semibold">{status.replicas.desired}</p>
                        </div>
                        <div className="bg-muted rounded p-2">
                          <p className="text-xs text-muted-foreground">Current</p>
                          <p className="font-semibold">{status.replicas.current}</p>
                        </div>
                        <div className="bg-muted rounded p-2">
                          <p className="text-xs text-muted-foreground">Ready</p>
                          <p className="font-semibold">{status.replicas.ready}</p>
                        </div>
                        <div className="bg-muted rounded p-2">
                          <p className="text-xs text-muted-foreground">Available</p>
                          <p className="font-semibold">{status.replicas.available}</p>
                        </div>
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>

              {/* Pods Section */}
              {status.pods && status.pods.length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">Pods</CardTitle>
                    <CardDescription>{status.pods.length} pod(s)</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {status.pods.map((pod) => (
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

              {/* Configuration Preview */}
              {status.config && Object.keys(status.config).length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">Current Configuration</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {Object.entries(status.config).map(([key, value]) => (
                        <div key={key} className="flex justify-between text-sm">
                          <span className="font-medium text-muted-foreground">{key}:</span>
                          <span className="font-mono text-xs">
                            {typeof value === 'object' && value !== null
                              ? JSON.stringify(value, null, 2)
                              : String(value)}
                          </span>
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
