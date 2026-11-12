import { Navigate, type RouteObject } from 'react-router-dom'
import { NotFoundPage, PagePad } from '@harns/core'
import { CollectionPage } from '@/features/collection/CollectionPage'
import { CollectionDetailPage } from '@/features/collection/CollectionDetailPage'
import { CollectionFormPage } from '@/features/collection/CollectionFormPage'
import { DriversPage } from '@/features/drivers/DriversPage'
import { LogsPage } from '@/features/logs/LogsPage'
import { OverviewPage } from '@/features/overview/OverviewPage'
import { PerformancePage } from '@/features/performance/PerformancePage'
import { PushPage } from '@/features/push/PushPage'

/** 网关应用路由(相对路径),独立部署在站点根路径。 */
export const gatewayRoutes: RouteObject[] = [
  {
    element: <PagePad />,
    children: [
      { index: true, element: <Navigate to="overview" replace /> },
      {
        path: 'overview',
        element: <OverviewPage />,
        handle: { titleKey: 'gateway.title.overview' },
      },
      {
        path: 'collections',
        element: <CollectionPage />,
        handle: { titleKey: 'gateway.title.collection' },
      },
      {
        path: 'collections/new',
        element: <CollectionFormPage mode="create" />,
        handle: { titleKey: 'gateway.title.collectionNew' },
      },
      {
        path: 'collections/:id',
        element: <CollectionDetailPage />,
        handle: { titleKey: 'gateway.title.collectionDetail' },
      },
      {
        path: 'collections/:id/edit',
        element: <CollectionFormPage mode="edit" />,
        handle: { titleKey: 'gateway.title.collectionEdit' },
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
        path: 'performance',
        element: <PerformancePage />,
        handle: { titleKey: 'gateway.title.performance' },
      },
      { path: '*', element: <NotFoundPage homeTo="/overview" /> },
    ],
  },
]
