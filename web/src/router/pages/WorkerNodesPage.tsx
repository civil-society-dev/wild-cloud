import { ErrorBoundary } from '../../components';
import { WorkerNodesComponent } from '../../components/WorkerNodesComponent';

export function WorkerNodesPage() {
  return (
    <ErrorBoundary>
      <WorkerNodesComponent />
    </ErrorBoundary>
  );
}
