import { ErrorBoundary } from '../../components';
import { ControlNodesComponent } from '../../components/ControlNodesComponent';

export function ControlNodesPage() {
  return (
    <ErrorBoundary>
      <ControlNodesComponent />
    </ErrorBoundary>
  );
}
