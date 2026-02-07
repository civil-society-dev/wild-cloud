import { useState, useEffect } from "react";
import { Card } from "./ui/card";
import { Button } from "./ui/button";
import { Cloud, HelpCircle, Edit2, Check, X, Loader2, AlertCircle } from "lucide-react";
import { Input, Label } from "./ui";
import { useInstanceConfig } from "../hooks";
import { useParams } from "react-router";

interface CloudConfig {
  domain: string;
  internalDomain: string;
  dhcpRange: string;
}

export function CloudComponent() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const { config: fullConfig, isLoading, error, updateConfig, isUpdating } = useInstanceConfig(instanceId);

  console.log('CloudComponent:', { instanceId, fullConfig, isLoading, error });

  // Extract cloud config from full config (canonical nested structure)
  const config = fullConfig?.cloud as CloudConfig | undefined;

  const [editingDomains, setEditingDomains] = useState(false);
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

  const updateFormValue = (key: keyof CloudConfig, value: string) => {
    if (!formValues) return;
    setFormValues(prev => prev ? { ...prev, [key]: value } : prev);
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
            Configure top-level cloud settings and domains
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



        </div>
      </Card>
    </div>
  );
}
