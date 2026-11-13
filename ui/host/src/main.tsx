import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router-dom'
import { Toaster } from 'sonner'
import {
  I18nProvider,
  ThemeProvider,
  coreMessages,
  mergeMessages,
  queryClient,
  registerMessages,
  useTheme,
} from '@harns/core'
import { AuthProvider, platformMessages } from '@harns/platform-ui'
import { router } from './routes'
import './index.css'

registerMessages(mergeMessages(coreMessages, platformMessages))

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
          <AuthProvider>
            <RouterProvider router={router} />
            <AppToaster />
          </AuthProvider>
        </ThemeProvider>
      </I18nProvider>
    </QueryClientProvider>
  </StrictMode>,
)
