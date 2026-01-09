import { ClusterNodesComponent } from './ClusterNodesComponent';

export function WorkerNodesComponent() {
  return <ClusterNodesComponent filterRole="worker" hideDiscoveryWhenNodesGte={undefined} showBootstrap={false} />;
}
