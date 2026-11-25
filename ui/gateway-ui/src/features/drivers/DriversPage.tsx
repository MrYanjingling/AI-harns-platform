import { useMemo, useState } from 'react'
import { FileJson, Plus, Trash2, Upload } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  ConfirmDialog,
  EnumBadge,
  ErrorState,
  ListTableCard,
  SearchInput,
  Pagination,
  TableSkeleton,
  formatDateTime,
  showErrorToast,
  useI18n,
} from '@harns/core'
import { TableActions } from '@harns/core'
import { useDeleteDriver, useDriverList } from '@/api/hooks'
import type { Driver } from '@/api/types'
import {
  DataTableBody,
  DataTableCell,
  DataTableHeadCell,
  DataTableHead,
  DataTableRow,
  DataTable,
} from '@harns/core'
import { ColumnResizeHandle, useColumnResize } from '@harns/core'
import { TableActionButton } from '@harns/core'
import { TruncatedCell } from '@/components/TruncatedCell'
import { driverModelLabel, driverTypeLabel } from '@/lib/constants'
import { DriverFormDialog } from './DriverFormDialog'
import { DriverPluginFileDialog } from './DriverPluginFileDialog'
import { DriverSchemaDialog } from './DriverSchemaDialog'
import {
  listViewKeys,
  useListViewRestore,
  useListPagination,
  useListPaginationClamp,
  usePersistedState,
} from '@/lib/list-view-state'

export function DriversPage() {
  const { t } = useI18n()
  const restore = useListViewRestore()
  const drivers = useDriverList()
  const remove = useDeleteDriver()
  const { page, pageSize, setPage, setPageSize } = useListPagination(
    listViewKeys.drivers,
    restore,
  )
  const [keywordInput, setKeywordInput] = usePersistedState(
    `${listViewKeys.drivers}:keywordInput`,
    '',
    restore,
  )
  const [keyword, setKeyword] = usePersistedState(`${listViewKeys.drivers}:keyword`, '', restore)
  const [createOpen, setCreateOpen] = useState(false)
  const [schemaTarget, setSchemaTarget] = useState<Driver | null>(null)
  const [pluginTarget, setPluginTarget] = useState<Driver | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Driver | null>(null)
  const { widths, totalWidth, resize } = useColumnResize({
    driverType: 140,
    model: 220,
    source: 90,
    plugin: 200,
    remark: 280,
    createdAt: 160,
    actions: 130,
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
  useListPaginationClamp(page, pageSize, total, setPage)
  const pageDrivers = filteredDrivers.slice((page - 1) * pageSize, page * pageSize)

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    setPage(1)
    void drivers.refetch()
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
      <div className="app-page-toolbar">
        <SearchInput
          wrapperClassName="w-[200px]"
          value={keywordInput}
          placeholder={t('gateway.drivers.searchPlaceholder')}
          onChange={(event) => setKeywordInput(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') applySearch()
          }}
        />
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

      <ListTableCard
        pagination={
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
        }
      >
          {drivers.isLoading ? (
            <TableSkeleton rows={6} cols={7} />
          ) : drivers.isError ? (
            <div className="p-5">
              <ErrorState error={drivers.error} onRetry={() => void drivers.refetch()} compact />
            </div>
          ) : total === 0 ? null : (
            <DataTable
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
                <col style={{ width: widths.remark }} />
                <col style={{ width: widths.createdAt }} />
                <col style={{ width: widths.actions }} />
              </colgroup>
              <DataTableHead>
                <tr>
                  <DataTableHeadCell>
                    {t('gateway.drivers.columns.driverType')}
                    <ColumnResizeHandle
                      column="driverType"
                      width={widths.driverType}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.drivers.columns.model')}
                    <ColumnResizeHandle column="model" width={widths.model} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.drivers.columns.source')}
                    <ColumnResizeHandle column="source" width={widths.source} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.drivers.columns.plugin')}
                    <ColumnResizeHandle column="plugin" width={widths.plugin} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.drivers.columns.remark')}
                    <ColumnResizeHandle
                      column="remark"
                      width={widths.remark}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.drivers.columns.createdAt')}
                    <ColumnResizeHandle
                      column="createdAt"
                      width={widths.createdAt}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell className="text-left">
                    {t('common.field.actions')}
                    <ColumnResizeHandle
                      column="actions"
                      width={widths.actions}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                </tr>
              </DataTableHead>
              <DataTableBody>
                {pageDrivers.map((driver) => (
                  <DataTableRow key={driver.id}>
                    <DataTableCell>
                      <TruncatedCell
                        text={driverTypeLabel(driver.driverType) || driver.driverType}
                      />
                    </DataTableCell>
                    <DataTableCell>
                      <TruncatedCell text={driverModelLabel(driver.model) || driver.model} />
                    </DataTableCell>
                    <DataTableCell>
                      {driver.builtinDriver ? (
                        <EnumBadge tone="info">{t('gateway.drivers.builtin')}</EnumBadge>
                      ) : (
                        <EnumBadge tone="violet">{t('gateway.drivers.custom')}</EnumBadge>
                      )}
                    </DataTableCell>
                    <DataTableCell>
                      <TruncatedCell text={driver.pluginName} />
                    </DataTableCell>
                    <DataTableCell>
                      <TruncatedCell text={driver.remark} />
                    </DataTableCell>
                    <DataTableCell className="whitespace-nowrap">
                      {formatDateTime(driver.createdTime)}
                    </DataTableCell>
                    <DataTableCell>
                      <TableActions>
                        <TableActionButton
                          label={t('gateway.drivers.viewSchema')}
                          icon={<FileJson />}
                          onClick={() => setSchemaTarget(driver)}
                        />
                        {!driver.builtinDriver ? (
                          <>
                            <TableActionButton
                              label={t('gateway.drivers.uploadPlugin')}
                              icon={<Upload />}
                              onClick={() => setPluginTarget(driver)}
                            />
                            <TableActionButton
                              label={t('common.action.delete')}
                              icon={<Trash2 />}
                              onClick={() => setDeleteTarget(driver)}
                            />
                          </>
                        ) : null}
                      </TableActions>
                    </DataTableCell>
                  </DataTableRow>
                ))}
              </DataTableBody>
            </DataTable>
          )}
      </ListTableCard>

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
