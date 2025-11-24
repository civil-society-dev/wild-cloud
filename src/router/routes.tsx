import { Navigate } from 'react-router';
import type { RouteObject } from 'react-router';
import { InstanceLayout } from './InstanceLayout';
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
import { AdvancedPage } from './pages/AdvancedPage';
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
        element: <ControlNodesPage />,
      },
      {
        path: 'worker',
        element: <WorkerNodesPage />,
      },
      {
        path: 'cluster',
        element: <ClusterPage />,
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
            element: <AppsPage />,
          },
          {
            path: 'installed',
            element: <AppsPage />,
          },
        ],
      },
      {
        path: 'backups',
        element: <BackupsPage />,
      },
      {
        path: 'advanced',
        element: <AdvancedPage />,
      },
    ],
  },
  {
    path: '*',
    element: <NotFoundPage />,
  },
];
