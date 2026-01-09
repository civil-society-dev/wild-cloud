import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../../components/ui/card';
import { Button } from '../../components/ui/button';
import { Badge } from '../../components/ui/badge';
import { Skeleton } from '../../components/ui/skeleton';
import { Activity, AlertCircle, Filter } from 'lucide-react';
import { useOperations } from '../../services/api';
import { OperationCard } from '../../components/operations/OperationCard';

type FilterType = 'all' | 'running' | 'completed' | 'failed';

export function OperationsPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const [filter, setFilter] = useState<FilterType>('all');

  const filterForApi = filter === 'all' ? undefined : filter;
  const { data, isLoading, error } = useOperations(instanceId || '', filterForApi);

  const getFilterCount = (type: FilterType) => {
    if (!data) return 0;
    if (type === 'all') return data.operations.length;

    if (type === 'running') {
      return data.operations.filter(op =>
        op.status === 'running' || op.status === 'pending'
      ).length;
    }

    return data.operations.filter(op => op.status === type).length;
  };

  const runningCount = getFilterCount('running');
  const completedCount = getFilterCount('completed');
  const failedCount = getFilterCount('failed');

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
        <h2 className="text-3xl font-bold tracking-tight">Operations</h2>
        <p className="text-muted-foreground">
          Monitor and manage operations for {instanceId}
        </p>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Running</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{runningCount}</div>
            <p className="text-xs text-muted-foreground mt-1">
              Active operations
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Completed</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">{completedCount}</div>
            <p className="text-xs text-muted-foreground mt-1">
              Successfully finished
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Failed</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">{failedCount}</div>
            <p className="text-xs text-muted-foreground mt-1">
              Encountered errors
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Activity className="h-5 w-5" />
                Operations
              </CardTitle>
              <CardDescription>
                Real-time operation monitoring with auto-refresh
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-muted-foreground" />
              <div className="flex gap-1">
                <Button
                  size="sm"
                  variant={filter === 'all' ? 'default' : 'outline'}
                  onClick={() => setFilter('all')}
                >
                  All
                  <Badge variant="secondary" className="ml-2">
                    {data?.operations.length || 0}
                  </Badge>
                </Button>
                <Button
                  size="sm"
                  variant={filter === 'running' ? 'default' : 'outline'}
                  onClick={() => setFilter('running')}
                >
                  Running
                  <Badge variant="secondary" className="ml-2">
                    {runningCount}
                  </Badge>
                </Button>
                <Button
                  size="sm"
                  variant={filter === 'completed' ? 'default' : 'outline'}
                  onClick={() => setFilter('completed')}
                >
                  Completed
                  <Badge variant="secondary" className="ml-2">
                    {completedCount}
                  </Badge>
                </Button>
                <Button
                  size="sm"
                  variant={filter === 'failed' ? 'default' : 'outline'}
                  onClick={() => setFilter('failed')}
                >
                  Failed
                  <Badge variant="secondary" className="ml-2">
                    {failedCount}
                  </Badge>
                </Button>
              </div>
            </div>
          </div>
        </CardHeader>

        <CardContent>
          {error ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <AlertCircle className="h-12 w-12 text-red-500 mb-3" />
              <p className="text-sm font-medium text-red-900 dark:text-red-100">
                Error loading operations
              </p>
              <p className="text-xs text-red-700 dark:text-red-300 mt-1">
                {error.message}
              </p>
            </div>
          ) : isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-32 w-full" />
              <Skeleton className="h-32 w-full" />
              <Skeleton className="h-32 w-full" />
            </div>
          ) : data && data.operations.length > 0 ? (
            <div className="space-y-3">
              {data.operations.map((operation) => (
                <OperationCard
                  key={operation.id}
                  operation={operation}
                  expandable={true}
                />
              ))}
            </div>
          ) : (
            <div className="text-center py-12 text-muted-foreground">
              <Activity className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p className="text-sm font-medium">No operations found</p>
              <p className="text-xs mt-1">
                {filter === 'all'
                  ? 'Operations will appear here as they are created'
                  : `No ${filter} operations at this time`}
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Auto-refresh indicator */}
      <div className="text-center">
        <p className="text-xs text-muted-foreground">
          Auto-refreshing every 3 seconds
        </p>
      </div>
    </div>
  );
}
