import { useEffect, useRef, useState, type ComponentType } from 'react'
import { createPortal } from 'react-dom'
import {
  CircleAlert,
  CircleCheck,
  CircleHelp,
  CirclePause,
  CircleSlash,
  CircleX,
} from 'lucide-react'
import { cn, useI18n } from '@harns/core'
import type { CollectStatus } from '@/api/types'
import { collectStatusMeta, connectorStatusMeta, type StatusTone } from '@/lib/constants'

const TOOLTIP_WIDTH = 200

const STATUS_ICONS: Record<string, ComponentType<{ className?: string }>> = {
  collecting: CircleCheck,
  running: CircleCheck,
  stopped: CirclePause,
  emptyVariable: CircleSlash,
  unconnected: CircleAlert,
  collectingError: CircleX,
  error: CircleX,
}

const TONE_CLASSES: Record<StatusTone, string> = {
  success: 'border-success/35 bg-success/10 text-success',
  warning: 'border-warning/35 bg-warning/10 text-warning',
  danger: 'border-danger/35 bg-danger/10 text-danger',
  neutral: 'border-border bg-surface-2 text-muted',
  info: 'border-info/35 bg-info/10 text-info',
}

function StatusDisplay({
  status,
  label,
  tooltip,
  tone,
  className,
}: {
  status: string
  label: string
  tooltip: string
  tone: StatusTone
  className?: string
}) {
  const rootRef = useRef<HTMLSpanElement>(null)
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0, above: false })
  const Icon = STATUS_ICONS[status] ?? CircleHelp
  const toneClass = TONE_CLASSES[tone]

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
    const rect = rootRef.current?.getBoundingClientRect()
    if (!rect) return
    const estimatedWidth = Math.min(TOOLTIP_WIDTH, tooltip.length * 12 + 16)
    const half = estimatedWidth / 2
    const center = rect.left + rect.width / 2
    const left = Math.min(
      Math.max(8 + half, center),
      Math.max(8 + half, window.innerWidth - 8 - half),
    )
    const above = rect.top > 40
    setPosition({
      top: above ? rect.top - 6 : rect.bottom + 6,
      left,
      above,
    })
    setOpen(true)
  }

  return (
    <span
      ref={rootRef}
      className={cn(
        'inline-flex min-w-0 items-center gap-1.5 rounded border px-2 py-0.5 text-sm font-medium whitespace-nowrap',
        toneClass,
        className,
      )}
      onMouseEnter={show}
      onMouseLeave={() => setOpen(false)}
      onFocus={(event) => {
        if (event.currentTarget.matches(':focus-visible')) show()
      }}
      onBlur={() => setOpen(false)}
      tabIndex={0}
      aria-label={`${label}: ${tooltip}`}
    >
      <Icon className="size-3.5 shrink-0" />
      <span className="truncate">{label}</span>
      {open
        ? createPortal(
            <div
              role="tooltip"
              style={{
                position: 'fixed',
                top: position.top,
                left: position.left,
                width: 'max-content',
                maxWidth: `min(${TOOLTIP_WIDTH}px, calc(100vw - 1rem))`,
                transform: position.above
                  ? 'translate(-50%, -100%)'
                  : 'translate(-50%, 0)',
              }}
              className="z-[300] rounded bg-header px-2 py-1 text-[11px] leading-4 text-header-fg shadow-lg"
            >
              {tooltip}
              <span
                className={cn(
                  'absolute left-1/2 size-2 -translate-x-1/2 rotate-45 bg-header',
                  position.above ? 'top-full -translate-y-1/2' : 'bottom-full translate-y-1/2',
                )}
              />
            </div>,
            document.body,
          )
        : null}
    </span>
  )
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
  const tooltip = (() => {
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
  })()

  return (
    <StatusDisplay
      status={status}
      label={t(meta.labelKey)}
      tooltip={tooltip}
      tone={meta.tone}
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

  return (
    <StatusDisplay
      status={status}
      label={t(meta.labelKey)}
      tooltip={tooltip}
      tone={meta.tone}
      className={className}
    />
  )
}
