import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'
import { Card } from '../ui/Card'

export function ListTableCard({
  children,
  pagination,
  className,
  scrollClassName,
}: {
  children: ReactNode
  pagination?: ReactNode
  className?: string
  scrollClassName?: string
}) {
  return (
    <Card className={cn('flex min-h-0 flex-1 flex-col overflow-hidden', className)}>
      <div className={cn('min-h-0 flex-1 overflow-auto', scrollClassName)}>{children}</div>
      {pagination ? <div className="shrink-0">{pagination}</div> : null}
    </Card>
  )
}
