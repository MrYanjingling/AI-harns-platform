import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'
import { Card } from '../ui/Card'

export function StatCard({
  label,
  value,
  hint,
  icon,
  className,
  footer,
}: {
  label: ReactNode
  value: ReactNode
  hint?: ReactNode
  icon?: ReactNode
  className?: string
  footer?: ReactNode
}) {
  return (
    <Card className={cn('group p-4 transition-colors hover:border-border-strong', className)}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-[11px] font-medium text-muted">{label}</p>
          <p className="mt-1.5 text-xl font-semibold tracking-tight tabular">{value}</p>
          {hint ? <p className="mt-0.5 text-[11px] text-faint">{hint}</p> : null}
        </div>
        {icon ? (
          <div className="flex size-8 shrink-0 items-center justify-center rounded border border-border bg-surface-2 text-muted transition-colors group-hover:text-primary">
            {icon}
          </div>
        ) : null}
      </div>
      {footer ? <div className="mt-3">{footer}</div> : null}
    </Card>
  )
}
