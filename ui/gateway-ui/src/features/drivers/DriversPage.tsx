import { useMemo, useState } from 'react'
import { Layers, Plus, Search } from 'lucide-react'
import { toast } from 'sonner'
import {
  Badge,
  Button,
  Card,
  ConfirmDialog,
  EmptyState,
  ErrorState,
  Input,
  Pagination,
  TableSkeleton,
  formatDateTime,
  showErrorToast,
  useI18n,
} from '@harns/core'
import { useDeleteDriver, useDriverList } from '@/api/hooks'
import type { Driver } from '@/api/types'
import {
  GatewayTBody,
  GatewayTD,
  GatewayTH,
  GatewayTHead,
  GatewayTR,
  GatewayTable,
} from '@/components/GatewayTable'
import { ColumnResizeHandle, useColumnResize } from '@/components/useColumnResize'
import { driverModelLabel, driverTypeLabel } from '@/lib/constants'
import { DriverFormDialog } from './DriverFormDialog'
import { DriverPluginFileDialog } from './DriverPluginFileDialog'
import { DriverSchemaDialog } from './DriverSchemaDialog'

export function DriversPage() {
  const { t } = useI18n()
  const drivers = useDriverList()
  const remove = useDeleteDriver()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [schemaTarget, setSchemaTarget] = useState<Driver | null>(null)
  const [pluginTarget, setPluginTarget] = useState<Driver | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Driver | null>(null)
  const { widths, totalWidth, resize } = useColumnResize({
    driverType: 140,
    model: 240,
    source: 100,
    plugin: 220,
    createdAt: 160,
    actions: 240,
  })

  const filteredDrivers = useMemo(() => {
    const query = keyword.trim().toLowerCase()
    const items = drivers.data ?? []
    const matched = query
      ? items.filter((driver) =>
          [driver.driverType, driver.model, driver.pluginName]
            .join(' ')
            .toLowerCase()
            .includes(query),
        )
      : items
    return [...matched].sort(
      (left, right) =>
        left.driverType.localeCompare(right.driverType) ||
        left.model.localeCompare(right.model),
    )
  }, [drivers.data, keyword])

  const total = filteredDrivers.length
  const pageDrivers = filteredDrivers.slice((page - 1) * pageSize, page * pageSize)
  const driverTypeSpans = new Map<number, number>()
  let groupStart = 0
  for (let index = 1; index <= pageDrivers.length; index += 1) {
    if (
      index === pageDrivers.length ||
      pageDrivers[index].driverType !== pageDrivers[groupStart].driverType
    ) {
      driverTypeSpans.set(groupStart, index - groupStart)
      groupStart = index
    }
  }

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    setPage(1)
  }

  const resetSearch = () => {
    setKeywordInput('')
    setKeyword('')
    setPage(1)
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await remove.mutateAsync({
        id: deleteTarget.id,
        title: `${deleteTarget.driverType}/${deleteTarget.model}`,
      })
      toast.success(t('gateway.drivers.toast.deleted'))
      setDeleteTarget(null)
    } catch (error) {
      showErrorToast(error)
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="mb-3 flex shrink-0 flex-wrap items-center gap-2">
        <div className="relative w-[200px]">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
          <Input
            value={keywordInput}
            placeholder={t('gateway.drivers.searchPlaceholder')}
            className="h-9 pl-9 text-sm"
            onChange={(event) => setKeywordInput(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') applySearch()
            }}
          />
        </div>
        <Button variant="primary" className="h-9" onClick={applySearch}>
          {t('common.action.search')}
        </Button>
        <Button variant="ghost" className="h-9" onClick={resetSearch}>
          {t('common.action.reset')}
        </Button>
        <Button
          variant="primary"
          className="ml-auto h-9"
          icon={<Plus className="size-3.5" />}
          onClick={() => setCreateOpen(true)}
        >
          {t('gateway.drivers.create')}
        </Button>
      </div>

      <Card className="flex min-h-0 flex-1 flex-col overflow-hidden">
        <div className="min-h-0 flex-1 overflow-auto">
          {drivers.isLoading ? (
            <TableSkeleton rows={6} cols={6} />
          ) : drivers.isError ? (
            <div className="p-5">
              <ErrorState error={drivers.error} onRetry={() => void drivers.refetch()} compact />
            </div>
          ) : total === 0 ? (
            <EmptyState
              icon={<Layers className="size-5" />}
              title={
                keyword
                  ? t('gateway.drivers.empty.filteredTitle')
                  : t('gateway.drivers.empty.title')
              }
              description={t('gateway.drivers.empty.description')}
              action={
                keyword ? (
                  <Button variant="outline" onClick={resetSearch}>
                    {t('common.action.reset')}
                  </Button>
                ) : (
                  <Button
                    variant="primary"
                    onClick={() => setCreateOpen(true)}
                    icon={<Plus className="size-3.5" />}
                  >
                    {t('gateway.drivers.create')}
                  </Button>
                )
              }
            />
          ) : (
            <GatewayTable
              style={{
                tableLayout: 'fixed',
                width: `max(100%, ${totalWidth}px)`,
              }}
            >
              <colgroup>
                <col style={{ width: widths.driverType }} />
                <col style={{ width: widths.model }} />
                <col style={{ width: widths.source }} />
                <col style={{ width: widths.plugin }} />
                <col style={{ width: widths.createdAt }} />
                <col style={{ width: widths.actions }} />
              </colgroup>
              <GatewayTHead>
                <tr>
                  <GatewayTH>
                    {t('gateway.drivers.columns.driverType')}
                    <ColumnResizeHandle
                      column="driverType"
                      width={widths.driverType}
                      onResize={resize}
                    />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.drivers.columns.model')}
                    <ColumnResizeHandle column="model" width={widths.model} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.drivers.columns.source')}
                    <ColumnResizeHandle column="source" width={widths.source} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.drivers.columns.plugin')}
                    <ColumnResizeHandle column="plugin" width={widths.plugin} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.drivers.columns.createdAt')}
                    <ColumnResizeHandle
                      column="createdAt"
                      width={widths.createdAt}
                      onResize={resize}
                    />
                  </GatewayTH>
                  <GatewayTH className="text-left">
                    {t('common.field.actions')}
                    <ColumnResizeHandle
                      column="actions"
                      width={widths.actions}
                      onResize={resize}
                    />
                  </GatewayTH>
                </tr>
              </GatewayTHead>
              <GatewayTBody>
                {pageDrivers.map((driver, index) => (
                  <GatewayTR key={driver.id}>
                    {driverTypeSpans.has(index) ? (
                      <GatewayTD rowSpan={driverTypeSpans.get(index)}>
                        <span className="block truncate text-fg">
                          {driverTypeLabel(driver.driverType) || driver.driverType}
                        </span>
                      </GatewayTD>
                    ) : null}
                    <GatewayTD>
                      <span className="block truncate text-fg">
                        {driverModelLabel(driver.model) || driver.model}
                      </span>
                    </GatewayTD>
                    <GatewayTD>
                      {driver.builtinDriver ? (
                        <Badge tone="info">{t('gateway.drivers.builtin')}</Badge>
                      ) : (
                        <Badge tone="violet">{t('gateway.drivers.custom')}</Badge>
                      )}
                    </GatewayTD>
                    <GatewayTD>
                      <span className="block truncate font-mono text-muted">
                        {driver.pluginName || '—'}
                      </span>
                    </GatewayTD>
                    <GatewayTD className="whitespace-nowrap text-muted">
                      {formatDateTime(driver.createdTime)}
                    </GatewayTD>
                    <GatewayTD>
                      <div className="flex items-center gap-2">
                        <button
                          type="button"
                          onClick={() => setSchemaTarget(driver)}
                          className="text-sm text-primary transition-colors hover:text-primary-hover"
                        >
                          {t('gateway.drivers.viewSchema')}
                        </button>
                        {!driver.builtinDriver ? (
                          <>
                            <button
                              type="button"
                              onClick={() => setPluginTarget(driver)}
                              className="text-sm text-primary transition-colors hover:text-primary-hover"
                            >
                              {t('gateway.drivers.uploadPlugin')}
                            </button>
                            <button
                              type="button"
                              onClick={() => setDeleteTarget(driver)}
                              className="text-sm text-danger transition-colors hover:opacity-80"
                            >
                              {t('common.action.delete')}
                            </button>
                          </>
                        ) : null}
                      </div>
                    </GatewayTD>
                  </GatewayTR>
                ))}
              </GatewayTBody>
            </GatewayTable>
          )}
        </div>
        <div className="shrink-0">
          <Pagination
            page={page}
            pageSize={pageSize}
            total={total}
            onPageChange={setPage}
            onPageSizeChange={(size) => {
              setPageSize(size)
              setPage(1)
            }}
            disabled={drivers.isFetching}
          />
        </div>
      </Card>

      <DriverFormDialog open={createOpen} onClose={() => setCreateOpen(false)} />
      <DriverPluginFileDialog driver={pluginTarget} onClose={() => setPluginTarget(null)} />
      <DriverSchemaDialog driver={schemaTarget} onClose={() => setSchemaTarget(null)} />
      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => void confirmDelete()}
        loading={remove.isPending}
        danger
        title={t('gateway.drivers.deleteTitle')}
        description={t('gateway.drivers.deleteDescription', {
          name: deleteTarget ? `${deleteTarget.driverType}/${deleteTarget.model}` : '',
        })}
        confirmText={t('common.action.delete')}
      />
    </div>
  )
}
