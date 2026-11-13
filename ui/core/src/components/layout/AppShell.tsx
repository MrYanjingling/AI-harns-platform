import { useEffect, useState, type ReactNode } from 'react'
import { Link, Outlet, useLocation, useMatches } from 'react-router-dom'
import { Menu, Search, X } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'
import { Button } from '../ui/Button'
import { LanguageSwitcher } from './LanguageSwitcher'
import { buildNavEntries, sectionForPath, titleKeyFromMatches, type NavSection } from './nav'
import { QuickSearchDialog } from './QuickSearchDialog'
import { Sidebar } from './Sidebar'
import { TechBackdrop } from './TechBackdrop'

export interface AppShellProps {
  /** 一级导航;每个子应用导出一组 section,由主应用合并后传入。 */
  sections: NavSection[]
  appName: string
  brand?: string
  homeTo?: string
  /** 账号/偏好菜单;子应用按自身认证能力注入。 */
  accountMenu?: ReactNode
  /** 顶栏右侧的额外内容,例如环境标识。 */
  headerExtra?: ReactNode
}

export function AppShell({
  sections,
  appName,
  brand = 'HARNS',
  homeTo = '/',
  accountMenu,
  headerExtra,
}: AppShellProps) {
  const { t } = useI18n()
  const [mobileOpen, setMobileOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const { pathname } = useLocation()
  const matches = useMatches()
  const section = sectionForPath(pathname, sections)
  const titleKey = titleKeyFromMatches(matches)
  const title = titleKey ? t(titleKey) : appName
  const isMock = import.meta.env.VITE_USE_MOCK === 'true'
  const searchEntries = buildNavEntries(sections)

  useEffect(() => {
    setMobileOpen(false)
  }, [pathname])

  useEffect(() => {
    document.title = t('layout.shell.documentTitle', { title, appName })
  }, [title, appName, t])

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
      <header className="relative z-30 flex h-12 shrink-0 items-center bg-header header-gradient text-header-fg">
        <Button
          variant="ghost"
          size="icon"
          className="ml-1 text-header-muted hover:bg-header-hover hover:text-header-fg lg:hidden"
          onClick={() => setMobileOpen(true)}
          aria-label={t('layout.shell.openMenu')}
        >
          <Menu className="size-4" />
        </Button>

        <Link to={homeTo} className="group flex shrink-0 items-center gap-2 pl-3 lg:pl-4">
          <span
            className="brand-mark size-5 rounded-[5px] transition-transform duration-300 group-hover:scale-110"
            aria-hidden="true"
          />
          <span className="text-lg font-bold tracking-wide">{brand}</span>
          <span className="hidden text-xs text-header-muted sm:inline">{appName}</span>
        </Link>

        {sections.length > 1 ? (
          <nav
            className="ml-3 hidden h-full items-stretch lg:flex"
            aria-label={t('layout.shell.primaryNav')}
          >
            {sections.map((item) => (
              <Link
                key={item.key}
                to={item.entry}
                className={cn(
                  'mx-0.5 my-2 flex items-center gap-1.5 rounded-lg px-3.5 text-sm transition-colors',
                  item.key === section.key
                    ? 'gradient-brand font-medium text-primary-fg hover:brightness-110'
                    : 'text-header-muted hover:bg-header-hover hover:text-header-fg',
                )}
              >
                {item.icon ? <span className="flex size-4 items-center justify-center">{item.icon}</span> : null}
                {t(item.labelKey)}
              </Link>
            ))}
          </nav>
        ) : null}

        <div className="ml-auto flex items-center gap-1 pr-2 sm:pr-3">
          {isMock ? (
            <span className="mr-1 hidden rounded border border-warning/30 bg-warning/10 px-1.5 py-0.5 text-[11px] text-warning sm:inline">
              {t('layout.shell.mockBadge')}
            </span>
          ) : null}
          {headerExtra}
          <button
            type="button"
            onClick={() => setSearchOpen(true)}
            aria-label={t('layout.shell.searchMenu')}
            title={t('layout.shell.searchMenuShortcut')}
            className="flex size-9 items-center justify-center rounded-lg text-header-muted transition-all hover:bg-header-hover hover:text-header-fg active:scale-95"
          >
            <Search className="size-4" />
          </button>
          <LanguageSwitcher />
          {accountMenu}
        </div>
      </header>

      <div className="relative z-10 flex min-h-0 flex-1">
        {/* 悬浮卡片式侧栏:与顶栏、正文留出间隙,靠圆角与阴影浮在底纹之上 */}
        <aside className="sidebar-gradient m-2 hidden w-56 shrink-0 overflow-hidden rounded-2xl border border-border shadow-lg shadow-black/5 lg:block">
          <Sidebar sections={sections} sectionKey={section.key} />
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
              'sidebar-gradient absolute inset-y-2 left-2 w-64 overflow-hidden rounded-2xl border border-border shadow-2xl transition-transform duration-200 ease-out',
              mobileOpen ? 'translate-x-0' : '-translate-x-full',
            )}
          >
            <div className="flex h-full flex-col">
              <div className="flex h-10 shrink-0 items-center justify-between border-b border-border px-3">
                <span className="text-sm font-semibold tracking-wide">
                  {t('layout.shell.mobileBrand')}
                </span>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => setMobileOpen(false)}
                  aria-label={t('layout.shell.closeMenu')}
                >
                  <X className="size-4" />
                </Button>
              </div>
              {sections.length > 1 ? (
                <div className="shrink-0 border-b border-border p-2">
                  <div className="grid grid-cols-2 gap-1">
                    {sections.map((item) => (
                      <Link
                        key={item.key}
                        to={item.entry}
                        onClick={() => setMobileOpen(false)}
                        className={cn(
                          'rounded-lg px-2 py-1.5 text-center text-xs transition-all',
                          item.key === section.key
                            ? 'gradient-brand font-medium text-primary-fg hover:brightness-110'
                            : 'bg-surface-2 text-muted hover:bg-surface-3 hover:text-fg',
                        )}
                      >
                        {t(item.labelKey)}
                      </Link>
                    ))}
                  </div>
                </div>
              ) : null}
              <div className="min-h-0 flex-1">
                <Sidebar
                  sections={sections}
                  sectionKey={section.key}
                  onNavigate={() => setMobileOpen(false)}
                />
              </div>
            </div>
          </aside>
        </div>

        <main className="min-w-0 flex-1 overflow-auto">
          <Outlet />
        </main>
      </div>

      <QuickSearchDialog
        open={searchOpen}
        onClose={() => setSearchOpen(false)}
        entries={searchEntries}
      />
    </div>
  )
}
