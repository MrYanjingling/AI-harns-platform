import type { ReactNode } from 'react'

export interface NavLeaf {
  to: string
  labelKey: string
  /** 快捷搜索的补充说明 */
  hintKey?: string
  /** React Router NavLink 的 end 语义 */
  end?: boolean
  icon?: ReactNode
}

export interface NavGroup {
  /** 有 labelKey 时渲染为可折叠分组,否则平铺 */
  labelKey?: string
  icon?: ReactNode
  items: NavLeaf[]
}

export interface NavSection {
  key: string
  labelKey: string
  /** 顶栏一级导航图标 */
  icon?: ReactNode
  /** 顶栏点击后进入的默认页面 */
  entry: string
  /** 用于判断当前路径属于哪个一级导航 */
  prefixes: string[]
  groups: NavGroup[]
}

export interface NavEntry extends NavLeaf {
  sectionKey: string
  sectionLabelKey: string
  groupLabelKey?: string
}

export function sectionForPath(pathname: string, sections: NavSection[]): NavSection {
  const fallback = sections[0]
  if (!fallback) throw new Error('AppShell requires at least one nav section')
  const matched = sections.find((section) =>
    section.prefixes.some((prefix) =>
      prefix === '/'
        ? pathname === '/'
        : pathname === prefix || pathname.startsWith(`${prefix}/`),
    ),
  )
  return matched ?? fallback
}

/** 快捷搜索使用的全量菜单项。 */
export function buildNavEntries(sections: NavSection[]): NavEntry[] {
  return sections.flatMap((section) =>
    section.groups.flatMap((group) =>
      group.items.map((item) => ({
        ...item,
        sectionKey: section.key,
        sectionLabelKey: section.labelKey,
        groupLabelKey: group.labelKey,
      })),
    ),
  )
}

/** 从 route handle 中读取页面标题 key。 */
export function titleKeyFromMatches(
  matches: ReadonlyArray<{ handle?: unknown }>,
): string | undefined {
  for (let index = matches.length - 1; index >= 0; index -= 1) {
    const handle = matches[index]?.handle
    if (handle && typeof handle === 'object' && 'titleKey' in handle) {
      const value = (handle as { titleKey?: unknown }).titleKey
      if (typeof value === 'string' && value) return value
    }
  }
  return undefined
}

export function navItemMatches(pathname: string, to: string, end = false): boolean {
  if (end) return pathname === to
  return pathname === to || pathname.startsWith(`${to}/`)
}

export function breadcrumbForPath(
  pathname: string,
  sections: NavSection[],
): { parentLabelKey?: string; itemLabelKey: string } | undefined {
  for (const section of sections) {
    for (const group of section.groups) {
      for (const item of group.items) {
        if (navItemMatches(pathname, item.to, item.end)) {
          return {
            parentLabelKey: group.labelKey,
            itemLabelKey: item.labelKey,
          }
        }
      }
    }
  }
  return undefined
}
