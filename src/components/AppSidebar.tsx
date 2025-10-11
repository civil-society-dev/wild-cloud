import { CheckCircle, Lock, Server, Play, Container, AppWindow, Settings, CloudLightning, Sun, Moon, Monitor, ChevronDown, Globe, Wifi, HardDrive } from 'lucide-react';
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

export type Phase = 'setup' | 'infrastructure' | 'cluster' | 'apps';
export type Tab = Phase | 'advanced' | 'cloud' | 'central' | 'dns' | 'dhcp' | 'pxe';

interface AppSidebarProps {
  currentTab: Tab;
  onTabChange: (tab: Tab) => void;
  completedPhases: Phase[];
}


export function AppSidebar({ currentTab, onTabChange, completedPhases }: AppSidebarProps) {
  const { theme, setTheme } = useTheme();

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

  const getTabStatus = (tab: Tab) => {
    // Non-phase tabs (like advanced and cloud) are always available
    if (tab === 'advanced' || tab === 'cloud') {
      return 'available';
    }
    
    // Central sub-tabs are available if setup phase is available or completed
    if (tab === 'central' || tab === 'dns' || tab === 'dhcp' || tab === 'pxe') {
      if (completedPhases.includes('setup')) {
        return 'completed';
      }
      return 'available';
    }
    
    // For phase tabs, check completion status
    if (completedPhases.includes(tab as Phase)) {
      return 'completed';
    }
    
    // Allow access to the first phase always
    if (tab === 'setup') {
      return 'available';
    }
    
    // Allow access to the next phase if the previous phase is completed
    if (tab === 'infrastructure' && completedPhases.includes('setup')) {
      return 'available';
    }
    
    if (tab === 'cluster' && completedPhases.includes('infrastructure')) {
      return 'available';
    }
    
    if (tab === 'apps' && completedPhases.includes('cluster')) {
      return 'available';
    }
    
    return 'locked';
  };

  return (
    <Sidebar variant="sidebar" collapsible="icon">
      <SidebarHeader>
        <div className="flex items-center gap-2 px-2">
          <div className="p-1 bg-primary/10 rounded-lg">
            <CloudLightning className="h-6 w-6 text-primary" />
          </div>
          <div className="group-data-[collapsible=icon]:hidden">
            <h2 className="text-lg font-bold text-foreground">Wild Cloud</h2>
            <p className="text-sm text-muted-foreground">Central</p>
          </div>
        </div>
      </SidebarHeader>
      
      <SidebarContent>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              isActive={currentTab === 'cloud'}
              onClick={() => {
                const status = getTabStatus('cloud');
                if (status !== 'locked') onTabChange('cloud');
              }}
              disabled={getTabStatus('cloud') === 'locked'}
              tooltip="Configure cloud settings and domains"
              className={cn(
                "transition-colors",
                getTabStatus('cloud') === 'locked' && "opacity-50 cursor-not-allowed"
              )}
            >
              <div className={cn(
                "p-1 rounded-md",
                currentTab === 'cloud' && "bg-primary/10",
                getTabStatus('cloud') === 'locked' && "bg-muted"
              )}>
                <CloudLightning className={cn(
                  "h-4 w-4",
                  currentTab === 'cloud' && "text-primary",
                  currentTab !== 'cloud' && "text-muted-foreground"
                )} />
              </div>
              <span className="truncate">Cloud</span>
            </SidebarMenuButton>
          </SidebarMenuItem>

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
                    <SidebarMenuSubButton
                      isActive={currentTab === 'central'}
                      onClick={() => {
                        const status = getTabStatus('central');
                        if (status !== 'locked') onTabChange('central');
                      }}
                      className={cn(
                        "transition-colors",
                        getTabStatus('central') === 'locked' && "opacity-50 cursor-not-allowed"
                      )}
                    >
                      <div className={cn(
                        "p-1 rounded-md",
                        currentTab === 'central' && "bg-primary/10",
                        getTabStatus('central') === 'locked' && "bg-muted"
                      )}>
                        <Server className={cn(
                          "h-4 w-4",
                          currentTab === 'central' && "text-primary",
                          currentTab !== 'central' && "text-muted-foreground"
                        )} />
                      </div>
                      <span className="truncate">Central</span>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton
                      isActive={currentTab === 'dns'}
                      onClick={() => {
                        const status = getTabStatus('dns');
                        if (status !== 'locked') onTabChange('dns');
                      }}
                      className={cn(
                        "transition-colors",
                        getTabStatus('dns') === 'locked' && "opacity-50 cursor-not-allowed"
                      )}
                    >
                      <div className={cn(
                        "p-1 rounded-md",
                        currentTab === 'dns' && "bg-primary/10",
                        getTabStatus('dns') === 'locked' && "bg-muted"
                      )}>
                        <Globe className={cn(
                          "h-4 w-4",
                          currentTab === 'dns' && "text-primary",
                          currentTab !== 'dns' && "text-muted-foreground"
                        )} />
                      </div>
                      <span className="truncate">DNS</span>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton
                      isActive={currentTab === 'dhcp'}
                      onClick={() => {
                        const status = getTabStatus('dhcp');
                        if (status !== 'locked') onTabChange('dhcp');
                      }}
                      className={cn(
                        "transition-colors",
                        getTabStatus('dhcp') === 'locked' && "opacity-50 cursor-not-allowed"
                      )}
                    >
                      <div className={cn(
                        "p-1 rounded-md",
                        currentTab === 'dhcp' && "bg-primary/10",
                        getTabStatus('dhcp') === 'locked' && "bg-muted"
                      )}>
                        <Wifi className={cn(
                          "h-4 w-4",
                          currentTab === 'dhcp' && "text-primary",
                          currentTab !== 'dhcp' && "text-muted-foreground"
                        )} />
                      </div>
                      <span className="truncate">DHCP</span>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton
                      isActive={currentTab === 'pxe'}
                      onClick={() => {
                        const status = getTabStatus('pxe');
                        if (status !== 'locked') onTabChange('pxe');
                      }}
                      className={cn(
                        "transition-colors",
                        getTabStatus('pxe') === 'locked' && "opacity-50 cursor-not-allowed"
                      )}
                    >
                      <div className={cn(
                        "p-1 rounded-md",
                        currentTab === 'pxe' && "bg-primary/10",
                        getTabStatus('pxe') === 'locked' && "bg-muted"
                      )}>
                        <HardDrive className={cn(
                          "h-4 w-4",
                          currentTab === 'pxe' && "text-primary",
                          currentTab !== 'pxe' && "text-muted-foreground"
                        )} />
                      </div>
                      <span className="truncate">PXE</span>
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
                  <Container className="h-4 w-4" />
                  Cluster
                  <ChevronDown className="ml-auto transition-transform group-data-[state=open]/collapsible:rotate-180" />
                </SidebarMenuButton>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <SidebarMenuSub>
                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton
                      isActive={currentTab === 'infrastructure'}
                      onClick={() => {
                        const status = getTabStatus('infrastructure');
                        if (status !== 'locked') onTabChange('infrastructure');
                      }}
                      className={cn(
                        "transition-colors",
                        getTabStatus('infrastructure') === 'locked' && "opacity-50 cursor-not-allowed"
                      )}
                    >
                      <div className={cn(
                        "p-1 rounded-md",
                        currentTab === 'infrastructure' && "bg-primary/10",
                        getTabStatus('infrastructure') === 'locked' && "bg-muted"
                      )}>
                        <Play className={cn(
                          "h-4 w-4",
                          currentTab === 'infrastructure' && "text-primary",
                          currentTab !== 'infrastructure' && "text-muted-foreground"
                        )} />
                      </div>
                      <span className="truncate">Cluster Nodes</span>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>

                  <SidebarMenuSubItem>
                    <SidebarMenuSubButton
                      isActive={currentTab === 'cluster'}
                      onClick={() => {
                        const status = getTabStatus('cluster');
                        if (status !== 'locked') onTabChange('cluster');
                      }}
                      className={cn(
                        "transition-colors",
                        getTabStatus('cluster') === 'locked' && "opacity-50 cursor-not-allowed"
                      )}
                    >
                      <div className={cn(
                        "p-1 rounded-md",
                        currentTab === 'cluster' && "bg-primary/10",
                        getTabStatus('cluster') === 'locked' && "bg-muted"
                      )}>
                        <Container className={cn(
                          "h-4 w-4",
                          currentTab === 'cluster' && "text-primary",
                          currentTab !== 'cluster' && "text-muted-foreground"
                        )} />
                      </div>
                      <span className="truncate">Cluster Services</span>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                </SidebarMenuSub>
              </CollapsibleContent>
            </SidebarMenuItem>
          </Collapsible>

          <SidebarMenuItem>
            <SidebarMenuButton
              isActive={currentTab === 'apps'}
              onClick={() => {
                const status = getTabStatus('apps');
                if (status !== 'locked') onTabChange('apps');
              }}
              disabled={getTabStatus('apps') === 'locked'}
              tooltip="Install and manage applications"
              className={cn(
                "transition-colors",
                getTabStatus('apps') === 'locked' && "opacity-50 cursor-not-allowed"
              )}
            >
              <div className={cn(
                "p-1 rounded-md",
                currentTab === 'apps' && "bg-primary/10",
                getTabStatus('apps') === 'locked' && "bg-muted"
              )}>
                <AppWindow className={cn(
                  "h-4 w-4",
                  currentTab === 'apps' && "text-primary",
                  currentTab !== 'apps' && "text-muted-foreground"
                )} />
              </div>
              <span className="truncate">Apps</span>
            </SidebarMenuButton>
          </SidebarMenuItem>

          <SidebarMenuItem>
            <SidebarMenuButton
              isActive={currentTab === 'advanced'}
              onClick={() => {
                const status = getTabStatus('advanced');
                if (status !== 'locked') onTabChange('advanced');
              }}
              disabled={getTabStatus('advanced') === 'locked'}
              tooltip="Advanced settings and system configuration"
              className={cn(
                "transition-colors",
                getTabStatus('advanced') === 'locked' && "opacity-50 cursor-not-allowed"
              )}
            >
              <div className={cn(
                "p-1 rounded-md",
                currentTab === 'advanced' && "bg-primary/10",
                getTabStatus('advanced') === 'locked' && "bg-muted"
              )}>
                <Settings className={cn(
                  "h-4 w-4",
                  currentTab === 'advanced' && "text-primary",
                  currentTab !== 'advanced' && "text-muted-foreground"
                )} />
              </div>
              <span className="truncate">Advanced</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
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