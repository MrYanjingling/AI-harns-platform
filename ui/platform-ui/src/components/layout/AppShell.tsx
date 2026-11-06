import { useEffect, useState } from 'react'
import { Link, Outlet, useLocation } from 'react-router-dom'
import { Menu, Search, X } from 'lucide-react'
import { useI18n, type MessageKey } from '@/i18n'
import { cn } from '@/lib/cn'
import { Button } from '@/components/ui/Button'
import { LanguageSwitcher } from './LanguageSwitcher'
import { NAV_SECTIONS, sectionForPath } from './nav-config'
import { QuickSearchDialog } from './QuickSearchDialog'
import { Sidebar } from './Sidebar'
import { TechBackdrop } from './TechBackdrop'
import { UserMenu } from './UserMenu'

const TITLES: Record<string, MessageKey> = {
  '/': 'layout.title.dashboard',
  '/devices': 'layout.title.devices',
  '/devices/new': 'layout.title.deviceNew',
  '/platform/thing-models': 'layout.title.thingModels',
  '/platform/thing-models/new': 'layout.title.thingModelNew',
  '/platform/things': 'layout.title.things',
  '/platform/things/new': 'layout.title.thingsNew',
  '/platform/users': 'layout.title.users',
  '/platform/roles': 'layout.title.roles',
  '/platform/orgs': 'layout.title.orgs',
  '/platform/audit-logs': 'layout.title.audit',
  '/data/metrics': 'layout.title.metrics',
  '/data/kpis': 'layout.title.kpis',
  '/data/tags': 'layout.title.tags',
  '/data/timeseries': 'layout.title.timeseries',
  '/settings': 'layout.title.settings',
}

function usePageTitle(): string {
  const { pathname } = useLocation()
  const { t } = useI18n()
  if (TITLES[pathname]) return t(TITLES[pathname])
  if (pathname.startsWith('/devices/') && pathname.endsWith('/edit')) {
    return t('layout.title.deviceEdit')
  }
  if (pathname.startsWith('/devices/')) return t('layout.title.deviceDetail')
  if (pathname.startsWith('/platform/thing-models/')) return t('layout.title.thingModelDetail')
  if (pathname.startsWith('/platform/things/')) return t('layout.title.thingDetail')
  return 'AI-harns-platform'
}

export function AppShell() {
  const { t } = useI18n()
  const [mobileOpen, setMobileOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const { pathname } = useLocation()
  const title = usePageTitle()
  const section = sectionForPath(pathname)
  const isMock = import.meta.env.VITE_USE_MOCK === 'true'

  useEffect(() => {
    setMobileOpen(false)
  }, [pathname])

  useEffect(() => {
    document.title = t('layout.title.document', { title })
  }, [title, t])

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        setSearchOpen((current) => !current)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  return (
    <div className="relative flex h-dvh flex-col overflow-hidden bg-bg text-fg">
      <TechBackdrop />
      <header className="relative z-10 flex h-11 shrink-0 items-center bg-header text-header-fg">
        <Button
          variant="ghost"
          size="icon"
          className="ml-1 text-header-muted hover:bg-header-hover hover:text-header-fg lg:hidden"
          onClick={() => setMobileOpen(true)}
          aria-label={t('layout.openMenu')}
        >
          <Menu className="size-4" />
        </Button>

        <Link to="/" className="flex shrink-0 items-center pl-3 lg:pl-4">
          <span className="mr-2 text-lg font-bold tracking-wide">{t('common.brand')}</span>
          <span className="hidden text-sm text-header-muted sm:inline">{t('common.appName')}</span>
        </Link>

        <nav className="ml-4 hidden h-full items-stretch lg:flex" aria-label={t('layout.primaryNav')}>
          {NAV_SECTIONS.map((item) => (
            <Link
              key={item.key}
              to={item.entry}
              className={cn(
                'flex items-center px-4 text-sm transition-colors',
                item.key === section.key
                  ? 'bg-primary font-medium text-primary-fg'
                  : 'text-header-muted hover:bg-header-hover hover:text-header-fg',
              )}
            >
              {t(item.labelKey)}
            </Link>
          ))}
        </nav>

        <div className="ml-auto flex items-center gap-1 pr-3 sm:pr-4">
          {isMock ? (
            <span className="mr-1 hidden rounded border border-amber-400/30 bg-amber-400/10 px-1.5 py-0.5 text-[11px] text-amber-200 sm:inline">
              {t('layout.mockBadge')}
            </span>
          ) : null}
          <button
            type="button"
            onClick={() => setSearchOpen(true)}
            aria-label={t('layout.searchMenu')}
            title={t('layout.searchMenuShortcut')}
            className="flex size-8 items-center justify-center rounded text-header-muted transition-colors hover:bg-header-hover hover:text-header-fg"
          >
            <Search className="size-4" />
          </button>
          <LanguageSwitcher />
          <UserMenu />
        </div>
      </header>

      <div className="relative z-10 flex min-h-0 flex-1">
        <aside className="hidden w-56 shrink-0 overflow-y-auto border-r border-border bg-surface lg:block">
          <Sidebar sectionKey={section.key} />
        </aside>

        <div
          className={cn(
            'fixed inset-0 z-40 lg:hidden',
            mobileOpen ? 'pointer-events-auto' : 'pointer-events-none',
          )}
        >
          <div
            className={cn(
              'absolute inset-0 bg-black/40 transition-opacity',
              mobileOpen ? 'opacity-100' : 'opacity-0',
            )}
            onClick={() => setMobileOpen(false)}
          />
          <aside
            className={cn(
              'absolute inset-y-0 left-0 w-64 border-r border-border bg-surface shadow-2xl transition-transform duration-200',
              mobileOpen ? 'translate-x-0' : '-translate-x-full',
            )}
          >
            <div className="flex h-full flex-col">
              <div className="flex h-11 shrink-0 items-center justify-between border-b border-border px-3">
                <span className="text-sm font-semibold tracking-wide">
                  {t('layout.mobileBrand')}
                </span>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => setMobileOpen(false)}
                  aria-label={t('layout.closeMenu')}
                >
                  <X className="size-4" />
                </Button>
              </div>
              <div className="shrink-0 border-b border-border p-2">
                <div className="grid grid-cols-2 gap-1">
                  {NAV_SECTIONS.map((item) => (
                    <Link
                      key={item.key}
                      to={item.entry}
                      onClick={() => setMobileOpen(false)}
                      className={cn(
                        'rounded px-2 py-1.5 text-center text-xs transition-colors',
                        item.key === section.key
                          ? 'bg-primary font-medium text-primary-fg'
                          : 'bg-surface-2 text-muted hover:bg-surface-3 hover:text-fg',
                      )}
                    >
                      {t(item.labelKey)}
                    </Link>
                  ))}
                </div>
              </div>
              <div className="min-h-0 flex-1">
                <Sidebar sectionKey={section.key} onNavigate={() => setMobileOpen(false)} />
              </div>
            </div>
          </aside>
        </div>

        <main className="min-w-0 flex-1 overflow-auto">
          <Outlet />
        </main>
      </div>

      <QuickSearchDialog open={searchOpen} onClose={() => setSearchOpen(false)} />
    </div>
  )
}
