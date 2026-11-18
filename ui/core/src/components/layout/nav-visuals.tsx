import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'

export function NavIcon({ children }: { children: ReactNode }) {
  return <span className="flex size-4 shrink-0 items-center justify-center">{children}</span>
}

export function navLeafClass(active: boolean, nested: boolean, iconOnly = false): string {
  return cn(
    'app-nav-item app-nav-leaf',
    iconOnly && 'app-nav-item--icon-only',
    !iconOnly && nested && 'app-nav-leaf--nested',
    active && 'app-nav-leaf--active',
  )
}
