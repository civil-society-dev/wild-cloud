import { useEffect } from 'react';
import { Outlet, useParams, Navigate } from 'react-router';
import { useInstanceContext } from '../hooks/useInstanceContext';
import { AppSidebar } from '../components/AppSidebar';
import { SidebarProvider, SidebarInset, SidebarTrigger } from '../components/ui/sidebar';

export function InstanceLayout() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const { setCurrentInstance } = useInstanceContext();

  useEffect(() => {
    if (instanceId) {
      setCurrentInstance(instanceId);
    }
    return () => {
      // Don't clear instance on unmount - let it persist
      // This allows the instance to stay selected when navigating
    };
  }, [instanceId, setCurrentInstance]);

  if (!instanceId) {
    return <Navigate to="/" replace />;
  }

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-16 shrink-0 items-center gap-2 px-4">
          <SidebarTrigger className="-ml-1" />
          <div className="flex items-center gap-2">
            <h1 className="text-lg font-semibold">Wild Cloud</h1>
          </div>
        </header>
        <div className="flex flex-1 flex-col gap-4 p-4">
          <Outlet />
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
