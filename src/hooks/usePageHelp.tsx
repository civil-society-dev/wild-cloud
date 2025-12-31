import { useEffect } from 'react';
import { useHelp } from '../contexts/HelpContext';
import type { ReactNode } from 'react';

interface PageHelpOptions {
  title: string;
  description: ReactNode;
  icon?: ReactNode;
  color?: string;
  actions?: ReactNode;
}

export function usePageHelp(options: PageHelpOptions | null) {
  const { setHelpContent } = useHelp();

  useEffect(() => {
    setHelpContent(options);

    // Clear help content when component unmounts
    return () => {
      setHelpContent(null);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Only run on mount/unmount
}
