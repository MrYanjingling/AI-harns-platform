import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { ChevronDown } from 'lucide-react'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/cn'
import { NAV_SECTIONS, type NavGroup, type NavSection } from './nav-config'

function groupKey(section: NavSection, group: NavGroup, index: number) {
  return `${section.key}-${group.labelKey ?? 'flat'}-${index}`
}

export function Sidebar({
  sectionKey,
  onNavigate,
}: {
  sectionKey: NavSection['key']
  onNavigate?: () => void
}) {
  const { t } = useI18n()
  const section = NAV_SECTIONS.find((item) => item.key === sectionKey) ?? NAV_SECTIONS[0]
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})

  const toggle = (key: string) => {
    setCollapsed((current) => ({ ...current, [key]: !current[key] }))
  }

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
                  className={({ isActive }) =>
                    cn(
                      'block px-4 py-2.5 text-sm transition-colors',
                      isActive
                        ? 'bg-primary font-medium text-primary-fg'
                        : 'text-fg hover:bg-surface-2',
                    )
                  }
                >
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
                      cn(
                        'block py-2 pr-4 pl-8 text-sm transition-colors',
                        isActive
                          ? 'bg-primary font-medium text-primary-fg'
                          : 'text-muted hover:bg-surface-2 hover:text-fg',
                      )
                    }
                  >
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
