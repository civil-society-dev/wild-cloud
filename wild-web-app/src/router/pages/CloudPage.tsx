import { ErrorBoundary } from '../../components';
import { CloudComponent } from '../../components/CloudComponent';

export function CloudPage() {
  return (
    <ErrorBoundary>
      <CloudComponent />
    </ErrorBoundary>
  );
}
