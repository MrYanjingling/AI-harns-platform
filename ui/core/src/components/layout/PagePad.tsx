import { Outlet, useLocation } from 'react-router-dom'
import { cn } from '../../lib/cn'

/** 管理后台常规页面的留白容器。workspace 用于需要撑满内容区的应用。 */
export function PagePad({ variant = 'default' }: { variant?: 'default' | 'workspace' }) {
  const { pathname } = useLocation()
  return (
    <div
      className={cn(
        variant === 'workspace'
          ? 'flex min-h-full flex-col bg-surface-3 px-4 pt-4 pb-0 sm:px-6 sm:pt-6 sm:pb-0'
          : 'p-2.5 sm:p-3',
      )}
    >
      <div
        key={pathname}
        className={cn(
          'animate-fade-up',
          variant === 'workspace' && 'flex min-h-0 flex-1 flex-col',
        )}
      >
        <Outlet />
      </div>
    </div>
  )
}
