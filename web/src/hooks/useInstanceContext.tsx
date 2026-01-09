import { useState, createContext, useContext, ReactNode } from 'react';

interface InstanceContextValue {
  currentInstance: string | null;
  setCurrentInstance: (name: string | null) => void;
}

const InstanceContext = createContext<InstanceContextValue | undefined>(undefined);

export function InstanceProvider({ children }: { children: ReactNode }) {
  const [currentInstance, setCurrentInstanceState] = useState<string | null>(
    () => localStorage.getItem('currentInstance')
  );

  const setCurrentInstance = (name: string | null) => {
    setCurrentInstanceState(name);
    if (name) {
      localStorage.setItem('currentInstance', name);
    } else {
      localStorage.removeItem('currentInstance');
    }
  };

  return (
    <InstanceContext.Provider value={{ currentInstance, setCurrentInstance }}>
      {children}
    </InstanceContext.Provider>
  );
}

export function useInstanceContext() {
  const context = useContext(InstanceContext);
  if (context === undefined) {
    throw new Error('useInstanceContext must be used within an InstanceProvider');
  }
  return context;
}
