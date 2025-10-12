import { Download } from 'lucide-react';
import { Button } from './ui/button';

interface DownloadButtonProps {
  content: string;
  filename: string;
  label?: string;
  variant?: 'default' | 'outline' | 'secondary' | 'ghost';
  disabled?: boolean;
}

export function DownloadButton({
  content,
  filename,
  label = 'Download',
  variant = 'default',
  disabled = false,
}: DownloadButtonProps) {
  const handleDownload = () => {
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <Button
      onClick={handleDownload}
      variant={variant}
      disabled={disabled}
    >
      <Download className="h-4 w-4" />
      {label}
    </Button>
  );
}
