import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { createBrowserRouter, Navigate, RouterProvider } from 'react-router-dom'
import { Toaster } from 'sonner'
import {
  AccountMenu,
  AppShell,
  coreMessages,
  I18nProvider,
  mergeMessages,
  queryClient,
  registerMessages,
  ThemeProvider,
  useTheme,
} from '@harns/core'
import { gatewayMessages } from '@/i18n/messages'
import { createGatewayNav } from '@/nav'
import { gatewayRoutes } from '@/routes'
import './index.css'

registerMessages(mergeMessages(coreMessages, gatewayMessages))

const sections = createGatewayNav()

const router = createBrowserRouter([
  {
    path: '/',
    element: (
      <AppShell
        sections={sections}
        appName="AI-harns-gateway"
        homeTo="/overview"
        accountMenu={<AccountMenu />}
      />
    ),
    children: [{ index: true, element: <Navigate to="/overview" replace /> }, ...gatewayRoutes],
  },
])

function AppToaster() {
  const { theme } = useTheme()
  return (
    <Toaster
      theme={theme}
      position="top-center"
      richColors
      closeButton
      toastOptions={{ style: { fontSize: '13px' } }}
    />
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <ThemeProvider>
          <RouterProvider router={router} />
          <AppToaster />
        </ThemeProvider>
      </I18nProvider>
    </QueryClientProvider>
  </StrictMode>,
)
