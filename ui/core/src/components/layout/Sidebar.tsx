import { useId, useState } from 'react'
import { NavLink } from 'react-router-dom'
import { ChevronDown } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'
import type { NavGroup, NavSection } from './nav'
import { NavIcon, navGroupArrowClass, navGroupHeaderClass, navLeafClass } from './nav-visuals'

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
  const idBase = useId()
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

        const panelId = `${idBase}-group-${index}`

        return (
          <div key={key} className="px-2 py-0.5">
            <button
              type="button"
              onClick={() => toggle(key)}
              aria-expanded={open}
              aria-controls={panelId}
              className={navGroupHeaderClass(open)}
            >
              <span className="truncate">{t(group.labelKey)}</span>
              <ChevronDown className={navGroupArrowClass(open)} aria-hidden="true" />
            </button>

            {/* 0fr → 1fr 的 grid 过渡可以按内容真实高度收起,不必测量高度 */}
            <div
              id={panelId}
              inert={!open}
              className={cn(
                'grid transition-[grid-template-rows,opacity] duration-300 ease-out',
                open ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0',
              )}
            >
              <div className="overflow-hidden">
                <div className="pt-0.5">
                  {group.items.map((item) => (
                    <NavLink
                      key={item.to}
                      to={item.to}
                      end={item.end}
                      onClick={onNavigate}
                      tabIndex={open ? undefined : -1}
                      className={({ isActive }) => navLeafClass(isActive, true)}
                    >
                      {item.icon ? <NavIcon>{item.icon}</NavIcon> : null}
                      {t(item.labelKey)}
                    </NavLink>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )
      })}
    </nav>
  )
}
