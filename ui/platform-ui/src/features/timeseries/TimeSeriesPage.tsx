import { useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { AreaChart, Search } from 'lucide-react'
import { useTags, useTimeSeries } from '@/api/hooks'
import type { DataPointTimeSeries, TagTimeSeriesRequest, TimeSeriesMode, TimeSeriesSource } from '@/api/types'
import { useI18n, type MessageKey } from '@/i18n'
import { formatDateTimeShort, formatNumber } from '@/lib/format'
import { cn } from '@/lib/cn'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { EmptyState } from '@/components/ui/EmptyState'
import { Field, Input, Select } from '@/components/ui/Field'
import { PageLoading } from '@/components/ui/Spinner'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'
import { ErrorState } from '@/components/shared/ErrorState'
import { PageHeader } from '@/components/shared/PageHeader'
import { TimeRangeFields, useTimeRange } from '@/components/shared/TimeRangeFields'

const SERIES_COLORS = [
  '#10b981',
  '#0ea5e9',
  '#f59e0b',
  '#8b5cf6',
  '#ef4444',
  '#14b8a6',
  '#6366f1',
  '#f43f5e',
  '#84cc16',
  '#06b6d4',
]

const MODES: Array<{ value: TimeSeriesMode; labelKey: MessageKey; hintKey: MessageKey }> = [
  { value: 'raw', labelKey: 'timeseries.mode.raw.label', hintKey: 'timeseries.mode.raw.hint' },
  { value: 'mean', labelKey: 'timeseries.mode.mean.label', hintKey: 'timeseries.mode.mean.hint' },
  { value: 'usage', labelKey: 'timeseries.mode.usage.label', hintKey: 'timeseries.mode.usage.hint' },
]

/**
 * 把多个测点的时序按时间戳合并成图表行。
 * 使用 s0/s1 作为列名,避免测点 ID 中的 "." 被 Recharts 当作嵌套路径。
 */
function mergeSeries(series: DataPointTimeSeries[]) {
  const rows = new Map<string, Record<string, string | number>>()
  series.forEach((item, index) => {
    const key = `s${index}`
    for (const point of item.data ?? []) {
      const value = Number(point.value)
      if (!Number.isFinite(value)) continue
      const row = rows.get(point.time) ?? { time: point.time }
      row[key] = value
      rows.set(point.time, row)
    }
  })
  return Array.from(rows.values()).sort((a, b) =>
    String(a.time).localeCompare(String(b.time)),
  )
}

function summarize(series: DataPointTimeSeries) {
  const values = (series.data ?? [])
    .map((point) => Number(point.value))
    .filter((value) => Number.isFinite(value))
  if (values.length === 0) return null
  return {
    count: values.length,
    min: Math.min(...values),
    max: Math.max(...values),
    latest: values[values.length - 1],
  }
}

export function TimeSeriesPage() {
  const { t } = useI18n()
  const [searchParams] = useSearchParams()
  const tags = useTags()
  const range = useTimeRange(24)
  const [source, setSource] = useState<TimeSeriesSource>('fmcs')
  const [selectText, setSelectText] = useState(searchParams.get('select') ?? '')
  const [mode, setMode] = useState<TimeSeriesMode>('raw')
  const [limit, setLimit] = useState(48)
  const [tagSearch, setTagSearch] = useState('')
  const [request, setRequest] = useState<TagTimeSeriesRequest | null>(null)
  const series = useTimeSeries(source, request)
  const initialized = useRef(Boolean(searchParams.get('select')))

  useEffect(() => {
    if (initialized.current || !tags.data?.length) return
    initialized.current = true
    const first = tags.data.find((tag) => tag.enable === 1 && tag.tagId) ?? tags.data[0]
    if (first?.tagId) setSelectText(first.tagId)
  }, [tags.data])

  const selectedIds = useMemo(
    () =>
      selectText
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean),
    [selectText],
  )

  const filteredTags = useMemo(() => {
    const keyword = tagSearch.trim().toLowerCase()
    return (tags.data ?? []).filter((tag) => {
      if (!keyword) return true
      return (
        (tag.tagId ?? '').toLowerCase().includes(keyword) ||
        (tag.tagName ?? '').toLowerCase().includes(keyword)
      )
    })
  }, [tags.data, tagSearch])

  const toggleTag = (tagId: string) => {
    const next = selectedIds.includes(tagId)
      ? selectedIds.filter((item) => item !== tagId)
      : [...selectedIds, tagId]
    setSelectText(next.join(','))
  }

  const runQuery = () => {
    const api = range.toApi()
    const select = selectText.trim()
    if (!api || !select) return
    setRequest({
      select,
      start: `${api.startTime}.000`,
      end: `${api.endTime}.000`,
      limit,
      mode,
    })
  }

  const chartData = useMemo(() => mergeSeries(series.data ?? []), [series.data])
  const chartSeries = series.data ?? []

  const canQuery = selectedIds.length > 0 && range.valid && limit >= 2

  return (
    <div>
      <PageHeader
        title={t('timeseries.title')}
        description={t('timeseries.description')}
        actions={
          <Button
            variant="primary"
            icon={<Search className="size-4" />}
            disabled={!canQuery}
            onClick={runQuery}
          >
            {t('timeseries.query')}
          </Button>
        }
      />

      <div className="grid gap-4 lg:grid-cols-[1.35fr_1fr]">
        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('timeseries.conditions')}</CardTitle>
              <p className="mt-0.5 text-xs text-muted">{t('timeseries.conditionsHint')}</p>
            </div>
            <Badge tone="neutral">{t('timeseries.tagCount', { count: selectedIds.length })}</Badge>
          </CardHeader>
          <CardContent className="space-y-4">
            <Field label={t('timeseries.selectLabel')} hint={t('timeseries.selectHint')}>
              <Input
                value={selectText}
                onChange={(event) => setSelectText(event.target.value)}
                placeholder={t('timeseries.selectPlaceholder')}
                className="font-mono text-xs"
              />
            </Field>

            <div className="flex flex-wrap items-end gap-3">
              <Field label={t('timeseries.source')} className="w-36">
                <Select
                  value={source}
                  onChange={(event) => setSource(event.target.value as TimeSeriesSource)}
                >
                  <option value="fmcs">FMCS</option>
                  <option value="escada">Escada</option>
                </Select>
              </Field>
              <Field label={t('timeseries.mode')} className="w-36">
                <Select
                  value={mode}
                  onChange={(event) => setMode(event.target.value as TimeSeriesMode)}
                >
                  {MODES.map((item) => (
                    <option key={item.value} value={item.value}>
                      {t(item.labelKey)}
                    </option>
                  ))}
                </Select>
              </Field>
              <Field
                label={t('timeseries.limit')}
                className="w-28"
                error={limit < 2 ? t('timeseries.limitError') : undefined}
              >
                <Input
                  type="number"
                  min={2}
                  max={2000}
                  value={limit}
                  onChange={(event) => setLimit(Number(event.target.value))}
                />
              </Field>
            </div>

            <TimeRangeFields range={range} />

            <p className="text-[11px] leading-relaxed text-faint">
              {t('timeseries.requestHint', {
                hint: t(MODES.find((item) => item.value === mode)?.hintKey ?? 'timeseries.mode.raw.hint'),
                source,
              })}
            </p>

            <Button
              className="w-full"
              variant="primary"
              icon={<Search className="size-4" />}
              disabled={!canQuery}
              onClick={runQuery}
            >
              {selectedIds.length > 0
                ? t('timeseries.queryCount', { count: selectedIds.length })
                : t('timeseries.queryPlain')}
            </Button>
          </CardContent>
        </Card>

        <Card className="flex flex-col">
          <CardHeader>
            <div>
              <CardTitle>{t('timeseries.available')}</CardTitle>
              <p className="mt-0.5 text-xs text-muted">{t('timeseries.availableHint')}</p>
            </div>
            <Input
              value={tagSearch}
              onChange={(event) => setTagSearch(event.target.value)}
              placeholder={t('timeseries.filterTags')}
              className="h-8 w-36 text-xs"
            />
          </CardHeader>
          <div className="max-h-96 overflow-y-auto px-2 pb-2">
            {tags.isLoading ? (
              <PageLoading label={t('timeseries.loadingTags')} />
            ) : tags.isError ? (
              <div className="p-3">
                <ErrorState error={tags.error} onRetry={tags.refetch} compact />
              </div>
            ) : filteredTags.length === 0 ? (
              <p className="px-3 py-8 text-center text-xs text-muted">
                {t('timeseries.noMatchingTags')}
              </p>
            ) : (
              <ul className="space-y-0.5">
                {filteredTags.map((tag) => {
                  const tagId = tag.tagId ?? ''
                  const checked = selectedIds.includes(tagId)
                  return (
                    <li key={tag.id}>
                      <label
                        className={cn(
                          'flex cursor-pointer items-start gap-2.5 rounded-lg px-3 py-2 transition-colors',
                          checked ? 'bg-primary-soft/60' : 'hover:bg-surface-2',
                        )}
                      >
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => toggleTag(tagId)}
                          className="mt-0.5 size-4 shrink-0 cursor-pointer accent-[var(--primary)]"
                        />
                        <span className="min-w-0">
                          <span className="block truncate text-xs font-medium">
                            {tag.tagName || tagId}
                          </span>
                          <span className="block truncate font-mono text-[10px] text-faint">
                            {tagId}
                          </span>
                        </span>
                      </label>
                    </li>
                  )
                })}
              </ul>
            )}
          </div>
        </Card>
      </div>

      <Card className="mt-4">
        <CardHeader>
          <div>
            <CardTitle>{t('timeseries.chartTitle')}</CardTitle>
            <p className="mt-0.5 text-xs text-muted">
              {chartSeries.length > 0
                ? t('timeseries.chartSummary', {
                    curves: chartSeries.length,
                    points: chartData.length,
                  })
                : t('timeseries.chartEmptyHint')}
            </p>
          </div>
          {request && series.isFetching ? (
            <span className="flex items-center gap-1.5 text-[11px] text-faint">
              <span className="size-1.5 animate-pulse rounded-full bg-primary" />
              {t('timeseries.loading')}
            </span>
          ) : null}
        </CardHeader>
        <CardContent>
          {!request ? (
            <EmptyState
              icon={<AreaChart className="size-5" />}
              title={t('timeseries.notQueried.title')}
              description={t('timeseries.notQueried.description')}
            />
          ) : series.isLoading ? (
            <PageLoading label={t('timeseries.querying')} />
          ) : series.isError ? (
            <ErrorState error={series.error} onRetry={series.refetch} compact />
          ) : chartSeries.length === 0 || chartData.length === 0 ? (
            <EmptyState
              title={t('timeseries.noData.title')}
              description={t('timeseries.noData.description')}
            />
          ) : (
            <div className="h-[340px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={chartData} margin={{ top: 8, right: 16, bottom: 4, left: 0 }}>
                  <CartesianGrid stroke="var(--color-border)" strokeDasharray="3 3" vertical={false} />
                  <XAxis
                    dataKey="time"
                    tickFormatter={(value: string) => value.slice(11, 16)}
                    tick={{ fill: 'var(--color-faint)', fontSize: 11 }}
                    tickLine={false}
                    axisLine={{ stroke: 'var(--color-border)' }}
                    minTickGap={28}
                  />
                  <YAxis
                    tick={{ fill: 'var(--color-faint)', fontSize: 11 }}
                    tickLine={false}
                    axisLine={false}
                    width={56}
                    tickFormatter={(value: number) => formatNumber(value, 2)}
                  />
                  <Tooltip
                    contentStyle={{
                      background: 'var(--color-surface)',
                      border: '1px solid var(--color-border)',
                      borderRadius: 12,
                      fontSize: 12,
                      color: 'var(--color-fg)',
                      boxShadow: '0 8px 24px rgba(0,0,0,0.12)',
                    }}
                    labelStyle={{ color: 'var(--color-muted)', marginBottom: 4 }}
                    labelFormatter={(value) => formatDateTimeShort(String(value))}
                    formatter={(value) => formatNumber(Number(value), 3)}
                  />
                  <Legend
                    iconType="plainline"
                    wrapperStyle={{ fontSize: 12, paddingTop: 8 }}
                  />
                  {chartSeries.map((item, index) => (
                    <Line
                      key={item.tagId}
                      type="monotone"
                      dataKey={`s${index}`}
                      name={item.tagName || item.tagId}
                      stroke={SERIES_COLORS[index % SERIES_COLORS.length]}
                      strokeWidth={2}
                      dot={false}
                      activeDot={{ r: 3 }}
                      connectNulls
                      isAnimationActive={false}
                    />
                  ))}
                </LineChart>
              </ResponsiveContainer>
            </div>
          )}
        </CardContent>
      </Card>

      {request && chartSeries.length > 0 ? (
        <Card className="mt-4 overflow-hidden">
          <CardHeader>
            <CardTitle>{t('timeseries.summaryTitle')}</CardTitle>
            <Badge tone="neutral">{source.toUpperCase()}</Badge>
          </CardHeader>
          <Table>
            <THead>
              <tr>
                <TH>{t('timeseries.table.tag')}</TH>
                <TH>ID</TH>
                <TH className="text-right">{t('timeseries.table.dataPoints')}</TH>
                <TH className="text-right">{t('timeseries.table.min')}</TH>
                <TH className="text-right">{t('timeseries.table.max')}</TH>
                <TH className="text-right">{t('timeseries.table.latest')}</TH>
              </tr>
            </THead>
            <TBody>
              {chartSeries.map((item, index) => {
                const summary = summarize(item)
                return (
                  <TR key={item.tagId}>
                    <TD>
                      <span className="flex items-center gap-2">
                        <span
                          className="size-2 shrink-0 rounded-full"
                          style={{ background: SERIES_COLORS[index % SERIES_COLORS.length] }}
                        />
                        <span className="font-medium">{item.tagName || item.tagId}</span>
                      </span>
                      {item.tagDesc ? (
                        <p className="mt-0.5 text-[11px] text-faint">{item.tagDesc}</p>
                      ) : null}
                    </TD>
                    <TD className="font-mono text-xs text-muted">{item.tagId}</TD>
                    <TD className="text-right text-xs tabular">{summary?.count ?? 0}</TD>
                    <TD className="text-right font-mono text-xs tabular">
                      {summary ? formatNumber(summary.min, 3) : '—'}
                    </TD>
                    <TD className="text-right font-mono text-xs tabular">
                      {summary ? formatNumber(summary.max, 3) : '—'}
                    </TD>
                    <TD className="text-right font-mono text-xs tabular">
                      {summary ? formatNumber(summary.latest, 3) : '—'}
                    </TD>
                  </TR>
                )
              })}
            </TBody>
          </Table>
        </Card>
      ) : null}
    </div>
  )
}
