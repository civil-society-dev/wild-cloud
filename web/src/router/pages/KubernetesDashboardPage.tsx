import { ErrorBoundary } from '../../components';
import { KubernetesDashboard } from '../../components/KubernetesDashboard';

export function KubernetesDashboardPage() {
  return (
    <ErrorBoundary>
      <KubernetesDashboard />
    </ErrorBoundary>
  );
}
