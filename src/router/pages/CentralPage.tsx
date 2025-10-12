import { ErrorBoundary } from '../../components';
import { CentralComponent } from '../../components/CentralComponent';

export function CentralPage() {
  return (
    <ErrorBoundary>
      <CentralComponent />
    </ErrorBoundary>
  );
}
