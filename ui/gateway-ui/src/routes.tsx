import { Navigate, type RouteObject } from 'react-router-dom'
import { PagePad } from '@harns/core'
import { CollectionPage } from '@/features/collection/CollectionPage'
import { CollectionDetailPage } from '@/features/collection/CollectionDetailPage'
import { DriversPage } from '@/features/drivers/DriversPage'
import { LogsPage } from '@/features/logs/LogsPage'
import { OverviewBreadcrumbAction } from '@/features/overview/OverviewBreadcrumbAction'
import { OverviewPage } from '@/features/overview/OverviewPage'
import { MonitoringPointsPage } from '@/features/monitoring/MonitoringPointsPage'
import { ConnectorsPage } from '@/features/push/ConnectorsPage'
import { PushGroupsPage } from '@/features/push/PushGroupsPage'
import { AIModelsPage } from '@/features/ai/AIModelsPage'
import { AboutPage } from '@/features/system/AboutPage'
import { SystemUpdatePage } from '@/features/system/SystemUpdatePage'
import { ModbusSimulatorPage } from '@/features/simulator/ModbusSimulatorPage'
import { UsersPage } from '@/features/system/UsersPage'

/** 网关应用路由(相对路径),独立部署在站点根路径。 */
export const gatewayRoutes: RouteObject[] = [
  { index: true, element: <Navigate to="overview" replace /> },
  {
    path: 'overview',
    element: <OverviewPage />,
    handle: {
      titleKey: 'gateway.title.overview',
      breadcrumbAction: <OverviewBreadcrumbAction />,
    },
  },
  {
    element: <PagePad variant="workspace" />,
    children: [
      {
        path: 'collections',
        element: <CollectionPage />,
        handle: { titleKey: 'gateway.title.collection' },
      },
      {
        path: 'collections/new',
        element: <Navigate to="/collections?create=1" replace />,
        handle: { titleKey: 'gateway.title.collectionNew' },
      },
      {
        path: 'collections/:id',
        element: <CollectionDetailPage />,
        handle: { titleKey: 'gateway.title.collectionDetail' },
      },
      {
        path: 'push',
        element: <ConnectorsPage />,
        handle: { titleKey: 'gateway.title.push' },
      },
      {
        path: 'push/groups',
        element: <PushGroupsPage />,
        handle: { titleKey: 'gateway.title.pushGroups' },
      },
      {
        path: 'drivers',
        element: <DriversPage />,
        handle: { titleKey: 'gateway.title.drivers' },
      },
      {
        path: 'logs',
        element: <LogsPage />,
        handle: { titleKey: 'gateway.title.logs' },
      },
      {
        path: 'monitoring/points',
        element: <MonitoringPointsPage />,
        handle: { titleKey: 'gateway.title.monitorPoints' },
      },
      {
        path: 'simulator/modbus',
        element: <ModbusSimulatorPage />,
        handle: { titleKey: 'gateway.title.simulatorModbus' },
      },
      {
        path: 'ai/models',
        element: <AIModelsPage />,
        handle: { titleKey: 'gateway.title.aiModels' },
      },
      {
        path: 'ai/mcp',
        element: <Navigate to="/ai/models" replace />,
      },
      {
        path: 'system/users',
        element: <UsersPage />,
        handle: { titleKey: 'gateway.title.users' },
      },
      {
        path: 'system/update',
        element: <SystemUpdatePage />,
        handle: { titleKey: 'gateway.title.systemUpdate' },
      },
      {
        path: 'system/about',
        element: <AboutPage />,
        handle: { titleKey: 'gateway.title.about' },
      },
    ],
  },
]
