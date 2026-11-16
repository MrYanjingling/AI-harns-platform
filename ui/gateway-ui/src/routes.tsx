import { Navigate, type RouteObject } from 'react-router-dom'
import { NotFoundPage } from '@harns/core'
import { GatewayPagePad } from '@/components/layout/GatewayPagePad'
import { CollectionPage } from '@/features/collection/CollectionPage'
import { CollectionDetailPage } from '@/features/collection/CollectionDetailPage'
import { DriversPage } from '@/features/drivers/DriversPage'
import { LogsPage } from '@/features/logs/LogsPage'
import { OverviewBreadcrumbAction } from '@/features/overview/OverviewBreadcrumbAction'
import { OverviewPage } from '@/features/overview/OverviewPage'
import { MonitoringPointsPage } from '@/features/monitoring/MonitoringPointsPage'
import { PushPage } from '@/features/push/PushPage'
import { AboutPage } from '@/features/system/AboutPage'
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
    element: <GatewayPagePad />,
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
        element: <PushPage />,
        handle: { titleKey: 'gateway.title.push' },
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
        path: 'system/users',
        element: <UsersPage />,
        handle: { titleKey: 'gateway.title.users' },
      },
      {
        path: 'system/about',
        element: <AboutPage />,
        handle: { titleKey: 'gateway.title.about' },
      },
      { path: '*', element: <NotFoundPage homeTo="/overview" /> },
    ],
  },
]
