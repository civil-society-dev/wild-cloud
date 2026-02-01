import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import {
  AppWindow,
  Database,
  Globe,
  Shield,
  BarChart3,
  MessageSquare,
  Search,
  ExternalLink,
  CheckCircle,
  AlertCircle,
  Download,
  Trash2,
  BookOpen,
  Loader2,
  Archive,
  RotateCcw,
  Settings,
  Eye,
  Activity,
  FileText,
} from 'lucide-react';
import { useInstanceContext } from '../hooks/useInstanceContext';
import { useAvailableApps, useDeployedApps, useAppBackups } from '../hooks/useApps';
import { BackupRestoreModal } from './BackupRestoreModal';
import { AppConfigDialog } from './apps/AppConfigDialog';
import { AppOverviewDialog } from './apps/AppOverviewDialog';
import { AppConfigurationDialog } from './apps/AppConfigurationDialog';
import { AppStatusDialog } from './apps/AppStatusDialog';
import { AppLogsDialog } from './apps/AppLogsDialog';
import type { App } from '../services/api';
import { appsApi } from '../services/api';
import { usePageHelp } from '../hooks/usePageHelp';

interface MergedApp extends App {
  deploymentStatus?: 'added' | 'deployed';
  url?: string;
}

type TabView = 'available' | 'installed';

export function AppsComponent() {
  const location = useLocation();
  const navigate = useNavigate();
  const { currentInstance } = useInstanceContext();
  const { data: availableAppsData, isLoading: loadingAvailable, error: availableError } = useAvailableApps();
  const {
    apps: deployedApps,
    isLoading: loadingDeployed,
    error: deployedError,
    addApp,
    isAdding,
    addingAppNames,
    deployApp,
    deployingAppNames,
    deleteApp,
    deletingAppNames
  } = useDeployedApps(currentInstance);

  // Determine active tab from URL path
  const activeTab: TabView = location.pathname.endsWith('/installed') ? 'installed' : 'available';
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [configDialogOpen, setConfigDialogOpen] = useState(false);
  const [selectedAppForConfig, setSelectedAppForConfig] = useState<App | null>(null);
  const [backupModalOpen, setBackupModalOpen] = useState(false);
  const [restoreModalOpen, setRestoreModalOpen] = useState(false);
  const [selectedAppForBackup, setSelectedAppForBackup] = useState<string | null>(null);
  const [overviewDialogOpen, setOverviewDialogOpen] = useState(false);
  const [configurationDialogOpen, setConfigurationDialogOpen] = useState(false);
  const [statusDialogOpen, setStatusDialogOpen] = useState(false);
  const [logsDialogOpen, setLogsDialogOpen] = useState(false);
  const [selectedAppForDialog, setSelectedAppForDialog] = useState<string | null>(null);

  usePageHelp({
    title: 'What are Apps in your Personal Cloud?',
    description: (
      <>
        <p className="mb-3 leading-relaxed">
          Apps are the useful programs that make your personal cloud valuable - like having a personal Netflix
          (media server), Google Drive (file storage), or Gmail (email server) running on your own hardware.
          Instead of relying on big tech companies, you control your data and services.
        </p>
        <p className="text-sm">
          Your cluster can run databases, web servers, photo galleries, password managers, backup services, and much more.
          Each app runs in its own secure container, so they don't interfere with each other and can be easily managed.
        </p>
      </>
    ),
    icon: <BookOpen className="h-6 w-6 text-pink-600 dark:text-pink-400" />,
    color: 'bg-gradient-to-r from-pink-50 to-rose-50 dark:from-pink-950/20 dark:to-rose-950/20',
    actions: (
      <Button
        variant="outline"
        size="sm"
        className="text-pink-700 border-pink-300 hover:bg-pink-100 dark:text-pink-300 dark:border-pink-700 dark:hover:bg-pink-900/20"
      >
        <ExternalLink className="h-4 w-4 mr-2" />
        Learn more about self-hosted applications
      </Button>
    ),
  });

  // Fetch backups for the selected app
  const {
    backups,
    isLoading: backupsLoading,
    backup: createBackup,
    isBackingUp,
    restore: restoreBackup,
    isRestoring,
  } = useAppBackups(currentInstance, selectedAppForBackup);

  // Merge available and deployed apps with URL from deployment
  const availableAppsMap = new Map((availableAppsData?.apps || []).map(app => [app.name, app]));

  // First, add all available apps with their deployment status
  const applications: MergedApp[] = Array.from(availableAppsMap.values()).map(app => {
    const deployedApp = deployedApps.find(d => d.name === app.name);
    return {
      ...app,
      deploymentStatus: deployedApp?.status,
      url: deployedApp?.url,
      // Prefer deployed app icon if it exists (allows custom icons in local manifests)
      icon: deployedApp?.icon || app.icon,
    } as MergedApp;
  });

  // Then, add deployed apps that aren't in the available list (custom apps)
  deployedApps.forEach(deployedApp => {
    if (!availableAppsMap.has(deployedApp.name)) {
      const customApp: MergedApp = {
        name: deployedApp.name,
        description: `Custom app: ${deployedApp.name}`,
        version: deployedApp.version || '',
        category: 'custom',
        deploymentStatus: deployedApp.status,
        url: deployedApp.url,
        icon: deployedApp.icon,
      };
      applications.push(customApp);
    }
  });

  const isLoading = loadingAvailable || loadingDeployed;

  // Filter for available apps (not added or deployed)
  const availableApps = applications.filter(app => !app.deploymentStatus).sort((a, b) => a.name.localeCompare(b.name));

  // Filter for installed apps (added or deployed)
  const installedApps = applications.filter(app => app.deploymentStatus).sort((a, b) => a.name.localeCompare(b.name));

  // Count running apps - apps that are deployed (not just added)
  const runningApps = installedApps.filter(app => app.deploymentStatus === 'deployed').length;

  const getStatusIcon = (status?: string) => {
    switch (status) {
      case 'running':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'error':
        return <AlertCircle className="h-5 w-5 text-red-500" />;
      case 'deploying':
        return <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />;
      case 'stopped':
        return <AlertCircle className="h-5 w-5 text-yellow-500" />;
      case 'added':
        return <Settings className="h-5 w-5 text-blue-500" />;
      case 'deployed':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'available':
        return <Download className="h-5 w-5 text-muted-foreground" />;
      default:
        return null;
    }
  };

  const getStatusBadge = (app: MergedApp) => {
    // Determine status: runtime status > deployment status > available
    const status = app.status?.status || app.deploymentStatus || 'available';

    const variants: Record<string, 'secondary' | 'default' | 'success' | 'destructive' | 'warning' | 'outline'> = {
      available: 'secondary',
      added: 'outline',
      deploying: 'default',
      running: 'success',
      error: 'destructive',
      stopped: 'warning',
      deployed: 'outline',
    };

    const labels: Record<string, string> = {
      available: 'Available',
      added: 'Added',
      deploying: 'Deploying',
      running: 'Running',
      error: 'Error',
      stopped: 'Stopped',
      deployed: 'Deployed',
    };

    return (
      <Badge variant={variants[status]}>
        {labels[status] || status}
      </Badge>
    );
  };

  const getCategoryIcon = (category?: string) => {
    switch (category) {
      case 'database':
        return <Database className="h-4 w-4" />;
      case 'web':
        return <Globe className="h-4 w-4" />;
      case 'security':
        return <Shield className="h-4 w-4" />;
      case 'monitoring':
        return <BarChart3 className="h-4 w-4" />;
      case 'communication':
        return <MessageSquare className="h-4 w-4" />;
      case 'storage':
        return <Database className="h-4 w-4" />;
      case 'custom':
        return <Settings className="h-4 w-4" />;
      default:
        return <AppWindow className="h-4 w-4" />;
    }
  };

  // Separate component for app icon with error handling
  const AppIcon = ({ app }: { app: MergedApp }) => {
    const [imageError, setImageError] = useState(false);

    return (
      <div className="h-12 w-12 rounded-lg bg-muted flex items-center justify-center overflow-hidden">
        {app.icon && !imageError ? (
          <img
            src={app.icon}
            alt={app.name}
            className="h-full w-full object-contain p-1"
            onError={() => setImageError(true)}
          />
        ) : (
          getCategoryIcon(app.category)
        )}
      </div>
    );
  };

  const handleAppAction = async (app: MergedApp, action: 'configure' | 'deploy' | 'delete' | 'backup' | 'restore' | 'overview' | 'configuration' | 'status' | 'logs' | 'viewBackups') => {
    if (!currentInstance) return;

    switch (action) {
      case 'configure':
        console.log('[AppsComponent] Configuring app:', {
          name: app.name,
          hasDefaultConfig: !!app.defaultConfig,
          defaultConfigKeys: app.defaultConfig ? Object.keys(app.defaultConfig) : [],
          fullApp: app,
          deploymentStatus: app.deploymentStatus,
        });

        // If app is already added, fetch its current config from the instance
        if (app.deploymentStatus === 'added' || app.deploymentStatus === 'deployed') {
          try {
            const instanceConfig = await appsApi.getConfig(currentInstance, app.name);
            console.log('[AppsComponent] Fetched instance config:', instanceConfig);
            // Merge the instance config into the app object
            setSelectedAppForConfig({
              ...app,
              config: instanceConfig,
            });
          } catch (error) {
            console.error('[AppsComponent] Failed to fetch instance config:', error);
            // Fall back to default config if fetch fails
            setSelectedAppForConfig(app);
          }
        } else {
          // For available apps, use default config
          setSelectedAppForConfig(app);
        }

        setConfigDialogOpen(true);
        break;
      case 'deploy':
        deployApp(app.name);
        break;
      case 'delete':
        if (confirm(`Are you sure you want to delete ${app.name}?`)) {
          deleteApp(app.name);
        }
        break;
      case 'backup':
        setSelectedAppForBackup(app.name);
        setBackupModalOpen(true);
        break;
      case 'restore':
        setSelectedAppForBackup(app.name);
        setRestoreModalOpen(true);
        break;
      case 'overview':
        setSelectedAppForDialog(app.name);
        setOverviewDialogOpen(true);
        break;
      case 'configuration':
        setSelectedAppForDialog(app.name);
        setConfigurationDialogOpen(true);
        break;
      case 'status':
        setSelectedAppForDialog(app.name);
        setStatusDialogOpen(true);
        break;
      case 'logs':
        setSelectedAppForDialog(app.name);
        setLogsDialogOpen(true);
        break;
      case 'viewBackups':
        navigate(`/instances/${currentInstance}/backups?app=${app.name}`);
        break;
    }
  };

  const handleConfigSave = (appName: string, config: Record<string, string>, requiredAppMappings?: Record<string, string>) => {
    if (!selectedAppForConfig) return;

    addApp({
      name: appName,
      config: config,
      requiredAppMappings: requiredAppMappings,
    });

    setConfigDialogOpen(false);
    setSelectedAppForConfig(null);
  };

  const handleBackupConfirm = () => {
    createBackup();
  };

  const handleRestoreConfirm = (backupId?: string) => {
    if (backupId) {
      restoreBackup(backupId);
    }
  };

  const categories = ['all', 'database', 'web', 'security', 'monitoring', 'communication', 'storage', 'custom'];

  const appsToDisplay = activeTab === 'available' ? availableApps : installedApps;

  const filteredApps = appsToDisplay.filter(app => {
    const matchesSearch = app.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         app.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || app.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  // Show message if no instance is selected
  if (!currentInstance) {
    return (
      <Card className="p-8 text-center">
        <AppWindow className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <h3 className="text-lg font-medium mb-2">No Instance Selected</h3>
        <p className="text-muted-foreground mb-4">
          Please select or create an instance to manage apps.
        </p>
      </Card>
    );
  }

  // Show error state
  if (availableError || deployedError) {
    return (
      <Card className="p-8 text-center">
        <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
        <h3 className="text-lg font-medium mb-2">Error Loading Apps</h3>
        <p className="text-muted-foreground mb-4">
          {(availableError as Error)?.message || (deployedError as Error)?.message || 'An error occurred'}
        </p>
        <Button onClick={() => window.location.reload()}>Reload Page</Button>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Applications</h2>
          <p className="text-muted-foreground">
            Install and manage applications on your Kubernetes cluster
          </p>
        </div>
      </div>

      <Card className="p-6">
        <div className="flex flex-col sm:flex-row gap-4 mb-6">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search applications..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border rounded-lg bg-background"
            />
          </div>
          <div className="flex gap-2 overflow-x-auto">
            {categories.map(category => (
              <Button
                key={category}
                variant={selectedCategory === category ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedCategory(category)}
                className="capitalize whitespace-nowrap"
              >
                {category}
              </Button>
            ))}
          </div>
        </div>

        <div className="flex items-center justify-between mb-4">
          <div className="text-sm text-muted-foreground">
            {isLoading ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                Loading apps...
              </span>
            ) : (
              `${runningApps} applications running • ${installedApps.length} installed • ${availableApps.length} available`
            )}
          </div>
        </div>
      </Card>

      {isLoading ? (
        <Card className="p-8 text-center">
          <Loader2 className="h-12 w-12 text-primary mx-auto mb-4 animate-spin" />
          <p className="text-muted-foreground">Loading applications...</p>
        </Card>
      ) : activeTab === 'available' ? (
        // Available Apps Grid
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredApps.map((app) => (
            <Card key={app.name} className="p-4 hover:shadow-lg transition-shadow">
              <div className="flex flex-col gap-3">
                <div className="flex items-start gap-3">
                  <AppIcon app={app} />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="font-medium truncate">{app.name}</h3>
                    </div>
                    {app.version && (
                      <Badge variant="outline" className="text-xs mb-2">
                        {app.version}
                      </Badge>
                    )}
                  </div>
                </div>
                <p className="text-sm text-muted-foreground line-clamp-2">{app.description}</p>
                <Button
                  size="sm"
                  onClick={() => handleAppAction(app, 'configure')}
                  disabled={addingAppNames.includes(app.name)}
                  className="w-full"
                >
                  {addingAppNames.includes(app.name) ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Configure & Add'}
                </Button>
              </div>
            </Card>
          ))}
        </div>
      ) : (
        // Installed Apps List
        <div className="space-y-3">
          {filteredApps.map((app) => (
            <Card key={app.name} className="p-4">
              <div className="flex items-start gap-3">
                <AppIcon app={app} />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <h3 className="font-medium truncate">{app.name}</h3>
                    {app.version && (
                      <Badge variant="outline" className="text-xs">
                        {app.version}
                      </Badge>
                    )}
                    {getStatusIcon(app.status?.status || app.deploymentStatus)}
                    {getStatusBadge(app)}
                  </div>
                  <p className="text-sm text-muted-foreground mb-2">{app.description}</p>

                  {/* Show ingress URL if available */}
                  {app.url && (
                    <a
                      href={app.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 text-xs text-primary hover:underline mb-2"
                    >
                      <ExternalLink className="h-3 w-3" />
                      {app.url}
                    </a>
                  )}

                  {app.status?.status === 'running' && (
                    <div className="space-y-1 text-xs text-muted-foreground mb-2">
                      {app.status.namespace && (
                        <div>Namespace: {app.status.namespace}</div>
                      )}
                      {app.status.replicas && (
                        <div>Replicas: {app.status.replicas}</div>
                      )}
                    </div>
                  )}

                  {/* Action buttons - horizontal layout */}
                  <div className="flex flex-wrap gap-2 mt-3">
                    {/* Added: in config but not deployed */}
                    {app.deploymentStatus === 'added' && (
                      <>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleAppAction(app, 'configure')}
                          title="Edit configuration"
                        >
                          <Settings className="h-4 w-4 sm:mr-1" />
                          <span className="hidden sm:inline">Configure</span>
                        </Button>
                        <Button
                          size="sm"
                          onClick={() => handleAppAction(app, 'deploy')}
                          disabled={deployingAppNames.includes(app.name)}
                        >
                          {deployingAppNames.includes(app.name) && <Loader2 className="h-4 w-4 animate-spin sm:mr-1" />}
                          <span className="hidden sm:inline">Deploy</span>
                        </Button>
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => handleAppAction(app, 'delete')}
                          disabled={deletingAppNames.includes(app.name)}
                          title="Delete"
                        >
                          {deletingAppNames.includes(app.name) ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                        </Button>
                      </>
                    )}

                    {/* Deployed: running in Kubernetes */}
                    {app.deploymentStatus === 'deployed' && (
                      <>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleAppAction(app, 'overview')}
                          title="Overview"
                        >
                          <Eye className="h-4 w-4 sm:mr-1" />
                          <span className="hidden sm:inline">Overview</span>
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleAppAction(app, 'configuration')}
                          title="Configuration"
                        >
                          <Settings className="h-4 w-4 sm:mr-1" />
                          <span className="hidden sm:inline">Config</span>
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleAppAction(app, 'status')}
                          title="Status"
                        >
                          <Activity className="h-4 w-4 sm:mr-1" />
                          <span className="hidden sm:inline">Status</span>
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleAppAction(app, 'logs')}
                          title="Logs"
                        >
                          <FileText className="h-4 w-4 sm:mr-1" />
                          <span className="hidden sm:inline">Logs</span>
                        </Button>
                        {app.status?.status === 'running' && (
                          <>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => handleAppAction(app, 'viewBackups')}
                              title="View all backups"
                            >
                              <Archive className="h-4 w-4 sm:mr-1" />
                              <span className="hidden sm:inline">Backups</span>
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => handleAppAction(app, 'backup')}
                              disabled={isBackingUp}
                              title="Create backup now"
                            >
                              {isBackingUp ? <Loader2 className="h-4 w-4 animate-spin sm:mr-1" /> : <Archive className="h-4 w-4 sm:mr-1" />}
                              <span className="hidden sm:inline">Backup Now</span>
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => handleAppAction(app, 'restore')}
                              disabled={isRestoring}
                              title="Restore from backup"
                            >
                              <RotateCcw className="h-4 w-4 sm:mr-1" />
                              <span className="hidden sm:inline">Restore</span>
                            </Button>
                          </>
                        )}
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => handleAppAction(app, 'delete')}
                          disabled={deletingAppNames.includes(app.name)}
                          title="Delete"
                        >
                          {deletingAppNames.includes(app.name) ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                        </Button>
                      </>
                    )}
                  </div>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {!isLoading && filteredApps.length === 0 && (
        <Card className="p-8 text-center">
          <AppWindow className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="text-lg font-medium mb-2">No applications found</h3>
          <p className="text-muted-foreground mb-4">
            {searchTerm || selectedCategory !== 'all'
              ? 'Try adjusting your search or category filter'
              : activeTab === 'available'
              ? 'All available apps have been installed'
              : 'No apps are currently installed'
            }
          </p>
        </Card>
      )}

      {/* Backup Modal */}
      <BackupRestoreModal
        isOpen={backupModalOpen}
        onClose={() => {
          setBackupModalOpen(false);
          setSelectedAppForBackup(null);
        }}
        mode="backup"
        appName={selectedAppForBackup || ''}
        onConfirm={handleBackupConfirm}
        isPending={isBackingUp}
      />

      {/* Restore Modal */}
      <BackupRestoreModal
        isOpen={restoreModalOpen}
        onClose={() => {
          setRestoreModalOpen(false);
          setSelectedAppForBackup(null);
        }}
        mode="restore"
        appName={selectedAppForBackup || ''}
        backups={backups?.backups || []}
        isLoading={backupsLoading}
        onConfirm={handleRestoreConfirm}
        isPending={isRestoring}
      />

      {/* App Configuration Dialog */}
      <AppConfigDialog
        open={configDialogOpen}
        onOpenChange={setConfigDialogOpen}
        app={selectedAppForConfig}
        existingConfig={selectedAppForConfig?.config}
        existingAppName={selectedAppForConfig?.deploymentStatus ? selectedAppForConfig.name : undefined}
        onSave={handleConfigSave}
        isSaving={isAdding}
      />

      {/* App Dialog Modals */}
      {selectedAppForDialog && currentInstance && (
        <>
          <AppOverviewDialog
            instanceName={currentInstance}
            appName={selectedAppForDialog}
            open={overviewDialogOpen}
            onClose={() => {
              setOverviewDialogOpen(false);
              setSelectedAppForDialog(null);
            }}
          />
          <AppConfigurationDialog
            instanceName={currentInstance}
            appName={selectedAppForDialog}
            open={configurationDialogOpen}
            onClose={() => {
              setConfigurationDialogOpen(false);
              setSelectedAppForDialog(null);
            }}
          />
          <AppStatusDialog
            instanceName={currentInstance}
            appName={selectedAppForDialog}
            open={statusDialogOpen}
            onClose={() => {
              setStatusDialogOpen(false);
              setSelectedAppForDialog(null);
            }}
          />
          <AppLogsDialog
            instanceName={currentInstance}
            appName={selectedAppForDialog}
            open={logsDialogOpen}
            onClose={() => {
              setLogsDialogOpen(false);
              setSelectedAppForDialog(null);
            }}
          />
        </>
      )}
    </div>
  );
}
