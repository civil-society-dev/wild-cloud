import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Loader2,
  Download,
  RefreshCw,
  Upload,
  Settings,
  Trash2,
  Activity,
  FileText,
  Info,
  Copy
} from 'lucide-react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { Service } from '@/services/api/types';
import { ServiceLifecycleBadges } from './ServiceLifecycleBadges';
import { ServiceConfigEditor } from './ServiceConfigEditor';
import { useServiceStatus, useService } from '@/hooks/useServices';
import { servicesApi } from '@/services/api';

interface ServiceDetailDialogProps {
  instanceName: string;
  serviceName: string;
  open: boolean;
  onClose: () => void;
  onFetch: (serviceName: string) => void;
  onCompile: (serviceName: string) => void;
  onDeploy: (serviceName: string) => void;
  onDelete: (serviceName: string) => void;
  onCleanFiles: (serviceName: string) => void;
  isFetching: boolean;
  isCompiling: boolean;
  isDeploying: boolean;
  isDeleting: boolean;
  isCleaningFiles: boolean;
  isOperating: boolean;
}

export function ServiceDetailDialog({
  instanceName,
  serviceName,
  open,
  onClose,
  onFetch,
  onCompile,
  onDeploy,
  onDelete,
  onCleanFiles,
  isFetching,
  isCompiling,
  isDeploying,
  isDeleting,
  isCleaningFiles,
  isOperating,
}: ServiceDetailDialogProps) {
  const [showConfigEditor, setShowConfigEditor] = useState(false);
  const [logs, setLogs] = useState<string[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [follow, setFollow] = useState(false);
  const [tail, setTail] = useState(100);
  const [container, setContainer] = useState<string | undefined>(undefined);
  const [autoScroll, setAutoScroll] = useState(true);
  const logsEndRef = useRef<HTMLDivElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  // Fetch live service data
  const { data: service, isLoading: serviceLoading } = useService(instanceName, serviceName);
  const { data: statusData } = useServiceStatus(instanceName, serviceName);

  // Extract unique container names from all pods
  const containers = useMemo(() => {
    if (!statusData?.pods) return [];
    const containerSet = new Set<string>();
    statusData.pods.forEach((pod) => {
      pod.containers?.forEach((c: string) => containerSet.add(c));
    });
    return Array.from(containerSet);
  }, [statusData?.pods]);

  // Set default container when containers become available
  useEffect(() => {
    if (containers.length > 0 && !container) {
      setContainer(containers[0]);
    }
  }, [containers, container]);

  // Scroll to bottom when logs change and autoScroll is enabled
  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  // Fetch initial buffered logs
  const fetchLogs = useCallback(async () => {
    if (!open || !serviceName) return;
    setLogsLoading(true);
    try {
      const url = servicesApi.getLogsUrl(instanceName, serviceName, tail, false, container);
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`Failed to fetch logs: ${response.statusText}`);
      }
      const data = await response.json();
      if (data.lines && Array.isArray(data.lines)) {
        setLogs(data.lines);
      } else {
        setLogs([]);
      }
    } catch (error) {
      console.error('Error fetching logs:', error);
      setLogs([`Error: ${error instanceof Error ? error.message : 'Failed to fetch logs'}`]);
    } finally {
      setLogsLoading(false);
    }
  }, [instanceName, serviceName, tail, open, container]);

  // Set up SSE streaming when follow is enabled
  useEffect(() => {
    if (!open || !serviceName) return;

    if (follow) {
      const url = servicesApi.getLogsUrl(instanceName, serviceName, tail, true, container);
      const eventSource = new EventSource(url);
      eventSourceRef.current = eventSource;

      eventSource.onmessage = (event) => {
        const line = event.data;
        if (line && line.trim() !== '') {
          setLogs((prev) => [...prev, line]);
        }
      };

      eventSource.onerror = (error) => {
        console.error('SSE error:', error);
        eventSource.close();
        setFollow(false);
      };

      return () => {
        eventSource.close();
      };
    } else {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    }
  }, [follow, instanceName, serviceName, tail, open, container]);

  // Fetch initial logs on mount and when parameters change
  useEffect(() => {
    if (open && !follow) {
      fetchLogs();
    }
  }, [fetchLogs, follow, open]);

  // Clean up on close
  useEffect(() => {
    if (!open) {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      setFollow(false);
    }
  }, [open]);

  const handleCopyLogs = () => {
    const text = logs.join('\n');
    navigator.clipboard.writeText(text);
  };

  const handleDownloadLogs = () => {
    const text = logs.join('\n');
    const blob = new Blob([text], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${serviceName}-logs.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const handleClearLogs = () => {
    setLogs([]);
  };

  const handleRefresh = () => {
    setLogs([]);
    fetchLogs();
  };

  // Show loading state while service data is being fetched
  if (serviceLoading || !service) {
    return (
      <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
        <DialogContent className="sm:max-w-4xl max-w-[95vw] max-h-[90vh] overflow-hidden flex flex-col">
          <DialogHeader>
            <DialogTitle>{serviceName}</DialogTitle>
          </DialogHeader>
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  if (showConfigEditor) {
    return (
      <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
        <DialogContent className="sm:max-w-4xl max-w-[95vw] max-h-[90vh] overflow-y-auto w-full">
          <ServiceConfigEditor
            instanceName={instanceName}
            serviceName={serviceName}
            manifest={service}
            onClose={() => setShowConfigEditor(false)}
            onSuccess={() => setShowConfigEditor(false)}
          />
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
              <span>{serviceName}</span>
              {service.version && (
                <Badge variant="outline">{service.version}</Badge>
              )}
            </div>
          </DialogTitle>
          {service.description && (
            <p className="text-sm text-muted-foreground">{service.description}</p>
          )}
        </DialogHeader>

        <Tabs defaultValue="overview" className="flex-1 flex flex-col overflow-hidden">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="overview">
              <Info className="h-4 w-4 mr-2" />
              Overview
            </TabsTrigger>
            <TabsTrigger value="status">
              <Activity className="h-4 w-4 mr-2" />
              Status
            </TabsTrigger>
            <TabsTrigger value="logs">
              <FileText className="h-4 w-4 mr-2" />
              Logs
            </TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="flex-1 overflow-y-auto space-y-6">
            {/* Lifecycle Status */}
            {service.lifecycle && (
              <div className="space-y-3">
                <h3 className="text-sm font-semibold">Lifecycle Status</h3>
                <ServiceLifecycleBadges lifecycle={service.lifecycle} />
              </div>
            )}

            {/* Action Buttons */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold">Actions</h3>
              <div className="flex flex-wrap gap-2">
                {/* Fetch button - show when templates not fetched or update available */}
                {(!service.lifecycle?.templates?.state ||
                  service.lifecycle?.templates?.state === 'not_fetched' ||
                  service.lifecycle?.templates?.state === 'update_available') && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => onFetch(serviceName)}
                    disabled={isOperating || isFetching}
                  >
                    {isFetching ? (
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    ) : (
                      <Download className="h-4 w-4 mr-2" />
                    )}
                    {service.lifecycle?.templates?.state === 'update_available' ? 'Update Templates' : 'Fetch Templates'}
                  </Button>
                )}

                {/* Compile button - show when templates are up to date and configuration not configured or needs recompiling */}
                {service.lifecycle?.templates?.state === 'up_to_date' &&
                 (!service.lifecycle?.configuration?.state ||
                  service.lifecycle?.configuration?.state === 'not_configured' ||
                  service.lifecycle?.configuration?.state === 'needs_recompile') && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => onCompile(serviceName)}
                    disabled={isOperating || isCompiling}
                  >
                    {isCompiling ? (
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    ) : (
                      <RefreshCw className="h-4 w-4 mr-2" />
                    )}
                    Compile Manifests
                  </Button>
                )}

                {/* Deploy button - only show when configuration is compiled and not deployed or needs redeploy */}
                {service.lifecycle?.configuration?.state === 'compiled' &&
                 (service.lifecycle?.deployment?.state === 'not_deployed' ||
                  service.lifecycle?.deployment?.state === 'needs_redeploy') && (
                  <Button
                    size="sm"
                    onClick={() => onDeploy(serviceName)}
                    disabled={isOperating || isDeploying}
                  >
                    {isDeploying ? (
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    ) : (
                      <Upload className="h-4 w-4 mr-2" />
                    )}
                    {service.lifecycle?.deployment?.state === 'needs_redeploy' ? 'Redeploy' : 'Deploy'}
                  </Button>
                )}

                {/* Config button */}
                {service.hasConfig && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setShowConfigEditor(true)}
                  >
                    <Settings className="h-4 w-4 mr-2" />
                    Edit Configuration
                  </Button>
                )}

                {/* Delete button */}
                {/* Only show Delete button if templates are fetched (up_to_date or update_available) and service is deployed */}
                {(service.lifecycle?.templates?.state === 'up_to_date' ||
                  service.lifecycle?.templates?.state === 'update_available') &&
                 service.lifecycle?.deployment?.state === 'deployed' && (
                  <Button
                    size="sm"
                    variant="destructive"
                    onClick={() => {
                      if (confirm(`Are you sure you want to delete service ${serviceName}?`)) {
                        onDelete(serviceName);
                      }
                    }}
                    disabled={isDeleting}
                  >
                    {isDeleting ? (
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    ) : (
                      <Trash2 className="h-4 w-4 mr-2" />
                    )}
                    Delete Service
                  </Button>
                )}

                {/* Clean files button - only show when templates are up to date */}
                {service.lifecycle?.templates?.state === 'up_to_date' && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => {
                      if (confirm(`Are you sure you want to clean cached files for ${serviceName}? This will remove fetched templates and compiled manifests.`)) {
                        onCleanFiles(serviceName);
                      }
                    }}
                    disabled={isCleaningFiles}
                  >
                    {isCleaningFiles ? (
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    ) : (
                      <Trash2 className="h-4 w-4 mr-2" />
                    )}
                    Clean Files
                  </Button>
                )}
              </div>
            </div>

            {/* Dependencies */}
            {service.dependencies && service.dependencies.length > 0 && (
              <div className="space-y-3">
                <h3 className="text-sm font-semibold">Dependencies</h3>
                <div className="flex flex-wrap gap-2">
                  {service.dependencies.map((dep) => (
                    <Badge key={dep} variant="secondary">{dep}</Badge>
                  ))}
                </div>
              </div>
            )}
          </TabsContent>

          <TabsContent value="status" className="flex-1 overflow-y-auto space-y-4">
            {statusData ? (
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
                        <p className="text-sm">{statusData.name}</p>
                      </div>
                      <div>
                        <p className="text-sm font-medium text-muted-foreground">Namespace</p>
                        <p className="text-sm">{statusData.namespace}</p>
                      </div>
                    </div>

                    {statusData.replicas && (
                      <div>
                        <p className="text-sm font-medium text-muted-foreground mb-2">Replicas</p>
                        <div className="grid grid-cols-4 gap-2 text-sm">
                          <div className="bg-muted rounded p-2">
                            <p className="text-xs text-muted-foreground">Desired</p>
                            <p className="font-semibold">{statusData.replicas.desired}</p>
                          </div>
                          <div className="bg-muted rounded p-2">
                            <p className="text-xs text-muted-foreground">Current</p>
                            <p className="font-semibold">{statusData.replicas.current}</p>
                          </div>
                          <div className="bg-muted rounded p-2">
                            <p className="text-xs text-muted-foreground">Ready</p>
                            <p className="font-semibold">{statusData.replicas.ready}</p>
                          </div>
                          <div className="bg-muted rounded p-2">
                            <p className="text-xs text-muted-foreground">Available</p>
                            <p className="font-semibold">{statusData.replicas.available}</p>
                          </div>
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>

                {/* Pods Section */}
                {statusData.pods && statusData.pods.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-lg">Pods</CardTitle>
                      <CardDescription>{statusData.pods.length} pod(s)</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        {statusData.pods.map((pod) => (
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
                              <Badge
                                variant="outline"
                                className={
                                  pod.status.toLowerCase().includes('running')
                                    ? 'text-green-600 dark:text-green-400'
                                    : pod.status.toLowerCase().includes('pending')
                                    ? 'text-yellow-600 dark:text-yellow-400'
                                    : pod.status.toLowerCase().includes('failed')
                                    ? 'text-red-600 dark:text-red-400'
                                    : 'text-muted-foreground'
                                }
                              >
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

                {/* Configuration Preview */}
                {statusData.config && Object.keys(statusData.config).length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-lg">Current Configuration</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        {Object.entries(statusData.config).map(([key, value]) => (
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
              <div className="flex items-center justify-center py-12">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            )}
          </TabsContent>

          <TabsContent value="logs" className="flex-1 flex flex-col overflow-hidden space-y-4">
            {/* Log controls */}
            <div className="flex flex-wrap gap-4 items-center">
              <div className="flex items-center gap-2">
                <Label htmlFor="tail-select">Lines:</Label>
                <Select value={tail.toString()} onValueChange={(v) => setTail(Number(v))}>
                  <SelectTrigger id="tail-select" className="w-24">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="50">50</SelectItem>
                    <SelectItem value="100">100</SelectItem>
                    <SelectItem value="200">200</SelectItem>
                    <SelectItem value="500">500</SelectItem>
                    <SelectItem value="1000">1000</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {containers.length > 1 && (
                <div className="flex items-center gap-2">
                  <Label htmlFor="container-select">Container:</Label>
                  <Select value={container} onValueChange={setContainer}>
                    <SelectTrigger id="container-select" className="w-40">
                      <SelectValue placeholder="Select container" />
                    </SelectTrigger>
                    <SelectContent>
                      {containers.map((c) => (
                        <SelectItem key={c} value={c}>
                          {c}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}

              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="follow-checkbox"
                  checked={follow}
                  onChange={(e) => setFollow(e.target.checked)}
                  className="rounded"
                />
                <Label htmlFor="follow-checkbox">Follow</Label>
              </div>

              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="autoscroll-checkbox"
                  checked={autoScroll}
                  onChange={(e) => setAutoScroll(e.target.checked)}
                  className="rounded"
                />
                <Label htmlFor="autoscroll-checkbox">Auto-scroll</Label>
              </div>

              <div className="flex gap-2 ml-auto">
                <Button variant="outline" size="sm" onClick={handleRefresh} disabled={follow}>
                  <RefreshCw className="h-4 w-4 sm:mr-1" />
                  <span className="hidden sm:inline">Refresh</span>
                </Button>
                <Button variant="outline" size="sm" onClick={handleCopyLogs}>
                  <Copy className="h-4 w-4 sm:mr-1" />
                  <span className="hidden sm:inline">Copy</span>
                </Button>
                <Button variant="outline" size="sm" onClick={handleDownloadLogs}>
                  <Download className="h-4 w-4 sm:mr-1" />
                  <span className="hidden sm:inline">Download</span>
                </Button>
                <Button variant="outline" size="sm" onClick={handleClearLogs}>
                  Clear
                </Button>
              </div>
            </div>

            {/* Log display */}
            <Card className="flex-1 overflow-hidden">
              <CardContent className="p-0 h-full">
                <div className="h-[400px] overflow-y-auto bg-slate-950 dark:bg-slate-900 p-4 font-mono text-xs text-green-400">
                  {logsLoading ? (
                    <div className="flex items-center justify-center h-full">
                      <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                    </div>
                  ) : logs.length === 0 ? (
                    <div className="text-slate-500">No logs available</div>
                  ) : (
                    logs.map((line, index) => (
                      <div key={index} className="whitespace-pre-wrap break-all">
                        {line}
                      </div>
                    ))
                  )}
                  <div ref={logsEndRef} />
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}
