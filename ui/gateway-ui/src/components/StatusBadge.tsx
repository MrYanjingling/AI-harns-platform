import { Badge, useI18n } from '@harns/core'
import type { CollectStatus } from '@/api/types'
import { collectStatusMeta, connectorStatusMeta } from '@/lib/constants'

export function CollectStatusBadge({ status }: { status: CollectStatus | string }) {
  const { t } = useI18n()
  const meta = collectStatusMeta(status)
  return (
    <Badge tone={meta.tone} dot title={t(meta.descriptionKey)}>
      {t(meta.labelKey)}
    </Badge>
  )
}

export function ConnectorStatusBadge({ status }: { status: string }) {
  const { t } = useI18n()
  const meta = connectorStatusMeta(status)
  return (
    <Badge tone={meta.tone} dot title={t(meta.descriptionKey)}>
      {t(meta.labelKey)}
    </Badge>
  )
}
