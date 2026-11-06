import type { DeviceStatus } from '@/api/types'
import { useI18n } from '@/i18n'
import { DEVICE_STATUS, UNKNOWN_STATUS } from '@/lib/constants'
import { cn } from '@/lib/cn'

export function StatusBadge({ status, className }: { status: DeviceStatus; className?: string }) {
  const { t } = useI18n()
  const meta = DEVICE_STATUS[status] ?? UNKNOWN_STATUS
  return (
    <span
      title={t(meta.descriptionKey)}
      className={cn(
        'inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-xs font-medium whitespace-nowrap',
        meta.className,
        className,
      )}
    >
      <span className={cn('size-1.5 rounded-full', meta.dotClassName)} />
      {t(meta.labelKey)}
    </span>
  )
}
