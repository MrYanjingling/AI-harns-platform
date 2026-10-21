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
import { useDevices } from '@/api/hooks'
import type {
  GatewayDeviceDetail,
  GatewayLogEntry,
  GatewayLogLevel,
  OperationLogEntry,
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

function eventStatus(value: string): EventStatus {
  return EVENT_STATUSES.includes(value as EventStatus) ? (value as EventStatus) : 'error'
}

function levelForEvent(status: EventStatus): GatewayLogLevel {
  if (status === 'collectingError' || status === 'error') return 'error'
  if (status === 'unconnected' || status === 'emptyVariable') return 'warn'
  return 'info'
}

function buildEventLogs(devices: GatewayDeviceDetail[], t: (key: string, vars?: Record<string, string | number>) => string): GatewayLogEntry[] {
  return devices
    .filter((device) => device.collectStatus !== 'collecting')
    .flatMap((device) => {
      const status = eventStatus(device.collectStatus)
      return [
        {
          id: `event-${device.id}-${status}`,
          time: device.modTime,
          level: levelForEvent(status),
          source: 'event' as const,
          title: t(`gateway.logs.event.title.${status}`),
          message: t(`gateway.logs.event.description.${status}`, { name: device.name }),
          deviceId: device.id,
        },
      ]
    })
}

function buildOperationLogs(
  entries: OperationLogEntry[],
  t: (key: string, vars?: Record<string, string | number>) => string,
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
    deviceId: undefined,
  }))
}

const LEVEL_TONE: Record<GatewayLogLevel, 'info' | 'warning' | 'danger'> = {
  info: 'info',
  warn: 'warning',
  error: 'danger',
}

export function LogsPage() {
  const { t } = useI18n()
  const devices = useDevices()
  const operationEntries = useOperationLogs()
  const [tab, setTab] = useState<TabValue>('events')
  const [keyword, setKeyword] = useState('')
  const [levelFilter, setLevelFilter] = useState<'all' | GatewayLogLevel>('all')
  const [clearOpen, setClearOpen] = useState(false)

  const allLogs = useMemo(() => {
    if (tab === 'events') {
      return buildEventLogs(devices.data ?? [], t)
    }
    return buildOperationLogs(operationEntries, t)
  }, [tab, devices.data, operationEntries, t])

  const filtered = useMemo(() => {
    const query = keyword.trim().toLowerCase()
    return allLogs.filter((entry) => {
      if (levelFilter !== 'all' && entry.level !== levelFilter) return false
      if (!query) return true
      return [entry.title, entry.message, entry.deviceId ?? '']
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

  const loading = tab === 'events' && devices.isLoading
  const error = tab === 'events' ? devices.error : null

  return (
    <div className="space-y-4">
      <PageHeader
        title={t('gateway.logs.title')}
        description={t('gateway.logs.description')}
        actions={
          <>
            <Button
              variant="outline"
              loading={devices.isFetching}
              onClick={() => void devices.refetch()}
              icon={<RefreshCw className="size-4" />}
            >
              {t('gateway.logs.refresh')}
            </Button>
            <Button variant="outline" onClick={exportLogs} icon={<Download className="size-4" />}>
              {t('gateway.logs.export')}
            </Button>
            {tab === 'operations' ? (
              <Button
                variant="dangerSoft"
                onClick={() => setClearOpen(true)}
                icon={<Trash2 className="size-4" />}
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
            <CardTitle>{t('gateway.logs.file.title')}</CardTitle>
            <CardDescription>{t('gateway.logs.file.description')}</CardDescription>
          </div>
          <FileText className="size-4 text-faint" />
        </CardHeader>
        <CardContent className="pt-0">
          <code className="rounded bg-surface-2 px-2 py-1 font-mono text-xs text-muted">
            ../log/harns-gateway.log
          </code>
        </CardContent>
      </Card>

      <div className="overflow-hidden rounded border border-border bg-surface">
        <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
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
            onChange={(event) =>
              setLevelFilter(event.target.value as 'all' | GatewayLogLevel)
            }
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
            <ErrorState error={error} onRetry={() => void devices.refetch()} />
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
                    {entry.deviceId ? (
                      <span className="font-mono">{entry.deviceId}</span>
                    ) : (
                      '—'
                    )}
                  </TD>
                  <TD>
                    <p className="text-sm text-fg">{entry.title}</p>
                    <p className="text-xs text-muted">{entry.message}</p>
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
        )}
      </div>

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
