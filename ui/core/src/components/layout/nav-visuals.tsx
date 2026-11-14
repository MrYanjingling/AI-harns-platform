import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'

export function NavIcon({ children }: { children: ReactNode }) {
  return <span className="flex size-4 shrink-0 items-center justify-center">{children}</span>
}

export function navLeafClass(active: boolean, nested: boolean, iconOnly = false): string {
  return cn(
    'flex h-[45px] items-center rounded-sm text-[14px] transition-all duration-150',
    iconOnly ? 'justify-center px-0' : 'gap-[10px]',
    !iconOnly && (nested ? 'pr-3 pl-11' : 'px-3'),
    active
      ? 'bg-primary/10 text-primary'
      : 'text-muted hover:bg-primary/5 hover:text-primary',
  )
}
