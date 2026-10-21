import { Badge, useI18n, type MessageKey } from '@harns/core'
import type { CollectStatus } from '@/api/types'
import { collectStatusMeta, UNKNOWN_COLLECT_STATUS } from '@/lib/constants'

export function DeviceStatusBadge({ status }: { status: CollectStatus | string }) {
  const { t } = useI18n()
  const meta = status ? collectStatusMeta(status) : UNKNOWN_COLLECT_STATUS
  return (
    <Badge tone={meta.tone} dot title={t(meta.descriptionKey as MessageKey)}>
      {t(meta.labelKey)}
    </Badge>
  )
}
