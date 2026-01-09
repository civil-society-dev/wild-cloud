import { ErrorBoundary } from '../../components';
import { AppsComponent } from '../../components/AppsComponent';

export function AppsPage() {
  // Note: onComplete callback removed as phase management will be handled differently with routing
  return (
    <ErrorBoundary>
      <AppsComponent />
    </ErrorBoundary>
  );
}
