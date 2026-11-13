import { createBrowserRouter, Navigate } from 'react-router-dom'
import { AppShell, NotFoundPage, useI18n } from '@harns/core'
import { LoginPage, PlatformAccountMenu, RequireAuth, createPlatformNav, platformRoutes } from '@harns/platform-ui'

/** 主应用装配平台子应用的导航。 */
const sections = createPlatformNav('/platform')

function HostShell() {
  const { t } = useI18n()
  return (
    <AppShell
      sections={sections}
      appName={t('common.appName')}
      homeTo="/platform"
      accountMenu={<PlatformAccountMenu />}
    />
  )
}

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/',
    element: (
      <RequireAuth>
        <HostShell />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <Navigate to="/platform" replace /> },
      { path: 'platform', children: platformRoutes },
      { path: '*', element: <NotFoundPage homeTo="/platform" /> },
    ],
  },
])
