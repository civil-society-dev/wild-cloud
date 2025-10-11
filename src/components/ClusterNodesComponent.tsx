import { useState } from 'react';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Cpu, HardDrive, Network, Monitor, Plus, CheckCircle, AlertCircle, Clock, BookOpen, ExternalLink } from 'lucide-react';

interface ClusterNodesComponentProps {
  onComplete?: () => void;
}

interface Node {
  id: string;
  name: string;
  type: 'controller' | 'worker' | 'unassigned';
  status: 'pending' | 'connecting' | 'connected' | 'healthy' | 'error';
  ipAddress?: string;
  macAddress: string;
  osVersion?: string;
  specs: {
    cpu: string;
    memory: string;
    storage: string;
  };
}

export function ClusterNodesComponent({ onComplete }: ClusterNodesComponentProps) {
  const [currentOsVersion, setCurrentOsVersion] = useState('v13.0.5');
  const [nodes, setNodes] = useState<Node[]>([
    {
      id: 'controller-1',
      name: 'Controller Node 1',
      type: 'controller',
      status: 'healthy',
      macAddress: '00:1A:2B:3C:4D:5E',
      osVersion: 'v13.0.4',
      specs: {
        cpu: '4 cores',
        memory: '8GB RAM',
        storage: '120GB SSD',
      },
    },
    {
      id: 'worker-1',
      name: 'Worker Node 1',
      type: 'worker',
      status: 'healthy',
      macAddress: '00:1A:2B:3C:4D:5F',
      osVersion: 'v13.0.5',
      specs: {
        cpu: '8 cores',
        memory: '16GB RAM',
        storage: '500GB SSD',
      },
    },
    {
      id: 'worker-2',
      name: 'Worker Node 2',
      type: 'worker',
      status: 'healthy',
      macAddress: '00:1A:2B:3C:4D:60',
      osVersion: 'v13.0.4',
      specs: {
        cpu: '8 cores',
        memory: '16GB RAM',
        storage: '500GB SSD',
      },
    },
    {
      id: 'node-1',
      name: 'Node 1',
      type: 'unassigned',
      status: 'pending',
      macAddress: '00:1A:2B:3C:4D:5E',
      osVersion: 'v13.0.5',
      specs: {
        cpu: '4 cores',
        memory: '8GB RAM',
        storage: '120GB SSD',
      },
    },
    {
      id: 'node-2',
      name: 'Node 2',
      type: 'unassigned',
      status: 'pending',
      macAddress: '00:1A:2B:3C:4D:5F',
      osVersion: 'v13.0.5',
      specs: {
        cpu: '8 cores',
        memory: '16GB RAM',
        storage: '500GB SSD',
      },
    },
  ]);

  const getStatusIcon = (status: Node['status']) => {
    switch (status) {
      case 'connected':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'error':
        return <AlertCircle className="h-5 w-5 text-red-500" />;
      case 'connecting':
        return <Clock className="h-5 w-5 text-blue-500 animate-spin" />;
      default:
        return <Monitor className="h-5 w-5 text-muted-foreground" />;
    }
  };

  const getStatusBadge = (status: Node['status']) => {
    const variants = {
      pending: 'secondary',
      connecting: 'default',
      connected: 'success',
      healthy: 'success',
      error: 'destructive',
    } as const;

    const labels = {
      pending: 'Pending',
      connecting: 'Connecting',
      connected: 'Connected',
      healthy: 'Healthy',
      error: 'Error',
    };

    return (
      <Badge variant={variants[status] as any}>
        {labels[status]}
      </Badge>
    );
  };

  const getTypeIcon = (type: Node['type']) => {
    return type === 'controller' ? (
      <Cpu className="h-4 w-4" />
    ) : (
      <HardDrive className="h-4 w-4" />
    );
  };

  const handleNodeAction = (nodeId: string, action: 'connect' | 'retry' | 'upgrade_node') => {
    console.log(`${action} node: ${nodeId}`);
  };

  const connectedNodes = nodes.filter(node => node.status === 'connected').length;
  const assignedNodes = nodes.filter(node => node.type !== 'unassigned');
  const unassignedNodes = nodes.filter(node => node.type === 'unassigned');
  const totalNodes = nodes.length;
  const isComplete = connectedNodes === totalNodes;

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

        <div className="space-y-4">
          <h2 className="text-lg font-medium mb-4">Assigned Nodes ({assignedNodes.length}/{totalNodes})</h2>
          {assignedNodes.map((node) => (
            <Card key={node.id} className="p-4">
              <div className="flex items-center gap-4">
                <div className="p-2 bg-muted rounded-lg">
                  {getTypeIcon(node.type)}
                </div>
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <h4 className="font-medium">{node.name}</h4>
                    <Badge variant="outline" className="text-xs">
                      {node.type}
                    </Badge>
                    {getStatusIcon(node.status)}
                  </div>
                  <div className="text-sm text-muted-foreground mb-2">
                    MAC: {node.macAddress}
                    {node.ipAddress && ` • IP: ${node.ipAddress}`}
                  </div>
                  <div className="flex items-center gap-4 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Cpu className="h-3 w-3" />
                      {node.specs.cpu}
                    </span>
                    <span className="flex items-center gap-1">
                      <Monitor className="h-3 w-3" />
                      {node.specs.memory}
                    </span>
                    <span className="flex items-center gap-1">
                      <HardDrive className="h-3 w-3" />
                      {node.specs.storage}
                    </span>
                    {node.osVersion && (
                      <span className="flex items-center gap-1">
                        <Badge variant="outline" className="text-xs">
                          OS: {node.osVersion}
                        </Badge>
                      </span>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  {getStatusBadge(node.status)}
                  {node.osVersion !== currentOsVersion && (
                    <Button
                      size="sm"
                      onClick={() => handleNodeAction(node.id, 'upgrade_node')}
                    >
                      Upgrade OS
                    </Button>
                  )}
                  {node.status === 'error' && (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleNodeAction(node.id, 'retry')}
                    >
                      Retry
                    </Button>
                  )}
                </div>
              </div>
            </Card>
          ))}
        </div>

        <h2 className="text-lg font-medium mb-4 mt-6">Unassigned Nodes ({unassignedNodes.length}/{totalNodes})</h2>
        <div className="space-y-4">
          {unassignedNodes.map((node) => (
            <Card key={node.id} className="p-4">
              <div className="flex items-center gap-4">
                <div className="p-2 bg-muted rounded-lg">
                  {getTypeIcon(node.type)}
                </div>
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-1">
                    <h4 className="font-medium">{node.name}</h4>
                    <Badge variant="outline" className="text-xs">
                      {node.type}
                    </Badge>
                    {getStatusIcon(node.status)}
                  </div>
                  <div className="text-sm text-muted-foreground mb-2">
                    MAC: {node.macAddress}
                    {node.ipAddress && ` • IP: ${node.ipAddress}`}
                  </div>
                  <div className="flex items-center gap-4 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <Cpu className="h-3 w-3" />
                      {node.specs.cpu}
                    </span>
                    <span className="flex items-center gap-1">
                      <Monitor className="h-3 w-3" />
                      {node.specs.memory}
                    </span>
                    <span className="flex items-center gap-1">
                      <HardDrive className="h-3 w-3" />
                      {node.specs.storage}
                    </span>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  {getStatusBadge(node.status)}
                  {node.status === 'pending' && (
                    <Button
                      size="sm"
                      onClick={() => handleNodeAction(node.id, 'connect')}
                    >
                      Assign
                    </Button>
                  )}
                  {node.status === 'error' && (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleNodeAction(node.id, 'retry')}
                    >
                      Retry
                    </Button>
                  )}
                </div>
              </div>
            </Card>
          ))}
        </div>

        {isComplete && (
          <div className="mt-6 p-4 bg-green-50 dark:bg-green-950 rounded-lg border border-green-200 dark:border-green-800">
            <div className="flex items-center gap-2 mb-2">
              <CheckCircle className="h-5 w-5 text-green-600" />
              <h3 className="font-medium text-green-800 dark:text-green-200">
                Infrastructure Ready!
              </h3>
            </div>
            <p className="text-sm text-green-700 dark:text-green-300 mb-3">
              All nodes are connected and ready for Kubernetes installation.
            </p>
            <Button onClick={onComplete} className="bg-green-600 hover:bg-green-700">
              Continue to Kubernetes Installation
            </Button>
          </div>
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