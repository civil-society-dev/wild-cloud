import { useState } from 'react';
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
  Plus, 
  Search, 
  Settings, 
  ExternalLink,
  CheckCircle,
  AlertCircle,
  Clock,
  Download,
  Trash2,
  BookOpen
} from 'lucide-react';

interface AppsComponentProps {
  onComplete?: () => void;
}

interface Application {
  id: string;
  name: string;
  description: string;
  category: 'database' | 'web' | 'security' | 'monitoring' | 'communication' | 'storage';
  status: 'available' | 'installing' | 'running' | 'error' | 'stopped';
  version?: string;
  namespace?: string;
  replicas?: number;
  resources?: {
    cpu: string;
    memory: string;
  };
  urls?: string[];
}

export function AppsComponent({ onComplete }: AppsComponentProps) {
  const [applications, setApplications] = useState<Application[]>([
    {
      id: 'postgres',
      name: 'PostgreSQL',
      description: 'Reliable, high-performance SQL database',
      category: 'database',
      status: 'running',
      version: 'v15.4',
      namespace: 'default',
      replicas: 1,
      resources: { cpu: '500m', memory: '1Gi' },
      urls: ['postgres://postgres.wildcloud.local:5432'],
    },
    {
      id: 'redis',
      name: 'Redis',
      description: 'In-memory data structure store',
      category: 'database',
      status: 'running',
      version: 'v7.2',
      namespace: 'default',
      replicas: 1,
      resources: { cpu: '250m', memory: '512Mi' },
    },
    {
      id: 'traefik-dashboard',
      name: 'Traefik Dashboard',
      description: 'Load balancer and reverse proxy dashboard',
      category: 'web',
      status: 'running',
      version: 'v3.0',
      namespace: 'kube-system',
      urls: ['https://traefik.wildcloud.local'],
    },
    {
      id: 'grafana',
      name: 'Grafana',
      description: 'Monitoring and observability dashboards',
      category: 'monitoring',
      status: 'installing',
      version: 'v10.2',
      namespace: 'monitoring',
    },
    {
      id: 'prometheus',
      name: 'Prometheus',
      description: 'Time-series monitoring and alerting',
      category: 'monitoring',
      status: 'running',
      version: 'v2.45',
      namespace: 'monitoring',
      replicas: 1,
      resources: { cpu: '1000m', memory: '2Gi' },
    },
    {
      id: 'vault',
      name: 'HashiCorp Vault',
      description: 'Secrets management and encryption',
      category: 'security',
      status: 'available',
      version: 'v1.15',
    },
    {
      id: 'minio',
      name: 'MinIO',
      description: 'High-performance object storage',
      category: 'storage',
      status: 'available',
      version: 'RELEASE.2023-12-07',
    },
  ]);

  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');

  const getStatusIcon = (status: Application['status']) => {
    switch (status) {
      case 'running':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'error':
        return <AlertCircle className="h-5 w-5 text-red-500" />;
      case 'installing':
        return <Clock className="h-5 w-5 text-blue-500 animate-spin" />;
      case 'stopped':
        return <AlertCircle className="h-5 w-5 text-yellow-500" />;
      default:
        return <Download className="h-5 w-5 text-muted-foreground" />;
    }
  };

  const getStatusBadge = (status: Application['status']) => {
    const variants = {
      available: 'secondary',
      installing: 'default',
      running: 'success',
      error: 'destructive',
      stopped: 'warning',
    } as const;

    const labels = {
      available: 'Available',
      installing: 'Installing',
      running: 'Running',
      error: 'Error',
      stopped: 'Stopped',
    };

    return (
      <Badge variant={variants[status] as any}>
        {labels[status]}
      </Badge>
    );
  };

  const getCategoryIcon = (category: Application['category']) => {
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
      default:
        return <AppWindow className="h-4 w-4" />;
    }
  };

  const handleAppAction = (appId: string, action: 'install' | 'start' | 'stop' | 'delete' | 'configure') => {
    console.log(`${action} app: ${appId}`);
  };

  const categories = ['all', 'database', 'web', 'security', 'monitoring', 'communication', 'storage'];
  
  const filteredApps = applications.filter(app => {
    const matchesSearch = app.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         app.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || app.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  const runningApps = applications.filter(app => app.status === 'running').length;

  return (
    <div className="space-y-6">
      {/* Educational Intro Section */}
      <Card className="p-6 bg-gradient-to-r from-pink-50 to-rose-50 dark:from-pink-950/20 dark:to-rose-950/20 border-pink-200 dark:border-pink-800">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-pink-100 dark:bg-pink-900/30 rounded-lg">
            <BookOpen className="h-6 w-6 text-pink-600 dark:text-pink-400" />
          </div>
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-pink-900 dark:text-pink-100 mb-2">
              What are Apps in your Personal Cloud?
            </h3>
            <p className="text-pink-800 dark:text-pink-200 mb-3 leading-relaxed">
              Apps are the useful programs that make your personal cloud valuable - like having a personal Netflix 
              (media server), Google Drive (file storage), or Gmail (email server) running on your own hardware. 
              Instead of relying on big tech companies, you control your data and services.
            </p>
            <p className="text-pink-700 dark:text-pink-300 mb-4 text-sm">
              Your cluster can run databases, web servers, photo galleries, password managers, backup services, and much more. 
              Each app runs in its own secure container, so they don't interfere with each other and can be easily managed.
            </p>
            <Button variant="outline" size="sm" className="text-pink-700 border-pink-300 hover:bg-pink-100 dark:text-pink-300 dark:border-pink-700 dark:hover:bg-pink-900/20">
              <ExternalLink className="h-4 w-4 mr-2" />
              Learn more about self-hosted applications
            </Button>
          </div>
        </div>
      </Card>

      <Card className="p-6">
        <div className="flex items-center gap-4 mb-6">
          <div className="p-2 bg-primary/10 rounded-lg">
            <AppWindow className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h2 className="text-2xl font-semibold">App Management</h2>
            <p className="text-muted-foreground">
              Install and manage applications on your Kubernetes cluster
            </p>
          </div>
        </div>

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
          <div className="flex gap-2">
            {categories.map(category => (
              <Button
                key={category}
                variant={selectedCategory === category ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedCategory(category)}
                className="capitalize"
              >
                {category}
              </Button>
            ))}
          </div>
        </div>

        <div className="flex items-center justify-between mb-4">
          <div className="text-sm text-muted-foreground">
            {runningApps} applications running • {applications.length} total available
          </div>
          <Button size="sm">
            <Plus className="h-4 w-4 mr-2" />
            Add App
          </Button>
        </div>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {filteredApps.map((app) => (
          <Card key={app.id} className="p-4">
            <div className="flex items-start gap-3">
              <div className="p-2 bg-muted rounded-lg">
                {getCategoryIcon(app.category)}
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <h3 className="font-medium truncate">{app.name}</h3>
                  {app.version && (
                    <Badge variant="outline" className="text-xs">
                      {app.version}
                    </Badge>
                  )}
                  {getStatusIcon(app.status)}
                </div>
                <p className="text-sm text-muted-foreground mb-2">{app.description}</p>
                
                {app.status === 'running' && (
                  <div className="space-y-1 text-xs text-muted-foreground">
                    {app.namespace && (
                      <div>Namespace: {app.namespace}</div>
                    )}
                    {app.replicas && (
                      <div>Replicas: {app.replicas}</div>
                    )}
                    {app.resources && (
                      <div>Resources: {app.resources.cpu} CPU, {app.resources.memory} RAM</div>
                    )}
                    {app.urls && app.urls.length > 0 && (
                      <div className="flex items-center gap-1">
                        <span>URLs:</span>
                        {app.urls.map((url, index) => (
                          <Button
                            key={index}
                            variant="link"
                            size="sm"
                            className="h-auto p-0 text-xs"
                            onClick={() => window.open(url, '_blank')}
                          >
                            <ExternalLink className="h-3 w-3 mr-1" />
                            Access
                          </Button>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
              
              <div className="flex flex-col gap-2">
                {getStatusBadge(app.status)}
                <div className="flex gap-1">
                  {app.status === 'available' && (
                    <Button
                      size="sm"
                      onClick={() => handleAppAction(app.id, 'install')}
                    >
                      Install
                    </Button>
                  )}
                  {app.status === 'running' && (
                    <>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleAppAction(app.id, 'configure')}
                      >
                        <Settings className="h-4 w-4" />
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleAppAction(app.id, 'stop')}
                      >
                        Stop
                      </Button>
                    </>
                  )}
                  {app.status === 'stopped' && (
                    <Button
                      size="sm"
                      onClick={() => handleAppAction(app.id, 'start')}
                    >
                      Start
                    </Button>
                  )}
                  {(app.status === 'running' || app.status === 'stopped') && (
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => handleAppAction(app.id, 'delete')}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </div>
            </div>
          </Card>
        ))}
      </div>

      {filteredApps.length === 0 && (
        <Card className="p-8 text-center">
          <AppWindow className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="text-lg font-medium mb-2">No applications found</h3>
          <p className="text-muted-foreground mb-4">
            {searchTerm || selectedCategory !== 'all' 
              ? 'Try adjusting your search or category filter'
              : 'Install your first application to get started'
            }
          </p>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Browse App Catalog
          </Button>
        </Card>
      )}
    </div>
  );
}