import type { PlatformStatus } from '@/api/types'
import { PLATFORM_STATUS, UNKNOWN_STATUS } from '@/lib/constants'
import { cn } from '@/lib/cn'

export function PlatformStatusBadge({
  status,
  className,
}: {
  status: PlatformStatus | string
  className?: string
}) {
  const meta = status === 'enabled' ? PLATFORM_STATUS.enabled : PLATFORM_STATUS.disabled
  const valid = status === 'enabled' || status === 'disabled'
  return (
    <span
      title={valid ? meta.description : undefined}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-medium whitespace-nowrap',
        valid ? meta.className : UNKNOWN_STATUS.className,
        className,
      )}
    >
      <span className={cn('size-1.5 rounded-full', valid ? meta.dotClassName : UNKNOWN_STATUS.dotClassName)} />
      {valid ? meta.label : String(status || '未知')}
    </span>
  )
}
