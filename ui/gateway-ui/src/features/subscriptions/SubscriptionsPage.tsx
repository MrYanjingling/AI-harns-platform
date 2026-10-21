import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Radio, Search } from 'lucide-react'
import {
  Badge,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  CopyButton,
  EmptyState,
  ErrorState,
  Input,
  PageHeader,
  Select,
  Table,
  TableSkeleton,
  TBody,
  TD,
  TH,
  THead,
  TR,
  useI18n,
} from '@harns/core'
import { useDevices, useGatewayMeta } from '@/api/hooks'
import type { SubscriptionSource, SubscriptionView } from '@/api/types'
import { DeviceStatusBadge } from '@/components/DeviceStatusBadge'
import { ProtocolLabel } from '@/components/ProtocolLabel'
import { defaultTopic } from '@/lib/constants'

export function SubscriptionsPage() {
  const { t } = useI18n()
  const devices = useDevices()
  const meta = useGatewayMeta()
  const [keyword, setKeyword] = useState('')
  const [sourceFilter, setSourceFilter] = useState<'all' | SubscriptionSource>('all')

  const gatewayId = meta.data?.id ?? ''

  const subscriptions = useMemo<SubscriptionView[]>(() => {
    const list = devices.data ?? []
    return list.map((device) => ({
      deviceId: device.id,
      deviceName: device.name,
      deviceCode: device.deviceCode ?? '',
      deviceType: device.deviceType,
      deviceModel: device.deviceModel,
      topic: device.topic || (gatewayId ? defaultTopic(gatewayId, device.id) : ''),
      source: device.topic ? 'custom' : 'auto',
      status: device.collectStatus,
    }))
  }, [devices.data, gatewayId])

  const filtered = useMemo(() => {
    const query = keyword.trim().toLowerCase()
    return subscriptions.filter((item) => {
      if (sourceFilter !== 'all' && item.source !== sourceFilter) return false
      if (!query) return true
      return [item.topic, item.deviceName, item.deviceCode, item.deviceId]
        .join(' ')
        .toLowerCase()
        .includes(query)
    })
  }, [subscriptions, keyword, sourceFilter])

  return (
    <div className="space-y-4">
      <PageHeader
        title={t('gateway.subscriptions.title')}
        description={t('gateway.subscriptions.description')}
        actions={
          <Badge tone="primary">
            {t('gateway.subscriptions.count', { count: subscriptions.length })}
          </Badge>
        }
      />

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('gateway.subscriptions.rule.title')}</CardTitle>
            <CardDescription>{t('gateway.subscriptions.rule.description')}</CardDescription>
          </div>
          <Radio className="size-4 text-faint" />
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <div>
            <p className="text-xs text-faint">{t('gateway.subscriptions.rule.gatewayId')}</p>
            <p className="mt-1 truncate font-mono text-sm text-fg">
              {gatewayId || t('gateway.common.notAvailable')}
            </p>
          </div>
          <div>
            <p className="text-xs text-faint">{t('gateway.subscriptions.rule.template')}</p>
            <p className="mt-1 truncate font-mono text-sm text-fg">
              data/&lt;gatewayId&gt;/v1/&lt;deviceId&gt;
            </p>
          </div>
        </CardContent>
      </Card>

      <div className="overflow-hidden rounded border border-border bg-surface">
        <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
          <div className="relative min-w-56 flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-faint" />
            <Input
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder={t('gateway.subscriptions.searchPlaceholder')}
              aria-label={t('common.action.search')}
              className="pl-8"
            />
          </div>
          <Select
            value={sourceFilter}
            onChange={(event) =>
              setSourceFilter(event.target.value as 'all' | SubscriptionSource)
            }
            aria-label={t('gateway.subscriptions.filter.source')}
            className="w-40"
          >
            <option value="all">{t('gateway.subscriptions.filter.allSources')}</option>
            <option value="custom">{t('gateway.subscriptions.source.custom')}</option>
            <option value="auto">{t('gateway.subscriptions.source.auto')}</option>
          </Select>
        </div>

        {devices.isLoading ? (
          <TableSkeleton rows={6} cols={5} />
        ) : devices.isError ? (
          <div className="p-3">
            <ErrorState error={devices.error} onRetry={() => void devices.refetch()} />
          </div>
        ) : subscriptions.length === 0 ? (
          <EmptyState
            title={t('gateway.subscriptions.empty.title')}
            description={t('gateway.subscriptions.empty.description')}
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            title={t('gateway.collection.empty.filteredTitle')}
            description={t('gateway.collection.empty.filteredDescription')}
          />
        ) : (
          <Table>
            <THead>
              <TR>
                <TH>{t('gateway.subscriptions.columns.topic')}</TH>
                <TH>{t('gateway.subscriptions.columns.device')}</TH>
                <TH>{t('gateway.subscriptions.columns.protocol')}</TH>
                <TH>{t('gateway.subscriptions.columns.status')}</TH>
                <TH>{t('gateway.subscriptions.columns.source')}</TH>
                <TH className="text-right">{t('gateway.subscriptions.columns.actions')}</TH>
              </TR>
            </THead>
            <TBody>
              {filtered.map((item) => (
                <TR key={item.deviceId}>
                  <TD>
                    <span className="block max-w-80 truncate font-mono text-xs text-fg" title={item.topic}>
                      {item.topic || t('gateway.common.notAvailable')}
                    </span>
                    <span className="block text-[11px] text-faint">
                      {t('gateway.subscriptions.note')}
                    </span>
                  </TD>
                  <TD>
                    <Link
                      to={`/gateway/collection/${item.deviceId}`}
                      className="block max-w-56 truncate text-sm font-medium text-fg hover:text-primary"
                    >
                      {item.deviceName}
                    </Link>
                    <span className="block max-w-56 truncate font-mono text-xs text-faint">
                      {item.deviceCode || item.deviceId}
                    </span>
                  </TD>
                  <TD>
                    <ProtocolLabel type={item.deviceType} model={item.deviceModel} />
                  </TD>
                  <TD>
                    <DeviceStatusBadge status={item.status} />
                  </TD>
                  <TD>
                    <Badge tone={item.source === 'custom' ? 'info' : 'neutral'}>
                      {item.source === 'custom'
                        ? t('gateway.subscriptions.source.custom')
                        : t('gateway.subscriptions.source.auto')}
                    </Badge>
                  </TD>
                  <TD>
                    <div className="flex items-center justify-end gap-1">
                      {item.topic ? (
                        <CopyButton
                          value={item.topic}
                          label={t('gateway.subscriptions.toast.copied')}
                        />
                      ) : null}
                      <Link
                        to={`/gateway/collection/${item.deviceId}`}
                        className="text-xs text-primary hover:underline"
                      >
                        {t('gateway.collection.action.detail')}
                      </Link>
                    </div>
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
        )}
      </div>
    </div>
  )
}
