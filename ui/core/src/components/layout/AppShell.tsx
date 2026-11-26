import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Link, Outlet, useLocation, useMatches } from 'react-router-dom'
import { ChevronsLeft, ChevronsRight, Search } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'
import { Button } from '../ui/Button'
import { LanguageSwitcher } from './LanguageSwitcher'
import {
  breadcrumbForPath,
  buildNavEntries,
  sectionForPath,
  titleKeyFromMatches,
  type NavSection,
} from './nav'
import { QuickSearchDialog } from './QuickSearchDialog'
import { Sidebar } from './Sidebar'
import { TechBackdrop } from './TechBackdrop'

export interface AppShellProps {
  /** 一级导航;每个子应用导出一组 section,由主应用合并后传入。 */
  sections: NavSection[]
  appName: string
  homeTo?: string
  documentTitle?: string
  /** 账号/偏好菜单;子应用按自身认证能力注入。 */
  accountMenu?: ReactNode
  /** 顶栏右侧的额外内容,例如环境标识。 */
  headerExtra?: ReactNode
  /** 替换默认语言切换按钮的顶栏工具。 */
  headerTools?: ReactNode
  /** 桌面侧边栏展开宽度,单位 px。 */
  sidebarWidth?: number
  /** 桌面侧边栏收起宽度,单位 px。 */
  sidebarCollapsedWidth?: number
  /** 是否显示侧边栏展开/收起按钮。 */
  sidebarCollapsible?: boolean
  /** 侧边栏底部右侧内容。 */
  sidebarFooter?: ReactNode
  /** 主内容区底部内容,预留给版权等信息。 */
  contentFooter?: ReactNode
  /** 是否允许拖动侧边栏调整宽度。 */
  sidebarResizable?: boolean
  /** 打开右侧辅助面板时自动收起左侧菜单；关闭后恢复原状态。 */
  autoCollapseSidebar?: boolean
  /** 顶栏视觉变体；surface 用于跟随内容区背景。 */
  headerVariant?: 'brand' | 'surface'
}

function hideBreadcrumbFromMatches(matches: ReadonlyArray<{ handle?: unknown }>): boolean {
  return matches.some((match) => {
    const handle = match.handle
    return Boolean(handle && typeof handle === 'object' && 'hideBreadcrumb' in handle && handle.hideBreadcrumb)
  })
}

function breadcrumbActionFromMatches(matches: ReadonlyArray<{ handle?: unknown }>): ReactNode {
  for (let index = matches.length - 1; index >= 0; index -= 1) {
    const handle = matches[index]?.handle
    if (handle && typeof handle === 'object' && 'breadcrumbAction' in handle) {
      return (handle as { breadcrumbAction?: ReactNode }).breadcrumbAction
    }
  }
  return undefined
}

export function AppShell({
  sections,
  appName,
  documentTitle,
  accountMenu,
  headerExtra,
  headerTools,
  sidebarWidth = 208,
  sidebarCollapsedWidth = 56,
  sidebarCollapsible = false,
  sidebarFooter,
  contentFooter,
  sidebarResizable = false,
  autoCollapseSidebar = false,
  headerVariant = 'brand',
}: AppShellProps) {
  const { t } = useI18n()
  const [searchOpen, setSearchOpen] = useState(false)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(
    () => typeof window !== 'undefined' && window.matchMedia('(max-width: 1023px)').matches,
  )
  const [sidebarActualWidth, setSidebarActualWidth] = useState(sidebarWidth)
  const [sidebarResizing, setSidebarResizing] = useState(false)
  const sidebarAutoCollapseRestoreRef = useRef(false)
  const { pathname } = useLocation()
  const matches = useMatches()
  const section = sectionForPath(pathname, sections)
  const titleKey = titleKeyFromMatches(matches)
  const title = titleKey ? t(titleKey) : appName
  const breadcrumb = breadcrumbForPath(pathname, sections)
  const breadcrumbAction = breadcrumbActionFromMatches(matches)
  const hideBreadcrumb = hideBreadcrumbFromMatches(matches)
  const isMock = import.meta.env.VITE_USE_MOCK === 'true'
  const surfaceHeader = headerVariant === 'surface'
  const searchEntries = buildNavEntries(sections)

  useEffect(() => {
    const query = window.matchMedia('(max-width: 1023px)')
    const handleChange = (event: MediaQueryListEvent) => {
      if (event.matches) setSidebarCollapsed(true)
    }
    query.addEventListener('change', handleChange)
    return () => query.removeEventListener('change', handleChange)
  }, [])

  useEffect(() => {
    if (autoCollapseSidebar) {
      sidebarAutoCollapseRestoreRef.current = !sidebarCollapsed
      if (!sidebarCollapsed) setSidebarCollapsed(true)
      return
    }
    if (sidebarAutoCollapseRestoreRef.current) {
      sidebarAutoCollapseRestoreRef.current = false
      setSidebarCollapsed(false)
    }
  }, [autoCollapseSidebar])

  useEffect(() => {
    if (!sidebarResizing) return
    const handlePointerMove = (event: PointerEvent) => {
      setSidebarActualWidth(Math.min(320, Math.max(172, event.clientX)))
    }
    const handlePointerUp = () => setSidebarResizing(false)
    window.addEventListener('pointermove', handlePointerMove)
    window.addEventListener('pointerup', handlePointerUp)
    return () => {
      window.removeEventListener('pointermove', handlePointerMove)
      window.removeEventListener('pointerup', handlePointerUp)
    }
  }, [sidebarResizing])

  useEffect(() => {
    document.title = documentTitle ?? t('layout.shell.documentTitle', { title, appName })
  }, [documentTitle, title, appName, t])

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
    <div
      className={cn(
        'relative flex h-dvh flex-col overflow-hidden bg-bg text-fg',
        sidebarResizing && 'cursor-col-resize select-none',
      )}
    >
      <TechBackdrop />
      <header
        className={cn(
          'app-header relative z-30 flex shrink-0 items-center border-b border-border',
          surfaceHeader ? 'bg-surface text-fg' : 'header-gradient text-header-fg',
        )}
      >
        {sections.length > 1 ? (
          <nav
            className="hidden h-full items-stretch pl-4 lg:flex"
            aria-label={t('layout.shell.primaryNav')}
          >
            {sections.map((item) => (
              <Link
                key={item.key}
                to={item.entry}
                className={cn(
                  'flex items-center gap-1.5 px-3.5 text-[length:var(--nav-font-size)] transition-colors',
                  item.key === section.key
                    ? 'bg-primary font-medium text-primary-fg'
                    : surfaceHeader
                      ? 'text-muted hover:bg-surface-2 hover:text-fg'
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
            <span
              className={cn(
                'mr-1 hidden rounded border px-1.5 py-0.5 text-[11px] sm:inline',
                surfaceHeader
                  ? 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-600/30 dark:bg-amber-500/10 dark:text-amber-300'
                  : 'border-amber-400/30 bg-amber-400/10 text-amber-200',
              )}
            >
              {t('layout.shell.mockBadge')}
            </span>
          ) : null}
          {headerExtra}
          <button
            type="button"
            onClick={() => setSearchOpen(true)}
            aria-label={t('layout.shell.searchMenu')}
            title={t('layout.shell.searchMenuShortcut')}
            className={cn(
              'flex size-9 items-center justify-center rounded transition-colors',
              surfaceHeader
                ? 'text-muted hover:bg-surface-2 hover:text-fg'
                : 'text-header-muted hover:bg-header-hover hover:text-header-fg',
            )}
          >
            <Search className="size-4" />
          </button>
          {headerTools ?? (
            <LanguageSwitcher
              className={
                surfaceHeader
                  ? 'flex h-8 items-center gap-1 rounded px-2 text-xs text-muted transition-colors hover:bg-surface-2 hover:text-fg'
                  : undefined
              }
            />
          )}
          {accountMenu}
        </div>
      </header>

      <div className="relative z-10 flex min-h-0 flex-1">
        <aside
          className={cn(
            'relative flex shrink-0 flex-col border-r border-border bg-surface',
            surfaceHeader && 'border-t',
          )}
          style={{ width: sidebarCollapsed ? sidebarCollapsedWidth : sidebarActualWidth }}
        >
          <div className="min-h-0 flex-1 overflow-y-auto">
            <Sidebar
              sections={sections}
              sectionKey={section.key}
              collapsed={sidebarCollapsed}
            />
          </div>
          {sidebarResizable && !sidebarCollapsed ? (
            <div
              role="separator"
              aria-label={t('layout.shell.resizeSidebar')}
              aria-orientation="vertical"
              aria-valuemin={172}
              aria-valuemax={320}
              aria-valuenow={Math.round(sidebarActualWidth)}
              tabIndex={0}
              onPointerDown={(event) => {
                event.preventDefault()
                setSidebarResizing(true)
              }}
              onKeyDown={(event) => {
                if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return
                event.preventDefault()
                const delta = event.key === 'ArrowLeft' ? -8 : 8
                setSidebarActualWidth((width) => Math.min(320, Math.max(172, width + delta)))
              }}
              className="group absolute inset-y-0 -right-1 z-20 w-2 cursor-col-resize touch-none"
            >
              <span className="absolute inset-y-0 left-1/2 w-0.5 -translate-x-1/2 bg-transparent transition-colors group-hover:bg-primary/60 group-focus-visible:bg-primary group-active:bg-primary" />
            </div>
          ) : null}
          {sidebarCollapsible || sidebarFooter ? (
            <div
              className={cn(
                'flex shrink-0 items-center border-t border-border py-2',
                sidebarCollapsed ? 'justify-center px-1' : 'gap-2 px-2',
              )}
            >
              {sidebarCollapsible ? (
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => setSidebarCollapsed((value) => !value)}
                  aria-label={
                    sidebarCollapsed
                      ? t('layout.shell.expandSidebar')
                      : t('layout.shell.collapseSidebar')
                  }
                  title={
                    sidebarCollapsed
                      ? t('layout.shell.expandSidebar')
                      : t('layout.shell.collapseSidebar')
                  }
                >
                  {sidebarCollapsed ? (
                    <ChevronsRight className="size-4" />
                  ) : (
                    <ChevronsLeft className="size-4" />
                  )}
                </Button>
              ) : null}
              {!sidebarCollapsed && sidebarFooter ? (
                <div className="min-w-0 flex-1 text-right">{sidebarFooter}</div>
              ) : null}
            </div>
          ) : null}
        </aside>

        <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
          {hideBreadcrumb ? null : <div className="app-breadcrumb-bar flex shrink-0 items-center justify-between gap-3 border-b border-border bg-surface-3 px-4 sm:px-6">
            <nav
              aria-label={t('layout.shell.breadcrumb')}
              className="flex min-w-0 items-center gap-2 text-sm"
            >
              {breadcrumb ? (
                <>
                  {breadcrumb.parentLabelKey ? (
                    <>
                      <span className="shrink-0 text-muted">
                        {t(breadcrumb.parentLabelKey)}
                      </span>
                      <span className="shrink-0 text-faint">/</span>
                    </>
                  ) : null}
                  <span className="truncate font-medium text-fg">
                    {t(breadcrumb.itemLabelKey)}
                  </span>
                </>
              ) : titleKey ? (
                <span className="truncate font-medium text-fg">{title}</span>
              ) : null}
            </nav>
            {breadcrumbAction ? <div className="shrink-0">{breadcrumbAction}</div> : null}
          </div>}
          <div className="min-h-0 flex-1 overflow-auto">
            <Outlet />
          </div>
          {contentFooter !== undefined ? (
            <div className="app-content-footer flex shrink-0 items-center justify-center bg-surface-3 px-4 sm:px-6">
              {contentFooter}
            </div>
          ) : null}
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
