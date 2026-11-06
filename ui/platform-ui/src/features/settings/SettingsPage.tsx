import type { ReactNode } from 'react'
import { Check, Database, Languages, Moon, RefreshCw, Server, Sun } from 'lucide-react'
import { useGatewayMeta, useMetrics } from '@/api/hooks'
import { useI18n } from '@/i18n'
import { useTheme } from '@/providers/ThemeProvider'
import { cn } from '@/lib/cn'
import { maskSecret } from '@/lib/format'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { CopyButton } from '@/components/ui/CopyButton'
import { PageHeader } from '@/components/shared/PageHeader'

const env = import.meta.env

const gatewayBase = env.VITE_GATEWAY_BASE || '/api/v1'
const dataBase = env.VITE_DATA_BASE || '/api/v1'
const gatewayTarget = env.VITE_GATEWAY_TARGET || 'http://127.0.0.1:32200'
const dataTarget = env.VITE_DATA_TARGET || 'http://127.0.0.1:32201'
const useMock = env.VITE_USE_MOCK === 'true'

export function SettingsPage() {
  const { t, locale, setLocale, locales } = useI18n()
  const { theme, setTheme } = useTheme()
  const gateway = useGatewayMeta()
  const dataService = useMetrics()

  return (
    <div>
      <PageHeader title={t('settings.title')} description={t('settings.description')} />

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('settings.appearance')}</CardTitle>
              <CardDescription>{t('settings.appearanceDescription')}</CardDescription>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <ThemeOption
                label={t('settings.theme.light')}
                active={theme === 'light'}
                onClick={() => setTheme('light')}
                icon={<Sun className="size-4" />}
                swatches={['bg-white', 'bg-zinc-100', 'bg-primary']}
              />
              <ThemeOption
                label={t('settings.theme.dark')}
                active={theme === 'dark'}
                onClick={() => setTheme('dark')}
                icon={<Moon className="size-4" />}
                swatches={['bg-zinc-900', 'bg-zinc-800', 'bg-primary']}
              />
            </div>
            <div>
              <p className="mb-2 text-xs font-medium">{t('settings.language')}</p>
              <div className="grid grid-cols-2 gap-3">
                {locales.map((item) => (
                  <ThemeOption
                    key={item.value}
                    label={item.label}
                    active={locale === item.value}
                    onClick={() => setLocale(item.value)}
                    icon={<Languages className="size-4" />}
                    swatches={[]}
                  />
                ))}
              </div>
              <p className="mt-2 text-[11px] leading-relaxed text-faint">
                {t('settings.languageDescription')}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('settings.runtime')}</CardTitle>
              <CardDescription>{t('settings.runtimeDescription')}</CardDescription>
            </div>
            {useMock ? (
              <Badge tone="warning">{t('settings.mockMode')}</Badge>
            ) : (
              <Badge tone="success">{t('settings.realBackend')}</Badge>
            )}
          </CardHeader>
          <CardContent className="space-y-3">
            <EnvRow
              label={t('settings.mode')}
              value={useMock ? t('settings.modeMock') : t('settings.modeProxy')}
            />
            <EnvRow label={t('settings.gatewayBase')} value={gatewayBase} />
            <EnvRow label={t('settings.dataBase')} value={dataBase} />
            <EnvRow
              label={t('settings.gatewayTarget')}
              value={useMock ? t('settings.disabled') : gatewayTarget}
            />
            <EnvRow
              label={t('settings.dataTarget')}
              value={useMock ? t('settings.disabled') : dataTarget}
            />
            <p className="text-[11px] leading-relaxed text-faint">{t('settings.proxyNote')}</p>
          </CardContent>
        </Card>
      </div>

      <Card className="mt-4">
        <CardHeader>
          <div>
            <CardTitle>{t('settings.service')}</CardTitle>
            <CardDescription>{t('settings.serviceDescription')}</CardDescription>
          </div>
          <Button
            size="sm"
            variant="outline"
            icon={<RefreshCw className="size-3.5" />}
            onClick={() => {
              void gateway.refetch()
              void dataService.refetch()
            }}
          >
            {t('settings.recheck')}
          </Button>
        </CardHeader>
        <CardContent className="space-y-3">
          <ServiceRow
            icon={<Server className="size-4" />}
            name={t('settings.gatewayService')}
            endpoint={t('settings.gatewayEndpoint', { base: gatewayBase })}
            loading={gateway.isLoading}
            online={gateway.isSuccess}
            labels={{
              checking: t('settings.checking'),
              online: t('settings.online'),
              offline: t('settings.unreachable'),
            }}
          >
            {gateway.data ? (
              <div className="mt-2 grid gap-x-6 gap-y-1 text-[11px] text-muted sm:grid-cols-2">
                <span>
                  {t('settings.name')} <span className="text-fg">{gateway.data.name}</span>
                </span>
                <span className="truncate">
                  Secret <span className="font-mono text-fg">{maskSecret(gateway.data.secret)}</span>
                </span>
                <span className="truncate">
                  ID <span className="font-mono text-fg">{gateway.data.id}</span>
                </span>
                <span>
                  eTag <span className="font-mono text-fg">{gateway.data.eTag}</span>
                </span>
              </div>
            ) : null}
          </ServiceRow>

          <ServiceRow
            icon={<Database className="size-4" />}
            name={t('settings.dataService')}
            endpoint={t('settings.dataEndpoint', { base: dataBase })}
            loading={dataService.isLoading}
            online={dataService.isSuccess}
            labels={{
              checking: t('settings.checking'),
              online: t('settings.online'),
              offline: t('settings.unreachable'),
            }}
          >
            {dataService.data ? (
              <p className="mt-2 text-[11px] text-muted">
                {t('settings.loadedMetrics', { count: dataService.data.length })}
              </p>
            ) : null}
          </ServiceRow>
        </CardContent>
      </Card>

      <Card className="mt-4">
        <CardHeader>
          <div>
            <CardTitle>{t('settings.deploy')}</CardTitle>
            <CardDescription>{t('settings.deployDescription')}</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-3 text-xs leading-relaxed text-muted">
          <p>{t('settings.deployIntro')}</p>
          <pre className="overflow-x-auto rounded border border-border bg-surface-2 p-4 font-mono text-[11px] leading-relaxed text-fg">
{`location / {
  try_files $uri /index.html;
}

location /api/v1/devices    { proxy_pass http://127.0.0.1:32200; }
location /api/v1/gateway    { proxy_pass http://127.0.0.1:32200; }
location /api/v1/metrics    { proxy_pass http://127.0.0.1:32201; }
location /api/v1/metricsData { proxy_pass http://127.0.0.1:32201; }
location /api/v1/kpi        { proxy_pass http://127.0.0.1:32201; }
location /api/v1/kpis       { proxy_pass http://127.0.0.1:32201; }
location /api/v1/tags       { proxy_pass http://127.0.0.1:32201; }
location /api/v1/fmcs       { proxy_pass http://127.0.0.1:32201; }
location /api/v1/escada     { proxy_pass http://127.0.0.1:32201; }`}
          </pre>
          <p className="text-faint">{t('settings.deployNote')}</p>
        </CardContent>
      </Card>

      <Card className="mt-4">
        <CardHeader>
          <CardTitle>{t('settings.about')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1.5 text-xs text-muted">
          <p>{t('settings.aboutVersion')}</p>
          <p>{t('settings.aboutStack')}</p>
          <p className="text-faint">{t('settings.aboutNote')}</p>
        </CardContent>
      </Card>
    </div>
  )
}

function ThemeOption({
  label,
  active,
  onClick,
  icon,
  swatches,
}: {
  label: string
  active: boolean
  onClick: () => void
  icon: ReactNode
  swatches: string[]
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        'rounded border p-3 text-left transition-colors',
        active
          ? 'border-primary bg-primary-soft/50 ring-1 ring-primary/30'
          : 'border-border hover:border-border-strong hover:bg-surface-2',
      )}
    >
      <span className="flex items-center justify-between">
        <span className="flex items-center gap-2 text-xs font-medium">
          {icon}
          {label}
        </span>
        {active ? <Check className="size-3.5 text-primary" /> : null}
      </span>
      {swatches.length > 0 ? (
        <span className="mt-3 flex gap-1.5">
          {swatches.map((swatch) => (
            <span
              key={swatch}
              className={cn(
                'h-7 flex-1 rounded-md border border-black/5 dark:border-white/10',
                swatch,
              )}
            />
          ))}
        </span>
      ) : null}
    </button>
  )
}

function EnvRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-border pb-2 last:border-0 last:pb-0">
      <span className="text-xs text-muted">{label}</span>
      <span className="flex min-w-0 items-center gap-1">
        <code className="truncate font-mono text-[11px] text-fg">{value}</code>
        <CopyButton value={value} />
      </span>
    </div>
  )
}

function ServiceRow({
  icon,
  name,
  endpoint,
  loading,
  online,
  labels,
  children,
}: {
  icon: ReactNode
  name: string
  endpoint: string
  loading: boolean
  online: boolean
  labels: { checking: string; online: string; offline: string }
  children?: ReactNode
}) {
  return (
    <div className="rounded border border-border px-4 py-3">
      <div className="flex flex-wrap items-center gap-3">
        <span className="flex size-8 items-center justify-center rounded-lg bg-surface-2 text-muted">
          {icon}
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium">{name}</p>
          <p className="truncate font-mono text-[11px] text-faint">{endpoint}</p>
        </div>
        {loading ? (
          <Badge tone="neutral" dot>
            {labels.checking}
          </Badge>
        ) : online ? (
          <Badge tone="success" dot>
            {labels.online}
          </Badge>
        ) : (
          <Badge tone="danger" dot>
            {labels.offline}
          </Badge>
        )}
      </div>
      {children}
    </div>
  )
}
