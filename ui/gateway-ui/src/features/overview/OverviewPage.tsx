import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import {
  Activity,
  ArrowRight,
  CircleCheck,
  CircleDashed,
  Cpu,
  HardDrive,
  MemoryStick,
  RadioTower,
  RefreshCw,
  Server,
  TriangleAlert,
  Waypoints,
} from 'lucide-react'
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  ErrorState,
  GaugeRing,
  PageHeader,
  ProgressBar,
  Skeleton,
  StatCard,
  formatDateTime,
  formatRelative,
  percentValue,
  toneForPercent,
  useI18n,
} from '@harns/core'
import {
  useCollections,
  useGatewayCpu,
  useGatewayDisk,
  useGatewayMem,
  useGatewayMeta,
  usePushConnectors,
  usePushGroups,
} from '@/api/hooks'
import type { Collection } from '@/api/types'
import { CollectStatusBadge } from '@/components/StatusBadge'
import { ProtocolLabel } from '@/components/ProtocolLabel'
import { driverTypeLabel, formatCycle, isAbnormalStatus } from '@/lib/constants'
import { useOperationLogs } from '@/lib/operation-log'

export function OverviewPage() {
  const { t } = useI18n()
  const meta = useGatewayMeta()
  const cpu = useGatewayCpu()
  const mem = useGatewayMem()
  const disk = useGatewayDisk()
  const collections = useCollections({ page: 1, pageSize: 100 })
  const connectors = usePushConnectors({ page: 1, pageSize: 100 })
  const groups = usePushGroups({ page: 1, pageSize: 100 })
  const operationLogs = useOperationLogs()

  const list = collections.data?.list ?? []
  const connectorList = connectors.data?.list ?? []

  const stats = useMemo(() => {
    const collecting = list.filter((item) => item.collectStatus === 'collecting').length
    return {
      total: collections.data?.total ?? list.length,
      collecting,
      abnormal: list.filter((item) => isAbnormalStatus(item.collectStatus)).length,
      stopped: list.filter((item) => item.collectStatus === 'stopped').length,
      runningConnectors: connectorList.filter((item) => item.status === 'running').length,
      connectors: connectors.data?.total ?? connectorList.length,
      groups: groups.data?.total ?? 0,
    }
  }, [list, collections.data, connectorList, connectors.data, groups.data])

  const protocolBreakdown = useMemo(() => {
    const counts = new Map<string, number>()
    for (const item of list) {
      counts.set(item.driverType, (counts.get(item.driverType) ?? 0) + 1)
    }
    return [...counts.entries()].sort((a, b) => b[1] - a[1])
  }, [list])

  const attention = useMemo(
    () =>
      list
        .filter((item) => isAbnormalStatus(item.collectStatus))
        .sort((a, b) => a.name.localeCompare(b.name))
        .slice(0, 5),
    [list],
  )

  const recentCollections = useMemo(
    () =>
      [...list]
        .sort((a, b) => new Date(b.modTime).getTime() - new Date(a.modTime).getTime())
        .slice(0, 5),
    [list],
  )

  const recentOperations = operationLogs.slice(0, 6)
  const refreshing =
    meta.isFetching ||
    cpu.isFetching ||
    mem.isFetching ||
    disk.isFetching ||
    collections.isFetching ||
    connectors.isFetching

  const refresh = () => {
    void meta.refetch()
    void cpu.refetch()
    void mem.refetch()
    void disk.refetch()
    void collections.refetch()
    void connectors.refetch()
    void groups.refetch()
  }

  const memData = mem.data?.mem
  const diskEntries = Object.entries(disk.data?.disk ?? {})
  const error = collections.error ?? connectors.error

  return (
    <div className="space-y-3">
      <PageHeader
        title={t('gateway.overview.title')}
        description={t('gateway.overview.description')}
        actions={
          <Button
            variant="outline"
            loading={refreshing}
            onClick={refresh}
            icon={<RefreshCw className="size-3.5" />}
          >
            {t('common.action.refresh')}
          </Button>
        }
      />

      <Card className="relative overflow-hidden">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,var(--primary-soft),transparent_62%)] opacity-70"
        />
        <div className="relative flex flex-col gap-4 p-4 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex size-11 shrink-0 items-center justify-center rounded-xl border border-border bg-surface-2 text-primary">
              <Server className="size-5" />
            </div>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                {meta.isLoading ? (
                  <Skeleton className="h-6 w-40" />
                ) : (
                  <h2 className="truncate text-lg font-semibold tracking-tight">
                    {meta.data?.name ?? t('gateway.overview.unknownGateway')}
                  </h2>
                )}
                <Badge tone={meta.isError ? 'danger' : 'success'} dot>
                  {meta.isError
                    ? t('gateway.overview.metaError')
                    : t('gateway.overview.metaOk')}
                </Badge>
              </div>
              <p className="mt-0.5 truncate font-mono text-[11px] text-faint">
                {meta.data?.id ?? '—'}
              </p>
            </div>
          </div>
          <div className="grid shrink-0 grid-cols-2 gap-x-6 gap-y-2 text-[11px] sm:grid-cols-3">
            <div>
              <p className="text-faint">{t('gateway.performance.field.version')}</p>
              <p className="mt-0.5 font-mono text-xs text-fg">{meta.data?.eTag ?? '—'}</p>
            </div>
            <div>
              <p className="text-faint">{t('gateway.performance.field.modTime')}</p>
              <p className="mt-0.5 text-xs text-fg">{formatDateTime(meta.data?.modTime)}</p>
            </div>
            <div className="hidden sm:block">
              <p className="text-faint">{t('gateway.overview.field.collectCycle')}</p>
              <p className="mt-0.5 text-xs text-fg">
                {list.length > 0
                  ? formatCycle(
                      Math.round(
                        list.reduce((sum, item) => sum + item.collectorCycleMs, 0) / list.length,
                      ),
                    )
                  : '—'}
              </p>
            </div>
          </div>
        </div>
      </Card>

      {error ? (
        <ErrorState error={error} onRetry={() => void collections.refetch()} compact />
      ) : null}

      <div className="grid gap-2.5 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label={t('gateway.overview.stat.collections')}
          value={stats.total}
          hint={t('gateway.overview.stat.collectionsHint', {
            collecting: stats.collecting,
            stopped: stats.stopped,
          })}
          icon={<Activity className="size-3.5" />}
        />
        <StatCard
          label={t('gateway.overview.stat.abnormal')}
          value={stats.abnormal}
          hint={t('gateway.overview.stat.abnormalHint')}
          icon={<TriangleAlert className="size-3.5" />}
          className={stats.abnormal > 0 ? 'border-danger/30' : undefined}
        />
        <StatCard
          label={t('gateway.overview.stat.connectors')}
          value={`${stats.runningConnectors}/${stats.connectors}`}
          hint={t('gateway.overview.stat.connectorsHint')}
          icon={<RadioTower className="size-3.5" />}
        />
        <StatCard
          label={t('gateway.overview.stat.groups')}
          value={stats.groups}
          hint={t('gateway.overview.stat.groupsHint')}
          icon={<Waypoints className="size-3.5" />}
        />
      </div>

      <div className="grid gap-3 xl:grid-cols-3">
        <Card>
          <CardHeader>
            <div>
              <CardTitle className="flex items-center gap-1.5">
                <Cpu className="size-4 text-faint" />
                {t('gateway.overview.resource.cpu')}
              </CardTitle>
              <CardDescription>{t('gateway.overview.resource.cpuHint')}</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            {cpu.isError ? (
              <p className="text-xs text-danger">{t('gateway.performance.cpu.unavailable')}</p>
            ) : (
              <div className="flex items-center justify-around gap-2">
                {(['1min', '5min', '15min'] as const).map((key) => {
                  const value = percentValue(cpu.data?.cpus?.[key])
                  return (
                    <GaugeRing
                      key={key}
                      value={value}
                      label={t(`gateway.performance.cpu.${key}`)}
                      size={72}
                      strokeWidth={6}
                      tone={toneForPercent(value)}
                    />
                  )
                })}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle className="flex items-center gap-1.5">
                <MemoryStick className="size-4 text-faint" />
                {t('gateway.overview.resource.memory')}
              </CardTitle>
              <CardDescription>{t('gateway.overview.resource.memoryHint')}</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            {mem.isError || !memData ? (
              <p className="text-xs text-danger">{t('gateway.performance.memory.unavailable')}</p>
            ) : (
              <div className="flex items-center gap-4">
                <GaugeRing
                  value={percentValue(memData.UsedPercent)}
                  label={memData.UsedPercent}
                  size={84}
                  tone={toneForPercent(percentValue(memData.UsedPercent))}
                />
                <div className="min-w-0 space-y-1 text-xs">
                  <p className="text-faint">{t('gateway.overview.resource.used')}</p>
                  <p className="font-medium text-fg tabular">{memData.Used}</p>
                  <p className="text-faint">{t('gateway.overview.resource.total')}</p>
                  <p className="font-medium text-fg tabular">{memData.Total}</p>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle className="flex items-center gap-1.5">
                <HardDrive className="size-4 text-faint" />
                {t('gateway.overview.resource.disk')}
              </CardTitle>
              <CardDescription>{t('gateway.overview.resource.diskHint')}</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            {disk.isError ? (
              <p className="text-xs text-danger">{t('gateway.performance.disk.unavailable')}</p>
            ) : diskEntries.length === 0 ? (
              <p className="text-xs text-faint">{t('gateway.performance.disk.empty')}</p>
            ) : (
              <div className="space-y-2.5">
                {diskEntries.slice(0, 3).map(([mount, info]) => {
                  const value = percentValue(info.UsedPercent)
                  return (
                    <div key={mount}>
                      <div className="mb-1 flex items-baseline justify-between gap-2 text-[11px]">
                        <span className="truncate font-mono text-fg">{mount}</span>
                        <span className="shrink-0 text-muted tabular">{info.UsedPercent}</span>
                      </div>
                      <ProgressBar
                        value={value}
                        tone={value >= 90 ? 'danger' : value >= 75 ? 'warning' : 'success'}
                      />
                    </div>
                  )
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="grid items-start gap-3 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('gateway.overview.attention.title')}</CardTitle>
              <CardDescription>{t('gateway.overview.attention.description')}</CardDescription>
            </div>
            <Link
              to="/collections"
              className="inline-flex items-center gap-1 text-[11px] text-muted transition-colors hover:text-fg"
            >
              {t('gateway.overview.viewAll')}
              <ArrowRight className="size-3" />
            </Link>
          </CardHeader>
          <CardContent>
            {collections.isLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 3 }).map((_, index) => (
                  <Skeleton key={index} className="h-10 w-full" />
                ))}
              </div>
            ) : attention.length === 0 ? (
              <div className="flex items-center gap-2 rounded-xl border border-emerald-200 bg-emerald-50/60 px-3 py-2.5 text-xs text-emerald-700 dark:border-emerald-600/30 dark:bg-emerald-500/10 dark:text-emerald-300">
                <CircleCheck className="size-4 shrink-0" />
                {t('gateway.overview.attention.empty')}
              </div>
            ) : (
              <div className="space-y-1.5">
                {attention.map((collection) => (
                  <AttentionRow key={collection.id} collection={collection} />
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('gateway.overview.recent.title')}</CardTitle>
              <CardDescription>{t('gateway.overview.recent.description')}</CardDescription>
            </div>
            <Link
              to="/collections"
              className="inline-flex items-center gap-1 text-[11px] text-muted transition-colors hover:text-fg"
            >
              {t('gateway.overview.viewAll')}
              <ArrowRight className="size-3" />
            </Link>
          </CardHeader>
          <CardContent>
            {collections.isLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 3 }).map((_, index) => (
                  <Skeleton key={index} className="h-10 w-full" />
                ))}
              </div>
            ) : recentCollections.length === 0 ? (
              <EmptyState
                icon={<CircleDashed className="size-5" />}
                title={t('gateway.collection.empty.title')}
                description={t('gateway.collection.empty.description')}
                className="py-6"
                action={
                  <Link
                    to="/collections/new"
                    className="inline-flex h-8 items-center rounded bg-primary px-3 text-sm font-medium text-primary-fg"
                  >
                    {t('gateway.collection.create')}
                  </Link>
                }
              />
            ) : (
              <div className="space-y-1.5">
                {recentCollections.map((collection) => (
                  <Link
                    key={collection.id}
                    to={`/collections/${collection.id}`}
                    className="flex items-center justify-between gap-2 rounded-lg border border-border px-2.5 py-2 transition-colors hover:border-border-strong hover:bg-surface-2"
                  >
                    <div className="min-w-0">
                      <p className="truncate text-xs font-medium text-fg">{collection.name}</p>
                      <p className="mt-0.5 truncate text-[10px] text-faint">
                        <ProtocolLabel
                          type={collection.driverType}
                          model={collection.driverModel}
                        />{' '}
                        · {formatRelative(collection.modTime)}
                      </p>
                    </div>
                    <CollectStatusBadge status={collection.collectStatus} />
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('gateway.overview.protocol.title')}</CardTitle>
              <CardDescription>{t('gateway.overview.protocol.description')}</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            {protocolBreakdown.length === 0 ? (
              <p className="text-xs text-faint">{t('gateway.overview.protocol.empty')}</p>
            ) : (
              <div className="space-y-2.5">
                {protocolBreakdown.map(([type, count]) => {
                  const ratio = stats.total > 0 ? (count / stats.total) * 100 : 0
                  return (
                    <div key={type}>
                      <div className="mb-1 flex items-center justify-between gap-2 text-xs">
                        <span className="text-fg">{driverTypeLabel(type)}</span>
                        <span className="text-muted tabular">
                          {count} · {Math.round(ratio)}%
                        </span>
                      </div>
                      <ProgressBar value={ratio} />
                    </div>
                  )
                })}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('gateway.overview.operations.title')}</CardTitle>
              <CardDescription>{t('gateway.overview.operations.description')}</CardDescription>
            </div>
            <Link
              to="/logs"
              className="inline-flex items-center gap-1 text-[11px] text-muted transition-colors hover:text-fg"
            >
              {t('gateway.overview.viewAll')}
              <ArrowRight className="size-3" />
            </Link>
          </CardHeader>
          <CardContent>
            {recentOperations.length === 0 ? (
              <p className="text-xs text-faint">{t('gateway.overview.operations.empty')}</p>
            ) : (
              <div className="space-y-1">
                {recentOperations.map((entry) => (
                  <div
                    key={entry.id}
                    className="flex items-center justify-between gap-2 rounded-lg px-1.5 py-1.5 transition-colors hover:bg-surface-2"
                  >
                    <div className="flex min-w-0 items-center gap-2">
                      <span
                        className={
                          entry.result === 'failure'
                            ? 'size-1.5 shrink-0 rounded-full bg-danger'
                            : 'size-1.5 shrink-0 rounded-full bg-emerald-500'
                        }
                      />
                      <span className="truncate text-xs text-fg">{entry.target || '—'}</span>
                      <Badge tone="neutral">
                        {t(`gateway.logs.operation.action.${entry.action}`)}
                      </Badge>
                    </div>
                    <span className="shrink-0 text-[10px] text-faint">
                      {formatRelative(entry.time)}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function AttentionRow({ collection }: { collection: Collection }) {
  return (
    <Link
      to={`/collections/${collection.id}`}
      className="flex items-center justify-between gap-2 rounded-lg border border-danger/20 bg-danger/5 px-2.5 py-2 transition-colors hover:border-danger/40"
    >
      <div className="min-w-0">
        <p className="truncate text-xs font-medium text-fg">{collection.name}</p>
        <p className="mt-0.5 truncate text-[10px] text-muted">
          <ProtocolLabel type={collection.driverType} model={collection.driverModel} />
        </p>
      </div>
      <CollectStatusBadge status={collection.collectStatus} />
    </Link>
  )
}
