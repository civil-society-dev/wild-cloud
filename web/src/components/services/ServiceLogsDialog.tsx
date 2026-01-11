import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Card, CardContent } from '@/components/ui/card';
import { ServiceStatusBadge } from './ServiceStatusBadge';
import { useServiceStatus } from '@/hooks/useServices';
import { servicesApi } from '@/services/api';
import { Copy, Download, RefreshCw } from 'lucide-react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

interface ServiceLogsDialogProps {
  instanceName: string;
  serviceName: string;
  open: boolean;
  onClose: () => void;
}

export function ServiceLogsDialog({
  instanceName,
  serviceName,
  open,
  onClose,
}: ServiceLogsDialogProps) {
  const { data: status } = useServiceStatus(instanceName, serviceName);
  const [logs, setLogs] = useState<string[]>([]);
  const [follow, setFollow] = useState(false);
  const [tail, setTail] = useState(100);
  const [container, setContainer] = useState<string | undefined>(undefined);
  const [autoScroll, setAutoScroll] = useState(true);
  const logsEndRef = useRef<HTMLDivElement>(null);
  const logsContainerRef = useRef<HTMLDivElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  // Extract unique container names from all pods
  const containers = useMemo(() => {
    if (!status?.pods) return [];
    const containerSet = new Set<string>();
    status.pods.forEach((pod) => {
      pod.containers?.forEach((c: string) => containerSet.add(c));
    });
    return Array.from(containerSet);
  }, [status?.pods]);

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
    if (!open) return;
    try {
      const url = servicesApi.getLogsUrl(instanceName, serviceName, tail, false, container);
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`Failed to fetch logs: ${response.statusText}`);
      }
      const data = await response.json();
      // API returns { lines: string[] }
      if (data.lines && Array.isArray(data.lines)) {
        setLogs(data.lines);
      } else {
        setLogs([]);
      }
    } catch (error) {
      console.error('Error fetching logs:', error);
      setLogs([`Error: ${error instanceof Error ? error.message : 'Failed to fetch logs'}`]);
    }
  }, [instanceName, serviceName, tail, open, container]);

  // Set up SSE streaming when follow is enabled
  useEffect(() => {
    if (!open) return;

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

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-5xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            {serviceName}
            {status && <ServiceStatusBadge status={status.deploymentStatus} />}
          </DialogTitle>
          <DialogDescription>Service logs</DialogDescription>
        </DialogHeader>

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

        <Card className="flex-1 overflow-hidden">
          <CardContent className="p-0 h-full">
            <div
              ref={logsContainerRef}
              className="h-[400px] overflow-y-auto bg-slate-950 dark:bg-slate-900 p-4 font-mono text-xs text-green-400"
            >
              {logs.length === 0 ? (
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
      </DialogContent>
    </Dialog>
  );
}
