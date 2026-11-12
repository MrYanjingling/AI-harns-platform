import {
  Boxes,
  Building2,
  History,
  LayoutDashboard,
  Layers,
  ShieldCheck,
  Users,
} from 'lucide-react'
import type { MessageKey, NavSection } from '@harns/core'

/**
 * 平台子应用的侧边栏结构。
 * base 由挂载方决定:主应用为 `/platform`,平台独立开发时为 `/platform`。
 */
export function createPlatformNav(base = '/platform'): NavSection[] {
  return [
    {
      key: 'platform',
      labelKey: 'layout.section.platform',
      icon: <ShieldCheck className="size-4" />,
      entry: base,
      prefixes: [base],
      groups: [
        {
          items: [
            {
              to: base,
              labelKey: 'layout.item.home',
              hintKey: 'layout.hint.home',
              end: true,
              icon: <LayoutDashboard className="size-4" />,
            },
          ],
        },
        {
          labelKey: 'layout.group.assets',
          items: [
            {
              to: `${base}/things`,
              labelKey: 'layout.item.things',
              hintKey: 'layout.hint.things',
              icon: <Boxes className="size-4" />,
            },
            {
              to: `${base}/thing-models`,
              labelKey: 'layout.item.thingModels',
              hintKey: 'layout.hint.thingModels',
              icon: <Layers className="size-4" />,
            },
          ],
        },
        {
          labelKey: 'layout.group.identity',
          items: [
            {
              to: `${base}/users`,
              labelKey: 'layout.item.users',
              hintKey: 'layout.hint.users',
              icon: <Users className="size-4" />,
            },
            {
              to: `${base}/roles`,
              labelKey: 'layout.item.roles',
              hintKey: 'layout.hint.roles',
              icon: <ShieldCheck className="size-4" />,
            },
            {
              to: `${base}/orgs`,
              labelKey: 'layout.item.orgs',
              hintKey: 'layout.hint.orgs',
              icon: <Building2 className="size-4" />,
            },
            {
              to: `${base}/audit-logs`,
              labelKey: 'layout.item.audit',
              hintKey: 'layout.hint.audit',
              icon: <History className="size-4" />,
            },
          ],
        },
      ],
    },
  ]
}

export type PlatformNavConfig = NavSection[]
export type PlatformNavLabelKey = MessageKey
