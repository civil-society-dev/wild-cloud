import { createContext, useContext, useState } from 'react';
import type { ReactNode } from 'react';

interface HelpContent {
  title: string;
  description: ReactNode;
  icon?: ReactNode;
  color?: string;
  actions?: ReactNode;
}

interface HelpContextType {
  helpContent: HelpContent | null;
  setHelpContent: (content: HelpContent | null) => void;
  isHelpOpen: boolean;
  setIsHelpOpen: (open: boolean) => void;
}

const HelpContext = createContext<HelpContextType | undefined>(undefined);

export function HelpProvider({ children }: { children: ReactNode }) {
  const [helpContent, setHelpContent] = useState<HelpContent | null>(null);
  const [isHelpOpen, setIsHelpOpen] = useState(false);

  return (
    <HelpContext.Provider value={{ helpContent, setHelpContent, isHelpOpen, setIsHelpOpen }}>
      {children}
    </HelpContext.Provider>
  );
}

export function useHelp() {
  const context = useContext(HelpContext);
  if (context === undefined) {
    throw new Error('useHelp must be used within a HelpProvider');
  }
  return context;
}
