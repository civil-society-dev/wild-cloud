import { NavLink, useParams } from 'react-router';
import { Server, Container, AppWindow, Settings, CloudLightning, Sun, Moon, Monitor, ChevronDown, Globe, Usb, Download, CheckCircle, Archive, Cpu, HardDrive, TerminalSquare, Cog, LayoutDashboard, Lock } from 'lucide-react';
import { cn } from '../lib/utils';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
} from './ui/sidebar';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from './ui/collapsible';
import { useTheme } from '../contexts/ThemeContext';
import { InstanceSwitcher } from './InstanceSwitcher';
import { useSetupStatus } from '../services/api';

export function AppSidebar() {
  const { theme, setTheme } = useTheme();
  const { instanceId } = useParams<{ instanceId: string }>();
  const { data: setupStatus } = useSetupStatus(instanceId || '', {
    enabled: !!instanceId,
    refetchInterval: 10000, // Poll every 10s in sidebar
  });

  const cycleTheme = () => {
    if (theme === 'light') {
      setTheme('dark');
    } else if (theme === 'dark') {
      setTheme('system');
    } else {
      setTheme('light');
    }
  };

  const getThemeIcon = () => {
    switch (theme) {
      case 'light':
        return <Sun className="h-4 w-4" />;
      case 'dark':
        return <Moon className="h-4 w-4" />;
      default:
        return <Monitor className="h-4 w-4" />;
    }
  };

  const getThemeLabel = () => {
    switch (theme) {
      case 'light':
        return 'Light mode';
      case 'dark':
        return 'Dark mode';
      default:
        return 'System theme';
    }
  };

  const getPhaseStatus = (phase: string) => {
    if (!setupStatus) return { available: true, complete: false };
    const check = setupStatus.phaseChecks[phase];
    return {
      available: check?.available ?? true,
      complete: check?.complete ?? false,
    };
  };

  const renderPhaseIndicator = (phase: string) => {
    const { available, complete } = getPhaseStatus(phase);
    if (complete) {
      return <></>;
    }
    if (!available) {
      return <Lock className="h-3 w-3 text-muted-foreground ml-auto" />;
    }
    return null;
  };

  // If no instanceId, we're not in an instance context
  if (!instanceId) {
    return null;
  }

  return (
    <Sidebar variant="sidebar" collapsible="icon">
      <SidebarHeader>
        <div className="flex items-center gap-2 px-2 pb-2">
          <div className="p-1 bg-primary/10 rounded-lg">
            <CloudLightning className="h-6 w-6 text-primary" />
          </div>
          <div className="group-data-[collapsible=icon]:hidden">
            <h2 className="text-lg font-bold text-foreground">Wild Cloud</h2>
          </div>
        </div>
      </SidebarHeader>

      <SidebarContent>
        <SidebarMenu>
          <Collapsible defaultOpen className="group/collapsible">
            <SidebarMenuItem>
              <CollapsibleTrigger asChild>
                <SidebarMenuButton>
                  <Server className="h-4 w-4" />
                  Central
                  <ChevronDown className="ml-auto transition-transform group-data-[state=open]/collapsible:rotate-180" />
                </SidebarMenuButton>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <SidebarMenuSub>
                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/central`} className={({ isActive }) => isActive ? "data-[active=true]" : ""}>
                        <div className="p-1 rounded-md">
                          <Server className="h-4 w-4" />
                        </div>
                        <span className="truncate">Central</span>
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/dns`}>
                        <div className="p-1 rounded-md">
                          <Globe className="h-4 w-4" />
                        </div>
                        <span className="truncate">DNS</span>
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/iso`}>
                        <div className="p-1 rounded-md">
                          <Usb className="h-4 w-4" />
                        </div>
                        <span className="truncate">ISO / USB</span>
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  {/* <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/dhcp`}>
                        <div className="p-1 rounded-md">
                          <Wifi className="h-4 w-4" />
                        </div>
                        <span className="truncate">DHCP</span>
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/pxe`}>
                        <div className="p-1 rounded-md">
                          <HardDrive className="h-4 w-4" />
                        </div>
                        <span className="truncate">PXE</span>
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem> */}
                </SidebarMenuSub>
              </CollapsibleContent>
            </SidebarMenuItem>
          </Collapsible>

          {/* Instance Selector and Configuration */}
          <SidebarMenuItem>
            <div className="px-2 py-2">
              <div className="flex items-center gap-2">
                <div className="flex-1 min-w-0">
                  <InstanceSwitcher />
                </div>
                <NavLink to={`/instances/${instanceId}/cloud`}>
                  {({ isActive }) => (
                    <SidebarMenuButton
                      isActive={isActive}
                      tooltip="Configure instance settings"
                      size="sm"
                      className="h-8 w-8 p-0"
                    >
                      <Settings className="h-4 w-4" />
                    </SidebarMenuButton>
                  )}
                </NavLink>
              </div>
            </div>
          </SidebarMenuItem>

          <SidebarMenuItem>
            <NavLink to={`/instances/${instanceId}/dashboard`}>
              {({ isActive }) => (
                <SidebarMenuButton
                  isActive={isActive}
                  tooltip="Instance dashboard and overview"
                >
                  <div className={cn(
                    "p-1 rounded-md",
                    isActive && "bg-primary/10"
                  )}>
                    <CloudLightning className={cn(
                      "h-4 w-4",
                      isActive && "text-primary",
                      !isActive && "text-muted-foreground"
                    )} />
                  </div>
                  <span className="truncate">Dashboard</span>
                </SidebarMenuButton>
              )}
            </NavLink>
          </SidebarMenuItem>

          <Collapsible defaultOpen className="group/collapsible">
            <SidebarMenuItem>
              <CollapsibleTrigger asChild>
                <SidebarMenuButton>
                  <Container className="h-4 w-4" />
                  Cluster
                  <ChevronDown className="ml-auto transition-transform group-data-[state=open]/collapsible:rotate-180" />
                </SidebarMenuButton>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <SidebarMenuSub>
                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/control`}>
                        <div className="p-1 rounded-md">
                          <Cpu className="h-4 w-4" />
                        </div>
                        <span className="truncate">Control Nodes</span>
                        {renderPhaseIndicator('control-nodes')}
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/worker`}>
                        <div className="p-1 rounded-md">
                          <HardDrive className="h-4 w-4" />
                        </div>
                        <span className="truncate">Worker Nodes</span>
                        {renderPhaseIndicator('control-nodes')}
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/cluster`}>
                        <div className="p-1 rounded-md">
                          <Container className="h-4 w-4" />
                        </div>
                        <span className="truncate">Cluster Services</span>
                        {renderPhaseIndicator('cluster-services')}
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                </SidebarMenuSub>
              </CollapsibleContent>
            </SidebarMenuItem>
          </Collapsible>

          <Collapsible defaultOpen className="group/collapsible">
            <SidebarMenuItem>
              <CollapsibleTrigger asChild>
                <SidebarMenuButton>
                  <AppWindow className="h-4 w-4" />
                  Apps
                  <ChevronDown className="ml-auto transition-transform group-data-[state=open]/collapsible:rotate-180" />
                </SidebarMenuButton>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <SidebarMenuSub>
                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/apps/available`}>
                        <div className="p-1 rounded-md">
                          <Download className="h-4 w-4" />
                        </div>
                        <span className="truncate">Available</span>
                        {renderPhaseIndicator('apps')}
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/apps/installed`}>
                        <div className="p-1 rounded-md">
                          <CheckCircle className="h-4 w-4" />
                        </div>
                        <span className="truncate">Installed</span>
                        {renderPhaseIndicator('apps')}
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                </SidebarMenuSub>
              </CollapsibleContent>
            </SidebarMenuItem>
          </Collapsible>

          <SidebarMenuItem>
            <SidebarMenuButton asChild tooltip="Manage backups and recovery">
              <NavLink to={`/instances/${instanceId}/backups`}>
                {({ isActive }) => (
                  <>
                    <div className={cn(
                      "p-1 rounded-md",
                      isActive && "bg-primary/10"
                    )}>
                      <Archive className={cn(
                        "h-4 w-4",
                        isActive && "text-primary",
                        !isActive && "text-muted-foreground"
                      )} />
                    </div>
                    <span className="truncate">Backups</span>
                  </>
                )}
              </NavLink>
            </SidebarMenuButton>
          </SidebarMenuItem>

          <Collapsible defaultOpen className="group/collapsible">
            <SidebarMenuItem>
              <CollapsibleTrigger asChild>
                <SidebarMenuButton>
                  <Settings className="h-4 w-4" />
                  Advanced
                  <ChevronDown className="ml-auto transition-transform group-data-[state=open]/collapsible:rotate-180" />
                </SidebarMenuButton>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <SidebarMenuSub>
                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/advanced/config`}>
                        <div className="p-1 rounded-md">
                          <Cog className="h-4 w-4" />
                        </div>
                        <span className="truncate">Configuration</span>
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/advanced/terminal`}>
                        <div className="p-1 rounded-md">
                          <TerminalSquare className="h-4 w-4" />
                        </div>
                        <span className="truncate">Terminal</span>
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/advanced/k8s-dashboard`}>
                        <div className="p-1 rounded-md">
                          <LayoutDashboard className="h-4 w-4" />
                        </div>
                        <span className="truncate">K8s Dashboard</span>
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                </SidebarMenuSub>
              </CollapsibleContent>
            </SidebarMenuItem>
          </Collapsible>
        </SidebarMenu>
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              onClick={cycleTheme}
              tooltip={`Current: ${getThemeLabel()}. Click to cycle themes.`}
            >
              {getThemeIcon()}
              <span>{getThemeLabel()}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
      <SidebarRail/>
    </Sidebar>
  );
}
