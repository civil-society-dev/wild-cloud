import { useParams, Link } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardAction } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Skeleton } from '../../components/ui/skeleton';
import { Activity, Server, AlertCircle, RefreshCw, FileText, TrendingUp } from 'lucide-react';
import { useInstance, useInstanceOperations, useInstanceClusterHealth, useClusterStatus } from '../../services/api';
import { OperationCard } from '../../components/operations/OperationCard';
import { HealthIndicator } from '../../components/operations/HealthIndicator';
import { SetupWizard } from '../../components/setup/SetupWizard';

export function DashboardPage() {
  const { instanceId } = useParams<{ instanceId: string }>();

  const { data: instance, isLoading: instanceLoading, refetch: refetchInstance } = useInstance(instanceId || '');
  const { data: operations, isLoading: operationsLoading } = useInstanceOperations(instanceId || '', 5);
  const { data: health, isLoading: healthLoading } = useInstanceClusterHealth(instanceId || '');
  const { data: status, isLoading: statusLoading } = useClusterStatus(instanceId || '');

  const handleRefresh = () => {
    refetchInstance();
  };

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
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Dashboard</h2>
          <p className="text-muted-foreground">
            Overview and quick status for {instanceId}
          </p>
        </div>
        <Button onClick={handleRefresh} variant="outline" size="sm">
          <RefreshCw className="h-4 w-4" />
          Refresh
        </Button>
      </div>

      {/* Setup Wizard - shown if setup is not complete */}
      <SetupWizard />

      {/* Status Cards Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {/* Instance Status */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">Instance Status</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </div>
          </CardHeader>
          <CardContent>
            {instanceLoading ? (
              <Skeleton className="h-8 w-24" />
            ) : instance ? (
              <div>
                <div className="text-2xl font-bold">Active</div>
                <p className="text-xs text-muted-foreground mt-1">
                  Instance configured
                </p>
              </div>
            ) : (
              <div>
                <div className="text-2xl font-bold text-muted-foreground">Unknown</div>
                <p className="text-xs text-muted-foreground mt-1">
                  Unable to load status
                </p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Cluster Health */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">Cluster Health</CardTitle>
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
            </div>
          </CardHeader>
          <CardContent>
            {healthLoading ? (
              <Skeleton className="h-8 w-24" />
            ) : health ? (
              <div>
                <div className="mb-2">
                  <HealthIndicator status={health.status} size="md" />
                </div>
                <p className="text-xs text-muted-foreground">
                  {health.checks.length} health checks
                </p>
              </div>
            ) : (
              <div>
                <div className="text-2xl font-bold text-muted-foreground">Unknown</div>
                <p className="text-xs text-muted-foreground mt-1">
                  Health data unavailable
                </p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Node Count */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">Nodes</CardTitle>
              <Server className="h-4 w-4 text-muted-foreground" />
            </div>
          </CardHeader>
          <CardContent>
            {statusLoading ? (
              <Skeleton className="h-8 w-24" />
            ) : status ? (
              <div>
                <div className="text-2xl font-bold">{status.nodes}</div>
                <p className="text-xs text-muted-foreground mt-1">
                  {status.controlPlaneNodes} control plane, {status.workerNodes} workers
                </p>
              </div>
            ) : (
              <div>
                <div className="text-2xl font-bold text-muted-foreground">-</div>
                <p className="text-xs text-muted-foreground mt-1">
                  No nodes detected
                </p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* K8s Version */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">Kubernetes</CardTitle>
              <FileText className="h-4 w-4 text-muted-foreground" />
            </div>
          </CardHeader>
          <CardContent>
            {statusLoading ? (
              <Skeleton className="h-8 w-24" />
            ) : status?.kubernetesVersion ? (
              <div>
                <div className="text-xl font-bold font-mono">{status.kubernetesVersion}</div>
                <p className="text-xs text-muted-foreground mt-1">
                  {status.status === 'ready' ? 'Ready' : 'Not ready'}
                </p>
              </div>
            ) : (
              <div>
                <div className="text-2xl font-bold text-muted-foreground">-</div>
                <p className="text-xs text-muted-foreground mt-1">
                  Version unknown
                </p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Cluster Health Details */}
      {health && health.checks.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Health Checks</CardTitle>
            <CardDescription>
              Detailed health status of cluster components
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {health.checks.map((check, index) => (
                <div key={index} className="flex items-center justify-between p-3 rounded-lg border">
                  <div className="flex items-center gap-3">
                    <HealthIndicator status={check.status} size="sm" />
                    <span className="font-medium text-sm">{check.name}</span>
                  </div>
                  {check.message && (
                    <span className="text-xs text-muted-foreground">{check.message}</span>
                  )}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Recent Operations */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Recent Operations</CardTitle>
              <CardDescription>
                Last 5 operations for this instance
              </CardDescription>
            </div>
            <CardAction>
              <Button asChild variant="outline" size="sm">
                <Link to={`/instances/${instanceId}/operations`}>
                  View All
                </Link>
              </Button>
            </CardAction>
          </div>
        </CardHeader>
        <CardContent>
          {operationsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-24 w-full" />
            </div>
          ) : operations && operations.operations.length > 0 ? (
            <div className="space-y-3">
              {operations.operations.map((operation) => (
                <OperationCard key={operation.id} operation={operation} />
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <Activity className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-sm">No operations found</p>
              <p className="text-xs mt-1">Operations will appear here as they are created</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>

  );
}
