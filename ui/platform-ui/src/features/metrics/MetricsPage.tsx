import { useEffect, useMemo, useRef, useState } from 'react'
import { Activity, BarChart3, Search } from 'lucide-react'
import { useMetrics, useMetricsValues } from '@/api/hooks'
import type { MetricsQueryRequest } from '@/api/types'
import { useI18n } from '@/i18n'
import { formatDateTimeShort, formatNumber } from '@/lib/format'
import { cn } from '@/lib/cn'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/EmptyState'
import { Skeleton, TableSkeleton } from '@/components/ui/Skeleton'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'
import { ErrorState } from '@/components/shared/ErrorState'
import { PageHeader } from '@/components/shared/PageHeader'
import { StatCard } from '@/components/shared/StatCard'
import { TimeRangeFields, useTimeRange } from '@/components/shared/TimeRangeFields'

export function MetricsPage() {
  const { t } = useI18n()
  const metrics = useMetrics()
  const range = useTimeRange(24)
  const [selected, setSelected] = useState<string[]>([])
  const [request, setRequest] = useState<MetricsQueryRequest | null>(null)
  const results = useMetricsValues(request)
  const initialized = useRef(false)

  useEffect(() => {
    if (initialized.current || !metrics.data) return
    initialized.current = true
    setSelected(
      metrics.data
        .filter((item) => item.enable)
        .slice(0, 4)
        .map((item) => item.metricsCode ?? '')
        .filter(Boolean),
    )
  }, [metrics.data])

  const metaByCode = useMemo(
    () =>
      new Map(
        (metrics.data ?? [])
          .filter((item) => item.metricsCode)
          .map((item) => [item.metricsCode as string, item] as const),
      ),
    [metrics.data],
  )

  const toggle = (code: string) => {
    setSelected((prev) =>
      prev.includes(code) ? prev.filter((item) => item !== code) : [...prev, code],
    )
  }

  const runQuery = () => {
    const api = range.toApi()
    if (!api || selected.length === 0) return
    setRequest({ metricsCodes: selected, granularity: 'h', ...api })
  }

  const allCodes = (metrics.data ?? [])
    .map((item) => item.metricsCode ?? '')
    .filter(Boolean)
  const allSelected = allCodes.length > 0 && selected.length === allCodes.length

  return (
    <div>
      <PageHeader
        title={t('metrics.title')}
        description={t('metrics.description')}
        actions={
          <Button
            variant="primary"
            icon={<Search className="size-4" />}
            disabled={selected.length === 0 || !range.valid}
            onClick={runQuery}
          >
            {t('metrics.query')}
          </Button>
        }
      />

      <div className="grid gap-4 lg:grid-cols-[1.4fr_1fr]">
        <Card className="overflow-hidden">
          <CardHeader>
            <div>
              <CardTitle>{t('metrics.listTitle')}</CardTitle>
              <p className="mt-0.5 text-xs text-muted">
                {t('metrics.selected', { selected: selected.length, total: allCodes.length })}
              </p>
            </div>
            <div className="flex items-center gap-1">
              <Button
                size="sm"
                variant="ghost"
                onClick={() => setSelected(allSelected ? [] : allCodes)}
              >
                {allSelected ? t('common.action.deselectAll') : t('common.action.selectAll')}
              </Button>
            </div>
          </CardHeader>

          {metrics.isLoading ? (
            <TableSkeleton rows={5} cols={4} />
          ) : metrics.isError ? (
            <div className="p-5">
              <ErrorState error={metrics.error} onRetry={metrics.refetch} compact />
            </div>
          ) : (metrics.data ?? []).length === 0 ? (
            <EmptyState
              title={t('metrics.empty.title')}
              description={t('metrics.empty.description')}
            />
          ) : (
            <Table>
              <THead>
                <tr>
                  <TH className="w-10">
                    <input
                      type="checkbox"
                      aria-label={t('metrics.selectAllAria')}
                      checked={allSelected}
                      onChange={() => setSelected(allSelected ? [] : allCodes)}
                      className="size-4 cursor-pointer accent-[var(--primary)]"
                    />
                  </TH>
                  <TH>{t('metrics.table.metric')}</TH>
                  <TH>{t('metrics.table.calculation')}</TH>
                  <TH>{t('metrics.table.unit')}</TH>
                  <TH>{t('common.field.status')}</TH>
                </tr>
              </THead>
              <TBody>
                {(metrics.data ?? []).map((item) => {
                  const code = item.metricsCode ?? item.id
                  const checked = selected.includes(item.metricsCode ?? '')
                  return (
                    <TR
                      key={item.id}
                      className={cn('cursor-pointer', checked && 'bg-primary-soft/40')}
                      onClick={() => toggle(item.metricsCode ?? '')}
                    >
                      <TD>
                        <input
                          type="checkbox"
                          aria-label={t('metrics.selectAria', { code })}
                          checked={checked}
                          onChange={() => toggle(item.metricsCode ?? '')}
                          onClick={(event) => event.stopPropagation()}
                          className="size-4 cursor-pointer accent-[var(--primary)]"
                        />
                      </TD>
                      <TD>
                        <p className="font-medium">{item.metricsName || code}</p>
                        <p className="font-mono text-[11px] text-faint">{code}</p>
                      </TD>
                      <TD className="text-xs text-muted">
                        {item.calculateType || '—'}
                        {item.metricsType ? (
                          <span className="ml-1.5 text-faint">/ {item.metricsType}</span>
                        ) : null}
                      </TD>
                      <TD className="text-xs text-muted">{item.measureType || '—'}</TD>
                      <TD>
                        {item.enable ? (
                          <Badge tone="success" dot>
                            {t('metrics.enabled')}
                          </Badge>
                        ) : (
                          <Badge tone="neutral" dot>
                            {t('metrics.disabled')}
                          </Badge>
                        )}
                      </TD>
                    </TR>
                  )
                })}
              </TBody>
            </Table>
          )}
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('metrics.conditions')}</CardTitle>
              <p className="mt-0.5 text-xs text-muted">{t('metrics.granularityHint')}</p>
            </div>
            <BarChart3 className="size-4 text-faint" />
          </CardHeader>
          <CardContent className="space-y-4">
            <TimeRangeFields range={range} />
            <p className="text-[11px] leading-relaxed text-faint">
              {t('metrics.conditionsHint')}
            </p>
            <Button
              className="w-full"
              variant="primary"
              icon={<Search className="size-4" />}
              disabled={selected.length === 0 || !range.valid}
              onClick={runQuery}
            >
              {selected.length > 0
                ? t('metrics.queryCount', { count: selected.length })
                : t('metrics.queryPlain')}
            </Button>
          </CardContent>
        </Card>
      </div>

      <div className="mt-6">
        <h2 className="mb-3 text-sm font-semibold">{t('metrics.results')}</h2>
        {!request ? (
          <Card>
            <EmptyState
              icon={<Activity className="size-5" />}
              title={t('metrics.notQueried.title')}
              description={t('metrics.notQueried.description')}
            />
          </Card>
        ) : results.isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {request.metricsCodes.map((code) => (
              <Skeleton key={code} className="h-28 rounded" />
            ))}
          </div>
        ) : results.isError ? (
          <ErrorState error={results.error} onRetry={results.refetch} />
        ) : (results.data ?? []).length === 0 ? (
          <Card>
            <EmptyState
              title={t('metrics.noData.title')}
              description={t('metrics.noData.description')}
            />
          </Card>
        ) : (
          <>
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {(results.data ?? []).map((result) => {
                const meta = metaByCode.get(result.metricsCode)
                return (
                  <StatCard
                    key={result.metricsCode}
                    label={meta?.metricsName || result.metricsCode}
                    value={
                      <span>
                        {formatNumber(result.value, 3)}
                        {meta?.measureType ? (
                          <span className="ml-1 text-sm font-normal text-muted">
                            {meta.measureType}
                          </span>
                        ) : null}
                      </span>
                    }
                    hint={`${formatDateTimeShort(result.startTime)} ~ ${formatDateTimeShort(result.endTime)}`}
                    icon={<Activity className="size-4" />}
                  />
                )
              })}
            </div>
            <p className="mt-3 text-[11px] text-faint">
              {t('metrics.rangeInfo', {
                start: formatDateTimeShort(request.startTime),
                end: formatDateTimeShort(request.endTime),
              })}
              {' · '}
              {t('metrics.granularity', { granularity: request.granularity })}
            </p>
          </>
        )}
      </div>
    </div>
  )
}
