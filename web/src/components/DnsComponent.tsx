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
  Loader2,
  Copy,
  ChevronDown,
  ChevronUp,
  Edit,
  ExternalLink
} from 'lucide-react';
import { useDnsmasq } from '../hooks/useDnsmasq';
import { useConfig } from '../hooks';
import { apiService } from '../services/api-legacy';
import { usePageHelp } from '../hooks/usePageHelp';

export function DnsComponent() {
  const { config: globalConfig, updateConfig, isUpdating } = useConfig();
  const dnsIp = globalConfig?.cloud?.dnsmasq?.ip;
  const routerIp = globalConfig?.cloud?.router?.ip;
  const dynamicDns = globalConfig?.cloud?.router?.dynamicDns || '';

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
  const [isEditingDdns, setIsEditingDdns] = useState(false);
  const [editedDdns, setEditedDdns] = useState(dynamicDns);
  const [ddnsSaveSuccess, setDdnsSaveSuccess] = useState(false);

  const isRunning = status?.status === 'active';


  // Update editedDdns when dynamicDns changes
  useEffect(() => {
    setEditedDdns(dynamicDns);
  }, [dynamicDns]);


  const handleCopyIp = () => {
    if (dnsIp) {
      navigator.clipboard.writeText(dnsIp);
      setCopiedIp(true);
      setTimeout(() => setCopiedIp(false), 2000);
    }
  };

  const handleStart = () => {
    // Generate config and apply it (overwrite=true)
    generateConfig(true);
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
            Configure external domain name resolution to your Wild Cloud from anywhere
          </p>
        </div>
      </div>

      {/* Global DNS Configuration */}
      <Card>
        <CardHeader>
          <CardTitle>Global Name Resolution</CardTitle>
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
          {status && status.last_restart && status.last_restart !== '0001-01-01T00:00:00Z' && (
            <div className="flex items-center justify-between text-xs text-muted-foreground p-3 bg-muted rounded-lg">
              <span>Last restart:</span>
              <span className="font-mono">
                {new Date(status.last_restart).toLocaleString()}
              </span>
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
          <div className="py-4">
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
