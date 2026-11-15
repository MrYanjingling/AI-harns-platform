import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { createBrowserRouter, Navigate, RouterProvider } from 'react-router-dom'
import { Toaster } from 'sonner'
import {
  AppShell,
  coreMessages,
  I18nProvider,
  mergeMessages,
  queryClient,
  registerMessages,
  ThemeProvider,
  useTheme,
} from '@harns/core'
import { useSystemInfo } from '@/api/hooks'
import { GatewayAccountMenu } from '@/components/layout/GatewayAccountMenu'
import { GatewaySettingsPopover } from '@/components/layout/GatewaySettingsPopover'
import { gatewayMessages } from '@/i18n/messages'
import { createGatewayNav } from '@/nav'
import { gatewayRoutes } from '@/routes'
import './index.css'

registerMessages(mergeMessages(coreMessages, gatewayMessages))

const sections = createGatewayNav()

function GatewaySidebarVersion() {
  const systemInfo = useSystemInfo()
  const version = systemInfo.data?.version || '—'
  return (
    <span
      title={version}
      className="block truncate font-mono text-[10px] text-faint"
    >
      V{version}
    </span>
  )
}

const router = createBrowserRouter([
  {
    path: '/',
    element: (
      <AppShell
        sections={sections}
        appName="AI-harns-gateway"
        documentTitle="HarnsGateway"
        homeTo="/overview"
        headerTools={<GatewaySettingsPopover />}
        accountMenu={<GatewayAccountMenu />}
        sidebarWidth={172}
        sidebarCollapsedWidth={56}
        sidebarCollapsible
        sidebarResizable
        sidebarFooter={<GatewaySidebarVersion />}
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
      closeButton
      richColors
      style={{ zIndex: 1000, width: 'min(560px, calc(100vw - 2rem))' }}
      toastOptions={{
        classNames: {
          toast: 'justify-center',
          content: 'flex-none!',
          title: 'text-center',
          description: 'text-center',
        },
        style: {
          left: '50%',
          translate: '-50% 0',
          width: 'max-content',
          minWidth: '240px',
          maxWidth: 'min(560px, calc(100vw - 2rem))',
          fontSize: '13px',
        },
      }}
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
