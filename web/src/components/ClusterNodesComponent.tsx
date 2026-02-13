import { useState, useEffect, useMemo } from 'react';
import { Card } from './ui/card';
import { EntityTile } from './ui/entity-tile';
import { Button } from './ui/button';
import { Alert } from './ui/alert';
import { Input } from './ui/input';
import { Network, CheckCircle, AlertCircle, BookOpen, ExternalLink, Loader2 } from 'lucide-react';
import { useInstanceContext } from '../hooks/useInstanceContext';
import { useNodes, useDiscoveryStatus } from '../hooks/useNodes';
import { useCluster } from '../hooks/useCluster';
import { useClusterStatus } from '../services/api/hooks/useCluster';
import { BootstrapModal } from './cluster/BootstrapModal';
import { deriveNodeStatus } from '../utils/deriveNodeStatus';
import { NodeStatus } from '../types/nodeStatus';
import { NodeFormDialog } from './nodes/NodeFormDialog';
import { ClusterSettings } from './nodes/ClusterSettings';
import type { NodeFormData } from './nodes/NodeForm';
import type { Node, HardwareInfo, DiscoveredNode } from '../services/api/types';
import { usePageHelp } from '../hooks/usePageHelp';

interface ClusterNodesComponentProps {
  filterRole?: 'controlplane' | 'worker';
  hideDiscoveryWhenNodesGte?: number;
  showBootstrap?: boolean;
}

export function ClusterNodesComponent({
  filterRole,
  hideDiscoveryWhenNodesGte,
  showBootstrap = true
}: ClusterNodesComponentProps = {}) {
  const { currentInstance } = useInstanceContext();
  const {
    nodes,
    isLoading,
    error,
    addNode,
    addError,
    deleteNode,
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

  const { data: clusterStatusData } = useClusterStatus(currentInstance || '');

  const [addNodeIp, setAddNodeIp] = useState('');

  usePageHelp({
    title: 'What are Cluster Nodes?',
    description: (
      <>
        <p className="mb-3 leading-relaxed">
          Think of cluster nodes as the "workers" in your personal cloud factory. Each node is a separate computer
          that contributes its processing power, memory, and storage to the overall cluster. Some nodes are "controllers"
          (like managers) that coordinate the work, while others are "workers" that do the heavy lifting.
        </p>
        <p className="text-sm">
          By connecting multiple computers together as nodes, you create a powerful, resilient system where if one
          computer fails, the others can pick up the work. This is how you scale your personal cloud from one machine to many.
        </p>
      </>
    ),
    icon: <BookOpen className="h-6 w-6 text-cyan-600 dark:text-cyan-400" />,
    color: 'bg-gradient-to-r from-cyan-50 to-blue-50 dark:from-cyan-950/20 dark:to-blue-950/20',
    actions: (
      <Button
        variant="outline"
        size="sm"
        className="text-cyan-700 border-cyan-300 hover:bg-cyan-100 dark:text-cyan-300 dark:border-cyan-700 dark:hover:bg-cyan-900/20"
      >
        <ExternalLink className="h-4 w-4 mr-2" />
        Learn more about distributed computing
      </Button>
    ),
  });
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
  const [drawerEverOpened, setDrawerEverOpened] = useState(false);

  const closeDrawer = () => setDrawerState({ ...drawerState, open: false });

  // Sync mutation errors to local state for display
  useEffect(() => {
    if (discoverMutationError) {
      const errorMsg = discoverMutationError instanceof Error ? discoverMutationError.message : 'Failed to discover nodes';
      setDiscoverError(errorMsg);
    }
  }, [discoverMutationError]);

  useEffect(() => {
    if (getHardwareError) {
      const errorMsg = getHardwareError instanceof Error ? getHardwareError.message : 'Failed to detect hardware';
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
        setDiscoverSuccess(`Discovery complete! No nodes were found.`);
      } else {
        setDiscoverSuccess(`Discovery complete! Found ${count} node${count !== 1 ? 's' : ''}.`);
      }
      setDiscoverError(null);
      refetch();

      const timer = setTimeout(() => setDiscoverSuccess(null), 5000);
      return () => clearTimeout(timer);
    }

    // Update previous state
    setPrevDiscoveryActive(isActive);
  }, [discoveryStatus, prevDiscoveryActive, refetch]);

  const handleAddFromDiscovery = async (discovered: DiscoveredNode) => {
    // Fetch full hardware details for the discovered node
    try {
      const hardware = await getHardware(discovered.ip);
      setDrawerEverOpened(true);
      setDrawerState({
        open: true,
        mode: 'add',
        detection: hardware,
      });
    } catch (err) {
      console.error('Failed to detect hardware:', err);
      setDetectError(err instanceof Error ? err.message : 'Failed to detect hardware');
    }
  };

  const handleAddNode = async () => {
    if (!addNodeIp) return;

    try {
      const hardware = await getHardware(addNodeIp);
      setDrawerEverOpened(true);
      setDrawerState({
        open: true,
        mode: 'add',
        detection: hardware,
      });
    } catch (err) {
      console.error('Failed to detect hardware:', err);
      setDetectError(err instanceof Error ? err.message : 'Failed to detect hardware');
    }
  };

  const getNodeStatusColor = (node: Node): string | null => {
    const status = deriveNodeStatus(node);
    switch (status) {
      case NodeStatus.DISCOVERED:
      case NodeStatus.PENDING:
      case NodeStatus.MAINTENANCE:
        return 'bg-white border border-black/20';
      case NodeStatus.CONFIGURING:
      case NodeStatus.CONFIGURED:
      case NodeStatus.APPLYING:
      case NodeStatus.PROVISIONING:
      case NodeStatus.REPROVISIONING:
      case NodeStatus.UNKNOWN:
        return 'bg-amber-500';
      case NodeStatus.READY:
      case NodeStatus.HEALTHY:
        return null;
      case NodeStatus.UNREACHABLE:
      case NodeStatus.DEGRADED:
      case NodeStatus.FAILED:
      case NodeStatus.ORPHANED:
        return 'bg-red-500';
    }
  };

  const handleConfigureNode = async (node: Node) => {
    // Try to detect hardware if target_ip is available
    if (node.target_ip) {
      try {
        const hardware = await getHardware(node.target_ip);
        setDrawerEverOpened(true);
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
    setDrawerEverOpened(true);
    setDrawerState({
      open: true,
      mode: 'configure',
      node,
    });
  };

  const handleAddSubmit = async (data: NodeFormData) => {
    const nodeData = {
      hostname: data.hostname,
      role: filterRole || data.role,
      disk: data.disk,
      target_ip: data.targetIp,
      interface: data.interface,
      schematic_id: data.schematicId,
      maintenance: data.maintenance,
    };

    // Add node configuration (if this fails, error is shown and drawer stays open)
    await addNode(nodeData);

    // Apply configuration immediately for new nodes
    try {
      await applyNode(data.hostname);
    } catch (applyError) {
      // Apply failed but node is added - user can use Apply button on card
      console.error('Failed to apply node configuration:', applyError);
    }

    closeDrawer();
    setAddNodeIp('');
  };

  const handleConfigureSubmit = async (data: NodeFormData) => {
    if (!drawerState.node) return;

    await updateNode({
      nodeName: drawerState.node.hostname,
      updates: {
        role: data.role,
        target_ip: data.targetIp,
        interface: data.interface,
        schematic_id: data.schematicId,
        maintenance: data.maintenance,
      },
    });
    closeDrawer();
  };

  const handleApply = async (data: NodeFormData) => {
    if (!drawerState.node) return;

    await handleConfigureSubmit(data);
    await applyNode(drawerState.node.hostname);
  };

  const handleDeleteNode = async (hostname: string) => {
    if (!currentInstance) return;
    if (confirm(`Reset and remove node ${hostname}?\n\nThis will reset the node and remove it from the cluster. The node will reboot to maintenance mode and can be reconfigured.`)) {
      await deleteNode(hostname);
    }
  };

  const handleDiscover = () => {
    setDiscoverError(null);
    setDiscoverSuccess(null);
    // Always use auto-detect to scan all local networks
    discover(undefined);
  };


  // Derive status from backend state flags for each node
  const assignedNodes = useMemo(() => {
    const allNodes = nodes.map(node => {
    // Get runtime status from cluster status
    const runtimeStatus = clusterStatusData?.node_statuses?.[node.hostname];

    let status = 'pending';
    if (node.maintenance) {
      status = 'provisioning';
    } else if (node.configured && !node.applied) {
      status = 'connecting';
    } else if (node.applied) {
      status = 'ready';
    }

    return {
      ...node,
      status,
      isReachable: runtimeStatus?.ready,
      inKubernetes: runtimeStatus?.ready, // Whether in cluster (from backend 'ready' field)
      kubernetesReady: runtimeStatus?.kubernetes_ready, // Whether K8s Ready condition is true
    };
  });

    // Filter by role if specified
    if (filterRole) {
      return allNodes.filter(node => node.role === filterRole);
    }
    return allNodes;
  }, [nodes, clusterStatusData, filterRole]);

  // Check if cluster needs bootstrap
  const needsBootstrap = useMemo(() => {
    // Find first ready control plane node
    const hasReadyControlPlane = assignedNodes.some(
      n => n.role === 'controlplane' && n.status === 'ready'
    );

    // Check if cluster is already bootstrapped using cluster status
    // The backend checks for kubeconfig existence and cluster connectivity
    // Status is "not_bootstrapped" when kubeconfig doesn't exist
    // Any other status (ready, degraded, unreachable) means cluster is bootstrapped
    const hasBootstrapped = clusterStatus?.status !== 'not_bootstrapped';

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
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">
            {filterRole === 'controlplane' ? 'Control Nodes' : filterRole === 'worker' ? 'Worker Nodes' : 'Cluster Nodes'}
          </h2>
          <p className="text-muted-foreground">
            Connect machines to your wild-cloud
          </p>
        </div>
      </div>

      {/* Bootstrap Alert */}
      {showBootstrap && needsBootstrap && firstReadyControl && (
        <Alert variant="info">
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

      {/* Cluster Settings - only show for control plane nodes */}
      {filterRole === 'controlplane' && currentInstance && (
        <ClusterSettings instanceId={currentInstance} />
      )}


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
            <p className="text-sm mt-1">{addError instanceof Error ? addError.message : 'An error occurred'}</p>
          </div>
        </Alert>
      )}

      {deleteError && (
        <Alert variant="error" onClose={() => {}} className="mb-4">
          <AlertCircle className="h-4 w-4" />
          <div>
            <strong>Failed to Remove Node</strong>
            <p className="text-sm mt-1">{deleteError instanceof Error ? deleteError.message : 'An error occurred'}</p>
          </div>
        </Alert>
      )}


      {/* ADD NODES SECTION - Discovery and manual add combined */}
      {(!hideDiscoveryWhenNodesGte || assignedNodes.length < hideDiscoveryWhenNodesGte) && (
        <Card className="p-6">
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">
            Add Nodes to Cluster
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            Discover nodes on the network or manually add by IP address
          </p>

          {/* Discovery button */}
          <div className="flex gap-2 mb-4">
            <Button
              onClick={handleDiscover}
              disabled={isDiscovering || discoveryStatus?.active}
              className="flex-1"
            >
              {isDiscovering || discoveryStatus?.active ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin mr-2" />
                  Discovering...
                </>
              ) : (
                'Discover Nodes'
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

          {/* Discovered nodes display */}
          {discoveryStatus?.nodes_found && discoveryStatus.nodes_found.length > 0 && (
            <div className="space-y-3 mb-4">
              {discoveryStatus.nodes_found.map((discovered) => (
                <div key={discovered.ip} className="border border-gray-300 dark:border-gray-600 rounded-lg p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="font-medium font-mono text-gray-900 dark:text-gray-100">{discovered.ip}</p>
                      {discovered.version && discovered.version !== 'maintenance' && (
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          {discovered.version}
                        </p>
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
          )}

          {/* Manual add by IP - styled like a list item */}
          <div className="border border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-4">
            <div className="flex items-center gap-3">
              <Input
                type="text"
                value={addNodeIp}
                onChange={(e) => setAddNodeIp(e.target.value)}
                placeholder="192.168.8.128"
                className="flex-1 font-mono"
              />
              <Button
                onClick={handleAddNode}
                disabled={isGettingHardware}
                size="sm"
              >
                {isGettingHardware ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    Detecting...
                  </>
                ) : (
                  'Add to Cluster'
                )}
              </Button>
            </div>
            <p className="text-xs text-gray-500 dark:text-gray-500 mt-2">
              Add a node by IP address if not discovered automatically
            </p>
          </div>
        </Card>
      )}

      {isLoading ? (
        <Card className="p-8 text-center">
          <Loader2 className="h-12 w-12 text-primary mx-auto mb-4 animate-spin" />
          <p className="text-muted-foreground">Loading nodes...</p>
        </Card>
      ) : (
        <>
          {assignedNodes.length === 0 ? (
            <Card className="p-8 text-center">
              <Network className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">No Nodes</h3>
              <p className="text-muted-foreground mb-4">
                Use the discover or auto-detect buttons above to find nodes on your network.
              </p>
            </Card>
          ) : (
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
              {assignedNodes.map((node) => (
                <EntityTile
                  key={node.hostname}
                  title={node.hostname}
                  description={node.target_ip}
                  statusIndicator={(() => { const color = getNodeStatusColor(node); return color ? <div className={`h-3 w-3 rounded-full ${color}`} /> : undefined; })()}
                  onClick={() => handleConfigureNode(node)}
                  tint="#ca95c8"
                >
                  {node.configured && !node.applied && (
                    <div className="pt-2 border-t" onClick={(e) => e.stopPropagation()}>
                      <Button
                        size="sm"
                        onClick={() => applyNode(node.hostname)}
                        disabled={isApplying}
                        variant="secondary"
                        className="w-full"
                      >
                        {isApplying ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Apply'}
                      </Button>
                    </div>
                  )}
                </EntityTile>
              ))}
            </div>
          )}
        </>
      )}

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

      {/* Node Form Dialog - only render after first open to prevent infinite loop on initial mount */}
      {drawerEverOpened && (
        <NodeFormDialog
          open={drawerState.open}
          onClose={closeDrawer}
          mode={drawerState.mode}
          node={drawerState.mode === 'configure' ? drawerState.node : undefined}
          detection={drawerState.detection}
          onSubmit={drawerState.mode === 'add' ? handleAddSubmit : handleConfigureSubmit}
          onApply={drawerState.mode === 'configure' ? handleApply : undefined}
          onDelete={drawerState.mode === 'configure' && drawerState.node ? async () => {
            await handleDeleteNode(drawerState.node!.hostname);
            closeDrawer();
          } : undefined}
          instanceName={currentInstance || ''}
        />
      )}
    </div>
  );
}