import { useState } from 'react'
import { Search, ScrollText } from 'lucide-react'
import { useAuditLogs } from '@/api/hooks'
import type { AuditLogInfo, AuditQuery } from '@/api/types'
import { useI18n, type MessageKey } from '@harns/core'
import { AUDIT_APPS, AUDIT_METHODS, OPERATION_TYPES } from '@/lib/constants'
import { formatApiTime, formatDateTime, fromDateTimeLocal } from '@harns/core'
import { Badge } from '@harns/core'
import { Button } from '@harns/core'
import { Card } from '@harns/core'
import { CopyButton } from '@harns/core'
import { Dialog } from '@harns/core'
import { EmptyState } from '@harns/core'
import { Field, Input, Select } from '@harns/core'
import { TableSkeleton } from '@harns/core'
import { TBody, TD, TH, THead, TR, Table } from '@harns/core'
import { ErrorState } from '@harns/core'
import { PageHeader } from '@harns/core'
import { Pagination } from '@harns/core'

type FilterKey = 'app' | 'method' | 'resultCode' | 'userId' | 'path' | 'start' | 'end'

const EMPTY_FILTERS: Record<FilterKey, string> = {
  app: '',
  method: '',
  resultCode: '',
  userId: '',
  path: '',
  start: '',
  end: '',
}

const OPERATION_TONE: Record<string, 'success' | 'danger' | 'warning' | 'info' | 'primary' | 'neutral'> = {
  CREATE: 'success',
  DELETE: 'danger',
  UPDATE: 'warning',
  LOGIN: 'info',
  LOGOUT: 'neutral',
  QUERY: 'primary',
}

const OPERATION_LABEL: Record<string, MessageKey> = Object.fromEntries(
  OPERATION_TYPES.map((item) => [item.value, item.labelKey]),
)

export function AuditLogsPage() {
  const { t } = useI18n()
  const [filters, setFilters] = useState<Record<FilterKey, string>>(EMPTY_FILTERS)
  const [query, setQuery] = useState<AuditQuery>({ page: 1, pageSize: 20 })
  const [detail, setDetail] = useState<AuditLogInfo | null>(null)

  const logs = useAuditLogs(query)
  const list = logs.data?.list ?? []
  const total = logs.data?.total ?? 0

  const setFilter = (key: FilterKey, value: string) =>
    setFilters((current) => ({ ...current, [key]: value }))

  const applyFilters = () => {
    const start = filters.start ? fromDateTimeLocal(filters.start) : null
    const end = filters.end ? fromDateTimeLocal(filters.end) : null
    setQuery({
      page: 1,
      pageSize: query.pageSize,
      app: (filters.app as AuditQuery['app']) || undefined,
      method: filters.method || undefined,
      resultCode: filters.resultCode ? Number(filters.resultCode) : undefined,
      userId: filters.userId.trim() || undefined,
      path: filters.path.trim() || undefined,
      start: start ? formatApiTime(start) : undefined,
      end: end ? formatApiTime(end) : undefined,
    })
  }

  const resetFilters = () => {
    setFilters(EMPTY_FILTERS)
    setQuery({ page: 1, pageSize: query.pageSize })
  }

  return (
    <div>
      <PageHeader
        title={t('audit.title')}
        description={t('audit.description')}
      />

      <Card className="mb-4 p-4">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label={t('audit.filter.app')}>
            <Select value={filters.app} onChange={(event) => setFilter('app', event.target.value)}>
              <option value="">{t('common.state.all')}</option>
              {AUDIT_APPS.map((app) => (
                <option key={app} value={app}>
                  {app}
                </option>
              ))}
            </Select>
          </Field>
          <Field label={t('audit.filter.method')}>
            <Select
              value={filters.method}
              onChange={(event) => setFilter('method', event.target.value)}
            >
              <option value="">{t('common.state.all')}</option>
              {AUDIT_METHODS.map((method) => (
                <option key={method} value={method}>
                  {method}
                </option>
              ))}
            </Select>
          </Field>
          <Field label={t('audit.filter.resultCode')}>
            <Input
              type="number"
              value={filters.resultCode}
              placeholder={t('audit.filter.resultCodePlaceholder')}
              onChange={(event) => setFilter('resultCode', event.target.value)}
            />
          </Field>
          <Field label={t('audit.filter.userId')}>
            <Input
              value={filters.userId}
              placeholder={t('audit.filter.userIdPlaceholder')}
              onChange={(event) => setFilter('userId', event.target.value)}
            />
          </Field>
          <Field label={t('audit.filter.path')} className="sm:col-span-2">
            <div className="relative">
              <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
              <Input
                value={filters.path}
                placeholder={t('audit.filter.pathPlaceholder')}
                className="pl-9"
                onChange={(event) => setFilter('path', event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') applyFilters()
                }}
              />
            </div>
          </Field>
          <Field label={t('audit.filter.start')}>
            <Input
              type="datetime-local"
              value={filters.start}
              onChange={(event) => setFilter('start', event.target.value)}
            />
          </Field>
          <Field label={t('audit.filter.end')}>
            <Input
              type="datetime-local"
              value={filters.end}
              onChange={(event) => setFilter('end', event.target.value)}
            />
          </Field>
        </div>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="ghost" onClick={resetFilters}>
            {t('common.action.reset')}
          </Button>
          <Button
            variant="primary"
            icon={<Search className="size-4" />}
            onClick={applyFilters}
          >
            {t('common.action.query')}
          </Button>
        </div>
      </Card>

      <Card className="overflow-hidden">
        {logs.isLoading ? (
          <TableSkeleton rows={8} cols={6} />
        ) : logs.isError ? (
          <div className="p-5">
            <ErrorState error={logs.error} onRetry={logs.refetch} compact />
          </div>
        ) : list.length === 0 ? (
          <EmptyState
            icon={<ScrollText className="size-5" />}
            title={t('audit.empty.title')}
            description={t('audit.empty.description')}
          />
        ) : (
          <>
            <Table>
              <THead>
                <tr>
                  <TH>{t('audit.table.time')}</TH>
                  <TH>{t('audit.table.user')}</TH>
                  <TH>{t('audit.table.operation')}</TH>
                  <TH>{t('audit.table.request')}</TH>
                  <TH>{t('audit.table.result')}</TH>
                  <TH>{t('audit.table.duration')}</TH>
                  <TH>IP</TH>
                </tr>
              </THead>
              <TBody>
                {list.map((log) => (
                  <TR key={log.id} className="cursor-pointer" onClick={() => setDetail(log)}>
                    <TD className="text-xs whitespace-nowrap text-muted">
                      {formatDateTime(log.createdTime)}
                    </TD>
                    <TD className="text-xs">
                      <p className="font-medium">{log.userName || '—'}</p>
                      <p className="font-mono text-[10px] text-faint">{log.userId}</p>
                    </TD>
                    <TD>
                      <div className="flex flex-col gap-1">
                        <Badge tone={OPERATION_TONE[log.operationType] ?? 'neutral'}>
                          {OPERATION_LABEL[log.operationType]
                            ? t(OPERATION_LABEL[log.operationType])
                            : log.operationType}
                        </Badge>
                        <span className="max-w-40 truncate text-[10px] text-faint">
                          {log.operationDesc || '—'}
                        </span>
                      </div>
                    </TD>
                    <TD className="max-w-72">
                      <p className="font-mono text-[11px]">
                        {log.method} {log.path}
                      </p>
                      {log.query ? (
                        <p className="truncate font-mono text-[10px] text-faint" title={log.query}>
                          ?{log.query}
                        </p>
                      ) : null}
                    </TD>
                    <TD>
                      {log.status >= 200 && log.status < 300 ? (
                        <Badge tone="success">{log.status}</Badge>
                      ) : (
                        <Badge tone="danger">{log.status}</Badge>
                      )}
                    </TD>
                    <TD className="text-xs text-muted tabular">{log.durationMs} ms</TD>
                    <TD className="font-mono text-[11px] text-muted">{log.ip}</TD>
                  </TR>
                ))}
              </TBody>
            </Table>
            <Pagination
              page={query.page}
              pageSize={query.pageSize}
              total={total}
              onPageChange={(page) => setQuery((current) => ({ ...current, page }))}
              onPageSizeChange={(pageSize) =>
                setQuery((current) => ({ ...current, page: 1, pageSize }))
              }
              disabled={logs.isFetching}
            />
          </>
        )}
      </Card>

      <Dialog
        open={Boolean(detail)}
        onClose={() => setDetail(null)}
        size="xl"
        title={t('audit.detail.title')}
        description={detail ? `${detail.method} ${detail.path}` : undefined}
      >
        {detail ? (
          <div className="max-h-[68vh] space-y-3 overflow-y-auto pr-1 text-xs">
            <div className="grid gap-2 sm:grid-cols-2">
              <DetailRow label={t('audit.detail.app')} value={detail.app} />
              <DetailRow
                label={t('audit.detail.operation')}
                value={`${detail.operationType} · ${detail.operationDesc}`}
              />
              <DetailRow
                label={t('audit.detail.user')}
                value={`${detail.userName || '—'} (${detail.userId})`}
              />
              <DetailRow label={t('audit.detail.time')} value={formatDateTime(detail.createdTime)} />
              <DetailRow label={t('audit.detail.status')} value={String(detail.status)} />
              <DetailRow
                label={t('audit.detail.businessCode')}
                value={String(detail.resultCode)}
              />
              <DetailRow label={t('audit.detail.duration')} value={`${detail.durationMs} ms`} />
              <DetailRow label="IP" value={detail.ip} />
              <DetailRow label={t('audit.detail.traceId')} value={detail.traceId} mono />
              <DetailRow label="User-Agent" value={detail.userAgent} />
            </div>
            {detail.errorMsg ? (
              <div className="rounded-lg border border-danger/25 bg-danger/5 px-3 py-2 text-danger">
                {detail.errorMsg}
              </div>
            ) : null}
            <BodyBlock label={t('audit.detail.query')} value={detail.query} />
            <BodyBlock label={t('audit.detail.requestBody')} value={detail.requestBody ?? ''} />
            <BodyBlock label={t('audit.detail.responseBody')} value={detail.responseBody ?? ''} />
          </div>
        ) : null}
      </Dialog>
    </div>
  )
}

function DetailRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-start justify-between gap-3 rounded-lg bg-surface-2/50 px-3 py-1.5">
      <span className="shrink-0 text-faint">{label}</span>
      <span className={mono ? 'min-w-0 truncate font-mono text-[11px]' : 'min-w-0 text-right'}>
        {value || '—'}
      </span>
    </div>
  )
}

function BodyBlock({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="mb-1 flex items-center justify-between">
        <p className="font-medium text-muted">{label}</p>
        {value ? <CopyButton value={value} /> : null}
      </div>
      <pre className="max-h-52 overflow-auto rounded-lg border border-border bg-surface-2/60 p-3 font-mono text-[11px] leading-relaxed whitespace-pre-wrap break-all">
        {value || '—'}
      </pre>
    </div>
  )
}
