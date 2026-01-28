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

interface ClusterConfig {
  hostnamePrefix?: string;
  nodes: {
    control: {
      vip: string;
    };
    talos: {
      version: string;
    };
  };
}

export function CloudComponent() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const { config: fullConfig, isLoading, error, updateConfig, isUpdating } = useInstanceConfig(instanceId);

  console.log('CloudComponent:', { instanceId, fullConfig, isLoading, error });

  // Extract cloud and cluster config from full config (canonical nested structure)
  const config = fullConfig?.cloud as CloudConfig | undefined;
  const clusterConfig = fullConfig?.cluster as ClusterConfig | undefined;

  const [editingDomains, setEditingDomains] = useState(false);
  const [editingNetwork, setEditingNetwork] = useState(false);
  const [editingCluster, setEditingCluster] = useState(false);
  const [formValues, setFormValues] = useState<CloudConfig | null>(null);
  const [clusterFormValues, setClusterFormValues] = useState<ClusterConfig | null>(null);

  // Sync form values when config loads or instance changes
  useEffect(() => {
    if (config) {
      setFormValues(config as CloudConfig);
    }
    if (clusterConfig) {
      setClusterFormValues(clusterConfig as ClusterConfig);
    }
  }, [config, clusterConfig, instanceId]);

  const handleDomainsEdit = () => {
    if (config) {
      setFormValues(config as CloudConfig);
      setEditingDomains(true);
    }
  };

  const handleNetworkEdit = () => {
    if (config) {
      setFormValues(config as CloudConfig);
      setEditingNetwork(true);
    }
  };

  const handleDomainsSave = async () => {
    if (!formValues || !fullConfig) return;

    try {
      // Update cloud section with new domain values
      await updateConfig({
        ...fullConfig,
        cloud: {
          ...fullConfig.cloud,
          domain: formValues.domain,
          internalDomain: formValues.internalDomain,
        },
      });
      setEditingDomains(false);
    } catch (err) {
      console.error('Failed to save domains:', err);
    }
  };

  const handleNetworkSave = async () => {
    if (!formValues || !fullConfig) return;

    try {
      // Update cloud section with new network values
      await updateConfig({
        ...fullConfig,
        cloud: {
          ...fullConfig.cloud,
          dhcpRange: formValues.dhcpRange,
        },
      });
      setEditingNetwork(false);
    } catch (err) {
      console.error('Failed to save network settings:', err);
    }
  };

  const handleDomainsCancel = () => {
    setFormValues(config as CloudConfig);
    setEditingDomains(false);
  };

  const handleNetworkCancel = () => {
    setFormValues(config as CloudConfig);
    setEditingNetwork(false);
  };

  const handleClusterEdit = () => {
    if (clusterConfig) {
      setClusterFormValues(clusterConfig as ClusterConfig);
      setEditingCluster(true);
    }
  };

  const handleClusterSave = async () => {
    if (!clusterFormValues || !fullConfig) return;

    try {
      // Update only the cluster section, preserving other config sections
      await updateConfig({
        ...fullConfig,
        cluster: clusterFormValues,
      });
      setEditingCluster(false);
    } catch (err) {
      console.error('Failed to save cluster settings:', err);
    }
  };

  const handleClusterCancel = () => {
    setClusterFormValues(clusterConfig as ClusterConfig);
    setEditingCluster(false);
  };

  const updateFormValue = (path: string, value: string) => {
    if (!formValues) return;

    setFormValues(prev => {
      if (!prev) return prev;

      // Handle nested paths like "dns.ip"
      const keys = path.split('.');
      if (keys.length === 1) {
        return { ...prev, [keys[0]]: value };
      }

      // Handle nested object updates
      const [parentKey, childKey] = keys;
      return {
        ...prev,
        [parentKey]: {
          ...(prev[parentKey as keyof CloudConfig] as Record<string, unknown>),
          [childKey]: value,
        },
      };
    });
  };

  const updateClusterFormValue = (path: string, value: string) => {
    if (!clusterFormValues) return;

    setClusterFormValues(prev => {
      if (!prev) return prev;

      // Handle nested paths like "nodes.talos.version" or "nodes.control.vip"
      const keys = path.split('.');
      if (keys.length === 1) {
        return { ...prev, [keys[0]]: value };
      }

      if (keys.length === 3 && keys[0] === 'nodes') {
        if (keys[1] === 'talos') {
          return {
            ...prev,
            nodes: {
              ...prev.nodes,
              talos: {
                ...prev.nodes.talos,
                [keys[2]]: value,
              },
            },
          };
        }
        if (keys[1] === 'control') {
          return {
            ...prev,
            nodes: {
              ...prev.nodes,
              control: {
                ...prev.nodes.control,
                [keys[2]]: value,
              },
            },
          };
        }
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
      <Card className="p-6">
        <div className="flex items-center gap-4 mb-6">
          <div className="p-2 bg-primary/10 rounded-lg">
            <Cloud className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold">Cloud Configuration</h2>
            <p className="text-muted-foreground">
              Configure top-level cloud settings and domains
            </p>
          </div>
        </div>

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

          {/* Network Configuration Section */}
          <Card className="p-4 border-l-4 border-l-green-500">
            <div className="flex items-center justify-between mb-3">
              <div>
                <h3 className="font-medium">Network Configuration</h3>
                <p className="text-sm text-muted-foreground">
                  Network settings and DHCP configuration
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="sm">
                  <HelpCircle className="h-4 w-4" />
                </Button>
                {!editingNetwork && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleNetworkEdit}
                    disabled={isUpdating}
                  >
                    <Edit2 className="h-4 w-4 mr-1" />
                    Edit
                  </Button>
                )}
              </div>
            </div>

            {editingNetwork ? (
              <div className="space-y-3">
                <div>
                  <Label htmlFor="dhcp-range-edit">DHCP Range</Label>
                  <Input
                    id="dhcp-range-edit"
                    value={formValues.dhcpRange}
                    onChange={(e) => updateFormValue('dhcpRange', e.target.value)}
                    placeholder="192.168.1.100,192.168.1.200"
                    className="mt-1"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Format: start_ip,end_ip
                  </p>
                </div>
                <div className="flex gap-2">
                  <Button size="sm" onClick={handleNetworkSave} disabled={isUpdating}>
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
                    onClick={handleNetworkCancel}
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
                  <Label>DHCP Range</Label>
                  <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                    {formValues.dhcpRange}
                  </div>
                </div>
              </div>
            )}
          </Card>

          {/* Cluster Configuration Section */}
          {clusterFormValues && (
            <Card className="p-4 border-l-4 border-l-purple-500">
              <div className="flex items-center justify-between mb-3">
                <div>
                  <h3 className="font-medium">Cluster Configuration</h3>
                  <p className="text-sm text-muted-foreground">
                    Kubernetes cluster and node settings
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="ghost" size="sm">
                    <HelpCircle className="h-4 w-4" />
                  </Button>
                  {!editingCluster && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleClusterEdit}
                      disabled={isUpdating}
                    >
                      <Edit2 className="h-4 w-4 mr-1" />
                      Edit
                    </Button>
                  )}
                </div>
              </div>

              {editingCluster ? (
                <div className="space-y-3">
                  <div>
                    <Label htmlFor="endpoint-ip-edit">Cluster Endpoint IP</Label>
                    <Input
                      id="endpoint-ip-edit"
                      value={clusterFormValues.nodes.control.vip}
                      onChange={(e) => updateClusterFormValue('nodes.control.vip', e.target.value)}
                      placeholder="192.168.1.60"
                      className="mt-1"
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                      Virtual IP for the Kubernetes API endpoint
                    </p>
                  </div>
                  <div>
                    <Label htmlFor="hostname-prefix-edit">Hostname Prefix (Optional)</Label>
                    <Input
                      id="hostname-prefix-edit"
                      value={clusterFormValues.hostnamePrefix || ''}
                      onChange={(e) => updateClusterFormValue('hostnamePrefix', e.target.value)}
                      placeholder="mycluster-"
                      className="mt-1"
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                      Prefix for auto-generated node hostnames (e.g., "mycluster-control-1")
                    </p>
                  </div>
                  <div>
                    <Label htmlFor="talos-version-edit">Talos Version</Label>
                    <Input
                      id="talos-version-edit"
                      value={clusterFormValues.nodes.talos.version}
                      onChange={(e) => updateClusterFormValue('nodes.talos.version', e.target.value)}
                      placeholder="v1.8.0"
                      className="mt-1"
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                      Talos Linux version for cluster nodes
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" onClick={handleClusterSave} disabled={isUpdating}>
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
                      onClick={handleClusterCancel}
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
                    <Label>Cluster Endpoint IP</Label>
                    <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                      {clusterFormValues.nodes.control.vip}
                    </div>
                  </div>
                  <div>
                    <Label>Hostname Prefix</Label>
                    <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                      {clusterFormValues.hostnamePrefix || '(none)'}
                    </div>
                  </div>
                  <div>
                    <Label>Talos Version</Label>
                    <div className="mt-1 p-2 bg-muted rounded-md font-mono text-sm">
                      {clusterFormValues.nodes.talos.version}
                    </div>
                  </div>
                </div>
              )}
            </Card>
          )}
        </div>
      </Card>
    </div>
  );
}
