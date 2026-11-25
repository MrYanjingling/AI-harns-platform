import { useEffect, useMemo, useRef } from 'react'
import { FileSearch } from 'lucide-react'
import {
  Card,
  CopyButton,
  EmptyState,
  ErrorState,
  Skeleton,
  cn,
  formatApiTimeMs,
  parseDate,
  useI18n,
} from '@harns/core'
import { useCollection } from '@/api/hooks'
import type { CollectionDetail, FrameEvent } from '@/api/types'
import { collectStatusMeta } from '@/lib/constants'
import { useFrameStreams } from './useFrameStreams'

const FRAME_STATUSES = new Set(['collecting', 'collectingError'])

function isFrameSupported(driverType: string | undefined): boolean {
  return Boolean(driverType) && driverType !== 'opcUa'
}

function frameCapable(collection: CollectionDetail | undefined): boolean {
  if (!collection) return false
  return isFrameSupported(collection.driverType) && FRAME_STATUSES.has(collection.collectStatus)
}

function formatFrameTime(value: string): string {
  const date = parseDate(value)
  return date ? formatApiTimeMs(date) : '—'
}

interface FrameExchange {
  traceId: number
  request?: FrameEvent
  response?: FrameEvent
}

function buildExchanges(requests: FrameEvent[], responses: FrameEvent[]): FrameExchange[] {
  const exchanges = new Map<number, FrameExchange>()
  const ensure = (traceId: number): FrameExchange => {
    const existing = exchanges.get(traceId)
    if (existing) return existing
    const exchange: FrameExchange = { traceId }
    exchanges.set(traceId, exchange)
    return exchange
  }
  for (const request of requests) ensure(request.traceId).request = request
  for (const response of responses) ensure(response.traceId).response = response
  return [...exchanges.values()].sort((left, right) => left.traceId - right.traceId)
}

function FrameBubble({ kind, event }: { kind: 'request' | 'response'; event: FrameEvent }) {
  const { t } = useI18n()
  const failed = kind === 'response' && Boolean(event.error)
  const copyValue = [
    `#${event.traceId} ${event.operation} ${formatFrameTime(event.time)} · ${event.length}B${
      event.durationMs !== undefined ? ` · ${event.durationMs}ms` : ''
    }`,
    event.hex,
    event.error ? `error: ${event.error}` : '',
  ]
    .filter(Boolean)
    .join('\n')
  return (
    <div
      className={cn(
        'w-fit max-w-[88%] rounded border px-3 py-2 text-sm leading-5',
        kind === 'request'
          ? 'mr-auto border-primary/20 bg-primary/5'
          : 'ml-auto border-border bg-surface-2',
        failed && 'border-danger/30 bg-danger/5',
      )}
    >
      <div className="mb-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 font-mono text-sm leading-5">
        <span className="font-semibold text-fg">
          {t(kind === 'request' ? 'gateway.frames.request' : 'gateway.frames.response')}
        </span>
        <span className="text-muted">{event.operation}</span>
        <span className="text-primary">#{event.traceId}</span>
        <span className="text-faint">{formatFrameTime(event.time)}</span>
        <span className="text-faint">· {event.length}B</span>
        {event.durationMs !== undefined ? (
          <span className="text-faint">· {event.durationMs}ms</span>
        ) : null}
        <CopyButton value={copyValue} className="ml-auto size-5" />
      </div>
      <p
        className={cn(
          'whitespace-pre-wrap break-all font-mono text-sm leading-5',
          failed ? 'text-danger' : 'text-fg',
        )}
      >
        {event.hex || '—'}
      </p>
      {event.error ? <p className="mt-1 break-all text-sm leading-5 text-danger">{event.error}</p> : null}
    </div>
  )
}

function PendingResponse() {
  const { t } = useI18n()
  return (
    <div className="ml-auto w-fit max-w-[70%] rounded border border-dashed border-border px-3 py-2 text-[11px] text-faint">
      {t('gateway.frames.pendingResponse')}
    </div>
  )
}

function FrameConversation({
  exchanges,
  emptyText,
  error,
  paused,
}: {
  exchanges: FrameExchange[]
  emptyText: string
  error: boolean
  paused: boolean
}) {
  const { t } = useI18n()
  const boxRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (paused) return
    const box = boxRef.current
    if (box) box.scrollTop = box.scrollHeight
  }, [exchanges, paused])

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded border-[1.5px] border-border-strong/75 bg-surface shadow-[0_1px_3px_rgba(28,47,76,0.06)]">
      <div
        ref={boxRef}
        role="log"
        aria-live="polite"
        className="min-h-0 flex-1 space-y-4 overflow-y-scroll p-3"
      >
        {error && exchanges.length === 0 ? (
          <p className="text-xs text-danger">{t('gateway.frames.streamError')}</p>
        ) : exchanges.length === 0 ? (
          <p className="text-xs text-faint">{emptyText}</p>
        ) : (
          exchanges.map((exchange) => (
            <div key={exchange.traceId} className="space-y-1.5">
              {exchange.request ? <FrameBubble kind="request" event={exchange.request} /> : null}
              {exchange.response ? (
                <FrameBubble kind="response" event={exchange.response} />
              ) : (
                <PendingResponse />
              )}
            </div>
          ))
        )}
      </div>
    </div>
  )
}

export interface FrameAnalysisHeaderState {
  paused: boolean
  count: number
  ready: boolean
  setPaused: (paused: boolean) => void
  clear: () => void
}

export function FrameAnalysisPanel({
  collectionId,
  active,
  onHeaderStateChange,
}: {
  collectionId: string
  active: boolean
  onHeaderStateChange?: (state: FrameAnalysisHeaderState) => void
}) {
  const { t } = useI18n()
  const collection = useCollection(active ? collectionId || undefined : undefined)
  const detail = collection.data
  const supported = Boolean(detail && isFrameSupported(detail.driverType))
  const frameReady = frameCapable(detail)
  const { requests, responses, error, paused, setPaused, clear } = useFrameStreams(
    collectionId || undefined,
    active && frameReady,
  )
  const exchanges = useMemo(
    () => buildExchanges(requests, responses),
    [requests, responses],
  )

  useEffect(() => {
    onHeaderStateChange?.({
      paused,
      count: requests.length + responses.length,
      ready: frameReady,
      setPaused,
      clear,
    })
  }, [clear, frameReady, onHeaderStateChange, paused, requests.length, responses.length, setPaused])

  if (collection.isLoading) {
    return (
      <div className="flex min-h-0 flex-1 flex-col gap-3">
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="min-h-0 flex-1 w-full" />
      </div>
    )
  }

  if (collection.isError || !detail) {
    return (
      <div className="flex min-h-0 flex-1 flex-col">
        <ErrorState error={collection.error} onRetry={() => void collection.refetch()} />
      </div>
    )
  }

  const meta = collectStatusMeta(detail.collectStatus)
  const emptyText = frameReady
    ? t('gateway.frames.waiting')
    : t('gateway.frames.notCollecting', { status: t(meta.labelKey) })

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      {!supported ? (
        <Card className="flex min-h-0 flex-1 items-center justify-center">
          <EmptyState
            icon={<FileSearch className="size-5" />}
            title={t('gateway.frames.unsupported')}
            description={t('gateway.frames.unsupportedDescription')}
          />
        </Card>
      ) : (
        <FrameConversation
          exchanges={exchanges}
          emptyText={emptyText}
          error={error}
          paused={paused}
        />
      )}
    </div>
  )
}
