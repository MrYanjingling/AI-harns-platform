import { useEffect, useMemo, useRef, useState } from 'react'
import { Search, Target } from 'lucide-react'
import { useKpis, useKpiValues } from '@/api/hooks'
import type { KpiQueryRequest } from '@/api/types'
import { useI18n, type MessageKey } from '@/i18n'
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

const GRAIN_LABEL: Record<string, MessageKey> = {
  h: 'kpi.grain.h',
  d: 'kpi.grain.d',
  m: 'kpi.grain.m',
  y: 'kpi.grain.y',
}

export function KpiPage() {
  const { t } = useI18n()
  const kpis = useKpis()
  const range = useTimeRange(24)
  const [selected, setSelected] = useState<string[]>([])
  const [request, setRequest] = useState<KpiQueryRequest | null>(null)
  const results = useKpiValues(request)
  const initialized = useRef(false)

  useEffect(() => {
    if (initialized.current || !kpis.data) return
    initialized.current = true
    setSelected(kpis.data.filter((item) => item.status === 'active').map((item) => item.id))
  }, [kpis.data])

  const metaById = useMemo(
    () => new Map((kpis.data ?? []).map((item) => [item.id, item] as const)),
    [kpis.data],
  )

  const toggle = (id: string) =>
    setSelected((prev) => (prev.includes(id) ? prev.filter((item) => item !== id) : [...prev, id]))

  const allIds = (kpis.data ?? []).map((item) => item.id)
  const allSelected = allIds.length > 0 && selected.length === allIds.length

  const runQuery = () => {
    const api = range.toApi()
    if (!api || selected.length === 0) return
    setRequest({ ids: selected, granularity: 'h', ...api })
  }

  return (
    <div>
      <PageHeader
        title={t('kpi.title')}
        description={t('kpi.description')}
        actions={
          <Button
            variant="primary"
            icon={<Search className="size-4" />}
            disabled={selected.length === 0 || !range.valid}
            onClick={runQuery}
          >
            {t('kpi.calculate')}
          </Button>
        }
      />

      <div className="grid gap-4 lg:grid-cols-[1.4fr_1fr]">
        <Card className="overflow-hidden">
          <CardHeader>
            <div>
              <CardTitle>{t('kpi.listTitle')}</CardTitle>
              <p className="mt-0.5 text-xs text-muted">
                {t('kpi.selected', { selected: selected.length, total: allIds.length })}
              </p>
            </div>
            <Button size="sm" variant="ghost" onClick={() => setSelected(allSelected ? [] : allIds)}>
              {allSelected ? t('common.action.deselectAll') : t('common.action.selectAll')}
            </Button>
          </CardHeader>

          {kpis.isLoading ? (
            <TableSkeleton rows={4} cols={5} />
          ) : kpis.isError ? (
            <div className="p-5">
              <ErrorState error={kpis.error} onRetry={kpis.refetch} compact />
            </div>
          ) : (kpis.data ?? []).length === 0 ? (
            <EmptyState
              title={t('kpi.empty.title')}
              description={t('kpi.empty.description')}
            />
          ) : (
            <Table>
              <THead>
                <tr>
                  <TH className="w-10">
                    <input
                      type="checkbox"
                      aria-label={t('kpi.selectAllAria')}
                      checked={allSelected}
                      onChange={() => setSelected(allSelected ? [] : allIds)}
                      className="size-4 cursor-pointer accent-[var(--primary)]"
                    />
                  </TH>
                  <TH>{t('kpi.table.name')}</TH>
                  <TH>{t('kpi.table.formula')}</TH>
                  <TH>{t('kpi.table.granularity')}</TH>
                  <TH>{t('common.field.status')}</TH>
                </tr>
              </THead>
              <TBody>
                {(kpis.data ?? []).map((item) => {
                  const checked = selected.includes(item.id)
                  return (
                    <TR
                      key={item.id}
                      className={cn('cursor-pointer', checked && 'bg-primary-soft/40')}
                      onClick={() => toggle(item.id)}
                    >
                      <TD>
                        <input
                          type="checkbox"
                          aria-label={t('kpi.selectAria', { name: item.name })}
                          checked={checked}
                          onChange={() => toggle(item.id)}
                          onClick={(event) => event.stopPropagation()}
                          className="size-4 cursor-pointer accent-[var(--primary)]"
                        />
                      </TD>
                      <TD>
                        <p className="font-medium">{item.name}</p>
                        <p className="font-mono text-[11px] text-faint">{item.id}</p>
                      </TD>
                      <TD>
                        <code
                          className="block max-w-48 truncate font-mono text-[11px] text-muted"
                          title={item.formula ?? item.formulaOri}
                        >
                          {item.formula || item.formulaOri}
                        </code>
                      </TD>
                      <TD className="text-xs text-muted">
                        {GRAIN_LABEL[item.timeGranularity]
                          ? t(GRAIN_LABEL[item.timeGranularity])
                          : item.timeGranularity}
                      </TD>
                      <TD>
                        <Badge
                          tone={item.status === 'active' ? 'success' : 'warning'}
                          dot
                        >
                          {item.status === 'active'
                            ? t('kpi.status.active')
                            : item.status === 'draft'
                              ? t('kpi.status.draft')
                              : item.status}
                        </Badge>
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
              <CardTitle>{t('kpi.conditions')}</CardTitle>
              <p className="mt-0.5 text-xs text-muted">{t('kpi.conditionsHint')}</p>
            </div>
            <Target className="size-4 text-faint" />
          </CardHeader>
          <CardContent className="space-y-4">
            <TimeRangeFields range={range} />
            <p className="text-[11px] leading-relaxed text-faint">{t('kpi.submitHint')}</p>
            <Button
              className="w-full"
              variant="primary"
              icon={<Search className="size-4" />}
              disabled={selected.length === 0 || !range.valid}
              onClick={runQuery}
            >
              {selected.length > 0
                ? t('kpi.calculateCount', { count: selected.length })
                : t('kpi.calculatePlain')}
            </Button>
          </CardContent>
        </Card>
      </div>

      <div className="mt-6">
        <h2 className="mb-3 text-sm font-semibold">{t('kpi.results')}</h2>
        {!request ? (
          <Card>
            <EmptyState
              icon={<Target className="size-5" />}
              title={t('kpi.notCalculated.title')}
              description={t('kpi.notCalculated.description')}
            />
          </Card>
        ) : results.isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {request.ids.map((id) => (
              <Skeleton key={id} className="h-28 rounded" />
            ))}
          </div>
        ) : results.isError ? (
          <ErrorState error={results.error} onRetry={results.refetch} />
        ) : (results.data ?? []).length === 0 ? (
          <Card>
            <EmptyState
              title={t('kpi.noResult.title')}
              description={t('kpi.noResult.description')}
            />
          </Card>
        ) : (
          <>
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {(results.data ?? []).map((result) => {
                const meta = metaById.get(result.id)
                const target = meta?.targetData ? Number(meta.targetData) : Number.NaN
                const hasTarget = Number.isFinite(target)
                return (
                  <StatCard
                    key={result.id}
                    label={meta?.name ?? result.id}
                    value={
                      <span>
                        {formatNumber(result.value, 2)}
                        {meta?.unit ? (
                          <span className="ml-1 text-sm font-normal text-muted">{meta.unit}</span>
                        ) : null}
                      </span>
                    }
                    hint={`${formatDateTimeShort(result.startTime)} ~ ${formatDateTimeShort(result.endTime)}`}
                    icon={<Target className="size-4" />}
                    footer={
                      hasTarget ? (
                        <div className="flex items-center justify-between text-[11px]">
                          <span className="text-faint">
                            {t('kpi.target', { value: `${meta?.targetData ?? ''}${meta?.unit ?? ''}` })}
                          </span>
                          <Badge tone={result.value >= target ? 'success' : 'warning'}>
                            {result.value >= target ? t('kpi.onTarget') : t('kpi.offTarget')}
                          </Badge>
                        </div>
                      ) : null
                    }
                  />
                )
              })}
            </div>
            <p className="mt-3 text-[11px] text-faint">
              {t('kpi.rangeInfo', {
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
