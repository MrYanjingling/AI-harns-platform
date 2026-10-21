import { useMemo, useState } from 'react'
import { Boxes, Search } from 'lucide-react'
import {
  Badge,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  ErrorState,
  Input,
  PageHeader,
  Table,
  TableSkeleton,
  TBody,
  TD,
  TH,
  THead,
  TR,
  useI18n,
} from '@harns/core'
import { useDevices } from '@/api/hooks'
import type { PluginCatalogItem } from '@/api/types'
import { PLUGIN_CATALOG, deviceTypeMeta } from '@/lib/constants'

function capabilityKey(key: string): string {
  return key
}

export function PluginsPage() {
  const { t } = useI18n()
  const devices = useDevices()
  const [keyword, setKeyword] = useState('')

  const counts = useMemo(() => {
    const result: Record<string, number> = {}
    for (const device of devices.data ?? []) {
      result[device.deviceType] = (result[device.deviceType] ?? 0) + 1
    }
    return result
  }, [devices.data])

  const filtered = useMemo<PluginCatalogItem[]>(() => {
    const query = keyword.trim().toLowerCase()
    if (!query) return PLUGIN_CATALOG
    return PLUGIN_CATALOG.filter((plugin) =>
      [plugin.name, ...plugin.models, ...plugin.dataTypes, ...plugin.transports]
        .join(' ')
        .toLowerCase()
        .includes(query),
    )
  }, [keyword])

  return (
    <div className="space-y-4">
      <PageHeader
        title={t('gateway.plugins.title')}
        description={t('gateway.plugins.description')}
        actions={
          <div className="relative w-64">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-faint" />
            <Input
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder={t('gateway.plugins.searchPlaceholder')}
              aria-label={t('common.action.search')}
              className="pl-8"
            />
          </div>
        }
      />

      {devices.isLoading ? (
        <Card>
          <TableSkeleton rows={4} cols={3} />
        </Card>
      ) : devices.isError ? (
        <ErrorState error={devices.error} onRetry={() => void devices.refetch()} />
      ) : filtered.length === 0 ? (
        <EmptyState
          title={t('gateway.plugins.empty.title')}
          description={t('gateway.plugins.empty.description')}
        />
      ) : (
        <div className="grid gap-3 lg:grid-cols-3">
          {filtered.map((plugin) => (
            <Card key={plugin.key}>
              <CardHeader>
                <div>
                  <CardTitle className="flex items-center gap-2">
                    <Boxes className="size-4 text-primary" />
                    {plugin.name}
                  </CardTitle>
                  <CardDescription>{t(plugin.descriptionKey)}</CardDescription>
                </div>
                <Badge tone="success">{t('gateway.plugins.builtin')}</Badge>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex flex-wrap items-center gap-2 text-xs text-muted">
                  <Badge tone="primary">
                    {t('gateway.plugins.deviceCount', {
                      count: counts[plugin.key] ?? 0,
                    })}
                  </Badge>
                  <span className="font-mono">{plugin.key}</span>
                </div>

                <div>
                  <p className="text-xs font-medium text-muted">
                    {t('gateway.plugins.models')}
                  </p>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {plugin.models.map((model) => {
                      const meta = deviceTypeMeta(plugin.key)
                      return (
                        <Badge key={model} tone="neutral">
                          {meta?.models.find((item) => item.value === model)?.label ?? model}
                        </Badge>
                      )
                    })}
                  </div>
                </div>

                <div>
                  <p className="text-xs font-medium text-muted">
                    {t('gateway.plugins.transports')}
                  </p>
                  <p className="mt-1 text-xs text-fg">{plugin.transports.join(' · ')}</p>
                </div>

                <div>
                  <p className="text-xs font-medium text-muted">
                    {t('gateway.plugins.dataTypes')}
                  </p>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {plugin.dataTypes.map((type) => (
                      <span
                        key={type}
                        className="rounded border border-border bg-surface-2 px-1.5 py-0.5 font-mono text-[11px] text-muted"
                      >
                        {type}
                      </span>
                    ))}
                  </div>
                </div>

                <div>
                  <p className="text-xs font-medium text-muted">
                    {t('gateway.plugins.capabilities')}
                  </p>
                  <ul className="mt-1 space-y-1 text-xs text-fg">
                    {plugin.capabilities.map((capability) => (
                      <li key={capability} className="flex items-center gap-1.5">
                        <span className="size-1 rounded-full bg-success" />
                        {t(capabilityKey(capability))}
                      </li>
                    ))}
                  </ul>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('gateway.plugins.registry')}</CardTitle>
            <CardDescription>{t('gateway.plugins.registryHint')}</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="px-0 pb-0">
          <Table>
            <THead>
              <TR>
                <TH>deviceType</TH>
                <TH>{t('gateway.form.field.deviceModel')}</TH>
                <TH>{t('gateway.plugins.devicesUsing')}</TH>
              </TR>
            </THead>
            <TBody>
              {PLUGIN_CATALOG.map((plugin) => (
                <TR key={plugin.key}>
                  <TD className="font-mono text-xs">{plugin.key}</TD>
                  <TD className="font-mono text-xs text-muted">{plugin.models.join(' / ')}</TD>
                  <TD>
                    {counts[plugin.key] ? (
                      <Badge tone="primary">
                        {t('gateway.plugins.deviceCount', { count: counts[plugin.key] })}
                      </Badge>
                    ) : (
                      <span className="text-xs text-faint">{t('gateway.plugins.noDevices')}</span>
                    )}
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
