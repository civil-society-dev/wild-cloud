import { useState, useEffect } from 'react';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { Input, Label } from './ui';
import { Server, HardDrive, Settings, Clock, CheckCircle, BookOpen, ExternalLink, Loader2, AlertCircle, Database, FolderTree, Mail, Router, Edit2, Check, X, Network, Globe } from 'lucide-react';
import { Badge } from './ui/badge';
import { useCentralStatus } from '../hooks/useCentralStatus';
import { useInstanceConfig, useInstanceContext, useConfig } from '../hooks';
import { usePageHelp } from '../hooks/usePageHelp';

interface GlobalConfigForm {
  operator?: {
    email?: string;
  };
  cloud?: {
    router?: {
      ip?: string;
      dynamicDns?: string;
    };
    dnsmasq?: {
      ip?: string;
      interface?: string;
    };
  };
}

export function CentralComponent() {
  const { currentInstance } = useInstanceContext();
  const { data: centralStatus, isLoading: statusLoading, error: statusError } = useCentralStatus();
  const { config: fullConfig, isLoading: configLoading } = useInstanceConfig(currentInstance);
  const { config: globalConfig, updateConfig: updateGlobalConfig, isUpdating } = useConfig();

  const [editingOperator, setEditingOperator] = useState(false);
  const [editingRouter, setEditingRouter] = useState(false);
  const [editingDnsmasq, setEditingDnsmasq] = useState(false);
  const [formValues, setFormValues] = useState<GlobalConfigForm>({});

  const serverConfig = fullConfig?.server as { host?: string; port?: number } | undefined;

  // Sync form values when globalConfig loads
  useEffect(() => {
    if (globalConfig) {
      setFormValues({
        operator: globalConfig.operator,
        cloud: {
          router: globalConfig.cloud?.router,
          dnsmasq: globalConfig.cloud?.dnsmasq,
        },
      });
    }
  }, [globalConfig]);

  const handleOperatorEdit = () => {
    setEditingOperator(true);
  };

  const handleOperatorSave = async () => {
    if (!globalConfig || !formValues.operator?.email) return;
    try {
      await updateGlobalConfig({
        ...globalConfig,
        operator: {
          ...globalConfig.operator,
          email: formValues.operator.email,
        },
      });
      setEditingOperator(false);
    } catch (err) {
      console.error('Failed to save operator:', err);
    }
  };

  const handleOperatorCancel = () => {
    if (globalConfig) {
      setFormValues(prev => ({
        ...prev,
        operator: globalConfig.operator,
      }));
    }
    setEditingOperator(false);
  };

  const handleRouterEdit = () => {
    setEditingRouter(true);
  };

  const handleRouterSave = async () => {
    if (!globalConfig || !formValues.cloud?.router) return;
    try {
      await updateGlobalConfig({
        ...globalConfig,
        cloud: {
          ...globalConfig.cloud,
          router: formValues.cloud.router,
        },
      });
      setEditingRouter(false);
    } catch (err) {
      console.error('Failed to save router:', err);
    }
  };

  const handleRouterCancel = () => {
    if (globalConfig) {
      setFormValues(prev => ({
        ...prev,
        cloud: {
          ...prev.cloud,
          router: globalConfig.cloud?.router,
        },
      }));
    }
    setEditingRouter(false);
  };

  const handleDnsmasqEdit = () => {
    setEditingDnsmasq(true);
  };

  const handleDnsmasqSave = async () => {
    if (!globalConfig || !formValues.cloud?.dnsmasq) return;
    try {
      await updateGlobalConfig({
        ...globalConfig,
        cloud: {
          ...globalConfig.cloud,
          dnsmasq: formValues.cloud.dnsmasq,
        },
      });
      setEditingDnsmasq(false);
    } catch (err) {
      console.error('Failed to save dnsmasq:', err);
    }
  };

  const handleDnsmasqCancel = () => {
    if (globalConfig) {
      setFormValues(prev => ({
        ...prev,
        cloud: {
          ...prev.cloud,
          dnsmasq: globalConfig.cloud?.dnsmasq,
        },
      }));
    }
    setEditingDnsmasq(false);
  };

  const updateFormValue = (path: string, value: string) => {
    setFormValues(prev => {
      const keys = path.split('.');
      if (keys.length === 2 && keys[0] === 'operator') {
        return {
          ...prev,
          operator: {
            ...prev.operator,
            [keys[1]]: value,
          },
        };
      }
      if (keys.length === 3 && keys[0] === 'cloud' && keys[1] === 'router') {
        return {
          ...prev,
          cloud: {
            ...prev.cloud,
            router: {
              ...prev.cloud?.router,
              [keys[2]]: value,
            },
          },
        };
      }
      if (keys.length === 3 && keys[0] === 'cloud' && keys[1] === 'dnsmasq') {
        return {
          ...prev,
          cloud: {
            ...prev.cloud,
            dnsmasq: {
              ...prev.cloud?.dnsmasq,
              [keys[2]]: value,
            },
          },
        };
      }
      return prev;
    });
  };

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

                <Card className="p-4 border-l-4 border-l-amber-500">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <Mail className="h-5 w-5 text-amber-500" />
                        <div className="text-sm text-muted-foreground">Operator Email</div>
                      </div>
                      {!editingOperator && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={handleOperatorEdit}
                          disabled={isUpdating}
                        >
                          <Edit2 className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                    {editingOperator ? (
                      <div className="space-y-3">
                        <div>
                          <Label htmlFor="operator-email">Email</Label>
                          <Input
                            id="operator-email"
                            type="email"
                            value={formValues.operator?.email || ''}
                            onChange={(e) => updateFormValue('operator.email', e.target.value)}
                            placeholder="email@example.com"
                            className="mt-1"
                          />
                        </div>
                        <div className="flex justify-end gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={handleOperatorCancel}
                            disabled={isUpdating}
                          >
                            <X className="h-4 w-4 mr-1" />
                            Cancel
                          </Button>
                          <Button
                            size="sm"
                            onClick={handleOperatorSave}
                            disabled={isUpdating || !formValues.operator?.email}
                          >
                            {isUpdating ? (
                              <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                            ) : (
                              <Check className="h-4 w-4 mr-1" />
                            )}
                            Save
                          </Button>
                        </div>
                      </div>
                    ) : (
                      <div className="font-medium font-mono text-sm ml-7">
                        {globalConfig?.operator?.email || (
                          <span className="text-muted-foreground italic">Not configured</span>
                        )}
                      </div>
                    )}
                  </Card>

                <Card className="p-4 border-l-4 border-l-teal-500">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <Router className="h-5 w-5 text-teal-500" />
                        <div className="text-sm text-muted-foreground">Router</div>
                      </div>
                      {!editingRouter && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={handleRouterEdit}
                          disabled={isUpdating}
                        >
                          <Edit2 className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                    {editingRouter ? (
                      <div className="space-y-3">
                        <div>
                          <Label htmlFor="router-ip">Router IP</Label>
                          <Input
                            id="router-ip"
                            value={formValues.cloud?.router?.ip || ''}
                            onChange={(e) => updateFormValue('cloud.router.ip', e.target.value)}
                            placeholder="192.168.1.1"
                            className="mt-1"
                          />
                        </div>
                        <div>
                          <Label htmlFor="router-ddns">Dynamic DNS</Label>
                          <Input
                            id="router-ddns"
                            value={formValues.cloud?.router?.dynamicDns || ''}
                            onChange={(e) => updateFormValue('cloud.router.dynamicDns', e.target.value)}
                            placeholder="example.ddns.com"
                            className="mt-1"
                          />
                        </div>
                        <div className="flex justify-end gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={handleRouterCancel}
                            disabled={isUpdating}
                          >
                            <X className="h-4 w-4 mr-1" />
                            Cancel
                          </Button>
                          <Button
                            size="sm"
                            onClick={handleRouterSave}
                            disabled={isUpdating}
                          >
                            {isUpdating ? (
                              <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                            ) : (
                              <Check className="h-4 w-4 mr-1" />
                            )}
                            Save
                          </Button>
                        </div>
                      </div>
                    ) : (
                      <div className="space-y-1 ml-7">
                        {globalConfig?.cloud?.router?.ip && (
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-muted-foreground w-16">IP:</span>
                            <span className="font-medium font-mono text-sm">{globalConfig.cloud.router.ip}</span>
                          </div>
                        )}
                        {globalConfig?.cloud?.router?.dynamicDns && (
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-muted-foreground w-16">DDNS:</span>
                            <span className="font-medium font-mono text-sm">{globalConfig.cloud.router.dynamicDns}</span>
                          </div>
                        )}
                        {!globalConfig?.cloud?.router?.ip && !globalConfig?.cloud?.router?.dynamicDns && (
                          <div className="text-sm text-muted-foreground italic">Not configured</div>
                        )}
                      </div>
                    )}
                  </Card>

                <Card className="p-4 border-l-4 border-l-green-500">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <Network className="h-5 w-5 text-green-500" />
                        <div className="text-sm text-muted-foreground">Dnsmasq</div>
                      </div>
                      {!editingDnsmasq && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={handleDnsmasqEdit}
                          disabled={isUpdating}
                        >
                          <Edit2 className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                    {editingDnsmasq ? (
                      <div className="space-y-3">
                        <div>
                          <Label htmlFor="dns-ip">DNS IP</Label>
                          <Input
                            id="dns-ip"
                            value={formValues.cloud?.dnsmasq?.ip || ''}
                            onChange={(e) => updateFormValue('cloud.dnsmasq.ip', e.target.value)}
                            placeholder="192.168.1.1"
                            className="mt-1"
                          />
                        </div>
                        <div>
                          <Label htmlFor="dnsmasq-interface">Network Interface</Label>
                          <Input
                            id="dnsmasq-interface"
                            value={formValues.cloud?.dnsmasq?.interface || ''}
                            onChange={(e) => updateFormValue('cloud.dnsmasq.interface', e.target.value)}
                            placeholder="eth0"
                            className="mt-1"
                          />
                        </div>
                        <div className="flex justify-end gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={handleDnsmasqCancel}
                            disabled={isUpdating}
                          >
                            <X className="h-4 w-4 mr-1" />
                            Cancel
                          </Button>
                          <Button
                            size="sm"
                            onClick={handleDnsmasqSave}
                            disabled={isUpdating}
                          >
                            {isUpdating ? (
                              <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                            ) : (
                              <Check className="h-4 w-4 mr-1" />
                            )}
                            Save
                          </Button>
                        </div>
                      </div>
                    ) : (<>
                      <div className="space-y-1 ml-7">
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-muted-foreground w-16">IP:</span>
                          {globalConfig?.cloud?.dnsmasq?.ip ? (
                            <span className="font-medium font-mono text-sm">{globalConfig.cloud.dnsmasq.ip}</span>
                          ) : (
                            <div className="text-sm text-muted-foreground italic">Not configured</div>
                          )}
                        </div>
                      </div>
                      <div className="font-medium font-mono text-sm ml-7">
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-muted-foreground w-16">Interface:</span>
                          {globalConfig?.cloud?.dnsmasq?.interface ? (
                            <span className="font-medium font-mono text-sm">{globalConfig.cloud.dnsmasq.interface}</span>
                          ) : (
                            <div className="text-sm text-muted-foreground italic">Not configured</div>
                          )}
                        </div>
                      </div>
                      </>
                    )}
                  </Card>
              </div>
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}
