import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, Cpu, Factory, Maximize2, MemoryStick, Minimize2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeviceDetails, useGatewayCpu, useGatewayMem, useGatewayMeta } from '@/api/hooks'
import type { DeviceStatus } from '@/api/types'
import { useI18n } from '@/i18n'
import { DEVICE_STATUS, type StatusMeta } from '@/lib/constants'
import { percentValue } from '@/lib/format'
import { cn } from '@/lib/cn'
import { EmptyState } from '@/components/ui/EmptyState'
import { Spinner } from '@/components/ui/Spinner'
import { Button } from '@/components/ui/Button'
import { ErrorState } from '@/components/shared/ErrorState'
import { DeviceSituationPanel } from './DeviceSituationPanel'
import { FactoryScene } from './FactoryScene'
import { ScreenClock } from './ScreenClock'
import { SelectedDevicePanel } from './SelectedDevicePanel'
import { buildFactoryLayout } from './layout'

const LEGEND = Object.entries(DEVICE_STATUS) as Array<[DeviceStatus, StatusMeta]>

export function FactoryScreenPage() {
  const { t } = useI18n()
  const details = useDeviceDetails()
  const cpu = useGatewayCpu()
  const mem = useGatewayMem()
  const meta = useGatewayMeta()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [isFullscreen, setIsFullscreen] = useState(false)

  const devices = useMemo(() => details.data ?? [], [details.data])
  const layout = useMemo(() => buildFactoryLayout(devices), [devices])
  const selected = useMemo(
    () => devices.find((device) => device.id === selectedId) ?? null,
    [devices, selectedId],
  )

  useEffect(() => {
    document.title = t('screen.documentTitle')
    return () => {
      document.title = 'AI-harns-platform'
    }
  }, [t])

  useEffect(() => {
    const onChange = () => setIsFullscreen(Boolean(document.fullscreenElement))
    document.addEventListener('fullscreenchange', onChange)
    return () => document.removeEventListener('fullscreenchange', onChange)
  }, [])

  useEffect(() => {
    if (selectedId && !devices.some((device) => device.id === selectedId)) {
      setSelectedId(null)
    }
  }, [devices, selectedId])

  const toggleFullscreen = async () => {
    try {
      if (document.fullscreenElement) await document.exitFullscreen()
      else await document.documentElement.requestFullscreen()
    } catch {
      toast.error(t('screen.fullscreenUnsupported'))
    }
  }

  const cpuPercent = cpu.data ? Math.round(percentValue(cpu.data.cpus?.['1min'])) : null
  const memPercent = mem.data ? Math.round(percentValue(mem.data.mem?.UsedPercent)) : null
  const online = meta.isSuccess && !meta.isError

  return (
    <div className="dark flex min-h-dvh flex-col bg-bg text-fg lg:h-dvh lg:overflow-hidden">
      <header className="flex h-16 shrink-0 items-center gap-2 border-b border-border bg-surface/70 px-3 backdrop-blur sm:gap-3 sm:px-4">
        <div className="flex min-w-0 items-center gap-2.5">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-xl border border-primary/30 bg-primary-soft text-primary">
            <Factory className="size-5" />
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold tracking-tight">{t('screen.title')}</p>
            <p className="hidden truncate text-[11px] text-muted sm:block">{t('screen.subtitle')}</p>
          </div>
        </div>

        <div className="ml-auto flex items-center gap-2">
          <div className="hidden items-center gap-3 rounded-full border border-border bg-surface px-3 py-1.5 text-[11px] text-muted lg:flex">
            <span className="flex items-center gap-1.5">
              <Cpu className="size-3.5 text-primary" />
              CPU{' '}
              <span className="font-medium text-fg tabular">
                {cpuPercent === null ? '—' : `${cpuPercent}%`}
              </span>
            </span>
            <span className="h-3 w-px bg-border" />
            <span className="flex items-center gap-1.5">
              <MemoryStick className="size-3.5 text-primary" />
              {t('screen.memory')}{' '}
              <span className="font-medium text-fg tabular">
                {memPercent === null ? '—' : `${memPercent}%`}
              </span>
            </span>
          </div>

          <span className="hidden items-center gap-1.5 rounded-full border border-border bg-surface px-2.5 py-1 text-[11px] text-muted md:inline-flex">
            <span
              className={cn(
                'size-1.5 rounded-full',
                meta.isError ? 'bg-rose-500' : online ? 'bg-emerald-500' : 'bg-amber-500',
              )}
            />
            <span className="max-w-36 truncate">
              {meta.isError ? t('screen.gatewayOffline') : (meta.data?.name ?? t('screen.connecting'))}
            </span>
          </span>

          <ScreenClock />

          <Button
            variant="ghost"
            size="icon"
            onClick={() => void toggleFullscreen()}
            aria-label={isFullscreen ? t('screen.exitFullscreen') : t('screen.enterFullscreen')}
            title={isFullscreen ? t('screen.exitFullscreen') : t('screen.enterFullscreen')}
          >
            {isFullscreen ? <Minimize2 className="size-4" /> : <Maximize2 className="size-4" />}
          </Button>

          <Link
            to="/"
            className="inline-flex h-8 items-center gap-1.5 rounded-lg border border-border bg-surface px-2.5 text-xs font-medium transition-colors hover:bg-surface-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
          >
            <ArrowLeft className="size-3.5" />
            <span className="hidden sm:inline">{t('screen.back')}</span>
          </Link>
        </div>
      </header>

      <main className="grid min-h-0 flex-1 gap-3 p-3 lg:grid-cols-[264px_minmax(0,1fr)_320px] lg:overflow-hidden">
        <div className="order-2 min-h-0 lg:order-1">
          <DeviceSituationPanel layout={layout} selectedId={selectedId} onSelect={setSelectedId} />
        </div>

        <section className="relative order-1 min-h-[420px] overflow-hidden rounded-2xl border border-border bg-bg lg:order-2 lg:min-h-0">
          <FactoryScene devices={layout} selectedId={selectedId} onSelect={setSelectedId} />

          <div className="pointer-events-none absolute top-3 right-3 z-20 flex items-center gap-1.5 rounded-full border border-border bg-bg/70 px-2.5 py-1 text-[10px] text-muted backdrop-blur">
            <span
              className={cn(
                'size-1.5 rounded-full',
                details.isFetching ? 'animate-pulse bg-primary' : 'bg-emerald-500',
              )}
            />
            {details.isFetching ? t('screen.syncing') : t('screen.live')}
          </div>

          <div className="pointer-events-none absolute bottom-3 left-3 z-20 max-w-[calc(100%-1.5rem)] rounded-xl border border-border bg-bg/70 px-3 py-2 backdrop-blur">
            <div className="flex flex-wrap gap-x-3 gap-y-1">
              {LEGEND.map(([status, info]) => (
                <span key={status} className="flex items-center gap-1.5 text-[10px] text-muted">
                  <span className={cn('size-1.5 rounded-full', info.dotClassName)} />
                  {t(info.labelKey)}
                </span>
              ))}
            </div>
            <p className="mt-1.5 text-[10px] text-faint">{t('screen.legendHint')}</p>
          </div>

          {details.isLoading ? (
            <div className="absolute inset-0 z-30 flex flex-col items-center justify-center gap-3 bg-bg/75 backdrop-blur-sm">
              <Spinner className="size-6 text-primary" />
              <p className="text-xs text-muted">{t('screen.loadingDevices')}</p>
            </div>
          ) : details.isError ? (
            <div className="absolute inset-0 z-30 flex items-center justify-center bg-bg/75 p-6 backdrop-blur-sm">
              <div className="w-full max-w-sm">
                <ErrorState error={details.error} onRetry={details.refetch} />
              </div>
            </div>
          ) : devices.length === 0 ? (
            <div className="absolute inset-0 z-30 flex items-center justify-center bg-bg/75 p-6 backdrop-blur-sm">
              <EmptyState
                title={t('screen.empty.title')}
                description={t('screen.empty.description')}
                action={
                  <Link
                    to="/devices/new"
                    className="inline-flex h-9 items-center rounded-lg bg-primary px-4 text-xs font-medium text-primary-fg transition-colors hover:bg-primary-hover"
                  >
                    {t('screen.empty.createDevice')}
                  </Link>
                }
              />
            </div>
          ) : null}
        </section>

        <div className="order-3 min-h-0">
          <SelectedDevicePanel device={selected} />
        </div>
      </main>
    </div>
  )
}
