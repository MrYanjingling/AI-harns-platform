import type { OrgInfo } from '@/api/types'

export interface OrgOption {
  id: string
  name: string
  code: string
  depth: number
}

/** 把组织树拍平成带缩进层级的下拉/勾选列表。 */
export function flattenOrgs(nodes: OrgInfo[], depth = 0): OrgOption[] {
  return nodes.flatMap((node) => [
    { id: node.id, name: node.name, code: node.code, depth },
    ...flattenOrgs(node.children ?? [], depth + 1),
  ])
}

export function findOrg(nodes: OrgInfo[], id: string): OrgInfo | undefined {
  for (const node of nodes) {
    if (node.id === id) return node
    const found = findOrg(node.children ?? [], id)
    if (found) return found
  }
  return undefined
}
