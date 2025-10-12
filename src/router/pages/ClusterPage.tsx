import { ErrorBoundary } from '../../components';
import { ClusterServicesComponent } from '../../components/ClusterServicesComponent';

export function ClusterPage() {
  // Note: onComplete callback removed as phase management will be handled differently with routing
  return (
    <ErrorBoundary>
      <ClusterServicesComponent />
    </ErrorBoundary>
  );
}
