import { useEffect, useMemo, useState } from 'react'
import { Cpu, Eye, EyeOff, HardDrive, MemoryStick, RefreshCw, Server } from 'lucide-react'
import { toast } from 'sonner'
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  CopyButton,
  GaugeRing,
  PageHeader,
  ProgressBar,
  Skeleton,
  formatDateTime,
  maskSecret,
  percentValue,
  toneForPercent,
  useI18n,
} from '@harns/core'
import { useGatewayCpu, useGatewayDisk, useGatewayMem, useGatewayMeta } from '@/api/hooks'

const CPU_GAUGES = [
  { key: '1min', labelKey: 'gateway.performance.cpu.1min' },
  { key: '5min', labelKey: 'gateway.performance.cpu.5min' },
  { key: '15min', labelKey: 'gateway.performance.cpu.15min' },
] as const

const HISTORY_LIMIT = 30

interface Sample {
  time: number
  value: number
}

function Sparkline({ samples }: { samples: Sample[] }) {
  const points = useMemo(() => {
    if (samples.length === 0) return ''
    const width = 300
    const height = 80
    const step = samples.length > 1 ? width / (samples.length - 1) : width
    return samples
      .map((sample, index) => {
        const x = index * step
        const y = height - (Math.min(100, Math.max(0, sample.value)) / 100) * height
        return `${x.toFixed(1)},${y.toFixed(1)}`
      })
      .join(' ')
  }, [samples])

  if (samples.length < 2) return null

  return (
    <svg viewBox="0 0 300 80" preserveAspectRatio="none" className="h-24 w-full" aria-hidden="true">
      <defs>
        <linearGradient id="gateway-cpu-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="currentColor" stopOpacity="0.28" />
          <stop offset="100%" stopColor="currentColor" stopOpacity="0" />
        </linearGradient>
      </defs>
      <polyline
        points={`0,80 ${points} 300,80`}
        fill="url(#gateway-cpu-fill)"
        stroke="none"
        className="text-primary"
      />
      <polyline
        points={points}
        fill="none"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
        strokeLinecap="round"
        className="text-primary"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  )
}

export function PerformancePage() {
  const { t } = useI18n()
  const meta = useGatewayMeta()
  const cpu = useGatewayCpu()
  const mem = useGatewayMem()
  const disk = useGatewayDisk()
  const [secretVisible, setSecretVisible] = useState(false)
  const [history, setHistory] = useState<Sample[]>([])

  const cpuValue = percentValue(cpu.data?.cpus?.['1min'])

  useEffect(() => {
    if (!cpu.data) return
    setHistory((current) => [
      ...current.slice(-(HISTORY_LIMIT - 1)),
      { time: Date.now(), value: percentValue(cpu.data?.cpus?.['1min']) },
    ])
  }, [cpu.data])

  const refreshing = meta.isFetching || cpu.isFetching || mem.isFetching || disk.isFetching

  const refresh = () => {
    void meta.refetch()
    void cpu.refetch()
    void mem.refetch()
    void disk.refetch()
    toast.success(t('gateway.performance.refreshSuccess'))
  }

  const memData = mem.data?.mem
  const diskData = disk.data?.disk ?? {}
  const metaData = meta.data

  return (
    <div className="space-y-4">
      <PageHeader
        title={t('gateway.performance.title')}
        description={t('gateway.performance.description')}
        actions={
          <Button
            variant="outline"
            loading={refreshing}
            onClick={refresh}
            icon={<RefreshCw className="size-4" />}
          >
            {t('gateway.performance.refresh')}
          </Button>
        }
      />

      <Card className="relative overflow-hidden">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,var(--primary-soft),transparent_60%)] opacity-70"
        />
        <div className="relative flex flex-col gap-5 p-5 md:flex-row md:items-start md:justify-between">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <Badge tone={meta.isError ? 'danger' : 'success'} dot>
                {meta.isError
                  ? t('gateway.performance.status.unavailable')
                  : meta.isSuccess
                    ? t('gateway.performance.status.running')
                    : t('gateway.performance.status.connecting')}
              </Badge>
              <Badge tone="neutral">{t('gateway.nav.section')}</Badge>
            </div>
            {meta.isLoading ? (
              <div className="mt-3 space-y-2">
                <Skeleton className="h-7 w-56" />
                <Skeleton className="h-4 w-72" />
              </div>
            ) : meta.isError || !metaData ? (
              <p className="mt-3 text-sm text-danger">{t('gateway.performance.loadError')}</p>
            ) : (
              <>
                <h2 className="mt-3 text-2xl font-semibold tracking-tight">{metaData.name}</h2>
                <div className="mt-1.5 flex flex-wrap items-center gap-2 text-xs text-muted">
                  <span className="font-mono">ID {metaData.id}</span>
                  <CopyButton value={metaData.id} />
                </div>
              </>
            )}
          </div>

          <div className="grid shrink-0 grid-cols-2 gap-x-8 gap-y-3 text-xs md:min-w-80">
            <div>
              <p className="text-faint">{t('gateway.performance.field.version')}</p>
              <p className="mt-1 font-mono text-fg">{metaData?.eTag ?? '—'}</p>
            </div>
            <div>
              <p className="text-faint">{t('gateway.performance.field.modTime')}</p>
              <p className="mt-1 text-fg">{formatDateTime(metaData?.modTime)}</p>
            </div>
            <div className="col-span-2">
              <p className="text-faint">{t('gateway.performance.field.secret')}</p>
              <div className="mt-1 flex items-center gap-1.5">
                <span className="truncate font-mono text-fg">
                  {metaData
                    ? secretVisible
                      ? metaData.secret || t('common.state.unset')
                      : maskSecret(metaData.secret)
                    : '—'}
                </span>
                {metaData?.secret ? (
                  <>
                    <button
                      type="button"
                      onClick={() => setSecretVisible((value) => !value)}
                      className="rounded p-1 text-faint hover:bg-surface-2 hover:text-fg"
                      aria-label={
                        secretVisible
                          ? t('gateway.performance.secret.hide')
                          : t('gateway.performance.secret.show')
                      }
                    >
                      {secretVisible ? (
                        <EyeOff className="size-3.5" />
                      ) : (
                        <Eye className="size-3.5" />
                      )}
                    </button>
                    <CopyButton
                      value={metaData.secret}
                      label={t('gateway.performance.secret.copied')}
                    />
                  </>
                ) : null}
              </div>
            </div>
          </div>
        </div>
      </Card>

      <div className="grid gap-3 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <div>
              <CardTitle className="flex items-center gap-1.5">
                <Cpu className="size-4 text-faint" />
                {t('gateway.performance.cpu.title')}
              </CardTitle>
              <CardDescription>{t('gateway.performance.cpu.description')}</CardDescription>
            </div>
            <Badge tone="neutral">
              {t('gateway.performance.sampleCount', { count: history.length })}
            </Badge>
          </CardHeader>
          <CardContent className="space-y-4">
            {cpu.isError ? (
              <p className="text-xs text-danger">{t('gateway.performance.cpu.unavailable')}</p>
            ) : (
              <>
                <div className="flex items-end justify-around gap-2">
                  {CPU_GAUGES.map((gauge) => {
                    const value = percentValue(cpu.data?.cpus?.[gauge.key])
                    return (
                      <GaugeRing
                        key={gauge.key}
                        value={value}
                        label={t(gauge.labelKey)}
                        size={84}
                        strokeWidth={7}
                        tone={toneForPercent(value)}
                      />
                    )
                  })}
                </div>
                <div>
                  <div className="mb-1 flex items-center justify-between text-xs">
                    <span className="font-medium text-muted">
                      {t('gateway.performance.cpu.trend')}
                    </span>
                    <span className="text-faint tabular">{Math.round(cpuValue)}%</span>
                  </div>
                  <Sparkline samples={history} />
                  <p className="mt-1 text-[11px] text-faint">
                    {t('gateway.performance.cpu.trendHint')}
                  </p>
                </div>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle className="flex items-center gap-1.5">
                <MemoryStick className="size-4 text-faint" />
                {t('gateway.performance.memory.title')}
              </CardTitle>
              <CardDescription>{t('gateway.performance.memory.description')}</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            {mem.isError || !memData ? (
              <p className="text-xs text-danger">{t('gateway.performance.memory.unavailable')}</p>
            ) : (
              <div className="flex flex-col items-center gap-3">
                <GaugeRing
                  value={percentValue(memData.UsedPercent)}
                  label={`${memData.Used} / ${memData.Total}`}
                  size={112}
                  tone={toneForPercent(percentValue(memData.UsedPercent))}
                />
                <div className="grid w-full grid-cols-3 gap-2 text-center text-xs">
                  <div className="rounded border border-border bg-surface-2 px-2 py-1.5">
                    <p className="text-faint">{t('gateway.performance.memory.title')}</p>
                    <p className="mt-0.5 text-fg tabular">{memData.UsedPercent}</p>
                  </div>
                  <div className="rounded border border-border bg-surface-2 px-2 py-1.5">
                    <p className="text-faint">{t('gateway.performance.memory.description')}</p>
                    <p className="mt-0.5 text-fg tabular">{memData.Used}</p>
                  </div>
                  <div className="rounded border border-border bg-surface-2 px-2 py-1.5">
                    <p className="text-faint">{t('gateway.performance.memory.title')}</p>
                    <p className="mt-0.5 text-fg tabular">{memData.Total}</p>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <div>
            <CardTitle className="flex items-center gap-1.5">
              <HardDrive className="size-4 text-faint" />
              {t('gateway.performance.disk.title')}
            </CardTitle>
            <CardDescription>{t('gateway.performance.disk.description')}</CardDescription>
          </div>
          <Server className="size-4 text-faint" />
        </CardHeader>
        <CardContent>
          {disk.isError ? (
            <p className="text-xs text-danger">{t('gateway.performance.disk.unavailable')}</p>
          ) : Object.keys(diskData).length === 0 ? (
            <p className="text-xs text-faint">{t('gateway.performance.disk.empty')}</p>
          ) : (
            <div className="space-y-3">
              {Object.entries(diskData).map(([mount, info]) => {
                const value = percentValue(info.UsedPercent)
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
  )
}
