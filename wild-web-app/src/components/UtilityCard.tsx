import { ReactNode } from 'react';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from './ui/card';
import { Button } from './ui/button';
import { Loader2, Copy, Check, AlertCircle } from 'lucide-react';
import { useState } from 'react';

interface UtilityCardProps {
  title: string;
  description: string;
  icon: ReactNode;
  action?: {
    label: string;
    onClick: () => void;
    disabled?: boolean;
    loading?: boolean;
  };
  children?: ReactNode;
  error?: Error | null;
  isLoading?: boolean;
}

export function UtilityCard({
  title,
  description,
  icon,
  action,
  children,
  error,
  isLoading,
}: UtilityCardProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-start gap-3">
          <div className="p-2 bg-primary/10 rounded-lg">
            {icon}
          </div>
          <div className="flex-1">
            <CardTitle>{title}</CardTitle>
            <CardDescription>{description}</CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading ? (
          <div className="flex items-center justify-center py-4">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : error ? (
          <div className="flex items-center gap-2 text-red-500 text-sm">
            <AlertCircle className="h-4 w-4" />
            <span>{error.message}</span>
          </div>
        ) : (
          children
        )}
        {action && (
          <Button
            onClick={action.onClick}
            disabled={action.disabled || action.loading || isLoading}
            className="w-full"
          >
            {action.loading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              action.label
            )}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

interface CopyableValueProps {
  value: string;
  label?: string;
  multiline?: boolean;
}

export function CopyableValue({ value, label, multiline = false }: CopyableValueProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-2">
      {label && <div className="text-sm font-medium">{label}</div>}
      <div className="flex items-start gap-2">
        <div
          className={`flex-1 p-3 bg-muted rounded-lg font-mono text-sm ${
            multiline ? '' : 'truncate'
          }`}
        >
          {multiline ? (
            <pre className="whitespace-pre-wrap break-all">{value}</pre>
          ) : (
            <span className="block truncate">{value}</span>
          )}
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={handleCopy}
          className="shrink-0"
        >
          {copied ? (
            <Check className="h-4 w-4 text-green-500" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
        </Button>
      </div>
    </div>
  );
}
