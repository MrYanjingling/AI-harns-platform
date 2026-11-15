import { useEffect, useState } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { ChevronDown } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'
import { navItemMatches, type NavGroup, type NavSection } from './nav'
import { NavIcon, navLeafClass } from './nav-visuals'

function groupKey(section: NavSection, group: NavGroup, index: number) {
  return `${section.key}-${group.labelKey ?? 'flat'}-${index}`
}

function collapsedGroups(section: NavSection | undefined): Record<string, boolean> {
  const next: Record<string, boolean> = {}
  section?.groups.forEach((group, index) => {
    if (group.labelKey) next[groupKey(section, group, index)] = true
  })
  return next
}

export function Sidebar({
  sections,
  sectionKey,
  onNavigate,
  collapsed = false,
}: {
  sections: NavSection[]
  sectionKey: string
  onNavigate?: () => void
  collapsed?: boolean
}) {
  const { t } = useI18n()
  const { pathname } = useLocation()
  const section = sections.find((item) => item.key === sectionKey) ?? sections[0]
  const [groupCollapsed, setGroupCollapsed] = useState<Record<string, boolean>>(() =>
    collapsedGroups(section),
  )

  useEffect(() => {
    setGroupCollapsed(collapsedGroups(section))
  }, [sectionKey])

  const groupKeys = section?.groups.map((group, index) =>
    group.labelKey ? groupKey(section, group, index) : undefined,
  ) ?? []

  const collapseOthers = (keepKey: string) => {
    const next: Record<string, boolean> = {}
    for (const key of groupKeys) {
      if (key) next[key] = key !== keepKey
    }
    setGroupCollapsed(next)
  }

  const toggle = (key: string) => {
    setGroupCollapsed((current) => ({ ...current, [key]: !current[key] }))
  }

  if (!section) return null

  return (
    <nav className="flex h-full flex-col overflow-y-auto py-1.5" aria-label={t(section.labelKey)}>
      {section.groups.map((group, index) => {
        const key = groupKey(section, group, index)
        const open = !groupCollapsed[key]
        const active = group.items.some((item) => navItemMatches(pathname, item.to, item.end))

        if (!group.labelKey) {
          return (
            <div key={key} className="space-y-1 px-2 py-0.5">
              {group.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  onClick={onNavigate}
                  title={collapsed ? t(item.labelKey) : undefined}
                  aria-label={collapsed ? t(item.labelKey) : undefined}
                  className={({ isActive }) => navLeafClass(isActive, false, collapsed)}
                >
                  {item.icon ? <NavIcon>{item.icon}</NavIcon> : null}
                  {collapsed ? null : t(item.labelKey)}
                </NavLink>
              ))}
            </div>
          )
        }

        return (
          <div key={key} className="space-y-1 px-2 py-0.5">
            {collapsed ? null : (
              <button
                type="button"
                onClick={() => toggle(key)}
                aria-expanded={open}
                className={cn(
                  'flex h-[45px] w-full items-center gap-[10px] rounded-sm px-3 text-[14px] transition-colors',
                  active
                    ? 'text-primary'
                    : 'text-muted hover:bg-primary/5 hover:text-primary',
                )}
              >
                {group.icon ? <NavIcon>{group.icon}</NavIcon> : null}
                <span className="min-w-0 flex-1 truncate text-left">{t(group.labelKey)}</span>
                <ChevronDown
                  className={cn(
                    'size-3.5 transition-transform duration-150',
                    open ? 'rotate-0' : '-rotate-90',
                  )}
                />
              </button>
            )}
            {open || collapsed ? (
              <div className="animate-fade-up space-y-1">
                {group.items.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    end={item.end}
                    onClick={() => {
                      onNavigate?.()
                      collapseOthers(key)
                    }}
                    title={collapsed ? t(item.labelKey) : undefined}
                    aria-label={collapsed ? t(item.labelKey) : undefined}
                    className={({ isActive }) => navLeafClass(isActive, true, collapsed)}
                  >
                    {collapsed && item.icon ? <NavIcon>{item.icon}</NavIcon> : null}
                    {collapsed ? null : t(item.labelKey)}
                  </NavLink>
                ))}
              </div>
            ) : null}
          </div>
        )
      })}
    </nav>
  )
}
