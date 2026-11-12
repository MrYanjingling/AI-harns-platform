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
    <nav className="flex h-full flex-col overflow-y-auto py-1.5" aria-label={t(section.labelKey)}>
      {section.groups.map((group, index) => {
        const key = groupKey(section, group, index)
        const open = !collapsed[key]

        if (!group.labelKey) {
          return (
            <div key={key} className="px-2 py-0.5">
              {group.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  onClick={onNavigate}
                  className={({ isActive }) => navLeafClass(isActive, false)}
                >
                  {item.icon ? <NavIcon>{item.icon}</NavIcon> : null}
                  {t(item.labelKey)}
                </NavLink>
              ))}
            </div>
          )
        }

        return (
          <div key={key} className="px-2 py-0.5">
            <button
              type="button"
              onClick={() => toggle(key)}
              aria-expanded={open}
              className="flex w-full items-center justify-between rounded-sm px-2 py-2 text-xs font-medium text-faint transition-colors hover:text-fg"
            >
              <span>{t(group.labelKey)}</span>
              <ChevronDown
                className={cn(
                  'size-3.5 transition-transform duration-150',
                  open ? 'rotate-0' : '-rotate-90',
                )}
              />
            </button>
            {open ? (
              <div className="animate-fade-up">
                {group.items.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    end={item.end}
                    onClick={onNavigate}
                    className={({ isActive }) => navLeafClass(isActive, true)}
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
