import {
  Activity,
  ArrowDownToLine,
  ArrowUpFromLine,
  Cable,
  Cpu,
  HardDrive,
  MemoryStick,
  Send,
  Server,
} from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { useI18n } from '@/providers/I18nProvider'

const rateBars = [28, 36, 31, 48, 42, 55, 45, 66, 52, 61, 44, 58, 70, 63, 75, 68]

function RateStat({
  title,
  unit,
  icon: Icon,
  accent,
}: {
  title: string
  unit: string
  icon: typeof Activity
  accent: 'primary' | 'soft'
}) {
  return (
    <div className="p-5">
      <div className="flex items-start gap-3">
        <span className="flex size-9 items-center justify-center rounded-lg bg-primary-soft text-primary">
          <Icon className="size-4.5" />
        </span>
        <div className="min-w-0">
          <div className="text-sm text-muted">{title}</div>
          <div className="mt-2 flex items-baseline gap-1.5">
            <span className="text-2xl font-semibold">--</span>
            <span className="text-xs text-faint">{unit}</span>
          </div>
        </div>
      </div>
      <div className="mt-5 flex h-20 items-end gap-1">
        {rateBars.map((height, index) => (
          <span
            key={index}
            style={{ height: `${height}%` }}
            className={`min-w-0 flex-1 rounded-t-sm ${accent === 'primary' ? 'bg-primary' : 'bg-primary/35'}`}
          />
        ))}
      </div>
    </div>
  )
}

function InlineMetric({
  title,
  icon: Icon,
}: {
  title: string
  icon: typeof Activity
}) {
  return (
    <div className="flex items-center gap-3 p-4">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-surface-2 text-muted">
        <Icon className="size-4.5" />
      </span>
      <div className="min-w-0">
        <div className="truncate text-xs text-muted">{title}</div>
        <div className="mt-1 text-lg font-semibold">--</div>
      </div>
    </div>
  )
}

function ResourceBar({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="mb-2 flex items-center justify-between text-xs">
        <span className="text-muted">{label}</span>
        <span className="text-faint">{value}</span>
      </div>
      <div className="h-1.5 overflow-hidden rounded-full bg-surface-3">
        <div className="h-full w-0 rounded-full bg-primary" />
      </div>
    </div>
  )
}

export function OverviewPage() {
  const { t } = useI18n()

  return (
    <div className="space-y-4">
      <Card className="overflow-hidden">
        <div className="grid divide-y divide-border lg:grid-cols-[1.1fr_1.4fr] lg:divide-x lg:divide-y-0">
          <div className="grid grid-cols-2 divide-x divide-border">
            <RateStat
              title={t('overview.collectionRate')}
              unit={t('overview.rateUnit')}
              icon={ArrowDownToLine}
              accent="primary"
            />
            <RateStat
              title={t('overview.pushRate')}
              unit={t('overview.pushUnit')}
              icon={ArrowUpFromLine}
              accent="soft"
            />
          </div>
          <div className="grid grid-cols-2 divide-x divide-y divide-border sm:grid-cols-3 sm:divide-y-0">
            <InlineMetric title={t('overview.totalCollections')} icon={Cable} />
            <InlineMetric title={t('overview.runningCollections')} icon={Activity} />
            <InlineMetric title={t('overview.pushGroups')} icon={Send} />
            <InlineMetric title={t('overview.runningConnectors')} icon={Server} />
            <InlineMetric title={t('overview.variables')} icon={Cpu} />
            <InlineMetric title={t('overview.errorCollections')} icon={Activity} />
          </div>
        </div>
      </Card>

      <div className="grid gap-4 xl:grid-cols-[1.7fr_1fr]">
        <Card className="p-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-base font-semibold">{t('overview.gatewayNode')}</h2>
              <p className="mt-1 text-xs text-muted">{t('overview.resourceHint')}</p>
            </div>
            <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-surface-2 px-2.5 py-1 text-xs text-muted">
              <span className="size-1.5 rounded-full bg-primary" />
              {t('overview.running')}
            </span>
          </div>

          <div className="mt-5 grid gap-4 sm:grid-cols-2">
            {[
              [t('overview.gatewayName'), 'harnsgateway'],
              [t('overview.gatewayId'), '--'],
              [t('overview.version'), '--'],
              [t('overview.uptime'), '--'],
            ].map(([label, value]) => (
              <div key={label} className="rounded-lg border border-border bg-surface-2/40 px-3 py-2.5">
                <div className="text-xs text-muted">{label}</div>
                <div className="mt-1 truncate text-sm font-medium">{value}</div>
              </div>
            ))}
          </div>

          <div className="mt-5 grid gap-4 sm:grid-cols-3">
            <ResourceBar label={t('overview.cpu')} value="--" />
            <ResourceBar label={t('overview.memory')} value="--" />
            <ResourceBar label={t('overview.disk')} value="--" />
          </div>
        </Card>

        <Card className="p-5">
          <div className="flex items-center gap-2">
            <span className="flex size-8 items-center justify-center rounded-lg bg-primary-soft text-primary">
              <MemoryStick className="size-4" />
            </span>
            <div>
              <h2 className="text-base font-semibold">{t('overview.recent')}</h2>
              <p className="text-xs text-muted">{t('overview.recentHint')}</p>
            </div>
          </div>
          <div className="mt-5 flex min-h-48 flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border bg-surface-2/30 text-center">
            <HardDrive className="size-6 text-faint" />
            <span className="text-sm text-muted">{t('overview.noAlerts')}</span>
          </div>
        </Card>
      </div>
    </div>
  )
}
