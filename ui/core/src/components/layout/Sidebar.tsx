import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { ChevronDown } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'
import type { NavGroup, NavSection } from './nav'
import { NavIcon, navLeafClass } from './nav-visuals'

function groupKey(section: NavSection, group: NavGroup, index: number) {
  return `${section.key}-${group.labelKey ?? 'flat'}-${index}`
}

export function Sidebar({
  sections,
  sectionKey,
  onNavigate,
}: {
  sections: NavSection[]
  sectionKey: string
  onNavigate?: () => void
}) {
  const { t } = useI18n()
  const section = sections.find((item) => item.key === sectionKey) ?? sections[0]
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})

  const toggle = (key: string) => {
    setCollapsed((current) => ({ ...current, [key]: !current[key] }))
  }

  if (!section) return null

  return (
    <nav className="flex h-full flex-col overflow-y-auto py-1" aria-label={t(section.labelKey)}>
      {section.groups.map((group, index) => {
        const key = groupKey(section, group, index)
        const open = !collapsed[key]

        if (!group.labelKey) {
          return (
            <div key={key} className="py-0.5">
              {group.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  onClick={onNavigate}
                  className={({ isActive }) => cn(navLeafClass(isActive, false), 'flex items-center gap-2')}
                >
                  {item.icon ? <NavIcon>{item.icon}</NavIcon> : null}
                  {t(item.labelKey)}
                </NavLink>
              ))}
            </div>
          )
        }

        return (
          <div key={key} className="py-0.5">
            <button
              type="button"
              onClick={() => toggle(key)}
              aria-expanded={open}
              className="flex w-full items-center justify-between px-4 py-2.5 text-sm text-fg transition-colors hover:bg-surface-2"
            >
              <span>{t(group.labelKey)}</span>
              <ChevronDown
                className={cn(
                  'size-3.5 text-faint transition-transform',
                  open ? 'rotate-0' : '-rotate-90',
                )}
              />
            </button>
            {open ? (
              <div>
                {group.items.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    end={item.end}
                    onClick={onNavigate}
                    className={({ isActive }) =>
                      cn(navLeafClass(isActive, true), 'flex items-center gap-2')
                    }
                  >
                    {item.icon ? <NavIcon>{item.icon}</NavIcon> : null}
                    {t(item.labelKey)}
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
