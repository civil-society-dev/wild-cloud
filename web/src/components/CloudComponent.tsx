import { useState, useEffect } from "react";
import { Card } from "./ui/card";
import { Button } from "./ui/button";
import { Checkbox } from "./ui/checkbox";
import { Cloud, HelpCircle, Edit2, Check, X, Loader2, AlertCircle, Mail } from "lucide-react";
import { Input, Label } from "./ui";
import { useInstanceConfig } from "../hooks";
import { useParams } from "react-router";

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

  // Extract cloud config from full config (canonical nested structure)
  const config = fullConfig?.cloud as CloudConfig | undefined;

  const [editingDomains, setEditingDomains] = useState(false);
  const [editingSmtp, setEditingSmtp] = useState(false);
  const [formValues, setFormValues] = useState<CloudConfig | null>(null);

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

      <Card className="p-6">
        <div className="space-y-6">
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
                </div>
                <div>
                  <Label>Internal Domain</Label>
                  <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                    {formValues.internalDomain}
                  </div>
                </div>
              </div>
            )}
          </Card>

          {/* SMTP Configuration Section */}
          <div className="space-y-4">
            <div>
              <h3 className="text-lg font-medium">Email Configuration</h3>
              <p className="text-sm text-muted-foreground">
                Configure SMTP settings for application email delivery
              </p>
            </div>

            {/* SMTP Configuration */}
            <Card className="p-4 border-l-4 border-l-green-500">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-green-500/10 rounded-lg">
                    <Mail className="h-5 w-5 text-green-500" />
                  </div>
                  <div>
                    <h3 className="font-medium">SMTP Configuration</h3>
                    <p className="text-sm text-muted-foreground">
                      Email delivery settings for applications
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

        </div>
      </Card>
    </div>
  );
}
