import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'
import { Tooltip } from './Tooltip'

export type StatusTone = 'success' | 'warning' | 'danger' | 'neutral' | 'info'

export function StatusBadge({
  label,
  tooltip,
  tone = 'neutral',
  icon,
  className,
}: {
  label: string
  tooltip: string
  tone?: StatusTone
  icon?: ReactNode
  className?: string
}) {
  return (
    <Tooltip
      content={tooltip}
      focusable
      ariaLabel={`${label}: ${tooltip}`}
      className={cn('min-w-0', className)}
    >
      <span className={cn('status-badge', `status-badge-${tone}`)}>
        {icon ? <span className="status-badge-icon">{icon}</span> : null}
        <span className="status-badge-label">{label}</span>
      </span>
    </Tooltip>
  )
}
