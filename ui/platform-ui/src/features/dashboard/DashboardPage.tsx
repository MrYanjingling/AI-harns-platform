import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  ArrowRight,
  Cpu,
  Eye,
  EyeOff,
  HardDrive,
  MemoryStick,
  RefreshCw,
  Router,
  Server,
  ShieldAlert,
  Activity,
} from 'lucide-react'
import { useGatewayCpu, useGatewayDisk, useGatewayMem, useGatewayMeta, useDevices } from '@/api/hooks'
import type { DeviceStatus } from '@/api/types'
import { useI18n, type MessageKey } from '@/i18n'
import { DEVICE_STATUS } from '@/lib/constants'
import { formatRelative, maskSecret, percentValue } from '@/lib/format'
import { cn } from '@/lib/cn'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { CopyButton } from '@/components/ui/CopyButton'
import { EmptyState } from '@/components/ui/EmptyState'
import { ProgressBar } from '@/components/ui/ProgressBar'
import { Skeleton, TableSkeleton } from '@/components/ui/Skeleton'
import { TD, TH, THead, TBody, TR, Table } from '@/components/ui/Table'
import { GaugeRing, toneForPercent } from '@/components/charts/GaugeRing'
import { PageHeader } from '@/components/shared/PageHeader'
import { StatCard } from '@/components/shared/StatCard'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { DeviceTypeBadge } from '@/components/shared/DeviceTypeBadge'
import { ErrorState } from '@/components/shared/ErrorState'
import { toast } from 'sonner'

const STATUS_ORDER: DeviceStatus[] = [
  'collecting',
  'collectingError',
  'unconnected',
  'emptyVariable',
  'stopped',
  'error',
]

const STATUS_BAR: Record<DeviceStatus, string> = {
  collecting: 'bg-emerald-500',
  collectingError: 'bg-amber-500',
  unconnected: 'bg-zinc-400',
  emptyVariable: 'bg-violet-500',
  stopped: 'bg-slate-400',
  error: 'bg-rose-500',
}

const CPU_GAUGES: Array<{ key: '1min' | '5min' | '15min'; labelKey: MessageKey }> = [
  { key: '1min', labelKey: 'dashboard.resources.cpu1m' },
  { key: '5min', labelKey: 'dashboard.resources.cpu5m' },
  { key: '15min', labelKey: 'dashboard.resources.cpu15m' },
]

export function DashboardPage() {
  const { t } = useI18n()
  const meta = useGatewayMeta()
  const cpu = useGatewayCpu()
  const mem = useGatewayMem()
  const disk = useGatewayDisk()
  const devices = useDeviceList()
  const [secretVisible, setSecretVisible] = useState(false)

  const list = devices.data ?? []
  const counts = useMemo(() => {
    const result = Object.fromEntries(STATUS_ORDER.map((s) => [s, 0])) as Record<DeviceStatus, number>
    for (const device of list) {
      if (device.collectStatus in result) result[device.collectStatus] += 1
      else result.error += 1
    }
    return result
  }, [list])

  const total = list.length
  const healthy = counts.collecting
  const abnormal = counts.collectingError + counts.error
  const inactive = counts.stopped + counts.unconnected + counts.emptyVariable

  const recent = useMemo(
    () =>
      [...list]
        .sort((a, b) => new Date(b.modTime).getTime() - new Date(a.modTime).getTime())
        .slice(0, 5),
    [list],
  )

  const refreshAll = () => {
    void meta.refetch()
    void cpu.refetch()
    void mem.refetch()
    void disk.refetch()
    void devices.refetch()
    toast.success(t('dashboard.refreshSuccess'))
  }

  const cpuData = cpu.data?.cpus
  const memData = mem.data?.mem
  const diskData = disk.data?.disk ?? {}

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('dashboard.title')}
        description={t('dashboard.description')}
        actions={
          <Button variant="outline" onClick={refreshAll} icon={<RefreshCw className="size-4" />}>
            {t('common.action.refresh')}
          </Button>
        }
      />

      {/* 网关身份 */}
      <Card className="relative overflow-hidden">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,var(--primary-soft),transparent_60%)] opacity-70"
        />
        <div className="relative flex flex-col gap-6 p-5 md:flex-row md:items-start md:justify-between">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <Badge tone={meta.isError ? 'danger' : 'success'} dot>
                {meta.isError
                  ? t('dashboard.gateway.unavailable')
                  : meta.isSuccess
                    ? t('dashboard.gateway.running')
                    : t('dashboard.gateway.connecting')}
              </Badge>
              <Badge tone="neutral">{t('dashboard.gateway.edgeNode')}</Badge>
            </div>

            {meta.isLoading ? (
              <div className="mt-3 space-y-2">
                <Skeleton className="h-7 w-56" />
                <Skeleton className="h-4 w-72" />
              </div>
            ) : meta.isError ? (
              <div className="mt-3">
                <h2 className="text-lg font-semibold">{t('dashboard.gateway.loadErrorTitle')}</h2>
                <p className="mt-1 text-xs text-muted">
                  {t('dashboard.gateway.loadErrorDescription')}
                </p>
              </div>
            ) : (
              <>
                <h2 className="mt-3 text-2xl font-semibold tracking-tight">{meta.data?.name}</h2>
                <div className="mt-1.5 flex flex-wrap items-center gap-2 text-xs text-muted">
                  <span className="font-mono">ID {meta.data?.id}</span>
                  <CopyButton value={meta.data?.id ?? ''} />
                </div>
              </>
            )}
          </div>

          <div className="grid shrink-0 grid-cols-2 gap-x-8 gap-y-3 text-xs md:min-w-72">
            <div>
              <p className="text-faint">{t('dashboard.gateway.version')}</p>
              <p className="mt-1 font-mono text-fg">{meta.data?.eTag ?? '—'}</p>
            </div>
            <div>
              <p className="text-faint">{t('dashboard.gateway.updatedAt')}</p>
              <p className="mt-1 text-fg">{formatRelative(meta.data?.modTime)}</p>
            </div>
            <div className="col-span-2">
              <p className="text-faint">{t('dashboard.gateway.secret')}</p>
              <div className="mt-1 flex items-center gap-1.5">
                <span className="truncate font-mono text-fg">
                  {meta.data
                    ? secretVisible
                      ? meta.data.secret || t('common.state.unset')
                      : maskSecret(meta.data.secret)
                    : '—'}
                </span>
                {meta.data?.secret ? (
                  <>
                    <button
                      type="button"
                      onClick={() => setSecretVisible((v) => !v)}
                      className="rounded p-1 text-faint hover:bg-surface-2 hover:text-fg"
                      aria-label={
                        secretVisible
                          ? t('dashboard.gateway.hideSecret')
                          : t('dashboard.gateway.showSecret')
                      }
                    >
                      {secretVisible ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                    </button>
                    <CopyButton
                      value={meta.data.secret}
                      label={t('dashboard.gateway.secretCopied')}
                    />
                  </>
                ) : null}
              </div>
            </div>
          </div>
        </div>
      </Card>

      {/* 设备统计 */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label={t('dashboard.stats.total')}
          value={devices.isLoading ? '—' : total}
          hint={t('dashboard.stats.totalHint')}
          icon={<Router className="size-4" />}
        />
        <StatCard
          label={t('dashboard.stats.healthy')}
          value={devices.isLoading ? '—' : healthy}
          hint={
            total
              ? t('dashboard.stats.healthyHint', { percent: Math.round((healthy / total) * 100) })
              : t('dashboard.stats.noDevices')
          }
          icon={<Activity className="size-4" />}
          footer={<ProgressBar value={total ? (healthy / total) * 100 : 0} tone="success" />}
        />
        <StatCard
          label={t('dashboard.stats.abnormal')}
          value={devices.isLoading ? '—' : abnormal}
          hint={t('dashboard.stats.abnormalHint')}
          icon={<ShieldAlert className="size-4" />}
          footer={<ProgressBar value={total ? (abnormal / total) * 100 : 0} tone="warning" />}
        />
        <StatCard
          label={t('dashboard.stats.inactive')}
          value={devices.isLoading ? '—' : inactive}
          hint={t('dashboard.stats.inactiveHint')}
          icon={<Cpu className="size-4" />}
          footer={<ProgressBar value={total ? (inactive / total) * 100 : 0} tone="primary" />}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-5">
        {/* 资源 */}
        <Card className="lg:col-span-3">
          <CardHeader>
            <div>
              <CardTitle>{t('dashboard.resources.title')}</CardTitle>
              <CardDescription>{t('dashboard.resources.description')}</CardDescription>
            </div>
            <Server className="size-4 text-faint" />
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="grid grid-cols-2 gap-4 rounded border border-border bg-surface-2/50 p-4">
              <div>
                <p className="mb-3 flex items-center gap-1.5 text-xs font-medium text-muted">
                  <Cpu className="size-3.5" /> {t('dashboard.resources.cpu')}
                </p>
                {cpu.isError ? (
                  <p className="text-xs text-danger">{t('dashboard.resources.cpuUnavailable')}</p>
                ) : (
                  <div className="flex items-end justify-around">
                    {CPU_GAUGES.map((gauge) => (
                      <GaugeRing
                        key={gauge.key}
                        value={percentValue(cpuData?.[gauge.key])}
                        label={t(gauge.labelKey)}
                        size={78}
                        strokeWidth={7}
                        tone={toneForPercent(percentValue(cpuData?.[gauge.key]))}
                      />
                    ))}
                  </div>
                )}
              </div>
              <div className="border-l border-border pl-4">
                <p className="mb-3 flex items-center gap-1.5 text-xs font-medium text-muted">
                  <MemoryStick className="size-3.5" /> {t('dashboard.resources.memory')}
                </p>
                {mem.isError || !memData ? (
                  <p className="text-xs text-danger">{t('dashboard.resources.memoryUnavailable')}</p>
                ) : (
                  <div className="flex flex-col items-center gap-2">
                    <GaugeRing
                      value={percentValue(memData.UsedPercent)}
                      label={`${memData.Used} / ${memData.Total}`}
                      size={92}
                      tone={toneForPercent(percentValue(memData.UsedPercent))}
                    />
                  </div>
                )}
              </div>
            </div>

            <div>
              <p className="mb-3 flex items-center gap-1.5 text-xs font-medium text-muted">
                <HardDrive className="size-3.5" /> {t('dashboard.resources.disk')}
              </p>
              {disk.isError ? (
                <p className="text-xs text-danger">{t('dashboard.resources.diskUnavailable')}</p>
              ) : Object.keys(diskData).length === 0 ? (
                <p className="text-xs text-faint">{t('dashboard.resources.noMounts')}</p>
              ) : (
                <div className="space-y-3">
                  {Object.entries(diskData).map(([mount, info]) => {
                    const percent = percentValue(info.UsedPercent)
                    return (
                      <div key={mount}>
                        <div className="mb-1.5 flex items-baseline justify-between gap-3 text-xs">
                          <span className="font-mono text-fg">{mount}</span>
                          <span className="text-muted tabular">
                            {info.Used} / {info.Total}
                            <span className="ml-2 font-medium text-fg">{info.UsedPercent}</span>
                          </span>
                        </div>
                        <ProgressBar
                          value={percent}
                          tone={
                            percent >= 90 ? 'danger' : percent >= 75 ? 'warning' : 'success'
                          }
                        />
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* 状态分布 */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <div>
              <CardTitle>{t('dashboard.status.title')}</CardTitle>
              <CardDescription>{t('dashboard.status.description')}</CardDescription>
            </div>
            <Link
              to="/devices"
              className="inline-flex items-center gap-1 text-xs text-muted transition-colors hover:text-fg"
            >
              {t('dashboard.status.viewAll')} <ArrowRight className="size-3" />
            </Link>
          </CardHeader>
          <CardContent>
            {devices.isLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-3 w-full" />
                {Array.from({ length: 4 }).map((_, i) => (
                  <Skeleton key={i} className="h-5 w-full" />
                ))}
              </div>
            ) : devices.isError ? (
              <ErrorState error={devices.error} onRetry={devices.refetch} compact />
            ) : total === 0 ? (
              <EmptyState
                title={t('dashboard.status.emptyTitle')}
                description={t('dashboard.status.emptyDescription')}
                className="py-8"
              />
            ) : (
              <>
                <div className="flex h-2.5 w-full overflow-hidden rounded-full bg-surface-3">
                  {STATUS_ORDER.map((status) =>
                    counts[status] > 0 ? (
                      <div
                        key={status}
                        className={cn('h-full transition-all', STATUS_BAR[status])}
                        style={{ width: `${(counts[status] / total) * 100}%` }}
                        title={`${t(DEVICE_STATUS[status].labelKey)} ${counts[status]}`}
                      />
                    ) : null,
                  )}
                </div>
                <ul className="mt-4 space-y-2.5">
                  {STATUS_ORDER.map((status) => (
                    <li key={status} className="flex items-center gap-2 text-xs">
                      <span className={cn('size-2 rounded-full', STATUS_BAR[status])} />
                      <span className="text-muted">{t(DEVICE_STATUS[status].labelKey)}</span>
                      <span className="ml-auto font-medium tabular">{counts[status]}</span>
                      <span className="w-10 text-right text-faint tabular">
                        {total ? Math.round((counts[status] / total) * 100) : 0}%
                      </span>
                    </li>
                  ))}
                </ul>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {/* 最近设备 */}
      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('dashboard.recent.title')}</CardTitle>
            <CardDescription>{t('dashboard.recent.description')}</CardDescription>
          </div>
          <Link
            to="/devices"
            className="inline-flex items-center gap-1 text-xs text-muted transition-colors hover:text-fg"
          >
            {t('dashboard.recent.link')} <ArrowRight className="size-3" />
          </Link>
        </CardHeader>
        {devices.isLoading ? (
          <TableSkeleton rows={4} cols={4} />
        ) : devices.isError ? (
          <div className="p-5">
            <ErrorState error={devices.error} onRetry={devices.refetch} compact />
          </div>
        ) : recent.length === 0 ? (
          <EmptyState
            title={t('dashboard.recent.emptyTitle')}
            description={t('dashboard.recent.emptyDescription')}
            action={
              <Link
                to="/devices/new"
                className="inline-flex h-9.5 items-center rounded-lg bg-primary px-4 text-sm font-medium text-primary-fg hover:bg-primary-hover"
              >
                {t('dashboard.recent.create')}
              </Link>
            }
          />
        ) : (
          <Table>
            <THead>
              <tr>
                <TH>{t('dashboard.table.name')}</TH>
                <TH>{t('dashboard.table.protocol')}</TH>
                <TH>{t('dashboard.table.status')}</TH>
                <TH>{t('dashboard.table.code')}</TH>
                <TH className="text-right">{t('dashboard.table.updatedAt')}</TH>
              </tr>
            </THead>
            <TBody>
              {recent.map((device) => (
                <TR key={device.id}>
                  <TD>
                    <Link
                      to={`/devices/${device.id}`}
                      className="font-medium transition-colors hover:text-primary"
                    >
                      {device.name}
                    </Link>
                  </TD>
                  <TD>
                    <DeviceTypeBadge type={device.deviceType} />
                  </TD>
                  <TD>
                    <StatusBadge status={device.collectStatus} />
                  </TD>
                  <TD className="font-mono text-xs text-muted">{device.deviceCode}</TD>
                  <TD className="text-right text-xs text-muted">{formatRelative(device.modTime)}</TD>
                </TR>
              ))}
            </TBody>
          </Table>
        )}
      </Card>
    </div>
  )
}

/** 独立 hook 便于后续替换为带过滤条件的列表查询。 */
function useDeviceList() {
  return useDevices()
}
