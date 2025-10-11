import { useEffect, useState } from 'react';
import { useConfig } from './hooks';
import {
  Advanced,
  ErrorBoundary
} from './components';
import { CloudComponent } from './components/CloudComponent';
import { CentralComponent } from './components/CentralComponent';
import { DnsComponent } from './components/DnsComponent';
import { DhcpComponent } from './components/DhcpComponent';
import { PxeComponent } from './components/PxeComponent';
import { ClusterNodesComponent } from './components/ClusterNodesComponent';
import { ClusterServicesComponent } from './components/ClusterServicesComponent';
import { AppsComponent } from './components/AppsComponent';
import { AppSidebar } from './components/AppSidebar';
import { SidebarProvider, SidebarInset, SidebarTrigger } from './components/ui/sidebar';
import type { Phase, Tab } from './components/AppSidebar';

function App() {
  const [currentTab, setCurrentTab] = useState<Tab>('cloud');
  const [completedPhases, setCompletedPhases] = useState<Phase[]>([]);

  const { config } = useConfig();

  // Update phase state from config when it changes
  useEffect(() => {
    console.log('Config changed:', config);
    console.log('config?.wildcloud:', config?.wildcloud);
    if (config?.wildcloud?.currentPhase) {
      console.log('Setting currentTab to:', config.wildcloud.currentPhase);
      setCurrentTab(config.wildcloud.currentPhase as Phase);
    }
    if (config?.wildcloud?.completedPhases) {
      console.log('Setting completedPhases to:', config.wildcloud.completedPhases);
      setCompletedPhases(config.wildcloud.completedPhases as Phase[]);
    }
  }, [config]);

  const handlePhaseComplete = (phase: Phase) => {
    if (!completedPhases.includes(phase)) {
      setCompletedPhases(prev => [...prev, phase]);
    }
    
    // Auto-advance to next phase (excluding advanced)
    const phases: Phase[] = ['setup', 'infrastructure', 'cluster', 'apps'];
    const currentIndex = phases.indexOf(phase);
    if (currentIndex < phases.length - 1) {
      setCurrentTab(phases[currentIndex + 1]);
    }
  };

  const renderCurrentTab = () => {
    switch (currentTab) {
      case 'cloud':
        return (
          <ErrorBoundary>
            <CloudComponent />
          </ErrorBoundary>
        );
      case 'central':
        return (
          <ErrorBoundary>
            <CentralComponent />
          </ErrorBoundary>
        );
      case 'dns':
        return (
          <ErrorBoundary>
            <DnsComponent />
          </ErrorBoundary>
        );
      case 'dhcp':
        return (
          <ErrorBoundary>
            <DhcpComponent />
          </ErrorBoundary>
        );
      case 'pxe':
        return (
          <ErrorBoundary>
            <PxeComponent />
          </ErrorBoundary>
        );
      case 'setup':
      case 'infrastructure':
        return (
          <ErrorBoundary>
            <ClusterNodesComponent onComplete={() => handlePhaseComplete('infrastructure')} />
          </ErrorBoundary>
        );
      case 'cluster':
        return (
          <ErrorBoundary>
            <ClusterServicesComponent onComplete={() => handlePhaseComplete('cluster')} />
          </ErrorBoundary>
        );
      case 'apps':
        return (
          <ErrorBoundary>
            <AppsComponent onComplete={() => handlePhaseComplete('apps')} />
          </ErrorBoundary>
        );
      case 'advanced':
        return (
          <ErrorBoundary>
            <Advanced />
          </ErrorBoundary>
        );
      default:
        return (
          <ErrorBoundary>
            <CloudComponent />
          </ErrorBoundary>
        );
    }
  };

  return (
    <SidebarProvider>
      <AppSidebar
        currentTab={currentTab}
        onTabChange={setCurrentTab}
        completedPhases={completedPhases}
      />
      <SidebarInset>
        <header className="flex h-16 shrink-0 items-center gap-2 px-4">
          <SidebarTrigger className="-ml-1" />
          <div className="flex items-center gap-2">
            <h1 className="text-lg font-semibold">Dashboard</h1>
          </div>
        </header>
        <div className="flex flex-1 flex-col gap-4 p-4">
          {renderCurrentTab()}
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}

export default App;