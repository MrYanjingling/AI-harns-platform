import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { Pause, Play, Send, Trash2 } from 'lucide-react'
import {
  Badge,
  Button,
  CopyButton,
  RightDrawer,
  cn,
  formatDateTime,
  useI18n,
} from '@harns/core'
import type { PushCommandEvent, PushPayloadEvent } from '@/api/types'
import { usePushGroupCommandStream } from './usePushGroupCommandStream'
import { usePushGroupPayloadStream } from './usePushGroupPayloadStream'

interface StreamEvent {
  time: string
  topic: string
  payload: unknown
  durationMs?: number
  error?: string
}

function payloadText(value: unknown): string {
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function EventSection({
  title,
  events,
  emptyText,
  paused,
}: {
  title: string
  events: StreamEvent[]
  emptyText: string
  paused: boolean
}) {
  const boxRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (paused) return
    const box = boxRef.current
    if (box) box.scrollTop = box.scrollHeight
  }, [events, paused])

  return (
    <section className="flex h-full min-h-0 flex-col overflow-hidden rounded border border-border bg-surface">
      <div className="flex shrink-0 items-center gap-2 border-b border-border bg-surface-2 px-3 py-2">
        <span className="text-sm font-medium text-fg">{title}</span>
        <Badge tone="neutral" className="border-[1.5px] px-2 py-1">
          {events.length}
        </Badge>
      </div>
      <div
        ref={boxRef}
        className="min-h-0 flex-1 overflow-y-scroll p-3"
        role="log"
        aria-live="polite"
      >
        {events.length === 0 ? (
          <p className="py-8 text-center text-sm text-faint">{emptyText}</p>
        ) : (
          <div className="space-y-3">
            {events.map((event, index) => (
              <div key={`${event.time}-${index}`} className="rounded border border-border">
                <div className="flex flex-wrap items-center gap-2 border-b border-border bg-surface-2 px-3 py-2">
                  <span className="font-mono text-[11px] text-muted">
                    {formatDateTime(event.time)}
                  </span>
                  <span className="min-w-0 truncate font-mono text-[11px] text-faint">
                    {event.topic}
                  </span>
                  {event.durationMs !== undefined ? (
                    <span className="text-[11px] text-faint">{event.durationMs}ms</span>
                  ) : null}
                  {event.error ? (
                    <span className="min-w-0 flex-1 truncate text-xs text-danger">
                      {event.error}
                    </span>
                  ) : null}
                  <CopyButton value={payloadText(event.payload)} className="ml-auto" />
                </div>
                <pre className="overflow-auto px-3 py-2 font-mono text-xs leading-5 whitespace-pre-wrap break-all text-fg">
                  {payloadText(event.payload)}
                </pre>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

export function PushPayloadDrawer({
  open,
  groupId,
  onClose,
}: {
  open: boolean
  groupId: string
  onClose: () => void
}) {
  const { t } = useI18n()
  const payload = usePushGroupPayloadStream(groupId || undefined, open)
  const command = usePushGroupCommandStream(groupId || undefined, open)
  const containerRef = useRef<HTMLDivElement>(null)
  const [topPercent, setTopPercent] = useState(50)
  const [resizing, setResizing] = useState(false)

  useEffect(() => {
    if (!resizing) return
    const handlePointerMove = (event: PointerEvent) => {
      const rect = containerRef.current?.getBoundingClientRect()
      if (!rect || rect.height <= 0) return
      const percent = ((event.clientY - rect.top) / rect.height) * 100
      setTopPercent(Math.min(85, Math.max(15, percent)))
    }
    const handlePointerUp = () => setResizing(false)
    window.addEventListener('pointermove', handlePointerMove)
    window.addEventListener('pointerup', handlePointerUp)
    return () => {
      window.removeEventListener('pointermove', handlePointerMove)
      window.removeEventListener('pointerup', handlePointerUp)
    }
  }, [resizing])

  const handleDividerKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== 'ArrowUp' && event.key !== 'ArrowDown') return
    event.preventDefault()
    const delta = event.key === 'ArrowUp' ? -5 : 5
    setTopPercent((current) => Math.min(85, Math.max(15, current + delta)))
  }

  const paused = payload.paused || command.paused
  const setPaused = (next: boolean) => {
    payload.setPaused(next)
    command.setPaused(next)
  }
  const clear = () => {
    payload.clear()
    command.clear()
  }
  const total = payload.events.length + command.events.length

  return (
    <RightDrawer
      open={open}
      onClose={onClose}
      title={t('gateway.push.payload.view')}
      resizeLabel={t('gateway.push.payload.resize')}
      icon={<Send className="size-4 text-primary" />}
      bodyClassName="p-3"
      headerActions={
        <>
          <Button
            variant="ghost"
            className="h-8"
            disabled={total === 0}
            icon={<Trash2 className="size-3.5" />}
            onClick={clear}
          >
            {t('gateway.push.payload.clear')}
          </Button>
          <Button
            variant="outline"
            className="h-8"
            icon={paused ? <Play className="size-3.5" /> : <Pause className="size-3.5" />}
            onClick={() => setPaused(!paused)}
          >
            {paused ? t('gateway.push.payload.resume') : t('gateway.push.payload.pause')}
          </Button>
        </>
      }
    >
      <div
        ref={containerRef}
        className={cn('grid min-h-0 flex-1 overflow-hidden', resizing && 'cursor-row-resize select-none')}
        style={{ gridTemplateRows: `${topPercent}% 8px minmax(0, 1fr)` }}
      >
        <EventSection
          title={t('gateway.push.payload.sentTitle')}
          events={payload.events as PushPayloadEvent[]}
          emptyText={t('gateway.push.payload.waiting')}
          paused={paused}
        />
        <div
          role="separator"
          aria-label={t('gateway.push.payload.resizeSections')}
          aria-orientation="horizontal"
          aria-valuemin={15}
          aria-valuemax={85}
          aria-valuenow={Math.round(topPercent)}
          tabIndex={0}
          onPointerDown={(event) => {
            event.preventDefault()
            setResizing(true)
          }}
          onKeyDown={handleDividerKeyDown}
          className="group relative flex cursor-row-resize touch-none items-center justify-center"
        >
          <span className="h-px w-full bg-border transition-colors group-hover:bg-primary/60 group-focus-visible:bg-primary group-active:bg-primary" />
        </div>
        <EventSection
          title={t('gateway.push.payload.receivedTitle')}
          events={command.events as PushCommandEvent[]}
          emptyText={t('gateway.push.payload.commandWaiting')}
          paused={paused}
        />
      </div>
    </RightDrawer>
  )
}
