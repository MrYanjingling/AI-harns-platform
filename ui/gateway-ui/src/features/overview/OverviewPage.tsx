import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Activity,
  ArrowRight,
  Boxes,
  CalendarClock,
  Check,
  Copy as CopyIcon,
  Cpu,
  HardDrive,
  KeyRound,
  MemoryStick,
  Server,
  Waypoints,
} from 'lucide-react'
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  CopyButton,
  Dialog,
  ErrorState,
  GaugeRing,
  ProgressBar,
  Skeleton,
  StatCard,
  cn,
  formatDateTime,
  formatRelative,
  parseDate,
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
  useLicense,
  usePushConnectors,
  usePushGroups,
} from '@/api/hooks'
import { CollectStatusBadge, ConnectorStatusBadge } from '@/components/StatusBadge'
import { driverModelLabel } from '@/lib/constants'
import { formatUptime } from '@/lib/uptime'

const overviewCardClass = 'border-[1.5px] border-border-strong/75 shadow-[0_1px_3px_rgba(28,47,76,0.06)]'

export function OverviewPage() {
  const { t, locale } = useI18n()
  const meta = useGatewayMeta()
  const cpu = useGatewayCpu()
  const mem = useGatewayMem()
  const disk = useGatewayDisk()
  const collections = useCollections({ page: 1, pageSize: 100 })
  const connectors = usePushConnectors({ page: 1, pageSize: 100 })
  const groups = usePushGroups({ page: 1, pageSize: 100 })
  const license = useLicense()
  const [secretOpen, setSecretOpen] = useState(false)

  const list = collections.data?.list ?? []
  const connectorList = connectors.data?.list ?? []
  const groupList = groups.data?.list ?? []
  const connectorTypeMap = useMemo(
    () => new Map(connectorList.map((item) => [item.id, item.type])),
    [connectorList],
  )

  const stats = useMemo(() => {
    return {
      total: collections.data?.total ?? list.length,
      normal: list.filter((item) => item.collectStatus === 'collecting').length,
      groups: groups.data?.total ?? groupList.length,
      normalGroups: groupList.filter((item) => item.status === 'running').length,
      currentVariables: list.reduce((sum, item) => sum + (item.variableCount ?? 0), 0),
    }
  }, [list, collections.data, groupList, groups.data])

  const recentGroups = useMemo(
    () =>
      [...groupList]
        .sort((a, b) => new Date(b.updatedTime).getTime() - new Date(a.updatedTime).getTime())
        .slice(0, 5),
    [groupList],
  )

  const recentCollections = useMemo(
    () =>
      [...list]
        .sort((a, b) => new Date(b.modTime).getTime() - new Date(a.modTime).getTime())
        .slice(0, 5),
    [list],
  )

  const memData = mem.data?.mem
  const diskEntries = Object.entries(disk.data?.disk ?? {})
  const licenseInfo = license.data?.license
  const deadline = parseDate(licenseInfo?.softwareRunDeadline)
  const expired = deadline ? deadline.getTime() < Date.now() : false
  const maxVariable = licenseInfo?.maxVariable
  const error = collections.error ?? connectors.error ?? groups.error

  return (
    <div className="min-h-full space-y-4 bg-surface-3 p-4 sm:p-6">
      <Card className={cn('relative overflow-hidden', overviewCardClass)}>
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,var(--primary-soft),transparent_62%)] opacity-70"
        />
        <div className="relative flex flex-col gap-4 p-4 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <div className="flex size-12 shrink-0 items-center justify-center rounded border border-border bg-surface-2 text-primary">
              <Server className="size-5" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                {meta.isLoading ? (
                  <Skeleton className="h-6 w-40" />
                ) : (
                  <h2 className="truncate text-lg font-semibold tracking-tight">
                    {meta.data?.name ?? t('gateway.overview.unknownGateway')}
                  </h2>
                )}
                <Button
                  variant="outline"
                  size="sm"
                  icon={<KeyRound className="size-3.5" />}
                  onClick={() => setSecretOpen(true)}
                  disabled={!meta.data?.secret}
                >
                  {t('gateway.overview.secret.show')}
                </Button>
              </div>
              <div className="flex h-5 min-w-0 items-center gap-1.5">
                <p className="min-w-0 truncate whitespace-nowrap font-mono text-xs text-fg">
                  {meta.data?.id ?? '—'}
                </p>
                {meta.data?.id ? <CopyButton value={meta.data.id} className="size-5" /> : null}
              </div>
            </div>
          </div>
          <div className="grid shrink-0 grid-cols-2 gap-x-6 gap-y-2 text-[11px]">
            <div>
              <CardDescription>{t('gateway.overview.field.currentUptime')}</CardDescription>
              <p className="font-mono text-xs text-fg">
                {formatUptime(meta.data?.currentUptime, locale)}
              </p>
            </div>
            <div>
              <CardDescription>{t('gateway.overview.field.totalUptime')}</CardDescription>
              <p className="font-mono text-xs text-fg">
                {formatUptime(meta.data?.totalUptime, locale)}
              </p>
            </div>
          </div>
        </div>
      </Card>

      {error ? (
        <ErrorState error={error} onRetry={() => void collections.refetch()} compact />
      ) : null}

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Link to="/collections" className="block">
          <StatCard
            label={t('gateway.overview.stat.collections')}
            value={<span className="text-lg">{`${stats.total}/${stats.normal}`}</span>}
            icon={<Activity />}
            variant="summary"
            tone="primary"
            className={overviewCardClass}
            footer={<CardDescription>{t('gateway.overview.stat.collectionsHint')}</CardDescription>}
          />
        </Link>
        <StatCard
          label={t('gateway.overview.stat.pushGroups')}
          value={<span className="text-lg">{`${stats.groups}/${stats.normalGroups}`}</span>}
          icon={<Waypoints />}
          variant="summary"
          tone="success"
          className={overviewCardClass}
          footer={<CardDescription>{t('gateway.overview.stat.pushGroupsHint')}</CardDescription>}
        />
        <StatCard
          label={t('gateway.overview.stat.connectors')}
          value={<span className="text-lg">{`${maxVariable ?? '—'}/${stats.currentVariables}`}</span>}
          icon={<Boxes />}
          variant="summary"
          tone="success"
          className={overviewCardClass}
          footer={<CardDescription>{t('gateway.overview.stat.variablesHint')}</CardDescription>}
        />
        <StatCard
          label={t('gateway.overview.stat.groups')}
          value={
            <span className={cn('text-lg', expired && 'text-danger')}>
              {deadline ? formatDateTime(deadline) : '—'}
            </span>
          }
          icon={<CalendarClock />}
          variant="summary"
          tone={expired ? 'danger' : 'neutral'}
          className={overviewCardClass}
          footer={<CardDescription>{t('gateway.overview.stat.groupsHint')}</CardDescription>}
        />
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        <Card className={overviewCardClass}>
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

        <Card className={cn(overviewCardClass, 'flex flex-col')}>
          <CardHeader>
            <div>
              <CardTitle className="flex items-center gap-1.5">
                <MemoryStick className="size-4 text-faint" />
                {t('gateway.overview.resource.memory')}
              </CardTitle>
              <CardDescription>{t('gateway.overview.resource.memoryHint')}</CardDescription>
            </div>
          </CardHeader>
          <CardContent className="flex flex-1 items-center gap-4">
            {mem.isError || !memData ? (
              <p className="text-xs text-danger">{t('gateway.performance.memory.unavailable')}</p>
            ) : (
              <>
                <GaugeRing
                  value={percentValue(memData.UsedPercent)}
                  size={84}
                  tone={toneForPercent(percentValue(memData.UsedPercent))}
                />
                <div className="min-w-0 space-y-1 text-xs">
                  <CardDescription>{t('gateway.overview.resource.used')}</CardDescription>
                  <p className="font-medium text-fg tabular">{memData.Used}</p>
                  <CardDescription>{t('gateway.overview.resource.total')}</CardDescription>
                  <p className="font-medium text-fg tabular">{memData.Total}</p>
                </div>
              </>
            )}
          </CardContent>
        </Card>

        <Card className={overviewCardClass}>
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
                    <div key={mount} title={mount}>
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

      <div className="grid items-start gap-4 lg:grid-cols-2">
        <Card className={overviewCardClass}>
          <CardHeader>
            <div>
              <CardTitle>{t('gateway.overview.recentGroups.title')}</CardTitle>
            </div>
            <Link
              to="/push/groups"
              className="inline-flex items-center gap-1 text-[13px] leading-relaxed text-muted transition-colors hover:text-fg"
            >
              {t('gateway.overview.viewAll')}
              <ArrowRight className="size-3" />
            </Link>
          </CardHeader>
          <CardContent className="min-h-[324px]">
            {groups.isLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 3 }).map((_, index) => (
                  <Skeleton key={index} className="h-10 w-full" />
                ))}
              </div>
            ) : recentGroups.length === 0 ? null : (
              <div className="space-y-1.5">
                {recentGroups.map((group) => (
                  <Link
                    key={group.id}
                    to={`/push/groups?connectorId=${encodeURIComponent(group.connectorId)}&groupId=${encodeURIComponent(group.id)}`}
                    className="flex items-center justify-between gap-2 rounded border border-border px-2.5 py-2 transition-colors hover:border-border-strong hover:bg-surface-2"
                  >
                    <div className="flex min-w-0 flex-1 items-center gap-2">
                      <span className="min-w-0 truncate text-sm leading-5 font-normal font-table text-table-fg">
                        {group.name}
                      </span>
                      <span className="min-w-0 truncate text-sm leading-5 font-normal font-table text-table-fg">
                        {connectorTypeMap.get(group.connectorId)?.toUpperCase() ?? '—'} ·{' '}
                        {formatRelative(group.updatedTime)}
                      </span>
                    </div>
                    <ConnectorStatusBadge status={group.status} error={group.lastError} />
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card className={overviewCardClass}>
          <CardHeader>
            <div>
              <CardTitle>{t('gateway.overview.recent.title')}</CardTitle>
            </div>
            <Link
              to="/collections"
              className="inline-flex items-center gap-1 text-[13px] leading-relaxed text-muted transition-colors hover:text-fg"
            >
              {t('gateway.overview.viewAll')}
              <ArrowRight className="size-3" />
            </Link>
          </CardHeader>
          <CardContent className="min-h-[324px]">
            {collections.isLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 3 }).map((_, index) => (
                  <Skeleton key={index} className="h-10 w-full" />
                ))}
              </div>
            ) : recentCollections.length === 0 ? null : (
              <div className="space-y-1.5">
                {recentCollections.map((collection) => (
                  <Link
                    key={collection.id}
                    to={`/collections/${collection.id}`}
                    className="flex items-center justify-between gap-2 rounded border border-border px-2.5 py-2 transition-colors hover:border-border-strong hover:bg-surface-2"
                  >
                    <div className="flex min-w-0 flex-1 items-center gap-2">
                      <span className="min-w-0 truncate text-sm leading-5 font-normal font-table text-table-fg">
                        {collection.name}
                      </span>
                      <span className="min-w-0 truncate text-sm leading-5 font-normal font-table text-table-fg">
                        {driverModelLabel(collection.driverModel) || collection.driverModel} ·{' '}
                        {formatRelative(collection.modTime)}
                      </span>
                    </div>
                    <CollectStatusBadge
                      status={collection.collectStatus}
                      error={collection.lastError}
                    />
                  </Link>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

      </div>

      <Dialog
        open={secretOpen}
        onClose={() => setSecretOpen(false)}
        title={t('gateway.overview.secret.title')}
        description={t('gateway.overview.secret.description')}
        size="md"
      >
        <div className="flex flex-nowrap items-center gap-2 rounded border border-border bg-surface-2 px-3 py-2">
          <code
            title={meta.data?.secret}
            className="block min-w-0 flex-1 truncate whitespace-nowrap font-mono text-xs text-fg"
          >
            {meta.data?.secret || t('common.state.unset')}
          </code>
          {meta.data?.secret ? <SecretCopyButton value={meta.data.secret} /> : null}
        </div>
      </Dialog>
    </div>
  )
}

function SecretCopyButton({ value }: { value: string }) {
  const { t } = useI18n()
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      /* ignore clipboard failures */
    }
  }

  return (
    <button
      type="button"
      onClick={copy}
      className="inline-flex shrink-0 items-center gap-1 whitespace-nowrap rounded p-1.5 text-xs text-muted transition-colors hover:bg-surface-3 hover:text-fg"
      aria-label={copied ? t('common.action.copied') : t('common.action.copy')}
    >
      {copied ? <Check className="size-3.5 text-primary" /> : <CopyIcon className="size-3.5" />}
      {copied ? <span>{t('common.action.copied')}</span> : null}
    </button>
  )
}
