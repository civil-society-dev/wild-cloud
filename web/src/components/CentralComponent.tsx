import { useState, useEffect } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from './ui/card';
import { Button } from './ui/button';
import { Input, Label } from './ui';
import { Textarea } from './ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';
import { HardDrive, Settings, Clock, CheckCircle, BookOpen, ExternalLink, Loader2, AlertCircle, Database, FolderTree, Mail, Router, Edit2, Check, X, Globe, Play, RotateCw, Copy, ChevronDown, ChevronUp, Edit } from 'lucide-react';
import { Badge } from './ui/badge';
import { useCentralStatus } from '../hooks/useCentralStatus';
import { useInstanceConfig, useInstanceContext, useConfig } from '../hooks';
import { usePageHelp } from '../hooks/usePageHelp';
import { useDnsmasq } from '../hooks/useDnsmasq';
import { apiService } from '../services/api-legacy';

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
  const { isLoading: configLoading } = useInstanceConfig(currentInstance);
  const { config: globalConfig, updateConfig: updateGlobalConfig, isUpdating } = useConfig();

  const [editingOperator, setEditingOperator] = useState(false);
  const [editingRouterDdns, setEditingRouterDdns] = useState(false);
  const [formValues, setFormValues] = useState<GlobalConfigForm>({});

  // DNS Service state
  const {
    status: dnsStatus,
    isLoadingStatus: isDnsStatusLoading,
    config: dnsConfig,
    fetchConfig: fetchDnsConfig,
    generateConfig: generateDnsConfig,
    isGenerating: isDnsGenerating,
    generateData: dnsGenerateData,
    restart: restartDns,
    isRestarting: isDnsRestarting,
    restartData: dnsRestartData,
    generateError: dnsGenerateError,
    restartError: dnsRestartError
  } = useDnsmasq();

  const [showDnsAdvanced, setShowDnsAdvanced] = useState(false);
  const [showDnsEditDialog, setShowDnsEditDialog] = useState(false);
  const [editedDnsConfig, setEditedDnsConfig] = useState('');
  const [copiedDnsIp, setCopiedDnsIp] = useState(false);

  const isDnsRunning = dnsStatus?.status === 'active';
  const dnsIp = dnsStatus?.ip;

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

  const handleRouterDdnsEdit = () => {
    setEditingRouterDdns(true);
  };

  const handleRouterDdnsSave = async () => {
    if (!globalConfig) return;
    try {
      await updateGlobalConfig({
        ...globalConfig,
        cloud: {
          ...globalConfig.cloud,
          router: {
            ...globalConfig.cloud?.router,
            dynamicDns: formValues.cloud?.router?.dynamicDns || '',
          },
        },
      });
      setEditingRouterDdns(false);
    } catch (err) {
      console.error('Failed to save router DDNS:', err);
    }
  };

  const handleRouterDdnsCancel = () => {
    if (globalConfig) {
      setFormValues(prev => ({
        ...prev,
        cloud: {
          ...prev.cloud,
          router: {
            ...prev.cloud?.router,
            dynamicDns: globalConfig.cloud?.router?.dynamicDns,
          },
        },
      }));
    }
    setEditingRouterDdns(false);
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

  // DNS Service handlers
  const handleCopyDnsIp = () => {
    if (dnsIp) {
      navigator.clipboard.writeText(dnsIp);
      setCopiedDnsIp(true);
      setTimeout(() => setCopiedDnsIp(false), 2000);
    }
  };

  const handleDnsStart = () => {
    generateDnsConfig(true);
  };

  const handleShowDnsAdvanced = () => {
    if (!showDnsAdvanced && !dnsConfig) {
      fetchDnsConfig();
    }
    setShowDnsAdvanced(!showDnsAdvanced);
  };

  const handleEditDnsConfig = () => {
    if (dnsConfig?.content) {
      setEditedDnsConfig(dnsConfig.content);
    } else if (dnsGenerateData?.config || dnsGenerateData?.content) {
      setEditedDnsConfig(dnsGenerateData.config || dnsGenerateData.content || '');
    } else {
      setEditedDnsConfig('');
    }
    setShowDnsEditDialog(true);
  };

  const handleSaveDnsConfig = async () => {
    try {
      await apiService.writeDnsmasqConfig(editedDnsConfig);
      setShowDnsEditDialog(false);
      fetchDnsConfig();
    } catch (error) {
      console.error('Failed to save config:', error);
    }
  };

  const handleSaveAndRestartDns = async () => {
    try {
      await apiService.writeDnsmasqConfig(editedDnsConfig);
      await apiService.restartDnsmasq();
      setShowDnsEditDialog(false);
      fetchDnsConfig();
    } catch (error) {
      console.error('Failed to save config and restart:', error);
    }
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
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Wild Central</h2>
          <p className="text-muted-foreground">
            Manage your Wild Central server
          </p>
        </div>
        {centralStatus && (
          <Badge variant="success" className="flex items-center gap-2">
            <CheckCircle className="h-4 w-4" />
            {centralStatus.status === 'running' ? 'Running' : centralStatus.status}
          </Badge>
        )}
      </div>

      {statusLoading ? (
        <Card className="p-6">
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        </Card>
      ) : (
        <>
          {/* Server Information */}
          <Card className="p-6">
            <h3 className="text-lg font-medium mb-4">Wild Central Status</h3>
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
          </Card>
        </>
      )}

      {configLoading ? (
        <Card className="p-6">
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        </Card>
      ) : (
        <>
          {/* Configuration */}
          <Card className="p-6">
            <h3 className="text-lg font-medium mb-4">Wild Central Configuration</h3>
            <div className="space-y-3">

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
            </div>
          </Card>

          {/* LAN Router Configuration */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div className="p-2 bg-primary/10 rounded-lg">
                    <Router className="h-6 w-6 text-primary" />
                  </div>
                  <div>
                    <CardTitle>LAN Router Configuration</CardTitle>
                    <p className="text-sm text-muted-foreground">
                      Configure your router to use Wild Central for DNS and dynamic DNS
                    </p>
                  </div>
                </div>
                <Badge
                  variant={isDnsRunning ? 'default' : 'secondary'}
                  className="gap-2"
                >
                  {isDnsStatusLoading ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : isDnsRunning ? (
                    <CheckCircle className="h-4 w-4" />
                  ) : (
                    <XCircle className="h-4 w-4" />
                  )}
                  DNS: {isDnsStatusLoading ? 'Checking...' : isDnsRunning ? 'Running' : 'Stopped'}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Router IP Display */}
              <div className="flex items-center justify-between p-4 bg-muted/30 rounded-lg">
                <div>
                  <div className="text-sm text-muted-foreground">Router IP Address</div>
                  <div className="font-mono text-lg font-medium mt-1">
                    {globalConfig?.cloud?.router?.ip || 'Auto-detecting...'}
                  </div>
                </div>
                {globalConfig?.cloud?.router?.ip && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => window.open(`http://${globalConfig.cloud.router.ip}`, '_blank')}
                    className="gap-2"
                  >
                    <ExternalLink className="h-4 w-4" />
                    Open Router Admin
                  </Button>
                )}
              </div>

              {/* Instructions */}
              <div className="p-4 bg-gradient-to-r from-cyan-50 to-blue-50 dark:from-cyan-900/20 dark:to-blue-900/20 rounded-lg border border-cyan-200 dark:border-cyan-800">
                <div className="flex items-start gap-3">
                  <BookOpen className="h-5 w-5 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div className="space-y-2">
                    <p className="text-sm font-medium text-cyan-900 dark:text-cyan-100">
                      Configuration Steps
                    </p>
                    <ol className="text-xs text-cyan-700 dark:text-cyan-300 space-y-1 list-decimal list-inside">
                      <li>Configure your router's DDNS hostname</li>
                      <li>Set Wild Central as your router's primary DNS server</li>
                    </ol>
                  </div>
                </div>
              </div>

              {/* Step 1: Router DDNS */}
              <Card className="p-4 border-l-4 border-l-teal-500">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-semibold bg-teal-100 dark:bg-teal-900/30 text-teal-700 dark:text-teal-300 px-2 py-1 rounded">STEP 1</span>
                    <div className="text-sm font-medium">Router Dynamic DNS</div>
                  </div>
                  {!editingRouterDdns && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleRouterDdnsEdit}
                      disabled={isUpdating}
                    >
                      <Edit2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
                {editingRouterDdns ? (
                  <div className="space-y-3 mt-3">
                    <div>
                      <Label htmlFor="router-ddns">Dynamic DNS Hostname</Label>
                      <Input
                        id="router-ddns"
                        value={formValues.cloud?.router?.dynamicDns || ''}
                        onChange={(e) => updateFormValue('cloud.router.dynamicDns', e.target.value)}
                        placeholder="example.ddns.com"
                        className="mt-1"
                      />
                      <p className="text-xs text-muted-foreground mt-1">
                        Find this in your router's DDNS settings
                      </p>
                    </div>
                    <div className="flex justify-end gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={handleRouterDdnsCancel}
                        disabled={isUpdating}
                      >
                        <X className="h-4 w-4 mr-1" />
                        Cancel
                      </Button>
                      <Button
                        size="sm"
                        onClick={handleRouterDdnsSave}
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
                  <div className="mt-3">
                    {globalConfig?.cloud?.router?.dynamicDns ? (
                      <div>
                        <span className="font-mono text-sm">{globalConfig.cloud.router.dynamicDns}</span>
                      </div>
                    ) : (
                      <div>
                        <div className="text-sm text-muted-foreground italic mb-2">Not configured</div>
                        <div className="p-2 bg-muted/50 rounded-md text-xs text-muted-foreground">
                          <p className="mb-1">To find your DDNS hostname:</p>
                          <ol className="list-decimal list-inside space-y-0.5 ml-2">
                            <li>Open your router admin panel</li>
                            <li>Look for "Dynamic DNS" or "DDNS" settings</li>
                            <li>Find your hostname (e.g., example.ddns.net)</li>
                          </ol>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </Card>

              {/* Step 2: DNS Configuration */}
              <Card className="p-4 border-l-4 border-l-green-500">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-semibold bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 px-2 py-1 rounded">STEP 2</span>
                    <div className="text-sm font-medium">Wild Central DNS IP</div>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleCopyDnsIp}
                    disabled={!dnsIp}
                  >
                    <Copy className="h-4 w-4 mr-1" />
                    {copiedDnsIp ? 'Copied!' : 'Copy'}
                  </Button>
                </div>
                <div className="mt-3">
                  <p className="text-xs text-muted-foreground mb-2">
                    Configure your router to use this IP as the primary DNS server:
                  </p>
                  <code className="text-lg font-mono font-bold bg-muted px-3 py-1 rounded inline-block">
                    {dnsIp || 'Auto-detecting...'}
                  </code>
                </div>
              </Card>

              {/* Action Buttons */}
              <div className="flex gap-2">
                {!isDnsRunning ? (
                  <Button
                    onClick={handleDnsStart}
                    disabled={isDnsGenerating}
                    className="gap-2"
                    size="lg"
                  >
                    {isDnsGenerating ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Play className="h-4 w-4" />
                    )}
                    Start DNS
                  </Button>
                ) : (
                  <Button
                    onClick={() => restartDns()}
                    disabled={isDnsRestarting}
                    variant="outline"
                    className="gap-2"
                  >
                    {isDnsRestarting ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <RotateCw className="h-4 w-4" />
                    )}
                    Restart DNS
                  </Button>
                )}
                <Button
                  onClick={handleShowDnsAdvanced}
                  variant="outline"
                  className="gap-2"
                >
                  {showDnsAdvanced ? (
                    <ChevronUp className="h-4 w-4" />
                  ) : (
                    <ChevronDown className="h-4 w-4" />
                  )}
                  Advanced Configuration
                </Button>
              </div>

              {/* Success/Error Messages */}
              {(dnsGenerateData || dnsRestartData) && (
                <Alert className="mt-4">
                  <CheckCircle className="h-4 w-4" />
                  <AlertDescription>
                    {dnsGenerateData?.message || dnsRestartData?.message}
                  </AlertDescription>
                </Alert>
              )}

              {(dnsGenerateError || dnsRestartError) && (
                <Alert variant="destructive" className="mt-4">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>
                    {dnsGenerateError || dnsRestartError}
                  </AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>

          {/* Advanced DNS Configuration */}
          {showDnsAdvanced && (
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>Advanced DNS Configuration</CardTitle>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleEditDnsConfig}
                    className="gap-2"
                  >
                    <Edit className="h-4 w-4" />
                    Edit Config
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-xs font-mono">
                  {isDnsGenerating && !dnsConfig && !dnsGenerateData && (
                    <div className="flex items-center justify-center py-4">
                      <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    </div>
                  )}
                  {(dnsConfig?.content || dnsGenerateData?.config || dnsGenerateData?.content) && (
                    dnsConfig?.content || dnsGenerateData?.config || dnsGenerateData?.content
                  )}
                  {!isDnsGenerating && !dnsGenerateData && !dnsConfig && (
                    <div className="text-center p-8 text-sm text-muted-foreground">
                      <p>Configuration preview will appear here</p>
                    </div>
                  )}
                </pre>
              </CardContent>
            </Card>
          )}
        </>
      )}

      {/* Edit DNS Config Dialog */}
      <Dialog open={showDnsEditDialog} onOpenChange={setShowDnsEditDialog}>
        <DialogContent className="max-w-4xl max-h-[80vh]">
          <DialogHeader>
            <DialogTitle>Edit DNS Configuration</DialogTitle>
            <DialogDescription>
              Modify the dnsmasq configuration. Changes will take effect after saving and restarting.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Textarea
              value={editedDnsConfig}
              onChange={(e) => setEditedDnsConfig(e.target.value)}
              className="font-mono text-xs min-h-[400px]"
              placeholder="# dnsmasq configuration"
            />
          </div>
          <DialogFooter className="gap-2">
            <Button
              variant="outline"
              onClick={() => setShowDnsEditDialog(false)}
            >
              Cancel
            </Button>
            <Button
              variant="outline"
              onClick={handleSaveDnsConfig}
            >
              Save
            </Button>
            <Button
              onClick={handleSaveAndRestartDns}
            >
              Save & Restart
            </Button>
          </DialogFooter>
          <p className="text-xs text-muted-foreground text-center">
            Save: Write config without restarting | Save & Restart: Write config and restart service
          </p>
        </DialogContent>
      </Dialog>
    </div>
  );
}
