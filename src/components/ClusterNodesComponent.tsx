import { useState, useEffect, useMemo } from 'react';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Alert } from './ui/alert';
import { Input } from './ui/input';
import { Cpu, HardDrive, Network, Monitor, CheckCircle, AlertCircle, BookOpen, ExternalLink, Loader2 } from 'lucide-react';
import { useInstanceContext } from '../hooks/useInstanceContext';
import { useNodes, useDiscoveryStatus } from '../hooks/useNodes';
import { useCluster } from '../hooks/useCluster';
import { BootstrapModal } from './cluster/BootstrapModal';
import { NodeStatusBadge } from './nodes/NodeStatusBadge';
import { NodeFormDrawer } from './nodes/NodeFormDrawer';
import type { NodeFormData } from './nodes/NodeForm';
import type { Node, HardwareInfo, DiscoveredNode } from '../services/api/types';

export function ClusterNodesComponent() {
  const { currentInstance } = useInstanceContext();
  const {
    nodes,
    isLoading,
    error,
    addNode,
    addError,
    deleteNode,
    isDeleting,
    deleteError,
    discover,
    isDiscovering,
    discoverError: discoverMutationError,
    getHardware,
    isGettingHardware,
    getHardwareError,
    cancelDiscovery,
    isCancellingDiscovery,
    updateNode,
    applyNode,
    isApplying,
    refetch
  } = useNodes(currentInstance);

  const {
    data: discoveryStatus
  } = useDiscoveryStatus(currentInstance);

  const {
    status: clusterStatus
  } = useCluster(currentInstance);

  const [discoverSubnet, setDiscoverSubnet] = useState('192.168.8.0/24');
  const [addNodeIp, setAddNodeIp] = useState('');
  const [discoverError, setDiscoverError] = useState<string | null>(null);
  const [detectError, setDetectError] = useState<string | null>(null);
  const [discoverSuccess, setDiscoverSuccess] = useState<string | null>(null);
  const [showBootstrapModal, setShowBootstrapModal] = useState(false);
  const [bootstrapNode, setBootstrapNode] = useState<{ name: string; ip: string } | null>(null);
  const [drawerState, setDrawerState] = useState<{
    open: boolean;
    mode: 'add' | 'configure';
    node?: Node;
    detection?: HardwareInfo;
  }>({
    open: false,
    mode: 'add',
  });

  const closeDrawer = () => setDrawerState({ ...drawerState, open: false });

  // Sync mutation errors to local state for display
  useEffect(() => {
    if (discoverMutationError) {
      const errorMsg = (discoverMutationError as any)?.message || 'Failed to discover nodes';
      setDiscoverError(errorMsg);
    }
  }, [discoverMutationError]);

  useEffect(() => {
    if (getHardwareError) {
      const errorMsg = (getHardwareError as any)?.message || 'Failed to detect hardware';
      setDetectError(errorMsg);
    }
  }, [getHardwareError]);

  // Track previous discovery status to detect completion
  const [prevDiscoveryActive, setPrevDiscoveryActive] = useState<boolean | null>(null);

  // Handle discovery completion (when active changes from true to false)
  useEffect(() => {
    const isActive = discoveryStatus?.active ?? false;

    // Discovery just completed (was active, now inactive)
    if (prevDiscoveryActive === true && isActive === false && discoveryStatus) {
      const count = discoveryStatus.nodes_found?.length || 0;
      if (count === 0) {
        setDiscoverSuccess(`Discovery complete! No nodes were found in the subnet.`);
      } else {
        setDiscoverSuccess(`Discovery complete! Found ${count} node${count !== 1 ? 's' : ''} in subnet.`);
      }
      setDiscoverError(null);
      refetch();

      const timer = setTimeout(() => setDiscoverSuccess(null), 5000);
      return () => clearTimeout(timer);
    }

    // Update previous state
    setPrevDiscoveryActive(isActive);
  }, [discoveryStatus, prevDiscoveryActive, refetch]);

  const getRoleIcon = (role: string) => {
    return role === 'controlplane' ? (
      <Cpu className="h-4 w-4" />
    ) : (
      <HardDrive className="h-4 w-4" />
    );
  };

  const handleAddFromDiscovery = async (discovered: DiscoveredNode) => {
    // Fetch full hardware details for the discovered node
    try {
      const hardware = await getHardware(discovered.ip);
      setDrawerState({
        open: true,
        mode: 'add',
        detection: hardware,
      });
    } catch (err) {
      console.error('Failed to detect hardware:', err);
      setDetectError((err as any)?.message || 'Failed to detect hardware');
    }
  };

  const handleAddNode = async () => {
    if (!addNodeIp) return;

    try {
      const hardware = await getHardware(addNodeIp);
      setDrawerState({
        open: true,
        mode: 'add',
        detection: hardware,
      });
    } catch (err) {
      console.error('Failed to detect hardware:', err);
      setDetectError((err as any)?.message || 'Failed to detect hardware');
    }
  };

  const handleConfigureNode = async (node: Node) => {
    // Try to detect hardware if target_ip is available
    if (node.target_ip) {
      try {
        const hardware = await getHardware(node.target_ip);
        setDrawerState({
          open: true,
          mode: 'configure',
          node,
          detection: hardware,
        });
        return;
      } catch (err) {
        console.error('Failed to detect hardware:', err);
        // Fall through to open drawer without detection data
      }
    }

    // Open drawer without detection data (either no target_ip or detection failed)
    setDrawerState({
      open: true,
      mode: 'configure',
      node,
    });
  };

  const handleAddSubmit = async (data: NodeFormData) => {
    await addNode({
      hostname: data.hostname,
      role: data.role,
      disk: data.disk,
      target_ip: data.targetIp,
      current_ip: data.currentIp,
      interface: data.interface,
      schematic_id: data.schematicId,
      maintenance: data.maintenance,
    });
    closeDrawer();
    setAddNodeIp('');
  };

  const handleConfigureSubmit = async (data: NodeFormData) => {
    if (!drawerState.node) return;

    await updateNode({
      nodeName: drawerState.node.hostname,
      updates: {
        role: data.role,
        config: {
          disk: data.disk,
          target_ip: data.targetIp,
          current_ip: data.currentIp,
          interface: data.interface,
          schematic_id: data.schematicId,
          maintenance: data.maintenance,
        },
      },
    });
    closeDrawer();
  };

  const handleApply = async (data: NodeFormData) => {
    if (!drawerState.node) return;

    await handleConfigureSubmit(data);
    await applyNode(drawerState.node.hostname);
  };

  const handleDeleteNode = (hostname: string) => {
    if (!currentInstance) return;
    if (confirm(`Are you sure you want to remove node ${hostname}?`)) {
      deleteNode(hostname);
    }
  };

  const handleDiscover = () => {
    setDiscoverError(null);
    setDiscoverSuccess(null);
    discover(discoverSubnet);
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

  // Check if cluster needs bootstrap
  const needsBootstrap = useMemo(() => {
    // Find first ready control plane node
    const hasReadyControlPlane = assignedNodes.some(
      n => n.role === 'controlplane' && n.status === 'ready'
    );

    // Check if cluster is already bootstrapped using cluster status
    // The backend checks for kubeconfig existence and cluster connectivity
    const hasBootstrapped = clusterStatus?.status !== 'not_bootstrapped' && clusterStatus?.status !== undefined;

    return hasReadyControlPlane && !hasBootstrapped;
  }, [assignedNodes, clusterStatus]);

  const firstReadyControl = useMemo(() => {
    return assignedNodes.find(
      n => n.role === 'controlplane' && n.status === 'ready'
    );
  }, [assignedNodes]);

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

      {/* Bootstrap Alert */}
      {needsBootstrap && firstReadyControl && (
        <Alert variant="info" className="mb-6">
          <CheckCircle className="h-5 w-5" />
          <div className="flex-1">
            <h3 className="font-semibold mb-1">First Control Plane Node Ready!</h3>
            <p className="text-sm text-muted-foreground mb-3">
              Your first control plane node ({firstReadyControl.hostname}) is ready.
              Bootstrap the cluster to initialize etcd and start Kubernetes control plane components.
            </p>
            <Button
              onClick={() => {
                setBootstrapNode({
                  name: firstReadyControl.hostname,
                  ip: firstReadyControl.target_ip
                });
                setShowBootstrapModal(true);
              }}
              size="sm"
            >
              Bootstrap Cluster
            </Button>
          </div>
        </Alert>
      )}

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
            {/* Error and Success Alerts */}
            {discoverError && (
              <Alert variant="error" onClose={() => setDiscoverError(null)} className="mb-4">
                <AlertCircle className="h-4 w-4" />
                <div>
                  <strong>Discovery Failed</strong>
                  <p className="text-sm mt-1">{discoverError}</p>
                </div>
              </Alert>
            )}

            {discoverSuccess && (
              <Alert variant="success" onClose={() => setDiscoverSuccess(null)} className="mb-4">
                <CheckCircle className="h-4 w-4" />
                <div>
                  <strong>Discovery Successful</strong>
                  <p className="text-sm mt-1">{discoverSuccess}</p>
                </div>
              </Alert>
            )}

            {detectError && (
              <Alert variant="error" onClose={() => setDetectError(null)} className="mb-4">
                <AlertCircle className="h-4 w-4" />
                <div>
                  <strong>Auto-Detect Failed</strong>
                  <p className="text-sm mt-1">{detectError}</p>
                </div>
              </Alert>
            )}


            {addError && (
              <Alert variant="error" onClose={() => {}} className="mb-4">
                <AlertCircle className="h-4 w-4" />
                <div>
                  <strong>Failed to Add Node</strong>
                  <p className="text-sm mt-1">{(addError as any)?.message || 'An error occurred'}</p>
                </div>
              </Alert>
            )}

            {deleteError && (
              <Alert variant="error" onClose={() => {}} className="mb-4">
                <AlertCircle className="h-4 w-4" />
                <div>
                  <strong>Failed to Remove Node</strong>
                  <p className="text-sm mt-1">{(deleteError as any)?.message || 'An error occurred'}</p>
                </div>
              </Alert>
            )}

            {/* DISCOVERY SECTION - Scan subnet for nodes */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 mb-6">
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">
                Discover Nodes on Network
              </h3>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                Scan a subnet to find nodes in maintenance mode
              </p>

              <div className="flex gap-3 mb-4">
                <Input
                  type="text"
                  value={discoverSubnet}
                  onChange={(e) => setDiscoverSubnet(e.target.value)}
                  placeholder="192.168.8.0/24"
                  className="flex-1"
                />
                <Button
                  onClick={handleDiscover}
                  disabled={isDiscovering || discoveryStatus?.active}
                >
                  {isDiscovering || discoveryStatus?.active ? (
                    <>
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                      Discovering...
                    </>
                  ) : (
                    'Discover'
                  )}
                </Button>
                {(isDiscovering || discoveryStatus?.active) && (
                  <Button
                    onClick={() => cancelDiscovery()}
                    disabled={isCancellingDiscovery}
                    variant="destructive"
                  >
                    {isCancellingDiscovery && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
                    Cancel
                  </Button>
                )}
              </div>

              {discoveryStatus?.nodes_found && discoveryStatus.nodes_found.length > 0 && (
                <div className="mt-6">
                  <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                    Discovered {discoveryStatus.nodes_found.length} node(s)
                  </h4>
                  <div className="space-y-3">
                    {discoveryStatus.nodes_found.map((discovered) => (
                      <div key={discovered.ip} className="border border-gray-300 dark:border-gray-600 rounded-lg p-4">
                        <div className="flex items-center justify-between">
                          <div>
                            <p className="font-medium font-mono text-gray-900 dark:text-gray-100">{discovered.ip}</p>
                            <p className="text-sm text-gray-600 dark:text-gray-400">
                              Maintenance mode{discovered.version ? ` • Talos ${discovered.version}` : ''}
                            </p>
                            {discovered.hostname && (
                              <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">{discovered.hostname}</p>
                            )}
                          </div>
                          <Button
                            onClick={() => handleAddFromDiscovery(discovered)}
                            size="sm"
                          >
                            Add to Cluster
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* ADD NODE SECTION - Add single node by IP */}
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 mb-6">
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">
                Add Single Node
              </h3>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                Add a node by IP address to detect hardware and configure
              </p>

              <div className="flex gap-3">
                <Input
                  type="text"
                  value={addNodeIp}
                  onChange={(e) => setAddNodeIp(e.target.value)}
                  placeholder="192.168.8.128"
                  className="flex-1"
                />
                <Button
                  onClick={handleAddNode}
                  disabled={isGettingHardware}
                  variant="secondary"
                >
                  {isGettingHardware ? (
                    <>
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                      Detecting...
                    </>
                  ) : (
                    'Add Node'
                  )}
                </Button>
              </div>
            </div>

            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-medium">Cluster Nodes ({assignedNodes.length})</h2>
              </div>

              {assignedNodes.map((node) => (
                <Card key={node.hostname} className="p-4 hover:shadow-md transition-shadow">
                  <div className="mb-2">
                    <NodeStatusBadge node={node} compact />
                  </div>

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
                      </div>
                      <div className="text-sm text-muted-foreground mb-2">
                        Target: {node.target_ip}
                      </div>
                      {node.disk && (
                        <div className="text-xs text-muted-foreground">
                          Disk: {node.disk}
                        </div>
                      )}
                      {node.hardware && (
                        <div className="flex items-center gap-4 text-xs text-muted-foreground mt-2">
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
                    <div className="flex flex-col gap-2">
                      <Button
                        size="sm"
                        onClick={() => handleConfigureNode(node)}
                      >
                        Configure
                      </Button>
                      {node.configured && !node.applied && (
                        <Button
                          size="sm"
                          onClick={() => applyNode(node.hostname)}
                          disabled={isApplying}
                          variant="secondary"
                        >
                          {isApplying ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Apply'}
                        </Button>
                      )}
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => handleDeleteNode(node.hostname)}
                        disabled={isDeleting}
                      >
                        {isDeleting ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Delete'}
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
          </>
        )}
      </Card>

      {/* Bootstrap Modal */}
      {showBootstrapModal && bootstrapNode && (
        <BootstrapModal
          instanceName={currentInstance!}
          nodeName={bootstrapNode.name}
          nodeIp={bootstrapNode.ip}
          onClose={() => {
            setShowBootstrapModal(false);
            setBootstrapNode(null);
            refetch();
          }}
        />
      )}

      {/* Node Form Drawer */}
      <NodeFormDrawer
        open={drawerState.open}
        onClose={closeDrawer}
        mode={drawerState.mode}
        node={drawerState.mode === 'configure' ? drawerState.node : undefined}
        detection={drawerState.detection}
        onSubmit={drawerState.mode === 'add' ? handleAddSubmit : handleConfigureSubmit}
        onApply={drawerState.mode === 'configure' ? handleApply : undefined}
        instanceName={currentInstance || ''}
      />
    </div>
  );
}