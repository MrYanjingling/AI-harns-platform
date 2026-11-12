import { useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { cn } from '@/lib/cn'
import { useI18n } from '@/providers/I18nProvider'
import type { MessageKey } from '@/i18n/messages'
import { SettingsPopover } from './SettingsPopover'
import { Sidebar } from './Sidebar'

const TITLE_KEYS: Record<string, MessageKey> = {
  '/overview': 'nav.overview',
  '/collections': 'nav.collections',
  '/push': 'nav.push',
  '/drivers': 'nav.drivers',
  '/users': 'nav.users',
  '/system': 'nav.system',
  '/settings': 'nav.settings',
}

const COLLAPSE_KEY = 'harns-gateway-sidebar-collapsed'

export function AppShell() {
  const [collapsed, setCollapsed] = useState(() => {
    try {
      return localStorage.getItem(COLLAPSE_KEY) === 'true'
    } catch {
      return false
    }
  })
  const { pathname } = useLocation()
  const { t } = useI18n()
  const title = t(TITLE_KEYS[pathname] ?? 'nav.overview')

  useEffect(() => {
    document.title = `${title} · Gateway`
  }, [pathname, title])

  useEffect(() => {
    try {
      localStorage.setItem(COLLAPSE_KEY, String(collapsed))
    } catch {
      /* ignore */
    }
  }, [collapsed])

  return (
    <div className="min-h-dvh bg-bg">
      <header className="sticky top-0 z-30 flex h-16 items-center justify-end border-b border-border bg-bg/85 px-4 backdrop-blur-md sm:px-6">
        <SettingsPopover />
      </header>

      <div className="flex min-h-[calc(100dvh-4rem)]">
        <aside
          className={cn(
            'sticky top-16 hidden h-[calc(100dvh-4rem)] shrink-0 self-start border-r border-border bg-surface transition-[width] duration-200 lg:block',
            collapsed ? 'w-14' : 'w-40',
          )}
        >
          <Sidebar collapsed={collapsed} onToggleCollapse={() => setCollapsed((current) => !current)} />
        </aside>

        <main className="min-w-0 flex-1 px-4 py-6 sm:px-6 lg:px-8">
          <div className="mx-auto w-full max-w-[1400px]">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}
