import { createBrowserRouter, Navigate } from 'react-router-dom'
import { AppShell } from '@harns/core'
import { createPlatformNav } from '@/components/layout/nav-config'
import { PlatformAccountMenu } from '@/components/layout/PlatformAccountMenu'
import { RequireAuth } from '@/features/auth/RequireAuth'
import { LoginPage } from '@/features/auth/LoginPage'
import { AuthProvider } from '@/providers/AuthProvider'
import { platformRoutes } from '@/routes'

const sections = createPlatformNav('/platform')

export const router = createBrowserRouter([
  {
    path: '/login',
    element: (
      <AuthProvider>
        <LoginPage />
      </AuthProvider>
    ),
  },
  {
    path: '/',
    element: (
      <AuthProvider>
        <RequireAuth>
          <AppShell
            sections={sections}
            appName="AI-harns-platform"
            accountMenu={<PlatformAccountMenu />}
          />
        </RequireAuth>
      </AuthProvider>
    ),
    children: [
      { index: true, element: <Navigate to="/platform" replace /> },
      { path: 'platform', children: platformRoutes },
    ],
  },
])
