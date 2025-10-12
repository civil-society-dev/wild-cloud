import { useState } from 'react';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Cpu, HardDrive, Network, Monitor, CheckCircle, AlertCircle, BookOpen, ExternalLink, Loader2 } from 'lucide-react';
import { useInstanceContext } from '../hooks/useInstanceContext';
import { useNodes, useDiscoveryStatus } from '../hooks/useNodes';

export function ClusterNodesComponent() {
  const { currentInstance } = useInstanceContext();
  const {
    nodes,
    isLoading,
    error,
    addNode,
    isAdding,
    deleteNode,
    isDeleting,
    discover,
    isDiscovering,
    detect,
    isDetecting
  } = useNodes(currentInstance);

  const {
    data: discoveryStatus
  } = useDiscoveryStatus(currentInstance);

  const [subnet, setSubnet] = useState('192.168.1.0/24');

  const getStatusIcon = (status?: string) => {
    switch (status) {
      case 'ready':
      case 'healthy':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'error':
        return <AlertCircle className="h-5 w-5 text-red-500" />;
      case 'connecting':
      case 'provisioning':
        return <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />;
      default:
        return <Monitor className="h-5 w-5 text-muted-foreground" />;
    }
  };

  const getStatusBadge = (status?: string) => {
    const variants: Record<string, 'secondary' | 'default' | 'success' | 'destructive'> = {
      pending: 'secondary',
      connecting: 'default',
      provisioning: 'default',
      ready: 'success',
      healthy: 'success',
      error: 'destructive',
    };

    const labels: Record<string, string> = {
      pending: 'Pending',
      connecting: 'Connecting',
      provisioning: 'Provisioning',
      ready: 'Ready',
      healthy: 'Healthy',
      error: 'Error',
    };

    return (
      <Badge variant={variants[status || 'pending']}>
        {labels[status || 'pending'] || status}
      </Badge>
    );
  };

  const getRoleIcon = (role: string) => {
    return role === 'controlplane' ? (
      <Cpu className="h-4 w-4" />
    ) : (
      <HardDrive className="h-4 w-4" />
    );
  };

  const handleAddNode = (ip: string, hostname: string, role: 'controlplane' | 'worker') => {
    if (!currentInstance) return;
    addNode({ target_ip: ip, hostname, role, disk: '/dev/sda' });
  };

  const handleDeleteNode = (hostname: string) => {
    if (!currentInstance) return;
    if (confirm(`Are you sure you want to remove node ${hostname}?`)) {
      deleteNode(hostname);
    }
  };

  const handleDiscover = () => {
    if (!currentInstance) return;
    discover(subnet);
  };

  const handleDetect = () => {
    if (!currentInstance) return;
    detect();
  };

  // Derive status from backend state flags for each node
  const assignedNodes = nodes.map(node => {
    let status = 'pending';
    if (node.maintenance) {
      status = 'provisioning';
    } else if (node.configured && !node.applied) {
      status = 'connecting';
    } else if (node.applied) {
      status = 'ready';
    }
    return { ...node, status };
  });

  // Extract IPs from discovered nodes
  const discoveredIps = discoveryStatus?.nodes_found?.map(n => n.ip) || [];

  // Show message if no instance is selected
  if (!currentInstance) {
    return (
      <Card className="p-8 text-center">
        <Network className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <h3 className="text-lg font-medium mb-2">No Instance Selected</h3>
        <p className="text-muted-foreground mb-4">
          Please select or create an instance to manage nodes.
        </p>
      </Card>
    );
  }

  // Show error state
  if (error) {
    return (
      <Card className="p-8 text-center">
        <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
        <h3 className="text-lg font-medium mb-2">Error Loading Nodes</h3>
        <p className="text-muted-foreground mb-4">
          {(error as Error)?.message || 'An error occurred'}
        </p>
        <Button onClick={() => window.location.reload()}>Reload Page</Button>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Educational Intro Section */}
      <Card className="p-6 bg-gradient-to-r from-cyan-50 to-blue-50 dark:from-cyan-950/20 dark:to-blue-950/20 border-cyan-200 dark:border-cyan-800">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-cyan-100 dark:bg-cyan-900/30 rounded-lg">
            <BookOpen className="h-6 w-6 text-cyan-600 dark:text-cyan-400" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-cyan-900 dark:text-cyan-100 mb-2">
              What are Cluster Nodes?
            </h3>
            <p className="text-cyan-800 dark:text-cyan-200 mb-3 leading-relaxed">
              Think of cluster nodes as the "workers" in your personal cloud factory. Each node is a separate computer 
              that contributes its processing power, memory, and storage to the overall cluster. Some nodes are "controllers" 
              (like managers) that coordinate the work, while others are "workers" that do the heavy lifting.
            </p>
            <p className="text-cyan-700 dark:text-cyan-300 mb-4 text-sm">
              By connecting multiple computers together as nodes, you create a powerful, resilient system where if one 
              computer fails, the others can pick up the work. This is how you scale your personal cloud from one machine to many.
            </p>
            <Button variant="outline" size="sm" className="text-cyan-700 border-cyan-300 hover:bg-cyan-100 dark:text-cyan-300 dark:border-cyan-700 dark:hover:bg-cyan-900/20">
              <ExternalLink className="h-4 w-4 mr-2" />
              Learn more about distributed computing
            </Button>
          </div>
        </div>
      </Card>

      <Card className="p-6">
        <div className="flex items-center gap-4 mb-6">
          <div className="p-2 bg-primary/10 rounded-lg">
            <Network className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold">Cluster Nodes</h2>
            <p className="text-muted-foreground">
              Connect machines to your wild-cloud
            </p>
          </div>
        </div>

        {isLoading ? (
          <Card className="p-8 text-center">
            <Loader2 className="h-12 w-12 text-primary mx-auto mb-4 animate-spin" />
            <p className="text-muted-foreground">Loading nodes...</p>
          </Card>
        ) : (
          <>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-medium">Cluster Nodes ({assignedNodes.length})</h2>
                <div className="flex gap-2">
                  <input
                    type="text"
                    placeholder="Subnet (e.g., 192.168.1.0/24)"
                    value={subnet}
                    onChange={(e) => setSubnet(e.target.value)}
                    className="px-3 py-1 text-sm border rounded-lg"
                  />
                  <Button
                    size="sm"
                    onClick={handleDiscover}
                    disabled={isDiscovering || discoveryStatus?.active}
                  >
                    {isDiscovering || discoveryStatus?.active ? (
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    ) : null}
                    {discoveryStatus?.active ? 'Discovering...' : 'Discover'}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={handleDetect}
                    disabled={isDetecting}
                  >
                    {isDetecting ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
                    Auto Detect
                  </Button>
                </div>
              </div>

              {assignedNodes.map((node) => (
                <Card key={node.hostname} className="p-4">
                  <div className="flex items-center gap-4">
                    <div className="p-2 bg-muted rounded-lg">
                      {getRoleIcon(node.role)}
                    </div>
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <h4 className="font-medium">{node.hostname}</h4>
                        <Badge variant="outline" className="text-xs">
                          {node.role}
                        </Badge>
                        {getStatusIcon(node.status)}
                      </div>
                      <div className="text-sm text-muted-foreground mb-2">
                        IP: {node.target_ip}
                      </div>
                      {node.hardware && (
                        <div className="flex items-center gap-4 text-xs text-muted-foreground">
                          {node.hardware.cpu && (
                            <span className="flex items-center gap-1">
                              <Cpu className="h-3 w-3" />
                              {node.hardware.cpu}
                            </span>
                          )}
                          {node.hardware.memory && (
                            <span className="flex items-center gap-1">
                              <Monitor className="h-3 w-3" />
                              {node.hardware.memory}
                            </span>
                          )}
                          {node.hardware.disk && (
                            <span className="flex items-center gap-1">
                              <HardDrive className="h-3 w-3" />
                              {node.hardware.disk}
                            </span>
                          )}
                        </div>
                      )}
                      {node.talosVersion && (
                        <div className="text-xs text-muted-foreground mt-1">
                          Talos: {node.talosVersion}
                          {node.kubernetesVersion && ` • K8s: ${node.kubernetesVersion}`}
                        </div>
                      )}
                    </div>
                    <div className="flex items-center gap-3">
                      {getStatusBadge(node.status)}
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => handleDeleteNode(node.hostname)}
                        disabled={isDeleting}
                      >
                        {isDeleting ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Remove'}
                      </Button>
                    </div>
                  </div>
                </Card>
              ))}

              {assignedNodes.length === 0 && (
                <Card className="p-8 text-center">
                  <Network className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                  <h3 className="text-lg font-medium mb-2">No Nodes</h3>
                  <p className="text-muted-foreground mb-4">
                    Use the discover or auto-detect buttons above to find nodes on your network.
                  </p>
                </Card>
              )}
            </div>

            {discoveredIps.length > 0 && (
              <div className="mt-6">
                <h3 className="text-lg font-medium mb-4">Discovered IPs ({discoveredIps.length})</h3>
                <div className="space-y-2">
                  {discoveredIps.map((ip) => (
                    <Card key={ip} className="p-3 flex items-center justify-between">
                      <span className="text-sm font-mono">{ip}</span>
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          onClick={() => handleAddNode(ip, `node-${ip}`, 'worker')}
                          disabled={isAdding}
                        >
                          Add as Worker
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleAddNode(ip, `controlplane-${ip}`, 'controlplane')}
                          disabled={isAdding}
                        >
                          Add as Control Plane
                        </Button>
                      </div>
                    </Card>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </Card>

      <Card className="p-6">
        <h3 className="text-lg font-medium mb-4">PXE Boot Instructions</h3>
        <div className="space-y-3 text-sm">
          <div className="flex items-start gap-3">
            <div className="w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-xs font-medium">
              1
            </div>
            <div>
              <p className="font-medium">Power on your nodes</p>
              <p className="text-muted-foreground">
                Ensure network boot (PXE) is enabled in BIOS/UEFI settings
              </p>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <div className="w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-xs font-medium">
              2
            </div>
            <div>
              <p className="font-medium">Connect to the wild-cloud network</p>
              <p className="text-muted-foreground">
                Nodes will automatically receive IP addresses via DHCP
              </p>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <div className="w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-xs font-medium">
              3
            </div>
            <div>
              <p className="font-medium">Boot Talos Linux</p>
              <p className="text-muted-foreground">
                Nodes will automatically download and boot Talos Linux via PXE
              </p>
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
}