import { useState } from 'react'
import { Search, ScrollText } from 'lucide-react'
import { useAuditLogs } from '@/api/hooks'
import type { AuditLogInfo, AuditQuery } from '@/api/types'
import { AUDIT_APPS, AUDIT_METHODS } from '@/lib/constants'
import { formatApiTime, formatDateTime, fromDateTimeLocal } from '@/lib/format'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { CopyButton } from '@/components/ui/CopyButton'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { Field, Input, Select } from '@/components/ui/Field'
import { TableSkeleton } from '@/components/ui/Skeleton'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'
import { ErrorState } from '@/components/shared/ErrorState'
import { PageHeader } from '@/components/shared/PageHeader'
import { Pagination } from '@/components/shared/Pagination'

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

export function AuditLogsPage() {
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
        title="审计日志"
        description="仅记录写操作(POST / PUT / PATCH / DELETE)。请求与响应体会脱敏并按 16 KiB 截断。"
      />

      <Card className="mb-4 p-4">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="应用">
            <Select value={filters.app} onChange={(event) => setFilter('app', event.target.value)}>
              <option value="">全部</option>
              {AUDIT_APPS.map((app) => (
                <option key={app} value={app}>
                  {app}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="请求方法">
            <Select
              value={filters.method}
              onChange={(event) => setFilter('method', event.target.value)}
            >
              <option value="">全部</option>
              {AUDIT_METHODS.map((method) => (
                <option key={method} value={method}>
                  {method}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="结果码">
            <Input
              type="number"
              value={filters.resultCode}
              placeholder="例如 200"
              onChange={(event) => setFilter('resultCode', event.target.value)}
            />
          </Field>
          <Field label="用户 ID">
            <Input
              value={filters.userId}
              placeholder="用户 ID"
              onChange={(event) => setFilter('userId', event.target.value)}
            />
          </Field>
          <Field label="路径" className="sm:col-span-2">
            <div className="relative">
              <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
              <Input
                value={filters.path}
                placeholder="模糊匹配,例如 /iot/things"
                className="pl-9"
                onChange={(event) => setFilter('path', event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') applyFilters()
                }}
              />
            </div>
          </Field>
          <Field label="开始时间">
            <Input
              type="datetime-local"
              value={filters.start}
              onChange={(event) => setFilter('start', event.target.value)}
            />
          </Field>
          <Field label="结束时间">
            <Input
              type="datetime-local"
              value={filters.end}
              onChange={(event) => setFilter('end', event.target.value)}
            />
          </Field>
        </div>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="ghost" onClick={resetFilters}>
            重置
          </Button>
          <Button
            variant="primary"
            icon={<Search className="size-4" />}
            onClick={applyFilters}
          >
            查询
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
            title="没有审计记录"
            description="调整筛选条件,或等待写操作产生日志。"
          />
        ) : (
          <>
            <Table>
              <THead>
                <tr>
                  <TH>时间</TH>
                  <TH>用户</TH>
                  <TH>操作</TH>
                  <TH>请求</TH>
                  <TH>结果</TH>
                  <TH>耗时</TH>
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
                          {log.operationType}
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
        title="审计详情"
        description={detail ? `${detail.method} ${detail.path}` : undefined}
      >
        {detail ? (
          <div className="max-h-[68vh] space-y-3 overflow-y-auto pr-1 text-xs">
            <div className="grid gap-2 sm:grid-cols-2">
              <DetailRow label="应用" value={detail.app} />
              <DetailRow label="操作" value={`${detail.operationType} · ${detail.operationDesc}`} />
              <DetailRow label="用户" value={`${detail.userName || '—'} (${detail.userId})`} />
              <DetailRow label="时间" value={formatDateTime(detail.createdTime)} />
              <DetailRow label="状态" value={String(detail.status)} />
              <DetailRow label="业务码" value={String(detail.resultCode)} />
              <DetailRow label="耗时" value={`${detail.durationMs} ms`} />
              <DetailRow label="IP" value={detail.ip} />
              <DetailRow label="Trace ID" value={detail.traceId} mono />
              <DetailRow label="User-Agent" value={detail.userAgent} />
            </div>
            {detail.errorMsg ? (
              <div className="rounded-lg border border-danger/25 bg-danger/5 px-3 py-2 text-danger">
                {detail.errorMsg}
              </div>
            ) : null}
            <BodyBlock label="Query" value={detail.query} />
            <BodyBlock label="请求体" value={detail.requestBody ?? ''} />
            <BodyBlock label="响应体" value={detail.responseBody ?? ''} />
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
