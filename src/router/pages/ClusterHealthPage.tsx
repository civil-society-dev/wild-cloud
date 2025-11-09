import { useParams } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../../components/ui/card';
import { Badge } from '../../components/ui/badge';
import { Skeleton } from '../../components/ui/skeleton';
import { HeartPulse, AlertCircle, Clock } from 'lucide-react';
import { useClusterHealth, useClusterStatus, useClusterNodes } from '../../services/api';
import { HealthIndicator } from '../../components/operations/HealthIndicator';
import { NodeStatusCard } from '../../components/operations/NodeStatusCard';

export function ClusterHealthPage() {
  const { instanceId } = useParams<{ instanceId: string }>();

  const { data: health, isLoading: healthLoading, error: healthError } = useClusterHealth(instanceId || '');
  const { data: status, isLoading: statusLoading } = useClusterStatus(instanceId || '');
  const { data: nodes, isLoading: nodesLoading } = useClusterNodes(instanceId || '');

  if (!instanceId) {
    return (
      <div className="flex items-center justify-center h-96">
        <Card className="p-6">
          <div className="flex items-center gap-3 text-muted-foreground">
            <AlertCircle className="h-5 w-5" />
            <p>No instance selected</p>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-3xl font-bold tracking-tight">Cluster Health</h2>
        <p className="text-muted-foreground">
          Monitor health metrics and node status for {instanceId}
        </p>
      </div>

      {/* Overall Health Status */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <HeartPulse className="h-5 w-5" />
                Overall Health
              </CardTitle>
              <CardDescription>
                Cluster health aggregated from all checks
              </CardDescription>
            </div>
            {health && (
              <HealthIndicator status={health.status} size="lg" />
            )}
          </div>
        </CardHeader>
        <CardContent>
          {healthError ? (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <AlertCircle className="h-12 w-12 text-red-500 mb-3" />
              <p className="text-sm font-medium text-red-900 dark:text-red-100">
                Error loading health data
              </p>
              <p className="text-xs text-red-700 dark:text-red-300 mt-1">
                {healthError.message}
              </p>
            </div>
          ) : healthLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-16 w-full" />
              <Skeleton className="h-16 w-full" />
              <Skeleton className="h-16 w-full" />
            </div>
          ) : health && health.checks.length > 0 ? (
            <div className="space-y-2">
              {health.checks.map((check, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between p-4 rounded-lg border hover:bg-accent/50 transition-colors"
                >
                  <div className="flex items-center gap-3 flex-1">
                    <HealthIndicator status={check.status} size="sm" />
                    <div className="flex-1">
                      <p className="font-medium text-sm">{check.name}</p>
                      {check.message && (
                        <p className="text-xs text-muted-foreground mt-0.5">
                          {check.message}
                        </p>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <AlertCircle className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-sm font-medium">No health data available</p>
              <p className="text-xs mt-1">
                Health checks will appear here once the cluster is running
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Cluster Information */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Cluster Status</CardTitle>
          </CardHeader>
          <CardContent>
            {statusLoading ? (
              <Skeleton className="h-8 w-24" />
            ) : status ? (
              <div>
                <Badge variant={status.status === 'ready' ? 'outline' : 'secondary'} className={status.status === 'ready' ? 'border-green-500' : ''}>
                  {status.status === 'ready' ? 'Ready' : 'Not Ready'}
                </Badge>
                <p className="text-xs text-muted-foreground mt-2">
                  {status.nodes} nodes total
                </p>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">Unknown</div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Kubernetes Version</CardTitle>
          </CardHeader>
          <CardContent>
            {statusLoading ? (
              <Skeleton className="h-8 w-32" />
            ) : status?.kubernetesVersion ? (
              <div>
                <div className="text-lg font-bold font-mono">
                  {status.kubernetesVersion}
                </div>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">Not available</div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Talos Version</CardTitle>
          </CardHeader>
          <CardContent>
            {statusLoading ? (
              <Skeleton className="h-8 w-32" />
            ) : status?.talosVersion ? (
              <div>
                <div className="text-lg font-bold font-mono">
                  {status.talosVersion}
                </div>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">Not available</div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Node Status */}
      <Card>
        <CardHeader>
          <CardTitle>Node Status</CardTitle>
          <CardDescription>
            Detailed status and information for each node
          </CardDescription>
        </CardHeader>
        <CardContent>
          {nodesLoading ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              <Skeleton className="h-48 w-full" />
              <Skeleton className="h-48 w-full" />
              <Skeleton className="h-48 w-full" />
            </div>
          ) : nodes && nodes.nodes.length > 0 ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {nodes.nodes.map((node) => (
                <NodeStatusCard key={node.hostname} node={node} showHardware={true} />
              ))}
            </div>
          ) : (
            <div className="text-center py-12 text-muted-foreground">
              <AlertCircle className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-sm font-medium">No nodes found</p>
              <p className="text-xs mt-1">
                Add nodes to your cluster to see them here
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Auto-refresh indicator */}
      <div className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
        <Clock className="h-3 w-3" />
        <p>Auto-refreshing every 10 seconds</p>
      </div>
    </div>
  );
}
