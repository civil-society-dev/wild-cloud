import { AlertCircle, CheckCircle, Info } from 'lucide-react';
import type { Message as MessageType } from '../types';
import { cn } from '@/lib/utils';

interface MessageProps {
  message?: MessageType;
}

export const Message = ({ message }: MessageProps) => {
  if (!message) return null;
  
  const getIcon = () => {
    switch (message.type) {
      case 'error':
        return <AlertCircle className="h-4 w-4" />;
      case 'success':
        return <CheckCircle className="h-4 w-4" />;
      default:
        return <Info className="h-4 w-4" />;
    }
  };

  const getVariantStyles = () => {
    switch (message.type) {
      case 'error':
        return 'border-destructive/50 text-destructive bg-destructive/10';
      case 'success':
        return 'border-green-500/50 text-green-700 bg-green-50 dark:bg-green-950 dark:text-green-400';
      default:
        return 'border-blue-500/50 text-blue-700 bg-blue-50 dark:bg-blue-950 dark:text-blue-400';
    }
  };
  
  return (
    <div className={cn(
      'flex items-center gap-2 p-3 rounded-md border text-sm',
      getVariantStyles()
    )}>
      {getIcon()}
      <span>{message.message}</span>
    </div>
  );
};