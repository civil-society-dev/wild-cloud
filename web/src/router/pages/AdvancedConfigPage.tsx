import { ErrorBoundary } from '../../components';
import { AdvancedConfig } from '../../components/AdvancedConfig';

export function AdvancedConfigPage() {
  return (
    <ErrorBoundary>
      <div className="h-[calc(100vh-8rem)]">
        <AdvancedConfig />
      </div>
    </ErrorBoundary>
  );
}
