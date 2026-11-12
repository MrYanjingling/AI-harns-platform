import { createBrowserRouter, Navigate } from 'react-router-dom'
import { AppShell } from '@/components/layout/AppShell'
import { OverviewPage } from '@/features/overview/OverviewPage'
import { CollectionsPage } from '@/features/collections/CollectionsPage'
import { PushPage } from '@/features/push/PushPage'
import { DriversPage } from '@/features/drivers/DriversPage'
import { UsersPage } from '@/features/users/UsersPage'
import { SystemPage } from '@/features/system/SystemPage'
import { SettingsPage } from '@/features/settings/SettingsPage'
import { NotFoundPage } from '@/features/NotFoundPage'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppShell />,
    children: [
      { index: true, element: <Navigate to="/overview" replace /> },
      { path: 'overview', element: <OverviewPage /> },
      { path: 'collections', element: <CollectionsPage /> },
      { path: 'push', element: <PushPage /> },
      { path: 'drivers', element: <DriversPage /> },
      { path: 'users', element: <UsersPage /> },
      { path: 'system', element: <SystemPage /> },
      { path: 'settings', element: <SettingsPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
