import { ErrorBoundary } from '../../components';
import { Terminal } from '../../components/Terminal';

export function TerminalPage() {
  return (
    <ErrorBoundary>
      <div className="h-[calc(100vh-8rem)]">
        <Terminal />
      </div>
    </ErrorBoundary>
  );
}
