import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Badge, cn, useI18n } from '@harns/core'
import type { CollectStatus } from '@/api/types'
import { collectStatusMeta, connectorStatusMeta } from '@/lib/constants'

const TOOLTIP_WIDTH = 224

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
  const badgeRef = useRef<HTMLSpanElement>(null)
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0, above: false })
  const lastError = error?.trim()
  const tooltip = (() => {
    switch (status) {
      case 'collecting':
        return t('gateway.status.tooltip.collecting')
      case 'emptyVariable':
        return t('gateway.status.tooltip.emptyVariable')
      case 'stopped':
        return t('gateway.status.tooltip.stopped')
      case 'collectingError':
      case 'unconnected':
      case 'error':
        return lastError || t(meta.descriptionKey)
      default:
        return lastError || t(meta.descriptionKey)
    }
  })()
  const tooltipTone =
    meta.tone === 'danger'
      ? 'border-danger/30'
      : meta.tone === 'warning'
        ? 'border-warning/30'
        : 'border-border'

  useEffect(() => {
    if (!open) return
    const close = () => setOpen(false)
    window.addEventListener('scroll', close, true)
    window.addEventListener('resize', close)
    return () => {
      window.removeEventListener('scroll', close, true)
      window.removeEventListener('resize', close)
    }
  }, [open])

  const show = () => {
    const rect = badgeRef.current?.getBoundingClientRect()
    if (!rect) return
    const width = Math.min(TOOLTIP_WIDTH, window.innerWidth - 16)
    const left = Math.min(
      Math.max(8, rect.left),
      Math.max(8, window.innerWidth - width - 8),
    )
    const above = rect.bottom + 88 > window.innerHeight
    setPosition({
      top: above ? rect.top - 6 : rect.bottom + 6,
      left,
      above,
    })
    setOpen(true)
  }

  return (
    <span
      ref={badgeRef}
      className="inline-flex"
      onMouseEnter={show}
      onMouseLeave={() => setOpen(false)}
    >
      <Badge
        tone={meta.tone}
        dot
        aria-label={`${t(meta.labelKey)}: ${tooltip}`}
        className={cn('cursor-help', className)}
      >
        {t(meta.labelKey)}
      </Badge>
      {open
        ? createPortal(
            <div
              role="tooltip"
              style={{
                position: 'fixed',
                top: position.top,
                left: position.left,
                width: 'max-content',
                maxWidth: `min(${TOOLTIP_WIDTH}px, calc(100vw - 2rem))`,
                transform: position.above ? 'translateY(-100%)' : undefined,
              }}
              className={cn(
                'z-[300] rounded border bg-surface px-2.5 py-2 text-xs leading-relaxed text-fg shadow-xl',
                tooltipTone,
              )}
            >
              {tooltip}
            </div>,
            document.body,
          )
        : null}
    </span>
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
