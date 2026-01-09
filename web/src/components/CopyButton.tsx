import { useState } from 'react';
import { Copy, Check } from 'lucide-react';
import { Button } from './ui/button';

interface CopyButtonProps {
  content: string;
  label?: string;
  variant?: 'default' | 'outline' | 'secondary' | 'ghost';
  disabled?: boolean;
}

export function CopyButton({
  content,
  label = 'Copy',
  variant = 'outline',
  disabled = false,
}: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error('Failed to copy:', error);
    }
  };

  return (
    <Button
      onClick={handleCopy}
      variant={variant}
      disabled={disabled}
    >
      {copied ? (
        <>
          <Check className="h-4 w-4" />
          Copied
        </>
      ) : (
        <>
          <Copy className="h-4 w-4" />
          {label}
        </>
      )}
    </Button>
  );
}
