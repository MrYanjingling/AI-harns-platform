import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router-dom'
import { Toaster } from 'sonner'
import {
  coreMessages,
  I18nProvider,
  mergeMessages,
  queryClient,
  registerMessages,
  ThemeProvider,
  useTheme,
} from '@harns/core'
import { platformMessages } from '@/i18n/messages'
import { router } from '@/router'
import './index.css'

registerMessages(mergeMessages(coreMessages, platformMessages))

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
