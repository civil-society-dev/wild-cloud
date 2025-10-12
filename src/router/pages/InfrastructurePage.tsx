import { ErrorBoundary } from '../../components';
import { ClusterNodesComponent } from '../../components/ClusterNodesComponent';

export function InfrastructurePage() {
  // Note: onComplete callback removed as phase management will be handled differently with routing
  return (
    <ErrorBoundary>
      <ClusterNodesComponent />
    </ErrorBoundary>
  );
}
