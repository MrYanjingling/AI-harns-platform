import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'
import { Card } from '../ui/Card'

type StatCardVariant = 'default' | 'summary'
type StatCardTone = 'primary' | 'success' | 'warning' | 'danger' | 'neutral'

const toneClasses: Record<StatCardTone, string> = {
  primary: 'text-primary',
  success: 'text-success',
  warning: 'text-warning',
  danger: 'text-danger',
  neutral: 'text-muted',
}

export function StatCard({
  label,
  value,
  hint,
  icon,
  className,
  footer,
  variant = 'default',
  tone = 'neutral',
}: {
  label: ReactNode
  value: ReactNode
  hint?: ReactNode
  icon?: ReactNode
  className?: string
  footer?: ReactNode
  variant?: StatCardVariant
  tone?: StatCardTone
}) {
  if (variant === 'summary') {
    return (
      <Card
        className={cn(
          'group flex min-h-[124px] flex-col justify-start rounded-lg px-5 py-4',
          'shadow-[0_10px_30px_-24px_rgba(15,23,42,0.32)] transition-colors hover:border-border-strong',
          className,
        )}
      >
        <div className="flex items-center justify-between gap-3">
          <p className="min-w-0 truncate text-sm font-medium text-muted">{label}</p>
          {icon ? (
            <span
              className={cn(
                'flex shrink-0 items-center justify-center [&_svg]:size-5',
                toneClasses[tone],
              )}
            >
              {icon}
            </span>
          ) : null}
        </div>
        <p className="mt-2 text-[32px] leading-9 font-bold tracking-tight text-fg tabular">
          {value}
        </p>
        {footer ? <div className="mt-2">{footer}</div> : null}
      </Card>
    )
  }

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
