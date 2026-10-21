import type { RouteObject } from 'react-router-dom'
import { NotFoundPage, PagePad } from '@harns/core'
import { AuditLogsPage } from '@/features/audit/AuditLogsPage'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { OrgsPage } from '@/features/identity/OrgsPage'
import { RolesPage } from '@/features/identity/RolesPage'
import { UsersPage } from '@/features/identity/UsersPage'
import { ThingCreatePage } from '@/features/things/ThingCreatePage'
import { ThingDetailPage } from '@/features/things/ThingDetailPage'
import { ThingsPage } from '@/features/things/ThingsPage'
import { ThingModelCreatePage } from '@/features/thing-models/ThingModelCreatePage'
import { ThingModelDetailPage } from '@/features/thing-models/ThingModelDetailPage'
import { ThingModelsPage } from '@/features/thing-models/ThingModelsPage'

/**
 * 平台子应用路由(相对路径)。
 * 主应用挂在 `/platform`,独立开发时也挂在 `/platform`,保证链接一致。
 */
export const platformRoutes: RouteObject[] = [
  {
    element: <PagePad />,
    children: [
      { index: true, element: <DashboardPage />, handle: { titleKey: 'layout.title.dashboard' } },
      {
        path: 'thing-models',
        element: <ThingModelsPage />,
        handle: { titleKey: 'layout.title.thingModels' },
      },
      {
        path: 'thing-models/new',
        element: <ThingModelCreatePage />,
        handle: { titleKey: 'layout.title.thingModelNew' },
      },
      {
        path: 'thing-models/:id',
        element: <ThingModelDetailPage />,
        handle: { titleKey: 'layout.title.thingModelDetail' },
      },
      {
        path: 'things',
        element: <ThingsPage />,
        handle: { titleKey: 'layout.title.things' },
      },
      {
        path: 'things/new',
        element: <ThingCreatePage />,
        handle: { titleKey: 'layout.title.thingsNew' },
      },
      {
        path: 'things/:id',
        element: <ThingDetailPage />,
        handle: { titleKey: 'layout.title.thingDetail' },
      },
      {
        path: 'users',
        element: <UsersPage />,
        handle: { titleKey: 'layout.title.users' },
      },
      {
        path: 'roles',
        element: <RolesPage />,
        handle: { titleKey: 'layout.title.roles' },
      },
      {
        path: 'orgs',
        element: <OrgsPage />,
        handle: { titleKey: 'layout.title.orgs' },
      },
      {
        path: 'audit-logs',
        element: <AuditLogsPage />,
        handle: { titleKey: 'layout.title.audit' },
      },
      { path: '*', element: <NotFoundPage homeTo="/platform" /> },
    ],
  },
]
