import { useState } from 'react'
import { Cpu, Eye, EyeOff, HardDrive, MemoryStick, RefreshCw, Server } from 'lucide-react'
import { toast } from 'sonner'
import { useGatewayCpu, useGatewayDisk, useGatewayMem, useGatewayMeta } from '@/api/hooks'
import { useI18n, type MessageKey } from '@harns/core'
import { formatRelative, maskSecret, percentValue } from '@harns/core'
import { Badge } from '@harns/core'
import { Button } from '@harns/core'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@harns/core'
import { CopyButton } from '@harns/core'
import { ProgressBar } from '@harns/core'
import { Skeleton } from '@harns/core'
import { GaugeRing, toneForPercent } from '@harns/core'
import { PageHeader } from '@harns/core'

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
  const [secretVisible, setSecretVisible] = useState(false)

  const refreshAll = () => {
    void meta.refetch()
    void cpu.refetch()
    void mem.refetch()
    void disk.refetch()
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

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('dashboard.resources.title')}</CardTitle>
            <CardDescription>{t('dashboard.resources.description')}</CardDescription>
          </div>
          <Server className="size-4 text-faint" />
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="grid grid-cols-2 gap-4 rounded-xl border border-border bg-surface-2/50 p-4">
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
    </div>
  )
}

