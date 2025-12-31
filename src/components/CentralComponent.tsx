import { Card } from './ui/card';
import { Button } from './ui/button';
import { Server, HardDrive, Settings, Clock, CheckCircle, BookOpen, ExternalLink, Loader2, AlertCircle, Database, FolderTree } from 'lucide-react';
import { Badge } from './ui/badge';
import { useCentralStatus } from '../hooks/useCentralStatus';
import { useInstanceConfig, useInstanceContext } from '../hooks';
import { usePageHelp } from '../hooks/usePageHelp';

export function CentralComponent() {
  const { currentInstance } = useInstanceContext();
  const { data: centralStatus, isLoading: statusLoading, error: statusError } = useCentralStatus();
  const { config: fullConfig, isLoading: configLoading } = useInstanceConfig(currentInstance);

  const serverConfig = fullConfig?.server as { host?: string; port?: number } | undefined;

  usePageHelp({
    title: 'What is the Central Service?',
    description: (
      <>
        <p className="mb-3 leading-relaxed">
          The Central Service is the "brain" of your personal cloud. It acts as the main coordinator that manages
          all the different services running on your network. Think of it like the control tower at an airport -
          it keeps track of what's happening, routes traffic between services, and ensures everything works together smoothly.
        </p>
        <p className="text-sm">
          This service handles configuration management, service discovery, and provides the web interface you're using right now.
        </p>
      </>
    ),
    icon: <BookOpen className="h-6 w-6 text-blue-600 dark:text-blue-400" />,
    color: 'bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-950/20 dark:to-indigo-950/20',
    actions: (
      <Button
        variant="outline"
        size="sm"
        className="text-blue-700 border-blue-300 hover:bg-blue-100 dark:text-blue-300 dark:border-blue-700 dark:hover:bg-blue-900/20"
      >
        <ExternalLink className="h-4 w-4 mr-2" />
        Learn more about service orchestration
      </Button>
    ),
  });

  const formatUptime = (seconds?: number) => {
    if (!seconds) return 'Unknown';

    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);

    const parts = [];
    if (days > 0) parts.push(`${days}d`);
    if (hours > 0) parts.push(`${hours}h`);
    if (minutes > 0) parts.push(`${minutes}m`);
    if (secs > 0 || parts.length === 0) parts.push(`${secs}s`);

    return parts.join(' ');
  };

  // Show error state
  if (statusError) {
    return (
      <Card className="p-8 text-center">
        <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
        <h3 className="text-lg font-medium mb-2">Error Loading Central Status</h3>
        <p className="text-muted-foreground mb-4">
          {(statusError as Error)?.message || 'An error occurred'}
        </p>
        <Button onClick={() => window.location.reload()}>Reload Page</Button>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <Card className="p-6">
        <div className="flex items-center gap-4 mb-6">
          <div className="p-2 bg-primary/10 rounded-lg">
            <Server className="h-6 w-6 text-primary" />
          </div>
          <div className="flex-1">
            <h2 className="text-2xl font-semibold">Central Service Status</h2>
            <p className="text-muted-foreground">
              Monitor the Wild Central server
            </p>
          </div>
          {centralStatus && (
            <Badge variant="success" className="flex items-center gap-2">
              <CheckCircle className="h-4 w-4" />
              {centralStatus.status === 'running' ? 'Running' : centralStatus.status}
            </Badge>
          )}
        </div>

        {statusLoading || configLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="space-y-6">
            {/* Server Information */}
            <div>
              <h3 className="text-lg font-medium mb-4">Server Information</h3>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <Card className="p-4 border-l-4 border-l-blue-500">
                  <div className="flex items-start gap-3">
                    <Settings className="h-5 w-5 text-blue-500 mt-0.5" />
                    <div className="flex-1">
                      <div className="text-sm text-muted-foreground mb-1">Version</div>
                      <div className="font-medium font-mono">{centralStatus?.version || 'Unknown'}</div>
                    </div>
                  </div>
                </Card>

                <Card className="p-4 border-l-4 border-l-green-500">
                  <div className="flex items-start gap-3">
                    <Clock className="h-5 w-5 text-green-500 mt-0.5" />
                    <div className="flex-1">
                      <div className="text-sm text-muted-foreground mb-1">Uptime</div>
                      <div className="font-medium">{formatUptime(centralStatus?.uptimeSeconds)}</div>
                    </div>
                  </div>
                </Card>

                <Card className="p-4 border-l-4 border-l-purple-500">
                  <div className="flex items-start gap-3">
                    <Database className="h-5 w-5 text-purple-500 mt-0.5" />
                    <div className="flex-1">
                      <div className="text-sm text-muted-foreground mb-1">Instances</div>
                      <div className="font-medium">{centralStatus?.instances.count || 0} configured</div>
                      {centralStatus?.instances.names && centralStatus.instances.names.length > 0 && (
                        <div className="text-xs text-muted-foreground mt-1">
                          {centralStatus.instances.names.join(', ')}
                        </div>
                      )}
                    </div>
                  </div>
                </Card>

              </div>
            </div>

            {/* Configuration */}
            <div>
              <h3 className="text-lg font-medium mb-4">Configuration</h3>
              <div className="space-y-3">
                <Card className="p-4 border-l-4 border-l-cyan-500">
                  <div className="flex items-start gap-3">
                    <Server className="h-5 w-5 text-cyan-500 mt-0.5" />
                    <div className="flex-1">
                      <div className="text-sm text-muted-foreground mb-1">Server Host</div>
                      <div className="font-medium font-mono">{serverConfig?.host || '0.0.0.0'}</div>
                    </div>
                    <div className="flex-1">
                      <div className="text-sm text-muted-foreground mb-1">Server Port</div>
                      <div className="font-medium font-mono">{serverConfig?.port || 5055}</div>
                    </div>
                  </div>
                </Card>

                <Card className="p-4 border-l-4 border-l-indigo-500">
                  <div className="flex items-start gap-3">
                    <HardDrive className="h-5 w-5 text-indigo-500 mt-0.5" />
                    <div className="flex-1">
                      <div className="text-sm text-muted-foreground mb-1">Data Directory</div>
                      <div className="font-medium font-mono text-sm break-all">
                        {centralStatus?.dataDir || '/var/lib/wild-central'}
                      </div>
                    </div>
                  </div>
                </Card>

                <Card className="p-4 border-l-4 border-l-pink-500">
                  <div className="flex items-start gap-3">
                    <FolderTree className="h-5 w-5 text-pink-500 mt-0.5" />
                    <div className="flex-1">
                      <div className="text-sm text-muted-foreground mb-1">Apps Directory</div>
                      <div className="font-medium font-mono text-sm break-all">
                        {centralStatus?.appsDir || '/opt/wild-cloud/apps'}
                      </div>
                    </div>
                  </div>
                </Card>
              </div>
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}
