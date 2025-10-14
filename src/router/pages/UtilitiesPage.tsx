import { useState } from 'react';
import { useParams } from 'react-router';
import { UtilityCard, CopyableValue } from '../../components/UtilityCard';
import { Button } from '../../components/ui/button';
import {
  Key,
  Info,
  Network,
  Server,
  Copy,
  AlertCircle,
} from 'lucide-react';
import {
  useDashboardToken,
  useClusterVersions,
  useNodeIPs,
  useControlPlaneIP,
  useCopySecret,
} from '../../services/api/hooks/useUtilities';

export function UtilitiesPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const [secretToCopy, setSecretToCopy] = useState('');
  const [sourceNamespace, setSourceNamespace] = useState('');
  const [destinationNamespace, setDestinationNamespace] = useState('');

  const dashboardToken = useDashboardToken(instanceId || '');
  const versions = useClusterVersions(instanceId || '');
  const nodeIPs = useNodeIPs(instanceId || '');
  const controlPlaneIP = useControlPlaneIP(instanceId || '');
  const copySecret = useCopySecret();

  const handleCopySecret = () => {
    if (secretToCopy && sourceNamespace && destinationNamespace && instanceId) {
      copySecret.mutate({
        instanceName: instanceId,
        secret: secretToCopy,
        sourceNamespace,
        destinationNamespace
      });
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-3xl font-bold tracking-tight">Utilities</h2>
        <p className="text-muted-foreground">
          Additional tools and utilities for cluster management
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Dashboard Token */}
        <UtilityCard
          title="Dashboard Access Token"
          description="Retrieve your Kubernetes dashboard authentication token"
          icon={<Key className="h-5 w-5 text-primary" />}
          isLoading={dashboardToken.isLoading}
          error={dashboardToken.error}
        >
          {dashboardToken.data && (
            <CopyableValue
              label="Token"
              value={dashboardToken.data.token}
              multiline
            />
          )}
        </UtilityCard>

        {/* Cluster Versions */}
        <UtilityCard
          title="Cluster Version Information"
          description="View Kubernetes and Talos versions running on your cluster"
          icon={<Info className="h-5 w-5 text-primary" />}
          isLoading={versions.isLoading}
          error={versions.error}
        >
          {versions.data && (
            <div className="space-y-3">
              <div className="flex justify-between items-center p-3 bg-muted rounded-lg">
                <span className="text-sm font-medium">Kubernetes</span>
                <span className="text-sm font-mono">{versions.data.version}</span>
              </div>
              {Object.entries(versions.data)
                .filter(([key]) => key !== 'version')
                .map(([key, value]) => (
                  <div
                    key={key}
                    className="flex justify-between items-center p-3 bg-muted rounded-lg"
                  >
                    <span className="text-sm font-medium capitalize">
                      {key.replace(/([A-Z])/g, ' $1').trim()}
                    </span>
                    <span className="text-sm font-mono">{String(value)}</span>
                  </div>
                ))}
            </div>
          )}
        </UtilityCard>

        {/* Node IPs */}
        <UtilityCard
          title="Node IP Addresses"
          description="List all node IP addresses in your cluster"
          icon={<Network className="h-5 w-5 text-primary" />}
          isLoading={nodeIPs.isLoading}
          error={nodeIPs.error}
        >
          {nodeIPs.data && (
            <div className="space-y-2">
              {nodeIPs.data.ips.map((ip, index) => (
                <CopyableValue key={index} value={ip} label={`Node ${index + 1}`} />
              ))}
              {nodeIPs.data.ips.length === 0 && (
                <div className="flex items-center gap-2 text-muted-foreground text-sm">
                  <AlertCircle className="h-4 w-4" />
                  <span>No nodes found</span>
                </div>
              )}
            </div>
          )}
        </UtilityCard>

        {/* Control Plane IP */}
        <UtilityCard
          title="Control Plane IP"
          description="Display the control plane endpoint IP address"
          icon={<Server className="h-5 w-5 text-primary" />}
          isLoading={controlPlaneIP.isLoading}
          error={controlPlaneIP.error}
        >
          {controlPlaneIP.data && (
            <CopyableValue label="Control Plane IP" value={controlPlaneIP.data.ip} />
          )}
        </UtilityCard>

        {/* Secret Copy Utility */}
        <UtilityCard
          title="Copy Secret"
          description="Copy a secret between namespaces"
          icon={<Copy className="h-5 w-5 text-primary" />}
        >
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium mb-2 block">Secret Name</label>
              <input
                type="text"
                placeholder="e.g., my-secret"
                value={secretToCopy}
                onChange={(e) => setSecretToCopy(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg bg-background"
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-2 block">
                Source Namespace
              </label>
              <input
                type="text"
                placeholder="e.g., default"
                value={sourceNamespace}
                onChange={(e) => setSourceNamespace(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg bg-background"
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-2 block">
                Destination Namespace
              </label>
              <input
                type="text"
                placeholder="e.g., production"
                value={destinationNamespace}
                onChange={(e) => setDestinationNamespace(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg bg-background"
              />
            </div>
            <Button
              onClick={handleCopySecret}
              disabled={!secretToCopy || !sourceNamespace || !destinationNamespace || copySecret.isPending}
              className="w-full"
            >
              {copySecret.isPending ? 'Copying...' : 'Copy Secret'}
            </Button>
            {copySecret.isSuccess && (
              <div className="text-sm text-green-600 dark:text-green-400">
                Secret copied successfully!
              </div>
            )}
            {copySecret.error && (
              <div className="flex items-center gap-2 text-red-500 text-sm">
                <AlertCircle className="h-4 w-4" />
                <span>{copySecret.error.message}</span>
              </div>
            )}
          </div>
        </UtilityCard>
      </div>
    </div>
  );
}
