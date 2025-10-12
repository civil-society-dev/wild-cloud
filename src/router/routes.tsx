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
import { BaseServicesPage } from './pages/BaseServicesPage';
import { UtilitiesPage } from './pages/UtilitiesPage';
import { CloudPage } from './pages/CloudPage';
import { CentralPage } from './pages/CentralPage';
import { DnsPage } from './pages/DnsPage';
import { DhcpPage } from './pages/DhcpPage';
import { PxePage } from './pages/PxePage';
import { IsoPage } from './pages/IsoPage';
import { InfrastructurePage } from './pages/InfrastructurePage';
import { ClusterPage } from './pages/ClusterPage';
import { AppsPage } from './pages/AppsPage';
import { AdvancedPage } from './pages/AdvancedPage';

export const routes: RouteObject[] = [
  {
    path: '/',
    element: <LandingPage />,
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
        path: 'services',
        element: <BaseServicesPage />,
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
        path: 'infrastructure',
        element: <InfrastructurePage />,
      },
      {
        path: 'cluster',
        element: <ClusterPage />,
      },
      {
        path: 'apps',
        element: <AppsPage />,
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
