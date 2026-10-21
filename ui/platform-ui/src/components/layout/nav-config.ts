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
            },
            {
              to: `${base}/thing-models`,
              labelKey: 'layout.item.thingModels',
              hintKey: 'layout.hint.thingModels',
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
            },
            {
              to: `${base}/roles`,
              labelKey: 'layout.item.roles',
              hintKey: 'layout.hint.roles',
            },
            {
              to: `${base}/orgs`,
              labelKey: 'layout.item.orgs',
              hintKey: 'layout.hint.orgs',
            },
            {
              to: `${base}/audit-logs`,
              labelKey: 'layout.item.audit',
              hintKey: 'layout.hint.audit',
            },
          ],
        },
      ],
    },
  ]
}

export type PlatformNavConfig = NavSection[]
export type PlatformNavLabelKey = MessageKey
