import { cn } from '../../lib/cn'

export function ProgressBar({
  value,
  tone = 'primary',
  className,
}: {
  value: number
  tone?: 'primary' | 'success' | 'warning' | 'danger'
  className?: string
}) {
  const clamped = Math.min(100, Math.max(0, Number.isFinite(value) ? value : 0))
  const colors = {
    primary: 'gradient-brand',
    success: 'bg-linear-to-r from-success to-success/55',
    warning: 'bg-linear-to-r from-warning to-warning/55',
    danger: 'bg-linear-to-r from-danger to-danger/55',
  }
  return (
    <div className={cn('h-1.5 w-full overflow-hidden rounded-full bg-surface-3', className)}>
      <div
        className={cn('h-full rounded-full transition-[width] duration-500', colors[tone])}
        style={{ width: `${clamped}%` }}
      />
    </div>
  )
}
