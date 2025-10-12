import { NavLink, useParams } from 'react-router';
import { Server, Play, Container, AppWindow, Settings, CloudLightning, Sun, Moon, Monitor, ChevronDown, Globe, Wifi, HardDrive, Usb } from 'lucide-react';
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

export function AppSidebar() {
  const { theme, setTheme } = useTheme();
  const { instanceId } = useParams<{ instanceId: string }>();

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

  // If no instanceId, we're not in an instance context
  if (!instanceId) {
    return null;
  }

  return (
    <Sidebar variant="sidebar" collapsible="icon">
      <SidebarHeader>
        <div className="flex items-center gap-2 px-2">
          <div className="p-1 bg-primary/10 rounded-lg">
            <CloudLightning className="h-6 w-6 text-primary" />
          </div>
          <div className="group-data-[collapsible=icon]:hidden">
            <h2 className="text-lg font-bold text-foreground">Wild Cloud</h2>
            <p className="text-sm text-muted-foreground">{instanceId}</p>
          </div>
        </div>
      </SidebarHeader>

      <SidebarContent>
        <SidebarMenu>
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

          <SidebarMenuItem>
            <NavLink to={`/instances/${instanceId}/cloud`}>
              {({ isActive }) => (
                <SidebarMenuButton
                  isActive={isActive}
                  tooltip="Configure cloud settings and domains"
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
                  <span className="truncate">Cloud</span>
                </SidebarMenuButton>
              )}
            </NavLink>
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
                    <SidebarMenuSubButton asChild>
                      <NavLink to={`/instances/${instanceId}/infrastructure`}>
                        <div className="p-1 rounded-md">
                          <Play className="h-4 w-4" />
                        </div>
                        <span className="truncate">Cluster Nodes</span>
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
                      </NavLink>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                </SidebarMenuSub>
              </CollapsibleContent>
            </SidebarMenuItem>
          </Collapsible>

          <SidebarMenuItem>
            <SidebarMenuButton asChild tooltip="Install and manage applications">
              <NavLink to={`/instances/${instanceId}/apps`}>
                <div className="p-1 rounded-md">
                  <AppWindow className="h-4 w-4" />
                </div>
                <span className="truncate">Apps</span>
              </NavLink>
            </SidebarMenuButton>
          </SidebarMenuItem>

          <SidebarMenuItem>
            <SidebarMenuButton asChild tooltip="Advanced settings and system configuration">
              <NavLink to={`/instances/${instanceId}/advanced`}>
                <div className="p-1 rounded-md">
                  <Settings className="h-4 w-4" />
                </div>
                <span className="truncate">Advanced</span>
              </NavLink>
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
