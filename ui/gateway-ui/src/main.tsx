import { StrictMode, useCallback, useMemo, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createBrowserRouter, Navigate, RouterProvider } from 'react-router-dom'
import {
  CircleCheck,
  CircleX,
  Info,
  LoaderCircle,
  TriangleAlert,
} from 'lucide-react'
import { Toaster } from 'sonner'
import {
  AppShell,
  Copyright,
  coreMessages,
  I18nProvider,
  mergeMessages,
  NotFoundPage,
  registerMessages,
  PreferencesPopover,
  ThemeProvider,
  useTheme,
} from '@harns/core'
import { useSystemInfo } from '@/api/hooks'
import { AuthGate } from '@/components/layout/AuthGate'
import { GatewayAccountMenu } from '@/components/layout/GatewayAccountMenu'
import { LoginPage } from '@/features/auth/LoginPage'
import { AIAssistant } from '@/features/ai/AIAssistant'
import { AIAssistantDrawer } from '@/features/ai/AIAssistantDrawer'
import { LicenseExpiryGuard } from '@/features/license/LicenseExpiryGuard'
import { FrameAnalysisDrawer } from '@/features/monitoring/FrameAnalysisDrawer'
import { FrameAnalysisProvider } from '@/features/monitoring/frame-analysis-context'
import { PushPayloadDrawer } from '@/features/push/PushPayloadDrawer'
import { PushPayloadProvider } from '@/features/push/push-payload-context'
import { gatewayMessages } from '@/i18n/messages'
import { createGatewayNav } from '@/nav'
import { AuthProvider } from '@/providers/AuthProvider'
import { gatewayRoutes } from '@/routes'
import './index.css'

registerMessages(mergeMessages(coreMessages, gatewayMessages))

const gatewayQueryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 0,
      gcTime: 0,
      retry: 1,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
})

function GatewaySidebarVersion() {
  const systemInfo = useSystemInfo()
  const version = systemInfo.data?.version || '—'
  return (
    <span
      title={version}
      className="block truncate font-mono text-[10px] font-semibold text-faint"
    >
      V{version}
    </span>
  )
}

const sections = createGatewayNav()

function GatewayShell() {
  const [assistantOpen, setAssistantOpen] = useState(false)
  const [analysisCollectionId, setAnalysisCollectionId] = useState<string | null>(null)
  const [pushPayloadGroupId, setPushPayloadGroupId] = useState<string | null>(null)

  const openAssistant = useCallback(() => {
    setAnalysisCollectionId(null)
    setPushPayloadGroupId(null)
    setAssistantOpen(true)
  }, [])
  const openFrameAnalysis = useCallback((collectionId: string) => {
    setAssistantOpen(false)
    setPushPayloadGroupId(null)
    setAnalysisCollectionId(collectionId)
  }, [])
  const closePushPayload = useCallback(() => setPushPayloadGroupId(null), [])
  const openPushPayload = useCallback((groupId: string) => {
    setAssistantOpen(false)
    setAnalysisCollectionId(null)
    setPushPayloadGroupId(groupId)
  }, [])
  const pushPayloadValue = useMemo(
    () => ({ open: openPushPayload, close: closePushPayload }),
    [closePushPayload, openPushPayload],
  )
  const updateFrameAnalysis = useCallback((collectionId: string) => {
    setAnalysisCollectionId((current) => (current === null ? null : collectionId))
  }, [])
  const closeFrameAnalysis = useCallback(() => setAnalysisCollectionId(null), [])
  const frameAnalysisValue = useMemo(
    () => ({
      open: openFrameAnalysis,
      update: updateFrameAnalysis,
      close: closeFrameAnalysis,
    }),
    [closeFrameAnalysis, openFrameAnalysis, updateFrameAnalysis],
  )

  return (
    <FrameAnalysisProvider value={frameAnalysisValue}>
      <PushPayloadProvider value={pushPayloadValue}>
        <div className="flex h-dvh min-h-0 w-full overflow-hidden">
        <div className="min-h-0 min-w-0 flex-1">
          <AppShell
            sections={sections}
            appName="AI-harns-gateway"
            documentTitle="HarnsGateway"
            homeTo="/overview"
            headerTools={
              <>
                <AIAssistant onOpen={openAssistant} />
                <PreferencesPopover />
              </>
            }
            accountMenu={<GatewayAccountMenu />}
            sidebarWidth={172}
            sidebarCollapsedWidth={56}
            sidebarCollapsible
            sidebarResizable
            autoCollapseSidebar={
              assistantOpen || analysisCollectionId !== null || pushPayloadGroupId !== null
            }
            sidebarFooter={<GatewaySidebarVersion />}
            contentFooter={<Copyright />}
            headerVariant="surface"
          />
        </div>
        <FrameAnalysisDrawer
          open={analysisCollectionId !== null}
          collectionId={analysisCollectionId ?? ''}
          onClose={closeFrameAnalysis}
        />
          <PushPayloadDrawer
            open={pushPayloadGroupId !== null}
            groupId={pushPayloadGroupId ?? ''}
            onClose={closePushPayload}
          />
          <AIAssistantDrawer open={assistantOpen} onClose={() => setAssistantOpen(false)} />
        </div>
      </PushPayloadProvider>
    </FrameAnalysisProvider>
  )
}

const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/404', element: <NotFoundPage homeTo="/overview" fullScreen /> },
  { path: '*', element: <Navigate to="/404" replace /> },
  {
    path: '/',
    element: (
      <AuthGate>
        <LicenseExpiryGuard>
          <GatewayShell />
        </LicenseExpiryGuard>
      </AuthGate>
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
      duration={3000}
      gap={8}
      offset={16}
      icons={{
        success: <CircleCheck className="size-4 text-success" />,
        error: <CircleX className="size-4 text-danger" />,
        info: <Info className="size-4 text-info" />,
        warning: <TriangleAlert className="size-4 text-warning" />,
        loading: <LoaderCircle className="size-4 animate-spin text-muted" />,
      }}
      style={{ zIndex: 1000, width: 'min(560px, calc(100vw - 2rem))' }}
      toastOptions={{
        classNames: {
          toast:
            'items-start! justify-start! gap-2! overflow-hidden! rounded-md! border! border-border! bg-surface! px-4! py-2.5! text-fg! shadow-lg!',
          content: 'min-w-0! flex-1!',
          title: 'text-sm! leading-5! font-normal! text-fg! break-words!',
          description: 'mt-1! text-xs! leading-5! text-muted! break-words!',
          icon: 'm-0! shrink-0!',
        },
        style: {
          left: '50%',
          translate: '-50% 0',
          width: 'max-content',
          minWidth: '160px',
          maxWidth: 'min(560px, calc(100vw - 2rem))',
        },
      }}
    />
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={gatewayQueryClient}>
      <I18nProvider>
        <ThemeProvider>
          <AuthProvider>
            <RouterProvider router={router} />
            <AppToaster />
          </AuthProvider>
        </ThemeProvider>
      </I18nProvider>
    </QueryClientProvider>
  </StrictMode>,
)
