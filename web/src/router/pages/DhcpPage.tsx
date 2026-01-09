import { ErrorBoundary } from '../../components';
import { DhcpComponent } from '../../components/DhcpComponent';

export function DhcpPage() {
  return (
    <ErrorBoundary>
      <DhcpComponent />
    </ErrorBoundary>
  );
}
