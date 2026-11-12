import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'

export function NavIcon({ children }: { children: ReactNode }) {
  return <span className="flex size-4 shrink-0 items-center justify-center">{children}</span>
}

export function navLeafClass(active: boolean, nested: boolean): string {
  return cn(
    'flex items-center gap-2.5 rounded-sm text-sm transition-all duration-150',
    nested ? 'py-2 pr-3 pl-7' : 'px-3 py-2',
    active
      ? 'bg-primary font-medium text-primary-fg shadow-sm'
      : nested
        ? 'text-muted hover:bg-surface-2 hover:text-fg'
        : 'text-fg hover:bg-surface-2',
  )
}
