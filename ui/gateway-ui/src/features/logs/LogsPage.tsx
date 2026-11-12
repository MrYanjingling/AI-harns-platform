import { useMemo, useState } from 'react'
import { Download, FileText, RefreshCw, Search, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  ConfirmDialog,
  EmptyState,
  ErrorState,
  Input,
  PageHeader,
  Select,
  Table,
  TableSkeleton,
  TBody,
  TD,
  TH,
  THead,
  TR,
  Tabs,
  formatDateTime,
  useI18n,
} from '@harns/core'
import {
  useCollections,
  usePushConnectors,
} from '@/api/hooks'
import type {
  Collection,
  GatewayLogEntry,
  GatewayLogLevel,
  OperationLogEntry,
  PushConnector,
} from '@/api/types'
import { clearOperationLogs, useOperationLogs } from '@/lib/operation-log'

type TabValue = 'events' | 'operations'

const EVENT_STATUSES = [
  'collectingError',
  'unconnected',
  'emptyVariable',
  'error',
  'stopped',
] as const

type EventStatus = (typeof EVENT_STATUSES)[number]

function isEventStatus(value: string): value is EventStatus {
  return EVENT_STATUSES.includes(value as EventStatus)
}

function levelForStatus(status: EventStatus): GatewayLogLevel {
  if (status === 'collectingError' || status === 'error') return 'error'
  if (status === 'unconnected' || status === 'emptyVariable') return 'warn'
  return 'info'
}

type Translate = ReturnType<typeof useI18n>['t']

function buildCollectionEvents(
  collections: Collection[],
  t: Translate,
): GatewayLogEntry[] {
  return collections
    .filter((collection) => collection.collectStatus !== 'collecting')
    .flatMap((collection) => {
      const status = isEventStatus(collection.collectStatus) ? collection.collectStatus : 'error'
      return [
        {
          id: `collection-${collection.id}-${status}`,
          time: collection.modTime,
          level: levelForStatus(status),
          source: 'event' as const,
          title: t(`gateway.logs.event.title.${status}`),
          message: t(`gateway.logs.event.description.${status}`, {
            name: collection.name,
            protocol: collection.driverModel,
          }),
          collectionId: collection.id,
        },
      ]
    })
}

function buildConnectorEvents(
  connectors: PushConnector[],
  t: Translate,
): GatewayLogEntry[] {
  return connectors
    .filter((connector) => connector.status !== 'running')
    .map((connector) => ({
      id: `connector-${connector.id}-${connector.status}`,
      time: connector.updatedTime,
      level: 'warn' as const,
      source: 'event' as const,
      title: t('gateway.logs.event.connectorStopped'),
      message: t('gateway.logs.event.connectorStoppedDescription', {
        name: connector.name,
      }),
    }))
}

function buildOperationLogs(
  entries: OperationLogEntry[],
  t: Translate,
): GatewayLogEntry[] {
  return entries.map((entry) => ({
    id: `operation-${entry.id}`,
    time: entry.time,
    level: entry.result === 'failure' ? 'error' : 'info',
    source: 'operation' as const,
    title: t(`gateway.logs.operation.action.${entry.action}`),
    message: entry.message
      ? `${t(`gateway.logs.operation.result.${entry.result}`)} · ${entry.message}`
      : t(`gateway.logs.operation.result.${entry.result}`),
  }))
}

const LEVEL_TONE: Record<GatewayLogLevel, 'info' | 'warning' | 'danger'> = {
  info: 'info',
  warn: 'warning',
  error: 'danger',
}

export function LogsPage() {
  const { t } = useI18n()
  const collections = useCollections({ page: 1, pageSize: 100 })
  const connectors = usePushConnectors({ page: 1, pageSize: 100 })
  const operationEntries = useOperationLogs()
  const [tab, setTab] = useState<TabValue>('events')
  const [keyword, setKeyword] = useState('')
  const [levelFilter, setLevelFilter] = useState<'all' | GatewayLogLevel>('all')
  const [clearOpen, setClearOpen] = useState(false)

  const allLogs = useMemo(() => {
    if (tab === 'events') {
      return [
        ...buildCollectionEvents(collections.data?.list ?? [], t),
        ...buildConnectorEvents(connectors.data?.list ?? [], t),
      ].sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime())
    }
    return buildOperationLogs(operationEntries, t)
  }, [tab, collections.data, connectors.data, operationEntries, t])

  const filtered = useMemo(() => {
    const query = keyword.trim().toLowerCase()
    return allLogs.filter((entry) => {
      if (levelFilter !== 'all' && entry.level !== levelFilter) return false
      if (!query) return true
      return [entry.title, entry.message, entry.collectionId ?? '']
        .join(' ')
        .toLowerCase()
        .includes(query)
    })
  }, [allLogs, keyword, levelFilter])

  const exportLogs = () => {
    const payload = JSON.stringify(filtered, null, 2)
    const blob = new Blob([payload], { type: 'application/json;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `gateway-logs-${tab}-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.json`
    anchor.click()
    URL.revokeObjectURL(url)
    toast.success(t('gateway.logs.toast.exported'))
  }

  const clear = () => {
    clearOperationLogs()
    setClearOpen(false)
    toast.success(t('gateway.logs.toast.cleared'))
  }

  const loading = tab === 'events'
    ? collections.isLoading || connectors.isLoading
    : false
  const error = tab === 'events' ? collections.error ?? connectors.error : null

  const refresh = () => {
    void collections.refetch()
    void connectors.refetch()
  }

  return (
    <div className="space-y-3">
      <PageHeader
        title={t('gateway.logs.title')}
        description={t('gateway.logs.description')}
        actions={
          <>
            <Button
              variant="outline"
              loading={collections.isFetching || connectors.isFetching}
              onClick={refresh}
              icon={<RefreshCw className="size-3.5" />}
            >
              {t('common.action.refresh')}
            </Button>
            <Button
              variant="outline"
              onClick={exportLogs}
              icon={<Download className="size-3.5" />}
            >
              {t('gateway.logs.export')}
            </Button>
            {tab === 'operations' ? (
              <Button
                variant="dangerSoft"
                onClick={() => setClearOpen(true)}
                icon={<Trash2 className="size-3.5" />}
              >
                {t('gateway.logs.clear')}
              </Button>
            ) : null}
          </>
        }
      />

      <Card>
        <CardHeader>
          <div>
            <CardTitle className="flex items-center gap-1.5">
              <FileText className="size-4 text-faint" />
              {t('gateway.logs.file.title')}
            </CardTitle>
            <CardDescription>{t('gateway.logs.file.description')}</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="pt-0">
          <code className="rounded bg-surface-2 px-2 py-1 font-mono text-xs text-muted">
            ../log/harns-gateway.log
          </code>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-2 border-b border-border p-2.5">
          <Tabs
            items={[
              { value: 'events', label: t('gateway.logs.tab.events') },
              { value: 'operations', label: t('gateway.logs.tab.operations') },
            ]}
            value={tab}
            onChange={setTab}
          />
          <div className="relative min-w-56 flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-faint" />
            <Input
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder={t('gateway.logs.searchPlaceholder')}
              aria-label={t('common.action.search')}
              className="pl-8"
            />
          </div>
          <Select
            value={levelFilter}
            onChange={(event) => setLevelFilter(event.target.value as 'all' | GatewayLogLevel)}
            aria-label={t('gateway.logs.filter.level')}
            className="w-36"
          >
            <option value="all">{t('gateway.logs.filter.allLevels')}</option>
            <option value="info">{t('gateway.logs.level.info')}</option>
            <option value="warn">{t('gateway.logs.level.warn')}</option>
            <option value="error">{t('gateway.logs.level.error')}</option>
          </Select>
        </div>

        {loading ? (
          <TableSkeleton rows={6} cols={5} />
        ) : error ? (
          <div className="p-3">
            <ErrorState error={error} onRetry={refresh} />
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            title={t('gateway.logs.empty.title')}
            description={t('gateway.logs.empty.description')}
          />
        ) : (
          <Table>
            <THead>
              <TR>
                <TH>{t('gateway.logs.columns.time')}</TH>
                <TH>{t('gateway.logs.columns.level')}</TH>
                <TH>{t('gateway.logs.columns.source')}</TH>
                <TH>{t('gateway.logs.columns.target')}</TH>
                <TH>{t('gateway.logs.columns.message')}</TH>
              </TR>
            </THead>
            <TBody>
              {filtered.map((entry) => (
                <TR key={entry.id}>
                  <TD className="whitespace-nowrap text-xs text-muted tabular">
                    {formatDateTime(entry.time)}
                  </TD>
                  <TD>
                    <Badge tone={LEVEL_TONE[entry.level]}>
                      {t(`gateway.logs.level.${entry.level}`)}
                    </Badge>
                  </TD>
                  <TD className="whitespace-nowrap text-xs text-muted">
                    {entry.source === 'event'
                      ? t('gateway.logs.source.event')
                      : t('gateway.logs.source.operation')}
                  </TD>
                  <TD className="max-w-48 truncate text-xs text-muted">
                    {entry.collectionId ? (
                      <span className="font-mono">{entry.collectionId}</span>
                    ) : (
                      '—'
                    )}
                  </TD>
                  <TD>
                    <p className="text-[13px] text-fg">{entry.title}</p>
                    <p className="text-xs text-muted">{entry.message}</p>
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <ConfirmDialog
        open={clearOpen}
        onClose={() => setClearOpen(false)}
        onConfirm={clear}
        danger
        title={t('gateway.logs.clearConfirm.title')}
        description={t('gateway.logs.clearConfirm.description')}
        confirmText={t('common.action.clear')}
      />
    </div>
  )
}
