import { useState, useEffect, useMemo } from 'react';
import ReactMarkdown from 'react-markdown';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Loader2,
  Settings,
  Trash2,
  Activity,
  FileText,
  Info,
  Upload,
  ExternalLink,
  AlertCircle,
  CheckCircle,
  RefreshCw,
  Archive,
  RotateCcw,
  Database,
} from 'lucide-react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useAppEnhanced, useAppReadme, useAppEvents, useAppLogs } from '@/hooks/useApps';
import { apiClient } from '@/services/api/client';

interface AppDetailPanelProps {
  instanceName: string;
  appName: string;
  open: boolean;
  onClose: () => void;
  onDeploy: (appName: string) => void;
  onUpdate: (appName: string) => void;
  onDelete: (appName: string) => void;
  onBackup: (appName: string) => void;
  onRestore: (appName: string) => void;
  onConfigure: (appName: string) => void;
  isDeploying: boolean;
  isUpdating: boolean;
  isDeleting: boolean;
  updateAvailable: boolean;
  availableVersion?: string;
}

export function AppDetailPanel({
  instanceName,
  appName,
  open,
  onClose,
  onDeploy,
  onUpdate,
  onDelete,
  onBackup,
  onRestore,
  onConfigure,
  isDeploying,
  isUpdating,
  isDeleting,
  updateAvailable,
  availableVersion,
}: AppDetailPanelProps) {
  const [showSecrets, setShowSecrets] = useState(false);
  const [secrets, setSecrets] = useState<Record<string, string>>({});
  const [loadingSecrets, setLoadingSecrets] = useState(false);
  const [logParams, setLogParams] = useState<{
    tail: number;
    sinceSeconds?: number;
    pod?: string;
    container?: string;
  }>({ tail: 100 });
  const [backupResources, setBackupResources] = useState<any[]>([]);
  const [loadingBackupResources, setLoadingBackupResources] = useState(false);
  const [hasLoadedBackupResources, setHasLoadedBackupResources] = useState(false);
  const [activeTab, setActiveTab] = useState('overview');

  // Disable polling when on the Data tab to prevent network loops
  const enablePolling = activeTab !== 'data';

  // Fetch app data
  const { data: appDetails, isLoading } = useAppEnhanced(instanceName, appName, { enablePolling });
  const { data: readmeContent, isLoading: readmeLoading } = useAppReadme(instanceName, appName);
  const { data: eventsData } = useAppEvents(instanceName, appName, 20, { enablePolling });
  const { data: logs, refetch: refetchLogs } = useAppLogs(
    instanceName,
    appName,
    open ? logParams : undefined
  );

  // Build a list of selectable "components" - each is a pod/container combination
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

  // Fetch secrets when needed
  useEffect(() => {
    if (showSecrets && Object.keys(secrets).length === 0) {
      setLoadingSecrets(true);
      apiClient.get(`/api/v1/instances/${instanceName}/secrets?raw=true`)
        .then((d) => {
          const data = d as { apps?: Record<string, Record<string, string>> };
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

  // Reset backup resources when app changes
  useEffect(() => {
    setBackupResources([]);
    setHasLoadedBackupResources(false);
  }, [appName]);

  // Fetch backup resources when Data tab is selected
  useEffect(() => {
    if (open && activeTab === 'data' && !hasLoadedBackupResources && !loadingBackupResources) {
      setLoadingBackupResources(true);
      apiClient.get(`/api/v1/instances/${instanceName}/apps/${appName}/backup/discover`)
        .then((response) => {
          const data = response as { data?: { resources?: any[] } };
          // Handle both empty array and actual resources
          const resources = data.data?.resources || [];
          setBackupResources(resources);
          setHasLoadedBackupResources(true);
        })
        .catch((error) => {
          console.error('Failed to fetch backup resources:', error);
          setBackupResources([]);
          setHasLoadedBackupResources(true);
        })
        .finally(() => {
          setLoadingBackupResources(false);
        });
    }
  }, [open, activeTab, instanceName, appName, hasLoadedBackupResources, loadingBackupResources]);

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

  // Show loading state while app data is being fetched
  if (isLoading || !appDetails) {
    return (
      <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
        <DialogContent className="sm:max-w-4xl max-w-[95vw] max-h-[90vh] overflow-hidden flex flex-col">
          <DialogHeader>
            <DialogTitle>{appName}</DialogTitle>
          </DialogHeader>
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-4xl max-w-[95vw] max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <span>{appName}</span>
              {appDetails.version && (
                <Badge variant="outline">{appDetails.version}</Badge>
              )}
              {getStatusBadge(appDetails.status)}
            </div>
          </DialogTitle>
          {appDetails.description && (
            <p className="text-sm text-muted-foreground">{appDetails.description}</p>
          )}
        </DialogHeader>

        <Tabs defaultValue="overview" value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col overflow-hidden">
          <TabsList className="grid w-full grid-cols-5">
            <TabsTrigger value="overview">
              <Info className="h-4 w-4 mr-2" />
              Overview
            </TabsTrigger>
            <TabsTrigger value="config">
              <Settings className="h-4 w-4 mr-2" />
              Config
            </TabsTrigger>
            <TabsTrigger value="status">
              <Activity className="h-4 w-4 mr-2" />
              Status
            </TabsTrigger>
            <TabsTrigger value="data">
              <Database className="h-4 w-4 mr-2" />
              Data
            </TabsTrigger>
            <TabsTrigger value="logs">
              <FileText className="h-4 w-4 mr-2" />
              Logs
            </TabsTrigger>
          </TabsList>

          {/* Overview Tab */}
          <TabsContent value="overview" className="flex-1 overflow-y-auto space-y-6">
            {/* Action Buttons */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold">Actions</h3>
              <div className="flex flex-wrap gap-2">
                {/* Added: configure and deploy */}
                {appDetails.status === 'added' && (
                  <>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => onConfigure(appName)}
                    >
                      <Settings className="h-4 w-4 mr-2" />
                      Configure
                    </Button>
                    <Button
                      size="sm"
                      onClick={() => onDeploy(appName)}
                      disabled={isDeploying}
                    >
                      {isDeploying ? (
                        <Loader2 className="h-4 w-4 animate-spin mr-2" />
                      ) : (
                        <Upload className="h-4 w-4 mr-2" />
                      )}
                      Deploy
                    </Button>
                  </>
                )}

                {/* Update available */}
                {updateAvailable && (appDetails.status === 'deployed' || appDetails.status === 'running') && (
                  <Button
                    size="sm"
                    variant="outline"
                    className="border-amber-500 text-amber-700 hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-950/20"
                    onClick={() => onUpdate(appName)}
                    disabled={isUpdating}
                  >
                    {isUpdating ? (
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    ) : (
                      <RefreshCw className="h-4 w-4 mr-2" />
                    )}
                    Update to {availableVersion}
                  </Button>
                )}

                {/* Deployed: backup and restore */}
                {(appDetails.status === 'deployed' || appDetails.status === 'running') && (
                  <>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => onBackup(appName)}
                    >
                      <Archive className="h-4 w-4 mr-2" />
                      Backup
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => onRestore(appName)}
                    >
                      <RotateCcw className="h-4 w-4 mr-2" />
                      Restore
                    </Button>
                  </>
                )}

                {/* Delete button */}
                <Button
                  size="sm"
                  variant="destructive"
                  onClick={() => {
                    if (confirm(`Are you sure you want to delete ${appName}?`)) {
                      onDelete(appName);
                    }
                  }}
                  disabled={isDeleting}
                >
                  {isDeleting ? (
                    <Loader2 className="h-4 w-4 animate-spin mr-2" />
                  ) : (
                    <Trash2 className="h-4 w-4 mr-2" />
                  )}
                  Delete
                </Button>
              </div>
            </div>

            {/* App Information */}
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
                    <p className="text-sm">
                      {appDetails.version || 'N/A'}
                      {updateAvailable && availableVersion && (
                        <span className="text-amber-600 dark:text-amber-400 ml-2">({availableVersion} available)</span>
                      )}
                    </p>
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

            {/* README */}
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
                    <div className="h-40 flex items-center justify-center">
                      <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                    </div>
                  ) : (
                    <div className="prose prose-sm max-w-none dark:prose-invert overflow-auto max-h-96 p-4 bg-muted/30 rounded-lg">
                      <ReactMarkdown
                        components={{
                          code: ({ children, ...props }) => (
                            <code className="bg-muted px-1 py-0.5 rounded text-sm" {...props}>
                              {children}
                            </code>
                          ),
                          a: ({ children, href, ...props }) => (
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
          </TabsContent>

          {/* Config Tab */}
          <TabsContent value="config" className="flex-1 overflow-y-auto space-y-4">
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
                    {appDetails.manifest.defaultSecrets?.map((secret) => {
                      const secretKey = typeof secret === 'string' ? secret : secret.key;
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

            {!appDetails.config && !appDetails.manifest?.defaultConfig && (
              <p className="text-center text-muted-foreground py-8">No configuration available</p>
            )}
          </TabsContent>

          {/* Status Tab */}
          <TabsContent value="status" className="flex-1 overflow-y-auto space-y-4">
            {appDetails?.runtime ? (
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
                              <Badge variant="outline" className={getPodStatusColor(pod.status)}>
                                {pod.status}
                              </Badge>
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
          </TabsContent>

          {/* Data Tab */}
          <TabsContent value="data" className="flex-1 overflow-y-auto space-y-6">
            <div className="space-y-4">
              <h3 className="text-lg font-semibold">Backup Resources</h3>

              {loadingBackupResources ? (
                <div className="flex items-center justify-center py-12">
                  <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                </div>
              ) : backupResources.length === 0 ? (
                <Card>
                  <CardContent className="py-8 text-center">
                    <p className="text-muted-foreground">No backup resources discovered for this app.</p>
                  </CardContent>
                </Card>
              ) : (
                <div className="space-y-4">
                  {backupResources.map((resource, idx) => (
                    <Card key={idx}>
                      <CardHeader>
                        <div className="flex items-center justify-between">
                          <CardTitle className="text-base flex items-center gap-2">
                            {resource.type === 'database' ? (
                              <Database className="h-4 w-4" />
                            ) : resource.type === 'pvc' ? (
                              <Archive className="h-4 w-4" />
                            ) : (
                              <FileText className="h-4 w-4" />
                            )}
                            {resource.name}
                          </CardTitle>
                          <Badge variant={resource.shouldBackup ? 'success' : 'secondary'}>
                            {resource.shouldBackup ? 'Backed Up' : 'Excluded'}
                          </Badge>
                        </div>
                        <CardDescription>
                          Type: {resource.type} | Plugin: {resource.plugin}
                          {resource.reason && ` | ${resource.reason}`}
                        </CardDescription>
                      </CardHeader>
                      <CardContent>
                        <div className="grid grid-cols-2 gap-4 text-sm">
                          {resource.source && Object.entries(resource.source).map(([key, value]) => (
                            <div key={key} className="space-y-1">
                              <p className="text-muted-foreground capitalize">
                                {key.replace(/([A-Z])/g, ' $1').trim()}:
                              </p>
                              <p className="font-mono text-xs">{String(value)}</p>
                            </div>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}
            </div>
          </TabsContent>

          {/* Logs Tab */}
          <TabsContent value="logs" className="flex-1 flex flex-col overflow-hidden space-y-4">
            {/* Log controls */}
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
                <Button variant="outline" size="icon" onClick={() => refetchLogs()} title="Refresh">
                  <RefreshCw className="h-4 w-4" />
                </Button>
              </div>
            </div>

            {/* Log display */}
            <Card className="flex-1 overflow-hidden">
              <CardContent className="p-4 h-full">
                <div className="h-[400px] overflow-y-auto bg-black text-green-400 font-mono text-xs p-4 rounded-lg">
                  {!logs ? (
                    <p className="text-gray-500">Loading logs...</p>
                  ) : logs.logs && Array.isArray(logs.logs) && logs.logs.length > 0 ? (
                    logs.logs.map((logLine: { message: string }, idx: number) => (
                      <div key={idx} className="whitespace-pre-wrap break-all">
                        {logLine.message}
                      </div>
                    ))
                  ) : (
                    <div className="text-gray-500">
                      <p>No logs available for pod: {logs.pod || 'unknown'}</p>
                      <p className="text-xs mt-2">
                        Try adjusting the time range or check if the pod has recently started.
                      </p>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}
