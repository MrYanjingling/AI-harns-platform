import { createBrowserRouter } from 'react-router-dom'
import { AppShell } from '@/components/layout/AppShell'
import { PagePad } from '@/components/layout/PagePad'
import { AuditLogsPage } from '@/features/audit/AuditLogsPage'
import { LoginPage } from '@/features/auth/LoginPage'
import { RequireAuth } from '@/features/auth/RequireAuth'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { OrgsPage } from '@/features/identity/OrgsPage'
import { RolesPage } from '@/features/identity/RolesPage'
import { UsersPage } from '@/features/identity/UsersPage'
import { ThingModelCreatePage } from '@/features/thing-models/ThingModelCreatePage'
import { ThingModelDetailPage } from '@/features/thing-models/ThingModelDetailPage'
import { ThingModelsPage } from '@/features/thing-models/ThingModelsPage'
import { ThingCreatePage } from '@/features/things/ThingCreatePage'
import { ThingDetailPage } from '@/features/things/ThingDetailPage'
import { ThingsPage } from '@/features/things/ThingsPage'
import { NotFoundPage } from '@/features/NotFoundPage'

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/',
    element: (
      <RequireAuth>
        <AppShell />
      </RequireAuth>
    ),
    children: [
      {
        element: <PagePad />,
        children: [
          { index: true, element: <DashboardPage /> },
          { path: 'platform/thing-models', element: <ThingModelsPage /> },
          { path: 'platform/thing-models/new', element: <ThingModelCreatePage /> },
          { path: 'platform/thing-models/:id', element: <ThingModelDetailPage /> },
          { path: 'platform/things', element: <ThingsPage /> },
          { path: 'platform/things/new', element: <ThingCreatePage /> },
          { path: 'platform/things/:id', element: <ThingDetailPage /> },
          { path: 'platform/users', element: <UsersPage /> },
          { path: 'platform/roles', element: <RolesPage /> },
          { path: 'platform/orgs', element: <OrgsPage /> },
          { path: 'platform/audit-logs', element: <AuditLogsPage /> },
          { path: '*', element: <NotFoundPage /> },
        ],
      },
    ],
  },
])
