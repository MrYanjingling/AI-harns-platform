import { Navigate, type RouteObject } from 'react-router-dom'
import { NotFoundPage, PagePad } from '@harns/core'
import { CollectionPage } from '@/features/collection/CollectionPage'
import { DeviceDetailPage } from '@/features/collection/DeviceDetailPage'
import { DeviceFormPage } from '@/features/collection/DeviceFormPage'
import { LogsPage } from '@/features/logs/LogsPage'
import { PerformancePage } from '@/features/performance/PerformancePage'
import { PluginsPage } from '@/features/plugins/PluginsPage'
import { SubscriptionsPage } from '@/features/subscriptions/SubscriptionsPage'

/** 网关子应用路由(相对路径),主应用挂在 /gateway。 */
export const gatewayRoutes: RouteObject[] = [
  {
    element: <PagePad />,
    children: [
      { index: true, element: <Navigate to="collection" replace /> },
      {
        path: 'collection',
        element: <CollectionPage />,
        handle: { titleKey: 'gateway.title.collection' },
      },
      {
        path: 'collection/new',
        element: <DeviceFormPage mode="create" />,
        handle: { titleKey: 'gateway.title.collectionNew' },
      },
      {
        path: 'collection/:id',
        element: <DeviceDetailPage />,
        handle: { titleKey: 'gateway.title.collectionDetail' },
      },
      {
        path: 'collection/:id/edit',
        element: <DeviceFormPage mode="edit" />,
        handle: { titleKey: 'gateway.title.collectionEdit' },
      },
      {
        path: 'subscriptions',
        element: <SubscriptionsPage />,
        handle: { titleKey: 'gateway.title.subscriptions' },
      },
      {
        path: 'plugins',
        element: <PluginsPage />,
        handle: { titleKey: 'gateway.title.plugins' },
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
      { path: '*', element: <NotFoundPage homeTo="/gateway/collection" /> },
    ],
  },
]
