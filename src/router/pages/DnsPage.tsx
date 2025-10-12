import { ErrorBoundary } from '../../components';
import { DnsComponent } from '../../components/DnsComponent';

export function DnsPage() {
  return (
    <ErrorBoundary>
      <DnsComponent />
    </ErrorBoundary>
  );
}
