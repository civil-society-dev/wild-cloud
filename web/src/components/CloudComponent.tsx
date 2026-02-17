import { useState, useEffect } from "react";
import { Card } from "./ui/card";
import { Button } from "./ui/button";
import { Checkbox } from "./ui/checkbox";
import { Cloud, HelpCircle, Edit2, Check, X, Loader2, AlertCircle, Mail, TestTube2, CheckCircle, XCircle, Globe, ExternalLink } from "lucide-react";
import { Input, Label } from "./ui";
import { useInstanceConfig, useConfig } from "../hooks";
import { useParams } from "react-router";
import { Alert, AlertDescription } from "./ui/alert";

interface SmtpConfig {
  host: string;
  port: string;
  user: string;
  from: string;
  tls: string;
  startTls: string;
}

interface CloudConfig {
  domain: string;
  internalDomain: string;
  smtp: SmtpConfig;
}

export function CloudComponent() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const { config: fullConfig, isLoading, error, updateConfig, isUpdating } = useInstanceConfig(instanceId);
  const { config: globalConfig } = useConfig();

  // Extract cloud config from full config (canonical nested structure)
  const config = fullConfig?.cloud as CloudConfig | undefined;
  const clusterLbIp = fullConfig?.cluster?.loadBalancerIp as string | undefined;
  // Get dynamicDns from global config, not instance config
  const dynamicDns = globalConfig?.cloud?.router?.dynamicDns as string | undefined;

  const [editingDomains, setEditingDomains] = useState(false);
  const [editingSmtp, setEditingSmtp] = useState(false);
  const [formValues, setFormValues] = useState<CloudConfig | null>(null);
  const [testingDomain, setTestingDomain] = useState<string | null>(null);
  const [testType, setTestType] = useState<'internal' | 'external' | null>(null);
  const [testResults, setTestResults] = useState<Record<string, {
    internal?: { success: boolean; message: string };
    external?: { success: boolean; message: string };
  }>>({});

  // Sync form values when config loads or instance changes
  useEffect(() => {
    if (config) {
      setFormValues(config as CloudConfig);
    }
  }, [config, instanceId]);

  const handleDomainsEdit = () => {
    if (config) {
      setFormValues(config as CloudConfig);
      setEditingDomains(true);
    }
  };

  const handleDomainsSave = async () => {
    if (!formValues || !fullConfig) return;

    try {
      // Update cloud section with new domain values
      const existingCloud = (fullConfig.cloud ?? {}) as Record<string, unknown>;
      await updateConfig({
        ...fullConfig,
        cloud: {
          ...existingCloud,
          domain: formValues.domain,
          internalDomain: formValues.internalDomain,
        },
      });
      setEditingDomains(false);
    } catch (err) {
      console.error('Failed to save domains:', err);
    }
  };

  const handleDomainsCancel = () => {
    setFormValues(config as CloudConfig);
    setEditingDomains(false);
  };

  const handleSmtpEdit = () => {
    if (config) {
      setFormValues(config as CloudConfig);
      setEditingSmtp(true);
    }
  };

  const handleSmtpSave = async () => {
    if (!formValues || !fullConfig) return;

    try {
      const existingCloud = (fullConfig.cloud ?? {}) as Record<string, unknown>;
      await updateConfig({
        ...fullConfig,
        cloud: {
          ...existingCloud,
          smtp: formValues.smtp,
        },
      });
      setEditingSmtp(false);
    } catch (err) {
      console.error('Failed to save SMTP config:', err);
    }
  };

  const handleSmtpCancel = () => {
    setFormValues(config as CloudConfig);
    setEditingSmtp(false);
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

          // For external, we expect it to resolve to the router's public IP (via DDNS)
          setTestResults(prev => ({
            ...prev,
            [domain]: {
              ...prev[domain],
              external: {
                success: true,
                message: `Resolves to ${resolvedIp} (router public IP via DDNS)`
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
                message: 'No public DNS record'
              }
            }
          }));
        }
      } else {
        // Test internal resolution via Wild Central API
        const response = await fetch(`/api/v1/network/resolve?domain=${domain}`);
        const data = await response.json();

        if (data.success && data.ip) {
          // Check if it resolves to the expected cluster LB IP
          const isCorrectIp = clusterLbIp && data.ip === clusterLbIp;
          setTestResults(prev => ({
            ...prev,
            [domain]: {
              ...prev[domain],
              internal: {
                success: isCorrectIp || !clusterLbIp, // Success if matches or if no LB IP configured yet
                message: clusterLbIp
                  ? (isCorrectIp
                    ? `Resolves to ${data.ip} (cluster LB IP)`
                    : `Resolves to ${data.ip} (expected ${clusterLbIp})`)
                  : `Resolves to ${data.ip}`
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
                message: data.error || 'Not resolved internally'
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

  const updateFormValue = (key: keyof CloudConfig, value: string) => {
    if (!formValues) return;
    setFormValues(prev => prev ? { ...prev, [key]: value } : prev);
  };

  const updateNestedFormValue = <T extends keyof CloudConfig>(
    section: T,
    key: keyof CloudConfig[T],
    value: string
  ) => {
    if (!formValues) return;
    setFormValues(prev => {
      if (!prev) return prev;
      const currentSection = prev[section];
      if (typeof currentSection === 'object' && currentSection !== null) {
        return {
          ...prev,
          [section]: {
            ...currentSection,
            [key]: value,
          },
        };
      }
      return prev;
    });
  };

  // Show message if no instance is selected
  if (!instanceId) {
    return (
      <Card className="p-8 text-center">
        <Cloud className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <h3 className="text-lg font-medium mb-2">No Instance Selected</h3>
        <p className="text-muted-foreground mb-4">
          Please select or create an instance to manage cloud configuration.
        </p>
      </Card>
    );
  }

  // Show loading state
  if (isLoading || !formValues) {
    return (
      <Card className="p-8 text-center">
        <Loader2 className="h-12 w-12 text-primary mx-auto mb-4 animate-spin" />
        <p className="text-muted-foreground">Loading cloud configuration...</p>
      </Card>
    );
  }

  // Show error state
  if (error) {
    return (
      <Card className="p-8 text-center">
        <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
        <h3 className="text-lg font-medium mb-2">Error Loading Configuration</h3>
        <p className="text-muted-foreground mb-4">
          {(error as Error)?.message || 'An error occurred'}
        </p>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Cloud Configuration</h2>
          <p className="text-muted-foreground">
            Configure domains and infrastructure settings
          </p>
        </div>
      </div>

      {/* Domains Section */}
      <Card className="p-4 border-l-4 border-l-blue-500">
            <div className="flex items-center justify-between mb-3">
              <div>
                <h3 className="font-medium">Domain Configuration</h3>
                <p className="text-sm text-muted-foreground">
                  Public and internal domain settings
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="sm">
                  <HelpCircle className="h-4 w-4" />
                </Button>
                {!editingDomains && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleDomainsEdit}
                    disabled={isUpdating}
                  >
                    <Edit2 className="h-4 w-4 mr-1" />
                    Edit
                  </Button>
                )}
              </div>
            </div>

            {/* Note about DNS regeneration */}
            {!editingDomains && (
              <Alert className="mb-3 border-blue-200 bg-blue-50 dark:bg-blue-950/30">
                <AlertCircle className="h-4 w-4 text-blue-600" />
                <AlertDescription className="text-blue-700 dark:text-blue-300 text-xs">
                  <span className="font-medium">Note:</span> After changing domains, regenerate the DNS configuration on the DNS page to apply changes.
                </AlertDescription>
              </Alert>
            )}

            {editingDomains ? (
              <div className="space-y-3">
                <div>
                  <Label htmlFor="domain-edit">Public Domain</Label>
                  <Input
                    id="domain-edit"
                    value={formValues.domain}
                    onChange={(e) => updateFormValue('domain', e.target.value)}
                    placeholder="example.com"
                    className="mt-1"
                  />
                </div>
                <div>
                  <Label htmlFor="internal-domain-edit">Internal Domain</Label>
                  <Input
                    id="internal-domain-edit"
                    value={formValues.internalDomain}
                    onChange={(e) => updateFormValue('internalDomain', e.target.value)}
                    placeholder="internal.example.com"
                    className="mt-1"
                  />
                </div>
                <div className="flex gap-2">
                  <Button size="sm" onClick={handleDomainsSave} disabled={isUpdating}>
                    {isUpdating ? (
                      <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                    ) : (
                      <Check className="h-4 w-4 mr-1" />
                    )}
                    Save
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleDomainsCancel}
                    disabled={isUpdating}
                  >
                    <X className="h-4 w-4 mr-1" />
                    Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <div className="space-y-3">
                <div>
                  <Label>Public Domain</Label>
                  <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                    {formValues.domain}
                  </div>

                  {/* Setup Instructions for Public Domain */}
                  {dynamicDns && (
                    <div className="mt-3 p-3 bg-blue-50 dark:bg-blue-950/30 rounded-md border border-blue-200 dark:border-blue-800">
                      <div className="text-xs space-y-2">
                        <p className="font-medium text-blue-900 dark:text-blue-100">
                          To enable external access:
                        </p>
                        <div className="space-y-1 text-blue-700 dark:text-blue-300">
                          <p>
                            1. Create a CNAME record in your DNS provider (e.g., Cloudflare)
                          </p>
                          <p className="pl-3 font-mono bg-white/50 dark:bg-gray-900/50 rounded px-2 py-1 inline-block">
                            *.{formValues.domain} → {dynamicDns}
                          </p>
                          <div className="pt-1">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => window.open('https://dash.cloudflare.com/', '_blank')}
                              className="h-6 text-xs gap-1 border-blue-300 hover:bg-blue-100 dark:border-blue-700 dark:hover:bg-blue-900/20"
                            >
                              <ExternalLink className="h-3 w-3" />
                              Open Cloudflare
                            </Button>
                          </div>
                        </div>
                      </div>
                    </div>
                  )}

                  {!dynamicDns && (
                    <Alert className="mt-3 border-amber-500 bg-amber-50 dark:bg-amber-950">
                      <AlertCircle className="h-4 w-4 text-amber-600" />
                      <AlertDescription className="text-amber-800 dark:text-amber-200 text-xs">
                        <span className="font-medium">Setup Required:</span> Configure your router's Dynamic DNS first (see DNS page), then create a CNAME record pointing to it.
                      </AlertDescription>
                    </Alert>
                  )}

                  {/* DNS Status Check for Public Domain */}
                  <div className="mt-3 space-y-2">
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleTestDomain(formValues.domain, 'external')}
                        disabled={testingDomain === formValues.domain && testType === 'external'}
                      >
                        {testingDomain === formValues.domain && testType === 'external' ? (
                          <Loader2 className="h-3 w-3 animate-spin mr-1" />
                        ) : (
                          <Globe className="h-3 w-3 mr-1" />
                        )}
                        Test External Resolution
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleTestDomain(formValues.domain, 'internal')}
                        disabled={testingDomain === formValues.domain && testType === 'internal'}
                      >
                        {testingDomain === formValues.domain && testType === 'internal' ? (
                          <Loader2 className="h-3 w-3 animate-spin mr-1" />
                        ) : (
                          <TestTube2 className="h-3 w-3 mr-1" />
                        )}
                        Test LAN Resolution
                      </Button>
                    </div>
                    {testResults[formValues.domain]?.external && (
                      <Alert className={`${
                        testResults[formValues.domain].external.success
                          ? 'border-green-500 bg-green-50 dark:bg-green-950'
                          : 'border-amber-500 bg-amber-50 dark:bg-amber-950'
                      }`}>
                        {testResults[formValues.domain].external.success ? (
                          <CheckCircle className="h-4 w-4 text-green-600" />
                        ) : (
                          <AlertCircle className="h-4 w-4 text-amber-600" />
                        )}
                        <AlertDescription className={
                          testResults[formValues.domain].external.success
                            ? 'text-green-800 dark:text-green-200'
                            : 'text-amber-800 dark:text-amber-200'
                        }>
                          <span className="font-medium">External: </span>
                          {testResults[formValues.domain].external.message}
                          {!testResults[formValues.domain].external.success && (
                            <div className="text-xs mt-1">
                              Set up a CNAME record pointing to your dynamic DNS hostname to enable external access
                            </div>
                          )}
                        </AlertDescription>
                      </Alert>
                    )}
                    {testResults[formValues.domain]?.internal && (
                      <Alert className={`${
                        testResults[formValues.domain].internal.success
                          ? 'border-green-500 bg-green-50 dark:bg-green-950'
                          : 'border-amber-500 bg-amber-50 dark:bg-amber-950'
                      }`}>
                        {testResults[formValues.domain].internal.success ? (
                          <CheckCircle className="h-4 w-4 text-green-600" />
                        ) : (
                          <AlertCircle className="h-4 w-4 text-amber-600" />
                        )}
                        <AlertDescription className={
                          testResults[formValues.domain].internal.success
                            ? 'text-green-800 dark:text-green-200'
                            : 'text-amber-800 dark:text-amber-200'
                        }>
                          <span className="font-medium">LAN: </span>
                          {testResults[formValues.domain].internal.message}
                          {!testResults[formValues.domain].internal.success && (
                            <div className="text-xs mt-1">
                              Ensure DNS service is configured and cluster has a load balancer IP
                            </div>
                          )}
                        </AlertDescription>
                      </Alert>
                    )}
                  </div>
                </div>
                <div>
                  <Label>Internal Domain</Label>
                  <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                    {formValues.internalDomain}
                  </div>

                  {/* Info about Internal Domain */}
                  <div className="mt-3 p-3 bg-green-50 dark:bg-green-950/30 rounded-md border border-green-200 dark:border-green-800">
                    <div className="text-xs space-y-1 text-green-700 dark:text-green-300">
                      <p className="font-medium text-green-900 dark:text-green-100">
                        LAN-only domain:
                      </p>
                      <p>• Only accessible from your local network</p>
                      <p>• Requires DNS service to be running</p>
                      <p>• Configure router to use Wild Central as DNS server</p>
                      {clusterLbIp && (
                        <p className="font-mono text-[11px] mt-1">Resolves to: {clusterLbIp}</p>
                      )}
                    </div>
                  </div>

                  {/* DNS Status Check for Internal Domain */}
                  <div className="mt-3">
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleTestDomain(formValues.internalDomain, 'internal')}
                        disabled={testingDomain === formValues.internalDomain && testType === 'internal'}
                      >
                        {testingDomain === formValues.internalDomain && testType === 'internal' ? (
                          <Loader2 className="h-3 w-3 animate-spin mr-1" />
                        ) : (
                          <TestTube2 className="h-3 w-3 mr-1" />
                        )}
                        Test LAN Resolution
                      </Button>
                    </div>
                    {testResults[formValues.internalDomain]?.internal && (
                      <Alert className={`mt-2 ${
                        testResults[formValues.internalDomain].internal.success
                          ? 'border-green-500 bg-green-50 dark:bg-green-950'
                          : 'border-amber-500 bg-amber-50 dark:bg-amber-950'
                      }`}>
                        {testResults[formValues.internalDomain].internal.success ? (
                          <CheckCircle className="h-4 w-4 text-green-600" />
                        ) : (
                          <AlertCircle className="h-4 w-4 text-amber-600" />
                        )}
                        <AlertDescription className={
                          testResults[formValues.internalDomain].internal.success
                            ? 'text-green-800 dark:text-green-200'
                            : 'text-amber-800 dark:text-amber-200'
                        }>
                          <span className="font-medium">LAN: </span>
                          {testResults[formValues.internalDomain].internal.message}
                          {!testResults[formValues.internalDomain].internal.success && (
                            <div className="text-xs mt-1">
                              Ensure the DNS service is running and configured properly. Internal domains only resolve on your LAN.
                            </div>
                          )}
                        </AlertDescription>
                      </Alert>
                    )}
                  </div>
                </div>
              </div>
            )}
      </Card>

      {/* SMTP Configuration Section */}
      <Card className="p-4 border-l-4 border-l-green-500">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-green-500/10 rounded-lg">
              <Mail className="h-5 w-5 text-green-500" />
            </div>
            <div>
              <h3 className="font-medium">Email Configuration</h3>
              <p className="text-sm text-muted-foreground">
                SMTP settings for application email delivery
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm">
              <HelpCircle className="h-4 w-4" />
            </Button>
            {!editingSmtp && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleSmtpEdit}
                disabled={isUpdating}
              >
                <Edit2 className="h-4 w-4 mr-1" />
                Edit
              </Button>
            )}
          </div>
        </div>

        {editingSmtp ? (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label htmlFor="smtp-host-edit">SMTP Host</Label>
                <Input
                  id="smtp-host-edit"
                  value={formValues.smtp?.host || ''}
                  onChange={(e) => updateNestedFormValue('smtp', 'host', e.target.value)}
                  placeholder="mail.example.com"
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="smtp-port-edit">SMTP Port</Label>
                <Input
                  id="smtp-port-edit"
                  value={formValues.smtp?.port || ''}
                  onChange={(e) => updateNestedFormValue('smtp', 'port', e.target.value)}
                  placeholder="587"
                  className="mt-1"
                />
              </div>
            </div>
            <div>
              <Label htmlFor="smtp-user-edit">SMTP Username</Label>
              <Input
                id="smtp-user-edit"
                value={formValues.smtp?.user || ''}
                onChange={(e) => updateNestedFormValue('smtp', 'user', e.target.value)}
                placeholder="username"
                className="mt-1"
              />
            </div>
            <div>
              <Label htmlFor="smtp-from-edit">From Address</Label>
              <Input
                id="smtp-from-edit"
                value={formValues.smtp?.from || ''}
                onChange={(e) => updateNestedFormValue('smtp', 'from', e.target.value)}
                placeholder="no-reply@example.com"
                className="mt-1"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="smtp-tls-edit"
                  checked={formValues.smtp?.tls === 'true'}
                  onCheckedChange={(checked) => updateNestedFormValue('smtp', 'tls', checked ? 'true' : 'false')}
                />
                <Label htmlFor="smtp-tls-edit" className="cursor-pointer">
                  Enable TLS
                </Label>
              </div>
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="smtp-starttls-edit"
                  checked={formValues.smtp?.startTls === 'true'}
                  onCheckedChange={(checked) => updateNestedFormValue('smtp', 'startTls', checked ? 'true' : 'false')}
                />
                <Label htmlFor="smtp-starttls-edit" className="cursor-pointer">
                  Enable STARTTLS
                </Label>
              </div>
            </div>
            <div className="flex gap-2">
              <Button size="sm" onClick={handleSmtpSave} disabled={isUpdating}>
                {isUpdating ? (
                  <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                ) : (
                  <Check className="h-4 w-4 mr-1" />
                )}
                Save
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleSmtpCancel}
                disabled={isUpdating}
              >
                <X className="h-4 w-4 mr-1" />
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label>SMTP Host</Label>
                <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                  {formValues.smtp?.host || 'Not configured'}
                </div>
              </div>
              <div>
                <Label>SMTP Port</Label>
                <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                  {formValues.smtp?.port || 'Not configured'}
                </div>
              </div>
            </div>
            <div>
              <Label>SMTP Username</Label>
              <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                {formValues.smtp?.user || 'Not configured'}
              </div>
            </div>
            <div>
              <Label>From Address</Label>
              <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                {formValues.smtp?.from || 'Not configured'}
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="flex items-center space-x-2">
                <Checkbox
                  checked={formValues.smtp?.tls === 'true'}
                  disabled
                />
                <Label>Enable TLS</Label>
              </div>
              <div className="flex items-center space-x-2">
                <Checkbox
                  checked={formValues.smtp?.startTls === 'true'}
                  disabled
                />
                <Label>Enable STARTTLS</Label>
              </div>
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}
