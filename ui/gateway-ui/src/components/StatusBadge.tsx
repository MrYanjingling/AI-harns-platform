import { useMemo, type ComponentType } from 'react'
import {
  CircleAlert,
  CircleCheck,
  CircleHelp,
  CirclePause,
  CircleSlash,
  CircleX,
  Download,
  RefreshCw,
} from 'lucide-react'
import { StatusBadge, useI18n } from '@harns/core'
import type { CollectStatus } from '@/api/types'
import { collectStatusMeta, connectorStatusMeta, otaStatusMeta } from '@/lib/constants'

const STATUS_ICONS: Record<string, ComponentType<{ className?: string }>> = {
  collecting: CircleCheck,
  running: CircleCheck,
  stopped: CirclePause,
  emptyVariable: CircleSlash,
  unconnected: CircleAlert,
  collectingError: CircleX,
  error: CircleX,
}

export function CollectStatusBadge({
  status,
  error,
  className,
}: {
  status: CollectStatus | string
  error?: string
  className?: string
}) {
  const { t } = useI18n()
  const meta = collectStatusMeta(status)
  const lastError = error?.trim()
  const tooltip = useMemo(() => {
    switch (status) {
      case 'collecting':
        return t('gateway.status.tooltip.collecting')
      case 'emptyVariable':
        return t('gateway.status.tooltip.emptyVariable')
      case 'stopped':
        return t('gateway.status.tooltip.stopped')
      case 'collectingError':
        return lastError || t('gateway.status.collectingError.description')
      case 'unconnected':
        return lastError || t('gateway.status.unconnected.description')
      case 'error':
        return lastError || t('gateway.status.error.description')
      default:
        return lastError || t(meta.descriptionKey)
    }
  }, [status, lastError, meta.descriptionKey, t])
  const Icon = STATUS_ICONS[status] ?? CircleHelp

  return (
    <StatusBadge
      label={t(meta.labelKey)}
      tooltip={tooltip}
      tone={meta.tone}
      icon={<Icon />}
      className={className}
    />
  )
}

export function ConnectorStatusBadge({
  status,
  error,
  className,
}: {
  status: string
  error?: string
  className?: string
}) {
  const { t } = useI18n()
  const meta = connectorStatusMeta(status)
  const tooltip = error?.trim() || t(meta.descriptionKey)
  const Icon = STATUS_ICONS[status] ?? CircleHelp

  return (
    <StatusBadge
      label={t(meta.labelKey)}
      tooltip={tooltip}
      tone={meta.tone}
      icon={<Icon />}
      className={className}
    />
  )
}

const OTA_ICONS: Record<string, ComponentType<{ className?: string }>> = {
  received: Download,
  downloading: Download,
  ready: Download,
  applying: RefreshCw,
  restarting: RefreshCw,
  succeeded: CircleCheck,
  failed: CircleX,
  rejected: CircleSlash,
  ignored: CircleSlash,
}

export function OtaStatusBadge({
  status,
  error,
  className,
}: {
  status: string
  error?: string
  className?: string
}) {
  const { t } = useI18n()
  const meta = otaStatusMeta(status)
  const tooltip = error?.trim() || t(meta.descriptionKey)
  const Icon = OTA_ICONS[status] ?? CircleHelp
  return (
    <StatusBadge
      label={t(meta.labelKey)}
      tooltip={tooltip}
      tone={meta.tone}
      icon={<Icon />}
      className={className}
    />
  )
}
