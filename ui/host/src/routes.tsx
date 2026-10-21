import { createBrowserRouter, Navigate } from 'react-router-dom'
import { AppShell, NotFoundPage, useI18n } from '@harns/core'
import {
  LoginPage,
  PlatformAccountMenu,
  RequireAuth,
  createPlatformNav,
  platformRoutes,
} from '@harns/platform-ui'
import { createGatewayNav, gatewayRoutes } from '@harns/gateway-ui'

/** 主应用合并两个子应用的一级导航:平台管理 / 网关管理。 */
const sections = [...createPlatformNav('/platform'), ...createGatewayNav('/gateway')]

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
      { path: 'gateway', children: gatewayRoutes },
      { path: '*', element: <NotFoundPage homeTo="/platform" /> },
    ],
  },
])
