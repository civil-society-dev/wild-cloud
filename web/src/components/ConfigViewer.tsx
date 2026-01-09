import { Card } from './ui/card';
import { cn } from '@/lib/utils';

interface ConfigViewerProps {
  content: string;
  className?: string;
}

export function ConfigViewer({ content, className }: ConfigViewerProps) {
  return (
    <Card className={cn('p-4', className)}>
      <pre className="text-xs overflow-auto max-h-96 whitespace-pre-wrap break-all">
        <code>{content}</code>
      </pre>
    </Card>
  );
}
