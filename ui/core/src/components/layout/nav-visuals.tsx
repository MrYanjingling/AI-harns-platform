import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'

export function NavIcon({ children }: { children: ReactNode }) {
  return <span className="flex size-4 shrink-0 items-center justify-center">{children}</span>
}

/** 一级分组标题:16px 加粗,和 14px 的二级条目拉开清晰层级。 */
export function navGroupHeaderClass(open: boolean): string {
  return cn(
    'flex w-full items-center justify-between gap-2 rounded-lg px-2 py-2 text-base font-semibold transition-colors duration-200',
    open ? 'text-sidebar-fg' : 'text-sidebar-muted',
    'hover:bg-sidebar-hover hover:text-sidebar-fg',
  )
}

/** 箭头旋转与分组的 grid 高度过渡共用 300ms,展开收起时两者同步收尾。 */
export function navGroupArrowClass(open: boolean): string {
  return cn(
    'size-4 shrink-0 transition-transform duration-300 ease-out',
    open ? 'rotate-0' : '-rotate-90',
  )
}

export function navLeafClass(active: boolean, nested: boolean): string {
  return cn(
    'flex items-center gap-2.5 rounded-lg transition-all duration-200',
    nested ? 'py-2 pr-3 pl-7 text-sm' : 'px-3 py-2.5 text-sm font-medium',
    active
      ? 'gradient-brand font-medium text-primary-fg shadow-sm hover:brightness-110'
      : nested
        ? 'text-sidebar-muted hover:bg-sidebar-hover hover:text-sidebar-fg'
        : 'text-sidebar-fg hover:bg-sidebar-hover',
  )
}
