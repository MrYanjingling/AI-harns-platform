import { useEffect, useRef, type ReactNode } from 'react'
import { Pause, Play, Trash2 } from 'lucide-react'
import {
  Button,
  CopyButton,
  FormDialog,
  cn,
  formatDateTime,
  useI18n,
} from '@harns/core'
import type { PushConnector, PushGroup } from '@/api/types'
import { usePushGroupCommandStream } from './usePushGroupCommandStream'
import { usePushGroupPayloadStream } from './usePushGroupPayloadStream'

function payloadText(value: unknown): string {
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function InfoItem({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 items-start gap-[5px]">
      <p className="shrink-0 text-sm leading-5 font-medium text-muted">{label}</p>
      <div className="min-w-0 flex-1 text-sm leading-5 text-fg">{children}</div>
    </div>
  )
}

export function PushGroupPayloadDialog({
  open,
  group,
  connector,
  onClose,
}: {
  open: boolean
  group: PushGroup | null
  connector?: PushConnector
  onClose: () => void
}) {
  const { t } = useI18n()
  const { events, connected, error, paused, setPaused, clear } =
    usePushGroupPayloadStream(group?.id, open)
  const commandStream = usePushGroupCommandStream(group?.id, open && Boolean(group?.commandTopic))
  const boxRef = useRef<HTMLDivElement>(null)
  const stickToBottom = useRef(true)

  useEffect(() => {
    if (!open || !stickToBottom.current) return
    const box = boxRef.current
    if (box) box.scrollTop = box.scrollHeight
  }, [events, open])

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={t('gateway.push.payload.title')}
      className="max-w-[1100px]"
      footer={
        <Button variant="outline" onClick={onClose}>
          {t('common.action.close')}
        </Button>
      }
    >
      <div className="grid grid-cols-1 gap-x-8 gap-y-3 rounded border border-border bg-surface-2 px-4 py-3 sm:grid-cols-2 xl:grid-cols-4">
        <InfoItem label={t('gateway.push.group.label')}>
          <span className="truncate font-medium">{group?.name ?? '—'}</span>
        </InfoItem>
        <InfoItem label={t('gateway.push.columns.connector')}>
          <span className="truncate">{connector?.name ?? group?.connectorId ?? '—'}</span>
        </InfoItem>
        <InfoItem label={t('gateway.push.field.dataTopic')}>
          <span className="block truncate font-mono">{group?.dataTopic || '—'}</span>
        </InfoItem>
        <InfoItem label={t('gateway.push.field.commandTopic')}>
          <span className="block truncate font-mono">{group?.commandTopic || '—'}</span>
        </InfoItem>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <span className="inline-flex items-center gap-1.5 text-sm font-medium text-muted">
          <span
            className={cn(
              'size-2 rounded-full',
              connected ? 'bg-success' : error ? 'bg-danger' : 'bg-warning',
            )}
          />
          {connected
            ? t('gateway.push.payload.connected')
            : error
              ? t('gateway.push.payload.connectionError')
              : t('gateway.push.payload.connecting')}
        </span>
        <span className="text-xs text-faint">
          {t('gateway.push.payload.count', { count: events.length })}
        </span>
        <div className="ml-auto flex items-center gap-2">
          <Button
            variant="outline"
            className="h-8"
            icon={paused ? <Play className="size-3.5" /> : <Pause className="size-3.5" />}
            onClick={() => setPaused(!paused)}
          >
            {paused ? t('gateway.push.payload.resume') : t('gateway.push.payload.pause')}
          </Button>
          <Button
            variant="ghost"
            className="h-8"
            icon={<Trash2 className="size-3.5" />}
            disabled={events.length === 0}
            onClick={clear}
          >
            {t('gateway.push.payload.clear')}
          </Button>
        </div>
      </div>

      <div
        ref={boxRef}
        role="log"
        aria-live="polite"
        onScroll={() => {
          const box = boxRef.current
          if (!box) return
          stickToBottom.current = box.scrollHeight - box.scrollTop - box.clientHeight < 24
        }}
        className="min-h-[360px] max-h-[56dvh] overflow-auto rounded border border-border bg-surface p-3"
      >
        {events.length === 0 ? (
          <p className="py-10 text-center text-sm text-faint">{t('gateway.push.payload.waiting')}</p>
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

      {group?.commandTopic ? (
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold text-fg">
              {t('gateway.push.payload.commandsTitle')}
            </h3>
            <span
              className={cn(
                'size-2 rounded-full',
                commandStream.connected
                  ? 'bg-success'
                  : commandStream.error
                    ? 'bg-danger'
                    : 'bg-warning',
              )}
            />
            <span className="text-xs text-faint">
              {t('gateway.push.payload.count', { count: commandStream.events.length })}
            </span>
          </div>
          <div className="min-h-[200px] max-h-[40dvh] overflow-auto rounded border border-border bg-surface p-3">
            {commandStream.events.length === 0 ? (
              <p className="py-8 text-center text-sm text-faint">
                {t('gateway.push.payload.commandWaiting')}
              </p>
            ) : (
              <div className="space-y-3">
                {commandStream.events.map((event, index) => (
                  <div key={`${event.time}-${index}`} className="rounded border border-border">
                    <div className="flex flex-wrap items-center gap-2 border-b border-border bg-surface-2 px-3 py-2">
                      <span className="font-mono text-[11px] text-muted">
                        {formatDateTime(event.time)}
                      </span>
                      <span className="min-w-0 truncate font-mono text-[11px] text-faint">
                        {event.topic}
                      </span>
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
        </div>
      ) : null}
    </FormDialog>
  )
}
