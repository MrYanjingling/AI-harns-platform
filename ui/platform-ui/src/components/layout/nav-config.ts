import type { MessageKey } from '@/i18n'

export type NavSectionKey = 'platform' | 'identity'

export interface NavLeaf {
  to: string
  labelKey: MessageKey
  /** 用于快捷搜索的补充说明 */
  hintKey?: MessageKey
  end?: boolean
}

export interface NavGroup {
  /** 有 labelKey 时渲染为可折叠分组,否则平铺 */
  labelKey?: MessageKey
  items: NavLeaf[]
}

export interface NavSection {
  key: NavSectionKey
  labelKey: MessageKey
  /** 顶栏点击后进入的默认页面 */
  entry: string
  /** 用于判断当前路径属于哪个一级导航 */
  prefixes: string[]
  groups: NavGroup[]
}

export const NAV_SECTIONS: NavSection[] = [
  {
    key: 'platform',
    labelKey: 'layout.section.platform',
    entry: '/',
    prefixes: ['/', '/devices', '/platform/things', '/platform/thing-models'],
    groups: [
      {
        items: [{ to: '/', labelKey: 'layout.item.home', hintKey: 'layout.hint.home', end: true }],
      },
      {
        labelKey: 'layout.group.assets',
        items: [
          { to: '/devices', labelKey: 'layout.item.devices', hintKey: 'layout.hint.devices' },
          {
            to: '/platform/thing-models',
            labelKey: 'layout.item.thingModels',
            hintKey: 'layout.hint.thingModels',
          },
        ],
      },
    ],
  },
  {
    key: 'identity',
    labelKey: 'layout.section.identity',
    entry: '/platform/users',
    prefixes: ['/platform/users', '/platform/roles', '/platform/orgs', '/platform/audit-logs'],
    groups: [
      {
        labelKey: 'layout.group.identity',
        items: [
          { to: '/platform/users', labelKey: 'layout.item.users', hintKey: 'layout.hint.users' },
          { to: '/platform/roles', labelKey: 'layout.item.roles', hintKey: 'layout.hint.roles' },
          { to: '/platform/orgs', labelKey: 'layout.item.orgs', hintKey: 'layout.hint.orgs' },
          {
            to: '/platform/audit-logs',
            labelKey: 'layout.item.audit',
            hintKey: 'layout.hint.audit',
          },
        ],
      },
    ],
  },
]

export function sectionForPath(pathname: string): NavSection {
  const matched = NAV_SECTIONS.find((section) =>
    section.prefixes.some((prefix) =>
      prefix === '/' ? pathname === '/' : pathname === prefix || pathname.startsWith(`${prefix}/`),
    ),
  )
  return matched ?? NAV_SECTIONS[0]
}

export interface NavEntry extends NavLeaf {
  sectionKey: NavSectionKey
  sectionLabelKey: MessageKey
  groupLabelKey?: MessageKey
}

/** 快捷搜索使用的全量菜单项。 */
export const NAV_ENTRIES: NavEntry[] = NAV_SECTIONS.flatMap((section) =>
  section.groups.flatMap((group) =>
    group.items.map((item) => ({
      ...item,
      sectionKey: section.key,
      sectionLabelKey: section.labelKey,
      groupLabelKey: group.labelKey,
    })),
  ),
)
