import { useState } from 'react';
import { Eye, EyeOff } from 'lucide-react';
import { Input } from './ui/input';
import { Button } from './ui/button';
import { cn } from '@/lib/utils';

interface SecretInputProps {
  value: string;
  onChange?: (value: string) => void;
  placeholder?: string;
  readOnly?: boolean;
  className?: string;
}

export function SecretInput({
  value,
  onChange,
  placeholder = '••••••••',
  readOnly = false,
  className,
}: SecretInputProps) {
  const [revealed, setRevealed] = useState(false);

  // If no onChange handler provided, the field should be read-only
  const isReadOnly = readOnly || !onChange;

  return (
    <div className="relative flex items-center gap-2">
      <Input
        type={revealed ? 'text' : 'password'}
        value={value}
        onChange={onChange ? (e) => onChange(e.target.value) : undefined}
        placeholder={placeholder}
        readOnly={isReadOnly}
        className={cn('pr-10', className)}
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="absolute right-0 h-full hover:bg-transparent"
        onClick={() => setRevealed(!revealed)}
        aria-label={revealed ? 'Hide value' : 'Show value'}
      >
        {revealed ? (
          <EyeOff className="h-4 w-4 text-muted-foreground" />
        ) : (
          <Eye className="h-4 w-4 text-muted-foreground" />
        )}
      </Button>
    </div>
  );
}
