import { useMemo } from 'react'
import { useI18n } from '@/i18n'
import { DEVICE_STATUS, PROTOCOLS, UNKNOWN_STATUS } from '@/lib/constants'
import { cn } from '@/lib/cn'
import {
  ZONE_BAR,
  ZONE_ORDER,
  ZONE_TEXT,
  type FactoryDevice,
} from './layout'

interface DeviceSituationPanelProps {
  layout: FactoryDevice[]
  selectedId: string | null
  onSelect: (id: string) => void
}

export function DeviceSituationPanel({ layout, selectedId, onSelect }: DeviceSituationPanelProps) {
  const { t } = useI18n()
  const { devices, counts, protocolCounts } = useMemo(() => {
    const devices = layout.map((item) => item.device)
    const counts = { total: devices.length, running: 0, abnormal: 0, inactive: 0 }

    for (const device of devices) {
      const status = DEVICE_STATUS[device.collectStatus] ? device.collectStatus : 'error'
      if (status === 'collecting') counts.running += 1
      else if (status === 'collectingError' || status === 'error') counts.abnormal += 1
      else counts.inactive += 1
    }

    const protocolCounts = ZONE_ORDER.map((type) => ({
      type,
      count: devices.filter((device) => device.deviceType === type).length,
    }))

    return { devices, counts, protocolCounts }
  }, [layout])

  const stats = [
    { label: t('screen.situation.devices'), value: counts.total, tone: 'text-fg' },
    { label: t('screen.situation.running'), value: counts.running, tone: 'text-success' },
    { label: t('screen.situation.abnormal'), value: counts.abnormal, tone: 'text-warning' },
    { label: t('screen.situation.inactive'), value: counts.inactive, tone: 'text-muted' },
  ]

  return (
    <aside className="flex max-h-[560px] min-h-0 flex-col gap-4 overflow-hidden rounded-2xl border border-border bg-surface/70 p-3 backdrop-blur lg:h-full lg:max-h-none">
      <div className="flex items-center justify-between">
        <h2 className="text-xs font-semibold tracking-wide">{t('screen.situation.title')}</h2>
        <span className="rounded-full border border-border px-2 py-0.5 text-[10px] text-muted tabular">
          {t('screen.situation.total', { count: counts.total })}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-2">
        {stats.map((stat) => (
          <div key={stat.label} className="rounded-xl border border-border bg-surface-2/60 px-3 py-2">
            <p className="text-[10px] text-muted">{stat.label}</p>
            <p className={cn('mt-0.5 text-xl font-semibold tabular', stat.tone)}>{stat.value}</p>
          </div>
        ))}
      </div>

      <div>
        <h3 className="mb-2 text-[10px] font-medium tracking-wider text-faint uppercase">
          {t('screen.situation.zones')}
        </h3>
        <div className="space-y-2">
          {protocolCounts.map(({ type, count }) => (
            <div key={type} className="flex items-center gap-2 text-[11px]">
              <span className={cn('w-12 shrink-0 font-medium', ZONE_TEXT[type])}>
                {PROTOCOLS[type].label}
              </span>
              <span className="h-1.5 flex-1 overflow-hidden rounded-full bg-surface-3">
                <span
                  className={cn('block h-full rounded-full transition-all', ZONE_BAR[type])}
                  style={{ width: `${counts.total ? (count / counts.total) * 100 : 0}%` }}
                />
              </span>
              <span className="w-5 text-right text-muted tabular">{count}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="flex min-h-0 flex-1 flex-col">
        <h3 className="mb-2 text-[10px] font-medium tracking-wider text-faint uppercase">
          {t('screen.situation.deviceList')}
        </h3>
        {devices.length === 0 ? (
          <p className="py-6 text-center text-xs text-muted">{t('screen.situation.noDevices')}</p>
        ) : (
          <div className="-mr-1 space-y-1 overflow-y-auto pr-1">
            {devices.map((device) => {
              const meta = DEVICE_STATUS[device.collectStatus] ?? UNKNOWN_STATUS
              const selected = device.id === selectedId
              return (
                <button
                  key={device.id}
                  type="button"
                  onClick={() => onSelect(device.id)}
                  aria-pressed={selected}
                  className={cn(
                    'flex w-full items-center gap-2 rounded-lg border px-2.5 py-2 text-left transition-colors',
                    selected
                      ? 'border-primary/50 bg-primary-soft/70'
                      : 'border-transparent hover:bg-surface-2',
                  )}
                >
                  <span className={cn('size-1.5 shrink-0 rounded-full', meta.dotClassName)} />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-xs font-medium">{device.name}</span>
                    <span className="mt-0.5 block truncate font-mono text-[10px] text-faint">
                      {device.deviceCode}
                    </span>
                  </span>
                  <span className="shrink-0 text-[10px] text-faint">
                    {PROTOCOLS[device.deviceType].short}
                  </span>
                </button>
              )
            })}
          </div>
        )}
      </div>
    </aside>
  )
}
