import { useState, useEffect } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from './ui/card';
import { Button } from './ui/button';
import { Alert, AlertDescription } from './ui/alert';
import { Badge } from './ui/badge';
import { Textarea } from './ui/textarea';
import { Input } from './ui/input';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog';
import {
  Globe,
  CheckCircle,
  XCircle,
  AlertCircle,
  Play,
  RotateCw,
  Settings,
  Loader2,
  Copy,
  ChevronDown,
  ChevronUp,
  Edit,
  TestTube2,
  ExternalLink
} from 'lucide-react';
import { useDnsmasq } from '../hooks/useDnsmasq';
import { useConfig } from '../hooks';
import { useInstances } from '../hooks/useInstances';
import { instancesApi } from '../services/api';
import { apiService } from '../services/api-legacy';
import { usePageHelp } from '../hooks/usePageHelp';
import type { InstanceConfig } from '../types';

export function DnsComponent() {
  const { config: globalConfig, updateConfig, isUpdating } = useConfig();
  const dnsIp = globalConfig?.cloud?.dnsmasq?.ip;
  const routerIp = globalConfig?.cloud?.router?.ip;
  const dynamicDns = globalConfig?.cloud?.router?.dynamicDns || '';
  const baseDomain = globalConfig?.cloud?.baseDomain;

  const { instances, isLoading: isLoadingInstances } = useInstances();

  usePageHelp({
    title: 'How DNS Works in Wild Cloud',
    description: (
      <>
        <p className="mb-3 leading-relaxed">
          The DNS service resolves domain names for your Wild Cloud instances. When running, it allows
          devices on your network to access services like{' '}
          <code className="bg-muted px-1 rounded">photos.cloud.local</code> instead
          of remembering IP addresses.
        </p>
        <p className="leading-relaxed">
          <strong>Router Setup:</strong> Configure your router's primary DNS server to{' '}
          <code className="bg-muted px-1 rounded font-semibold">{dnsIp || 'the IP shown above'}</code> so
          all devices on your network can resolve Wild Cloud domains automatically.
        </p>
      </>
    ),
    icon: <Globe className="h-6 w-6" />,
  });

  const {
    status,
    isLoadingStatus,
    config,
    fetchConfig,
    generateConfig,
    isGenerating,
    generateData,
    restart,
    isRestarting,
    restartData,
    generateError,
    restartError
  } = useDnsmasq();

  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [editedConfig, setEditedConfig] = useState('');
  const [copiedIp, setCopiedIp] = useState(false);
  const [isLoadingFromInstances, setIsLoadingFromInstances] = useState(false);
  const [testingDomain, setTestingDomain] = useState<string | null>(null);
  const [testType, setTestType] = useState<'internal' | 'external' | null>(null);
  const [testResults, setTestResults] = useState<Record<string, {
    internal?: { success: boolean; message: string };
    external?: { success: boolean; message: string };
  }>>({});
  const [isEditingDdns, setIsEditingDdns] = useState(false);
  const [editedDdns, setEditedDdns] = useState(dynamicDns);
  const [ddnsSaveSuccess, setDdnsSaveSuccess] = useState(false);
  const [instanceConfigs, setInstanceConfigs] = useState<Record<string, InstanceConfig>>({});

  const isRunning = status?.status === 'active';

  // Fetch config on mount to populate Configured Instances section
  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  // Update editedDdns when dynamicDns changes
  useEffect(() => {
    setEditedDdns(dynamicDns);
  }, [dynamicDns]);

  // Fetch configs for all instances
  useEffect(() => {
    const fetchInstanceConfigs = async () => {
      if (!instances || instances.length === 0) return;

      const configs: Record<string, InstanceConfig> = {};
      await Promise.all(
        instances.map(async (instanceName) => {
          try {
            const config = await instancesApi.getConfig(instanceName) as InstanceConfig;
            configs[instanceName] = config;
          } catch (error) {
            console.error(`Failed to fetch config for instance ${instanceName}:`, error);
          }
        })
      );
      setInstanceConfigs(configs);
    };

    fetchInstanceConfigs();
  }, [instances]);

  // Build list of configured domains from instance configs
  const configuredDomains = (() => {
    return instances.map(instanceName => {
      const config = instanceConfigs[instanceName];
      const domain = config?.cloud?.domain || instanceName;
      const hasLoadBalancer = !!config?.cluster?.loadBalancerIp;

      return {
        domain,
        instanceName,
        isCommented: !hasLoadBalancer,
        loadBalancerIp: config?.cluster?.loadBalancerIp
      };
    });
  })();

  const handleCopyIp = () => {
    if (dnsIp) {
      navigator.clipboard.writeText(dnsIp);
      setCopiedIp(true);
      setTimeout(() => setCopiedIp(false), 2000);
    }
  };

  const handleTestDomain = async (domain: string, type: 'internal' | 'external') => {
    setTestingDomain(domain);
    setTestType(type);

    try {
      if (type === 'external') {
        // Use Cloudflare DNS-over-HTTPS to test external resolution
        const response = await fetch(`https://cloudflare-dns.com/dns-query?name=${domain}&type=A`, {
          headers: { 'accept': 'application/dns-json' }
        });
        const data = await response.json();

        if (data.Status === 0 && data.Answer && data.Answer.length > 0) {
          // Find the A record (type 1) in the answer chain (may have CNAME first)
          const aRecord = data.Answer.find((ans: { type: number; data: string }) => ans.type === 1);
          const resolvedIp = aRecord ? aRecord.data : data.Answer[data.Answer.length - 1].data;
          setTestResults(prev => ({
            ...prev,
            [domain]: {
              ...prev[domain],
              external: {
                success: true,
                message: resolvedIp
              }
            }
          }));
        } else {
          setTestResults(prev => ({
            ...prev,
            [domain]: {
              ...prev[domain],
              external: {
                success: false,
                message: 'No public DNS record (LAN-only)'
              }
            }
          }));
        }
      } else {
        // Test internal resolution via Wild Central API
        const response = await fetch(`http://localhost:5055/api/v1/network/resolve?domain=${domain}`);
        const data = await response.json();

        if (data.success && data.ip) {
          setTestResults(prev => ({
            ...prev,
            [domain]: {
              ...prev[domain],
              internal: {
                success: true,
                message: data.ip
              }
            }
          }));
        } else {
          setTestResults(prev => ({
            ...prev,
            [domain]: {
              ...prev[domain],
              internal: {
                success: false,
                message: data.error || 'Not found'
              }
            }
          }));
        }
      }
    } catch {
      setTestResults(prev => ({
        ...prev,
        [domain]: {
          ...prev[domain],
          [type]: {
            success: false,
            message: 'Test failed'
          }
        }
      }));
    } finally {
      setTestingDomain(null);
      setTestType(null);
    }
  };

  const handleStart = () => {
    // Generate config and apply it (overwrite=true)
    generateConfig(true);
  };

  const handleLoadFromInstances = async () => {
    setIsLoadingFromInstances(true);
    try {
      // Generate config without overwriting, load into editor
      const result = await apiService.generateDnsmasqConfig(false);
      const configText = result.config || result.content || '';
      if (configText) {
        setEditedConfig(configText);
      } else {
        console.error('No config content in response:', result);
      }
    } catch (error) {
      console.error('Failed to load config from instances:', error);
    } finally {
      setIsLoadingFromInstances(false);
    }
  };

  const handleShowAdvanced = () => {
    if (!showAdvanced && !config) {
      fetchConfig();
    }
    setShowAdvanced(!showAdvanced);
  };

  const handleEditConfig = () => {
    // Load current config into editor
    if (config?.content) {
      setEditedConfig(config.content);
    } else if (generateData?.config || generateData?.content) {
      setEditedConfig(generateData.config || generateData.content || '');
    }
    setShowEditDialog(true);
  };

  const handleSaveConfig = async () => {
    try {
      await apiService.writeDnsmasqConfig(editedConfig);
      setShowEditDialog(false);
      // Refetch config to show updated content
      fetchConfig();
    } catch (error) {
      console.error('Failed to save config:', error);
    }
  };

  const handleSaveAndRestart = async () => {
    try {
      await apiService.writeDnsmasqConfig(editedConfig);
      await apiService.restartDnsmasq();
      setShowEditDialog(false);
      // Refetch status and config
      fetchConfig();
    } catch (error) {
      console.error('Failed to save and restart:', error);
    }
  };

  const handleSaveDdns = async () => {
    if (!globalConfig) return;

    try {
      // Create updated config with the new dynamicDns value
      const updatedConfig = {
        ...globalConfig,
        cloud: {
          ...globalConfig.cloud,
          router: {
            ...globalConfig.cloud?.router,
            dynamicDns: editedDdns
          }
        }
      };

      await updateConfig(updatedConfig);
      setIsEditingDdns(false);
      setDdnsSaveSuccess(true);
      setTimeout(() => setDdnsSaveSuccess(false), 3000);
    } catch (error) {
      console.error('Failed to save dynamic DNS:', error);
    }
  };

  const handleCancelDdnsEdit = () => {
    setEditedDdns(dynamicDns);
    setIsEditingDdns(false);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">DNS Configuration</h2>
          <p className="text-muted-foreground">
            Configure external access to your Wild Cloud from anywhere
          </p>
        </div>
      </div>

      {/* Global DNS Configuration */}
      <Card>
        <CardHeader>
          <CardTitle>Global DNS (DDNS)</CardTitle>
          <p className="text-sm text-muted-foreground">
            How machines globally find your wild cloud
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* DDNS Value Configuration */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium">Dynamic DNS Hostname</p>
                <p className="text-xs text-muted-foreground">Your router's dynamic DNS address</p>
              </div>
              {!isEditingDdns && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setIsEditingDdns(true)}
                  className="gap-2"
                >
                  <Edit className="h-4 w-4" />
                  Edit
                </Button>
              )}
            </div>

            {isEditingDdns ? (
              <div className="space-y-3">
                <Input
                  value={editedDdns}
                  onChange={(e) => setEditedDdns(e.target.value)}
                  placeholder="example.dyndns.org"
                  className="font-mono"
                />
                <div className="flex gap-2">
                  <Button
                    onClick={handleSaveDdns}
                    disabled={isUpdating}
                    size="sm"
                  >
                    {isUpdating ? (
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    ) : null}
                    Save
                  </Button>
                  <Button
                    onClick={handleCancelDdnsEdit}
                    variant="outline"
                    size="sm"
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <div className="p-3 bg-muted rounded-md border">
                {dynamicDns ? (
                  <code className="text-sm font-mono">{dynamicDns}</code>
                ) : (
                  <p className="text-sm text-muted-foreground">Not configured</p>
                )}
              </div>
            )}

            {ddnsSaveSuccess && (
              <Alert className="border-green-500 bg-green-50 dark:bg-green-950">
                <CheckCircle className="h-4 w-4 text-green-600" />
                <AlertDescription className="text-green-800 dark:text-green-200">
                  Dynamic DNS hostname saved successfully
                </AlertDescription>
              </Alert>
            )}
          </div>

          {/* Setup Instructions */}
          <div className="space-y-4 pt-4 border-t">
            <p className="text-sm font-medium">Setup Steps</p>

            {/* Step 1: Router DDNS */}
            <div className="space-y-2">
              <div className="flex items-start gap-3">
                <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground">
                  1
                </div>
                <div className="flex-1 space-y-2">
                  <p className="text-sm font-medium">Configure Router DDNS</p>
                  <p className="text-xs text-muted-foreground">
                    Enable Dynamic DNS in your router's settings and note the hostname
                  </p>
                  {routerIp && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => window.open(`http://${routerIp}`, '_blank')}
                      className="gap-2"
                    >
                      <ExternalLink className="h-3 w-3" />
                      Open Router ({routerIp})
                    </Button>
                  )}
                </div>
              </div>
            </div>

            {/* Step 2: Update Config */}
            <div className="space-y-2">
              <div className="flex items-start gap-3">
                <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground">
                  2
                </div>
                <div className="flex-1">
                  <p className="text-sm font-medium">Save DDNS Hostname</p>
                  <p className="text-xs text-muted-foreground">
                    Enter your router's dynamic DNS hostname in the field above
                  </p>
                </div>
              </div>
            </div>

            {/* Step 3: Cloudflare CNAME */}
            <div className="space-y-2">
              <div className="flex items-start gap-3">
                <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground">
                  3
                </div>
                <div className="flex-1 space-y-2">
                  <p className="text-sm font-medium">Update Cloudflare DNS</p>
                  <p className="text-xs text-muted-foreground">
                    Create a CNAME record for your Wild Cloud domain pointing to your DDNS hostname
                  </p>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => window.open(`https://dash.cloudflare.com/`, '_blank')}
                    className="gap-2"
                  >
                    <ExternalLink className="h-3 w-3" />
                    Cloudflare Dashboard
                  </Button>
                  {dynamicDns && baseDomain && (
                    <div className="p-3 bg-muted rounded-md border">
                      <p className="text-xs font-mono">
                        CNAME: *.{baseDomain} → {dynamicDns}
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* LAN DNS Service Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="p-2 bg-primary/10 rounded-lg">
                <Globe className="h-6 w-6 text-primary" />
              </div>
              <div>
                <CardTitle>LAN DNS</CardTitle>
                <p className="text-sm text-muted-foreground">
                  Local domain name resolution for your Wild Cloud
                </p>
              </div>
            </div>
            <Badge
              variant={isRunning ? 'default' : 'secondary'}
              className="gap-2"
            >
              {isLoadingStatus ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : isRunning ? (
                <CheckCircle className="h-4 w-4" />
              ) : (
                <XCircle className="h-4 w-4" />
              )}
              {isLoadingStatus ? 'Checking...' : isRunning ? 'Running' : 'Stopped'}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* DNS IP Address - Prominent Display */}
          {dnsIp && (
            <div className="p-6 bg-gradient-to-br from-blue-50 to-cyan-50 dark:from-blue-950/30 dark:to-cyan-950/30 rounded-lg border-2 border-blue-200 dark:border-blue-800">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-2">
                    <Globe className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                    <p className="text-sm font-semibold text-blue-900 dark:text-blue-100">
                      DNS Server IP Address
                    </p>
                  </div>
                  <p className="text-xs text-blue-700 dark:text-blue-300 mb-3">
                    Configure your router to use this IP as the primary DNS server
                  </p>
                  <code className="text-2xl font-mono font-bold bg-white dark:bg-gray-900 px-4 py-2 rounded border border-blue-300 dark:border-blue-700 inline-block">
                    {dnsIp}
                  </code>
                </div>
                <Button
                  variant="outline"
                  size="lg"
                  onClick={handleCopyIp}
                  className="ml-4 border-blue-300 hover:bg-blue-100 dark:border-blue-700 dark:hover:bg-blue-900/20"
                >
                  <Copy className="h-5 w-5 mr-2" />
                  {copiedIp ? 'Copied!' : 'Copy IP'}
                </Button>
              </div>
            </div>
          )}

          {/* Status Details */}
          {status && (
            <div className="space-y-4">
              <div className="p-4 bg-muted rounded-lg border">
                <p className="text-sm text-muted-foreground mb-2">Configured Instances</p>
                {isLoadingInstances ? (
                  <div className="flex items-center justify-center py-4">
                    <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                  </div>
                ) : configuredDomains.length > 0 ? (
                  <div className="space-y-2">
                    {configuredDomains.map(({ domain, instanceName, isCommented }) => (
                      <div key={instanceName} className="space-y-1">
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex items-center gap-2 flex-1">
                            <div className={`h-1.5 w-1.5 rounded-full ${isCommented ? 'bg-muted-foreground' : 'bg-primary'}`} />
                            <code className={`text-sm font-mono ${isCommented ? 'text-muted-foreground' : ''}`}>{domain}</code>
                            {isCommented && (
                              <Badge variant="outline" className="text-xs">No Load Balancer</Badge>
                            )}
                          </div>
                          <div className="flex items-center gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleTestDomain(domain, 'internal')}
                              disabled={testingDomain === domain && testType === 'internal'}
                              className="h-7 px-2"
                              title="Test internal DNS"
                            >
                              {testingDomain === domain && testType === 'internal' ? (
                                <Loader2 className="h-3 w-3 animate-spin" />
                              ) : (
                                <TestTube2 className="h-3 w-3" />
                              )}
                              <span className="ml-1 text-xs">Int</span>
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleTestDomain(domain, 'external')}
                              disabled={testingDomain === domain && testType === 'external'}
                              className="h-7 px-2"
                              title="Test external DNS"
                            >
                              {testingDomain === domain && testType === 'external' ? (
                                <Loader2 className="h-3 w-3 animate-spin" />
                              ) : (
                                <TestTube2 className="h-3 w-3" />
                              )}
                              <span className="ml-1 text-xs">Ext</span>
                            </Button>
                          </div>
                        </div>
                        {testResults[domain] && (
                          <div className="ml-5 space-y-0.5">
                            {testResults[domain].internal && (
                              <div className={`text-xs flex items-center gap-1 ${testResults[domain].internal.success ? 'text-green-600' : 'text-red-600'}`}>
                                <span className="font-medium">Internal:</span>
                                <span className="font-mono">{testResults[domain].internal.message}</span>
                              </div>
                            )}
                            {testResults[domain].external && (
                              <div className={`text-xs flex items-center gap-1 ${testResults[domain].external.success ? 'text-green-600' : 'text-red-600'}`}>
                                <span className="font-medium">External:</span>
                                <span className="font-mono">{testResults[domain].external.message}</span>
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No instances configured. Create a Wild Cloud instance to see it listed here.</p>
                )}
              </div>
              {status.last_restart && status.last_restart !== '0001-01-01T00:00:00Z' && (
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>Last restart:</span>
                  <span className="font-mono">
                    {new Date(status.last_restart).toLocaleString()}
                  </span>
                </div>
              )}
            </div>
          )}

          {/* Action Buttons */}
          <div className="flex gap-2">
            {!isRunning ? (
              <Button
                onClick={handleStart}
                disabled={isGenerating}
                className="gap-2"
                size="lg"
              >
                {isGenerating ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Play className="h-4 w-4" />
                )}
                Start DNS
              </Button>
            ) : (
              <Button
                onClick={() => restart()}
                disabled={isRestarting}
                variant="outline"
                className="gap-2"
              >
                {isRestarting ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RotateCw className="h-4 w-4" />
                )}
                Restart
              </Button>
            )}
          </div>

          {/* Success Messages */}
          {restartData && (
            <Alert className="border-green-500 bg-green-50 dark:bg-green-950">
              <CheckCircle className="h-4 w-4 text-green-600" />
              <AlertDescription className="text-green-800 dark:text-green-200">
                {restartData.message || 'DNS service restarted successfully'}
              </AlertDescription>
            </Alert>
          )}

          {/* Error Messages */}
          {generateError && (
            <Alert variant="error">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                Failed to generate config: {generateError.message}
              </AlertDescription>
            </Alert>
          )}

          {restartError && (
            <Alert variant="error">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                Failed to restart service: {restartError.message}
              </AlertDescription>
            </Alert>
          )}
        </CardContent>
      </Card>

      {/* Advanced Section */}
      <Card>
        <CardHeader>
          <Button
            variant="ghost"
            className="w-full justify-between p-0 h-auto"
            onClick={handleShowAdvanced}
          >
            <CardTitle className="text-lg">Advanced Configuration</CardTitle>
            {showAdvanced ? (
              <ChevronUp className="h-5 w-5" />
            ) : (
              <ChevronDown className="h-5 w-5" />
            )}
          </Button>
        </CardHeader>
        {showAdvanced && (
          <CardContent className="space-y-4">
            {/* Current Config Display */}
            {config && (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium">Current Configuration</p>
                    <code className="text-xs text-muted-foreground font-mono">
                      {config.config_file}
                    </code>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleEditConfig}
                    className="gap-2"
                  >
                    <Edit className="h-4 w-4" />
                    Edit
                  </Button>
                </div>
                <pre className="p-4 bg-muted rounded-md text-xs overflow-auto max-h-96 font-mono border">
                  {config.content}
                </pre>
              </div>
            )}

            {!isGenerating && !generateData && !config && (
              <div className="text-center p-8 text-sm text-muted-foreground">
                <p>Configuration preview will appear here</p>
              </div>
            )}
          </CardContent>
        )}
      </Card>

      {/* Edit Config Dialog */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent className="max-w-4xl max-h-[80vh]">
          <DialogHeader>
            <DialogTitle>Edit DNS Configuration</DialogTitle>
            <DialogDescription>
              Modify the dnsmasq configuration. Changes will take effect after saving and restarting.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4 space-y-3">
            <div className="flex justify-end">
              <Button
                variant="outline"
                size="sm"
                onClick={handleLoadFromInstances}
                disabled={isLoadingFromInstances}
                className="gap-2"
              >
                {isLoadingFromInstances ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Settings className="h-4 w-4" />
                )}
                Load from Instances
              </Button>
            </div>
            <Textarea
              value={editedConfig}
              onChange={(e) => setEditedConfig(e.target.value)}
              className="font-mono text-xs min-h-[400px]"
              placeholder="# dnsmasq configuration"
            />
          </div>
          <DialogFooter className="gap-2">
            <Button
              variant="outline"
              onClick={() => setShowEditDialog(false)}
            >
              Cancel
            </Button>
            <Button
              variant="outline"
              onClick={handleSaveConfig}
            >
              Save
            </Button>
            <Button
              onClick={handleSaveAndRestart}
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
