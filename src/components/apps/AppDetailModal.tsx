import { useState, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
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
import { useAppEnhanced, useAppLogs, useAppEvents, useAppReadme } from '@/hooks/useApps';
import { apiClient } from '@/services/api/client';
import {
  RefreshCw,
  Eye,
  Settings,
  Activity,
  FileText,
  ExternalLink,
  AlertCircle,
  CheckCircle,
} from 'lucide-react';

interface AppDetailModalProps {
  instanceName: string;
  appName: string;
  open: boolean;
  onClose: () => void;
}

type ViewMode = 'overview' | 'configuration' | 'status' | 'logs';

export function AppDetailModal({
  instanceName,
  appName,
  open,
  onClose,
}: AppDetailModalProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('overview');
  const [showSecrets, setShowSecrets] = useState(false);
  const [secrets, setSecrets] = useState<Record<string, string>>({});
  const [loadingSecrets, setLoadingSecrets] = useState(false);
  const [logParams, setLogParams] = useState({ tail: 100, sinceSeconds: 3600 });

  const { data: appDetails, isLoading, refetch } = useAppEnhanced(instanceName, appName);
  const { data: logs, refetch: refetchLogs } = useAppLogs(
    instanceName,
    appName,
    viewMode === 'logs' ? logParams : undefined
  );
  const { data: eventsData } = useAppEvents(instanceName, appName, 20);
  const { data: readmeContent, isLoading: readmeLoading } = useAppReadme(instanceName, appName);

  // Fetch secrets when show is toggled
  useEffect(() => {
    if (showSecrets && Object.keys(secrets).length === 0) {
      setLoadingSecrets(true);
      apiClient.get(`/api/v1/instances/${instanceName}/secrets?raw=true`)
        .then((data: Record<string, any>) => {
          // Extract only app-specific secrets
          if (data.apps && data.apps[appName]) {
            setSecrets(data.apps[appName]);
          }
        })
        .catch((error) => {
          console.error('Failed to fetch secrets:', error);
        })
        .finally(() => {
          setLoadingSecrets(false);
        });
    }
  }, [showSecrets, instanceName, appName, secrets]);

  const getPodStatusColor = (status: string | undefined) => {
    if (!status) return 'text-muted-foreground';
    const lowerStatus = status.toLowerCase();
    if (lowerStatus.includes('running')) return 'text-green-600 dark:text-green-400';
    if (lowerStatus.includes('pending')) return 'text-yellow-600 dark:text-yellow-400';
    if (lowerStatus.includes('failed')) return 'text-red-600 dark:text-red-400';
    return 'text-muted-foreground';
  };

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
            {appDetails?.description || 'Application details and configuration'}
          </DialogDescription>
        </DialogHeader>

        {/* View Mode Selector */}
        <div className="flex gap-2 border-b pb-4">
          <Button
            variant={viewMode === 'overview' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('overview')}
          >
            <Eye className="h-4 w-4 mr-2" />
            Overview
          </Button>
          <Button
            variant={viewMode === 'configuration' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('configuration')}
          >
            <Settings className="h-4 w-4 mr-2" />
            Configuration
          </Button>
          <Button
            variant={viewMode === 'status' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('status')}
          >
            <Activity className="h-4 w-4 mr-2" />
            Status
          </Button>
          <Button
            variant={viewMode === 'logs' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('logs')}
          >
            <FileText className="h-4 w-4 mr-2" />
            Logs
          </Button>
        </div>

        {/* Overview Tab */}
        {viewMode === 'overview' && (
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
            ) : appDetails ? (
              <>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">Application Information</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <p className="text-sm font-medium text-muted-foreground">Name</p>
                        <p className="text-sm">{appDetails.name}</p>
                      </div>
                      <div>
                        <p className="text-sm font-medium text-muted-foreground">Version</p>
                        <p className="text-sm">{appDetails.version || 'N/A'}</p>
                      </div>
                      <div>
                        <p className="text-sm font-medium text-muted-foreground">Namespace</p>
                        <p className="text-sm">{appDetails.namespace}</p>
                      </div>
                      <div>
                        <p className="text-sm font-medium text-muted-foreground">Status</p>
                        <p className="text-sm">{appDetails.status}</p>
                      </div>
                    </div>

                    {appDetails.url && (
                      <div>
                        <p className="text-sm font-medium text-muted-foreground mb-1">URL</p>
                        <a
                          href={appDetails.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-1 text-sm text-primary hover:underline"
                        >
                          <ExternalLink className="h-3 w-3" />
                          {appDetails.url}
                        </a>
                      </div>
                    )}

                    {appDetails.description && (
                      <div>
                        <p className="text-sm font-medium text-muted-foreground mb-1">Description</p>
                        <p className="text-sm">{appDetails.description}</p>
                      </div>
                    )}

                    {appDetails.manifest?.dependencies && appDetails.manifest.dependencies.length > 0 && (
                      <div>
                        <p className="text-sm font-medium text-muted-foreground mb-2">Dependencies</p>
                        <div className="flex flex-wrap gap-2">
                          {appDetails.manifest.dependencies.map((dep) => (
                            <Badge key={dep} variant="outline">
                              {dep}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>

                {/* README Documentation */}
                {readmeContent && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-lg flex items-center gap-2">
                        <FileText className="h-5 w-5" />
                        README
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      {readmeLoading ? (
                        <Skeleton className="h-40 w-full" />
                      ) : (
                        <div className="prose prose-sm max-w-none dark:prose-invert overflow-auto max-h-96 p-4 bg-muted/30 rounded-lg">
                          <ReactMarkdown
                            components={{
                              // Style code blocks
                              code: ({inline, children, ...props}) => {
                                return inline ? (
                                  <code className="bg-muted px-1 py-0.5 rounded text-sm" {...props}>
                                    {children}
                                  </code>
                                ) : (
                                  <code className="block bg-muted p-3 rounded text-sm overflow-x-auto" {...props}>
                                    {children}
                                  </code>
                                );
                              },
                              // Make links open in new tab
                              a: ({children, href, ...props}) => (
                                <a href={href} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline" {...props}>
                                  {children}
                                </a>
                              ),
                            }}
                          >
                            {readmeContent}
                          </ReactMarkdown>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                )}
              </>
            ) : (
              <p className="text-center text-muted-foreground py-8">No information available</p>
            )}
          </div>
        )}

        {/* Configuration Tab */}
        {viewMode === 'configuration' && (
          <div className="space-y-4">
            <div className="flex justify-end">
              <Button variant="outline" size="sm" onClick={() => refetch()}>
                <RefreshCw className="h-4 w-4 mr-2" />
                Refresh
              </Button>
            </div>

            {isLoading ? (
              <div className="space-y-4">
                <Skeleton className="h-48 w-full" />
                <Skeleton className="h-32 w-full" />
              </div>
            ) : appDetails ? (
              <>
                {/* Configuration Values */}
                {((appDetails.config && Object.keys(appDetails.config).length > 0) ||
                  (appDetails.manifest?.defaultConfig && Object.keys(appDetails.manifest.defaultConfig).length > 0)) && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-lg">Configuration</CardTitle>
                      <CardDescription>Current configuration values</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        {Object.entries(appDetails.config || appDetails.manifest?.defaultConfig || {}).map(([key, value]) => {
                          // Check if value is a nested object
                          if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
                            return (
                              <div key={key} className="space-y-2">
                                <div className="font-medium text-muted-foreground">{key}:</div>
                                <div className="ml-4 space-y-1 border-l-2 border-muted pl-4">
                                  {Object.entries(value).map(([nestedKey, nestedValue]) => (
                                    <div key={`${key}.${nestedKey}`} className="flex justify-between text-sm">
                                      <span className="text-muted-foreground">{nestedKey}:</span>
                                      <span className="font-mono text-xs">
                                        {typeof nestedValue === 'object' && nestedValue !== null
                                          ? JSON.stringify(nestedValue)
                                          : String(nestedValue)}
                                      </span>
                                    </div>
                                  ))}
                                </div>
                              </div>
                            );
                          }

                          // Render simple values
                          return (
                            <div key={key} className="flex justify-between text-sm border-b pb-2">
                              <span className="font-medium text-muted-foreground">{key}:</span>
                              <span className="font-mono text-xs break-all">
                                {Array.isArray(value)
                                  ? JSON.stringify(value)
                                  : String(value)}
                              </span>
                            </div>
                          );
                        })}
                      </div>
                    </CardContent>
                  </Card>
                )}

                {/* Secrets */}
                {appDetails.manifest?.defaultSecrets && appDetails.manifest.defaultSecrets.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-lg flex items-center justify-between">
                        <span>Secrets</span>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setShowSecrets(!showSecrets)}
                          disabled={loadingSecrets}
                        >
                          {loadingSecrets ? 'Loading...' : showSecrets ? 'Hide' : 'Show'}
                        </Button>
                      </CardTitle>
                      <CardDescription>Sensitive configuration values</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        {appDetails.manifest.defaultSecrets.map((secret) => {
                          const secretKey = typeof secret === 'object' && secret.key ? secret.key : secret;
                          const secretValue = secrets[secretKey];
                          return (
                            <div key={secretKey} className="flex justify-between text-sm border-b pb-2">
                              <span className="font-medium text-muted-foreground">{secretKey}:</span>
                              <span className="font-mono text-xs break-all max-w-md text-right">
                                {showSecrets && secretValue ? secretValue : '••••••••'}
                              </span>
                            </div>
                          );
                        })}
                      </div>
                    </CardContent>
                  </Card>
                )}
              </>
            ) : (
              <p className="text-center text-muted-foreground py-8">No configuration available</p>
            )}
          </div>
        )}

        {/* Status Tab */}
        {viewMode === 'status' && (
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
                {/* Replicas */}
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

                {/* Pods */}
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

                {/* Resource Usage */}
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

                {/* Recent Events */}
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
        )}

        {/* Logs Tab */}
        {viewMode === 'logs' && (
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
                      // Handle both string format and object format {timestamp, message, pod}
                      if (typeof line === 'string') {
                        return (
                          <div key={idx} className="whitespace-pre-wrap break-all">
                            {line}
                          </div>
                        );
                      } else if (line && typeof line === 'object' && 'message' in line) {
                        // Display timestamp and message nicely
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
                    // Handle case where logs might be an object with different structure
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
        )}
      </DialogContent>
    </Dialog>
  );
}
