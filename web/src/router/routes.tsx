import { Navigate } from 'react-router';
import type { RouteObject } from 'react-router';
import { InstanceLayout } from './InstanceLayout';
import { PhaseGuard } from './PhaseGuard';
import { LandingPage } from './pages/LandingPage';
import { NotFoundPage } from './pages/NotFoundPage';
import { DashboardPage } from './pages/DashboardPage';
import { OperationsPage } from './pages/OperationsPage';
import { ClusterHealthPage } from './pages/ClusterHealthPage';
import { ClusterAccessPage } from './pages/ClusterAccessPage';
import { SecretsPage } from './pages/SecretsPage';
import { UtilitiesPage } from './pages/UtilitiesPage';
import { CloudPage } from './pages/CloudPage';
import { CentralPage } from './pages/CentralPage';
import { DnsPage } from './pages/DnsPage';
import { DhcpPage } from './pages/DhcpPage';
import { PxePage } from './pages/PxePage';
import { IsoPage } from './pages/IsoPage';
import { ControlNodesPage } from './pages/ControlNodesPage';
import { WorkerNodesPage } from './pages/WorkerNodesPage';
import { ClusterPage } from './pages/ClusterPage';
import { AppsPage } from './pages/AppsPage';
import { BackupsPage } from './pages/BackupsPage';
import { AdvancedConfigPage } from './pages/AdvancedConfigPage';
import { TerminalPage } from './pages/TerminalPage';
import { KubernetesDashboardPage } from './pages/KubernetesDashboardPage';
import { AssetsIsoPage } from './pages/AssetsIsoPage';
import { AssetsPxePage } from './pages/AssetsPxePage';

export const routes: RouteObject[] = [
  {
    path: '/',
    element: <LandingPage />,
  },
  {
    path: '/iso',
    element: <AssetsIsoPage />,
  },
  {
    path: '/pxe',
    element: <AssetsPxePage />,
  },
  {
    path: '/instances/:instanceId',
    element: <InstanceLayout />,
    children: [
      {
        index: true,
        element: <Navigate to="dashboard" replace />,
      },
      {
        path: 'dashboard',
        element: <DashboardPage />,
      },
      {
        path: 'operations',
        element: <OperationsPage />,
      },
      {
        path: 'cluster/health',
        element: <ClusterHealthPage />,
      },
      {
        path: 'cluster/access',
        element: <ClusterAccessPage />,
      },
      {
        path: 'secrets',
        element: <SecretsPage />,
      },
      {
        path: 'utilities',
        element: <UtilitiesPage />,
      },
      {
        path: 'cloud',
        element: <CloudPage />,
      },
      {
        path: 'central',
        element: <CentralPage />,
      },
      {
        path: 'dns',
        element: <DnsPage />,
      },
      {
        path: 'dhcp',
        element: <DhcpPage />,
      },
      {
        path: 'pxe',
        element: <PxePage />,
      },
      {
        path: 'iso',
        element: <IsoPage />,
      },
      {
        path: 'control',
        element: (
          <PhaseGuard requiredPhase="control-nodes">
            <ControlNodesPage />
          </PhaseGuard>
        ),
      },
      {
        path: 'worker',
        element: (
          <PhaseGuard requiredPhase="control-nodes">
            <WorkerNodesPage />
          </PhaseGuard>
        ),
      },
      {
        path: 'cluster',
        element: (
          <PhaseGuard requiredPhase="cluster-services">
            <ClusterPage />
          </PhaseGuard>
        ),
      },
      {
        path: 'apps',
        children: [
          {
            index: true,
            element: <Navigate to="available" replace />,
          },
          {
            path: 'available',
            element: (
              <PhaseGuard requiredPhase="apps">
                <AppsPage />
              </PhaseGuard>
            ),
          },
          {
            path: 'installed',
            element: (
              <PhaseGuard requiredPhase="apps">
                <AppsPage />
              </PhaseGuard>
            ),
          },
        ],
      },
      {
        path: 'backups',
        element: <BackupsPage />,
      },
      {
        path: 'advanced',
        children: [
          {
            index: true,
            element: <Navigate to="config" replace />,
          },
          {
            path: 'config',
            element: <AdvancedConfigPage />,
          },
          {
            path: 'terminal',
            element: <TerminalPage />,
          },
          {
            path: 'k8s-dashboard',
            element: <KubernetesDashboardPage />,
          },
        ],
      },
    ],
  },
  {
    path: '*',
    element: <NotFoundPage />,
  },
];
