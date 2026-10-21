import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'

export function NavIcon({ children }: { children: ReactNode }) {
  return <span className="flex size-4 shrink-0 items-center justify-center">{children}</span>
}

export function navLeafClass(active: boolean, nested: boolean): string {
  return cn(
    'block text-sm transition-colors',
    nested ? 'py-2 pr-4 pl-8' : 'px-4 py-2.5',
    active
      ? 'bg-primary font-medium text-primary-fg'
      : nested
        ? 'text-muted hover:bg-surface-2 hover:text-fg'
        : 'text-fg hover:bg-surface-2',
  )
}
