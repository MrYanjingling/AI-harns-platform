import {
  Cable,
  Cpu,
  LayoutDashboard,
  ChevronsLeft,
  ChevronsRight,
  Send,
  Settings2,
  SlidersHorizontal,
  Users,
} from 'lucide-react'
import { NavLink } from 'react-router-dom'
import { cn } from '@/lib/cn'
import { useI18n } from '@/providers/I18nProvider'
import type { MessageKey } from '@/i18n/messages'

const items: Array<{ to: string; label: MessageKey; icon: typeof LayoutDashboard }> = [
  { to: '/overview', label: 'nav.overview', icon: LayoutDashboard },
  { to: '/collections', label: 'nav.collections', icon: Cable },
  { to: '/push', label: 'nav.push', icon: Send },
  { to: '/drivers', label: 'nav.drivers', icon: Cpu },
  { to: '/users', label: 'nav.users', icon: Users },
  { to: '/system', label: 'nav.system', icon: SlidersHorizontal },
  { to: '/settings', label: 'nav.settings', icon: Settings2 },
]

export function Sidebar({
  onNavigate,
  collapsed = false,
  onToggleCollapse,
}: {
  onNavigate?: () => void
  collapsed?: boolean
  onToggleCollapse?: () => void
}) {
  const { t } = useI18n()

  return (
    <div className="flex h-full flex-col">
      <nav className={cn('flex-1 space-y-2 overflow-y-auto py-5', collapsed ? 'px-2' : 'px-3')}>
        {items.map((item) => {
          const Icon = item.icon
          return (
            <NavLink
              key={item.to}
              to={item.to}
              onClick={onNavigate}
              title={collapsed ? t(item.label) : undefined}
              className={({ isActive }) =>
                cn(
                  'flex items-center rounded-lg text-sm text-muted transition-colors',
                  collapsed ? 'justify-center px-2 py-2.5' : 'gap-2.5 px-3.5 py-2.5',
                  'hover:bg-surface-2 hover:text-fg',
                  isActive && 'bg-surface-2 font-medium text-fg',
                )
              }
            >
              <Icon className="size-4.5 shrink-0" />
              {!collapsed ? <span className="truncate">{t(item.label)}</span> : null}
            </NavLink>
          )
        })}
      </nav>
      <div className={cn('flex items-center gap-2 border-t border-border p-3', collapsed && 'justify-center')}>
        {onToggleCollapse ? (
          <button
            type="button"
            onClick={onToggleCollapse}
            title={collapsed ? '展开菜单' : '收起菜单'}
            aria-label={collapsed ? '展开菜单' : '收起菜单'}
            className="inline-flex size-9 shrink-0 items-center justify-center rounded-lg text-muted transition-colors hover:bg-surface-2 hover:text-fg"
          >
            {collapsed ? <ChevronsRight className="size-4.5" /> : <ChevronsLeft className="size-4.5" />}
          </button>
        ) : null}
        {!collapsed ? (
          <span className="ml-auto shrink-0 text-xs font-bold text-fg">V1.0.0</span>
        ) : null}
      </div>
    </div>
  )
}
