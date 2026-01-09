import { ClusterNodesComponent } from './ClusterNodesComponent';

export function ControlNodesComponent() {
  return <ClusterNodesComponent filterRole="controlplane" hideDiscoveryWhenNodesGte={3} showBootstrap={true} />;
}
