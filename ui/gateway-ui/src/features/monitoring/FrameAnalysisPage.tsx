import { useEffect, useMemo, useRef, type ReactNode } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { AlertTriangle, FileSearch, Pause, Play, Trash2 } from 'lucide-react'
import {
  Badge,
  Button,
  Card,
  CardContent,
  CopyButton,
  EmptyState,
  ErrorState,
  Skeleton,
  cn,
  formatApiTimeMs,
  parseDate,
  useI18n,
  type Translate,
} from '@harns/core'
import { useCollection } from '@/api/hooks'
import type { CollectionDetail, FrameEvent } from '@/api/types'
import { CollectStatusBadge } from '@/components/StatusBadge'
import {
  collectStatusMeta,
  driverModelLabel,
  driverTypeLabel,
  formatCycle,
} from '@/lib/constants'
import { useFrameStreams } from './useFrameStreams'

const FRAME_STATUSES = new Set(['collecting', 'collectingError'])

function isFrameSupported(driverType: string | undefined): boolean {
  return Boolean(driverType) && driverType !== 'opcUa'
}

function frameCapable(collection: CollectionDetail | undefined): boolean {
  if (!collection) return false
  return isFrameSupported(collection.driverType) && FRAME_STATUSES.has(collection.collectStatus)
}

function endpointOf(collection: CollectionDetail): string {
  const location = collection.protocolConfig.location
  const port = collection.protocolConfig.port
  if (typeof location !== 'string' || !location) return '—'
  return typeof port === 'number' || typeof port === 'string' ? `${location}:${port}` : location
}

function formatFrameTime(value: string): string {
  const date = parseDate(value)
  return date ? formatApiTimeMs(date) : '—'
}

function InfoItem({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 items-start gap-[5px]">
      <p className="shrink-0 text-sm leading-5 font-medium text-muted">{label}</p>
      <div className="min-w-0 flex-1 text-sm leading-5 text-fg">{children}</div>
    </div>
  )
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
}: {
  exchanges: FrameExchange[]
  emptyText: string
  error: boolean
}) {
  const { t } = useI18n()
  const boxRef = useRef<HTMLDivElement>(null)
  const stickToBottom = useRef(true)

  useEffect(() => {
    if (!stickToBottom.current) return
    const box = boxRef.current
    if (box) box.scrollTop = box.scrollHeight
  }, [exchanges])

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded border-[1.5px] border-border-strong/75 bg-surface shadow-[0_1px_3px_rgba(28,47,76,0.06)]">
      <div
        ref={boxRef}
        role="log"
        aria-live="polite"
        onScroll={() => {
          const box = boxRef.current
          if (!box) return
          stickToBottom.current = box.scrollHeight - box.scrollTop - box.clientHeight < 24
        }}
        className="min-h-0 flex-1 space-y-4 overflow-auto p-3"
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

function statusText(
  t: Translate,
  frameReady: boolean,
  connected: boolean,
  error: boolean,
): string {
  if (!frameReady) return t('gateway.frames.waitingForCollecting')
  if (error) return t('gateway.frames.streamError')
  if (!connected) return t('gateway.frames.connecting')
  return t('gateway.frames.connected')
}

export function FrameAnalysisPage() {
  const { t } = useI18n()
  const [searchParams] = useSearchParams()
  const collectionId = searchParams.get('collectionId') ?? ''
  const collection = useCollection(collectionId || undefined)
  const detail = collection.data
  const supported = Boolean(detail && isFrameSupported(detail.driverType))
  const frameReady = frameCapable(detail)
  const { requests, responses, connected, error, paused, setPaused, clear } = useFrameStreams(
    collectionId || undefined,
    frameReady,
  )
  const exchanges = useMemo(
    () => buildExchanges(requests, responses),
    [requests, responses],
  )

  if (!collectionId) {
    return (
      <div className="flex min-h-0 flex-1 flex-col">
        <Card className="flex min-h-0 flex-1 items-center justify-center">
          <EmptyState
            icon={<FileSearch className="size-5" />}
            title={t('gateway.frames.missingCollection')}
            action={
              <Link
                to="/monitoring/points"
                className="text-sm text-primary transition-colors hover:text-primary-hover"
              >
                {t('gateway.frames.backToMonitoring')}
              </Link>
            }
          />
        </Card>
      </div>
    )
  }

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
      <Card className="shrink-0">
        <CardContent className="pt-4">
          <div className="grid grid-cols-1 gap-x-8 gap-y-3 sm:grid-cols-2 xl:grid-cols-4">
            <InfoItem label={t('gateway.frames.info.collection')}>
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                <span className="truncate text-sm font-semibold text-fg">{detail.name}</span>
                <CollectStatusBadge status={detail.collectStatus} error={detail.lastError} />
              </div>
            </InfoItem>
            <InfoItem label={t('gateway.frames.info.protocol')}>
              <p
                className="truncate text-sm leading-5 text-fg"
                title={`${driverTypeLabel(detail.driverType) || '—'} / ${
                  driverModelLabel(detail.driverModel) || '—'
                }`}
              >
                {driverTypeLabel(detail.driverType) || '—'} /{' '}
                {driverModelLabel(detail.driverModel) || '—'}
              </p>
            </InfoItem>
            <InfoItem label={t('gateway.frames.info.cycle')}>
              <span className="font-mono">{formatCycle(detail.collectorCycleMs)}</span>
            </InfoItem>
            <InfoItem label={t('gateway.frames.info.endpoint')}>
              <span className="block max-w-[260px] truncate font-mono" title={endpointOf(detail)}>
                {endpointOf(detail)}
              </span>
            </InfoItem>
          </div>
          {detail.lastError ? (
            <p className="mt-3 flex items-start gap-1.5 text-xs leading-5 text-danger">
              <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
              <span className="min-w-0 break-all">{detail.lastError}</span>
            </p>
          ) : null}
        </CardContent>
      </Card>

      {!supported ? (
        <Card className="flex min-h-0 flex-1 items-center justify-center">
          <EmptyState
            icon={<FileSearch className="size-5" />}
            title={t('gateway.frames.unsupported')}
            description={t('gateway.frames.unsupportedDescription')}
          />
        </Card>
      ) : (
        <>
          <div className="flex shrink-0 flex-wrap items-center gap-2">
            <span className="inline-flex items-center gap-1.5 text-sm font-medium text-muted">
              <span
                className={cn(
                  'size-2 rounded-full',
                  frameReady
                    ? error
                      ? 'bg-danger'
                      : connected
                        ? 'bg-success'
                        : 'bg-warning'
                    : 'bg-border-strong',
                )}
              />
              {statusText(t, frameReady, connected, error)}
            </span>
            <div className="ml-auto flex items-center gap-2">
              <Button
                variant="outline"
                className="h-8"
                disabled={!frameReady}
                icon={paused ? <Play className="size-3.5" /> : <Pause className="size-3.5" />}
                onClick={() => setPaused(!paused)}
              >
                {paused ? t('gateway.frames.resume') : t('gateway.frames.pause')}
              </Button>
              <Button
                variant="ghost"
                className="h-8"
                disabled={requests.length + responses.length === 0}
                icon={<Trash2 className="size-3.5" />}
                onClick={clear}
              >
                {t('gateway.frames.clear')}
              </Button>
              <Badge tone="neutral" className="border-[1.5px] px-2 py-1">
                {t('gateway.frames.items', { count: requests.length + responses.length })}
              </Badge>
            </div>
          </div>

          <FrameConversation
            exchanges={exchanges}
            emptyText={emptyText}
            error={error}
          />
        </>
      )}
    </div>
  )
}
