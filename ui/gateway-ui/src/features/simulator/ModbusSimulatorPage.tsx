import { useEffect, useRef, useState } from 'react'
import { Plus, RefreshCw, Trash2 } from 'lucide-react'
import {
  Badge,
  Button,
  ColumnResizeHandle,
  DataTable,
  DataTableBody,
  DataTableCell,
  DataTableHead,
  DataTableHeadCell,
  DataTableRow,
  ErrorState,
  Input,
  ListTableCard,
  Pagination,
  SearchSelect,
  TableActions,
  TableActionButton,
  TableSkeleton,
  showErrorToast,
  useColumnResize,
  useI18n,
} from '@harns/core'
import { toast } from 'sonner'
import {
  useCloseSimulator,
  useOpenSimulator,
  useSaveSimulatorPoints,
  useSimulator,
} from '@/api/hooks'
import type { SimulatorDataType, SimulatorMode } from '@/api/types'

const modes: SimulatorMode[] = ['random', 'increment', 'decrement']
const dataTypes: SimulatorDataType[] = ['int16', 'int32', 'float']

interface PointRow {
  key: string
  name: string
  address: string
  dataType: SimulatorDataType
  description: string
  mode: SimulatorMode
  saved: boolean
  dirty: boolean
}

function dataTypeWords(dataType: SimulatorDataType): number {
  return dataType === 'int16' ? 1 : 2
}

function nextAddress(rows: PointRow[]): string {
  const end = rows.reduce(
    (value, row) => Math.max(value, (Number(row.address) || 0) + dataTypeWords(row.dataType) - 1),
    0,
  )
  return String(end + 1)
}

function recalculateDraftAddresses(rows: PointRow[]): PointRow[] {
  const savedEnd = rows.reduce(
    (value, row) =>
      row.saved
        ? Math.max(value, (Number(row.address) || 0) + dataTypeWords(row.dataType) - 1)
        : value,
    0,
  )
  let next = savedEnd + 1
  return rows.map((row) => {
    if (row.saved) return row
    const address = String(next)
    next += dataTypeWords(row.dataType)
    return { ...row, address }
  })
}

export function ModbusSimulatorPage() {
  const { t } = useI18n()
  const simulator = useSimulator()
  const openSimulator = useOpenSimulator()
  const closeSimulator = useCloseSimulator()
  const savePoints = useSaveSimulatorPoints()

  const state = simulator.data
  const running = state?.running ?? false
  const [port, setPort] = useState('2502')
  const [rows, setRows] = useState<PointRow[]>([])
  const [deletedAddresses, setDeletedAddresses] = useState<number[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const draftId = useRef(0)
  const { widths, totalWidth, resize } = useColumnResize({
    order: 64,
    name: 160,
    address: 120,
    dataType: 140,
    registerType: 160,
    mode: 140,
    actions: 140,
  })

  const newDraftRow = (address: string, dirty = true): PointRow => {
    draftId.current += 1
    return {
      key: `draft-${draftId.current}`,
      name: '',
      address,
      dataType: 'int16',
      description: '',
      mode: 'random',
      saved: false,
      dirty,
    }
  }

  useEffect(() => {
    if (state?.port) setPort(String(state.port))
  }, [state?.port])

  useEffect(() => {
    const points = state?.points ?? []
    setRows(
      points.length > 0
        ? points.map((point) => ({
            key: `saved-${point.address}`,
            name: point.name,
            address: String(point.address),
            dataType: point.dataType,
            description: point.description,
            mode: point.mode,
            saved: true,
            dirty: false,
          }))
        : [newDraftRow('1')],
    )
    setDeletedAddresses([])
    setPage(1)
  }, [state?.points])

  const open = async () => {
    const value = Number(port)
    if (!Number.isInteger(value) || value < 1 || value > 65535) {
      showErrorToast(new Error(t('gateway.simulator.validation.port')))
      return
    }
    try {
      await openSimulator.mutateAsync(value)
      toast.success(t('gateway.simulator.toast.opened'))
    } catch (err) {
      showErrorToast(err)
    }
  }

  const close = async () => {
    try {
      await closeSimulator.mutateAsync()
      toast.success(t('gateway.simulator.toast.closed'))
    } catch (err) {
      showErrorToast(err)
    }
  }

  const addRow = () => {
    setRows((current) => [...current, newDraftRow(nextAddress(current), true)])
    setPage(Math.max(1, Math.ceil((rows.length + 1) / pageSize)))
  }

  const changeRow = (key: string, patch: Partial<PointRow>) => {
    setRows((current) => {
      const next = current.map((row) => {
        if (row.key !== key) return row
        return { ...row, ...patch, dirty: true }
      })
      return patch.dataType ? recalculateDraftAddresses(next) : next
    })
  }

  const removeRow = (row: PointRow) => {
    if (row.saved) {
      setDeletedAddresses((current) =>
        current.includes(Number(row.address)) ? current : [...current, Number(row.address)],
      )
      setRows((current) =>
        recalculateDraftAddresses(current.filter((item) => item.key !== row.key)),
      )
      return
    }
    setRows((current) => {
      const next = current.filter((item) => item.key !== row.key)
      return recalculateDraftAddresses(next.length > 0 ? next : [newDraftRow('1')])
    })
  }

  const activeRows = rows.filter((row) => row.saved || row.dirty)

  const validateRows = (): boolean => {
    const ranges: [number, number][] = []
    for (const row of activeRows) {
      const address = Number(row.address)
      const end = address + dataTypeWords(row.dataType) - 1
      if (!Number.isInteger(address) || address < 1 || end > 65535) {
        showErrorToast(new Error(t('gateway.simulator.validation.address')))
        return false
      }
      if (ranges.some(([start, finish]) => address <= finish && end >= start)) {
        showErrorToast(new Error(t('gateway.simulator.validation.duplicateAddress')))
        return false
      }
      ranges.push([address, end])
    }
    return true
  }

  const dirty = deletedAddresses.length > 0 || rows.some((row) => !row.saved && row.dirty)
  const pageCount = Math.max(1, Math.ceil(rows.length / pageSize))
  const safePage = Math.min(page, pageCount)
  const pageStart = (safePage - 1) * pageSize
  const pageRows = rows.slice(pageStart, pageStart + pageSize)

  const save = async () => {
    if (!dirty || !validateRows()) return
    const created = activeRows
      .filter((row) => !row.saved)
      .map((row) => ({
        address: Number(row.address),
        name: row.name.trim(),
        description: row.description.trim(),
        dataType: row.dataType,
        mode: row.mode,
      }))
    try {
      await savePoints.mutateAsync({ created, deleted: deletedAddresses })
      toast.success(t('gateway.simulator.toast.pointsSaved'))
      setDeletedAddresses([])
      await simulator.refetch()
    } catch (err) {
      showErrorToast(err)
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="app-page-toolbar">
        <label className="flex items-center gap-2 text-sm font-medium text-muted">
          {t('gateway.simulator.port')}
          <Input
            type="number"
            min={1}
            max={65535}
            value={port}
            disabled={running}
            className="h-9 w-[120px]"
            onChange={(event) => setPort(event.target.value)}
          />
        </label>
        {running ? (
          <Button variant="outline" className="h-9" loading={closeSimulator.isPending} onClick={() => void close()}>
            {t('gateway.simulator.close')}
          </Button>
        ) : (
          <Button variant="primary" className="h-9" loading={openSimulator.isPending} onClick={() => void open()}>
            {t('gateway.simulator.open')}
          </Button>
        )}
        <Badge tone={running ? 'success' : 'neutral'}>
          {running ? t('gateway.simulator.status.running') : t('gateway.simulator.status.stopped')}
        </Badge>
        <Button
          variant="ghost"
          className="h-9"
          icon={<RefreshCw className="size-3.5" />}
          onClick={() => void simulator.refetch()}
        >
          {t('gateway.simulator.refresh')}
        </Button>
      </div>

      <ListTableCard
        pagination={
          <Pagination
            page={safePage}
            pageSize={pageSize}
            total={rows.length}
            onPageChange={setPage}
            onPageSizeChange={(size) => {
              setPageSize(size)
              setPage(1)
            }}
            disabled={simulator.isFetching || savePoints.isPending}
            actions={
              <Button
                size="sm"
                variant="primary"
                loading={savePoints.isPending}
                disabled={!dirty || simulator.isLoading}
                onClick={() => void save()}
              >
                {t('common.action.save')}
              </Button>
            }
          />
        }
      >
        {simulator.isLoading ? (
          <TableSkeleton rows={6} cols={7} />
        ) : simulator.isError ? (
          <div className="p-5">
            <ErrorState error={simulator.error} onRetry={() => void simulator.refetch()} compact />
          </div>
        ) : (
          <DataTable
            style={{
              tableLayout: 'fixed',
              width: `max(100%, ${totalWidth}px)`,
            }}
          >
            <colgroup>
              <col style={{ width: widths.order }} />
              <col style={{ width: widths.name }} />
              <col style={{ width: widths.address }} />
              <col style={{ width: widths.dataType }} />
              <col style={{ width: widths.registerType }} />
              <col style={{ width: widths.mode }} />
              <col style={{ width: widths.actions }} />
            </colgroup>
            <DataTableHead>
              <tr>
                <DataTableHeadCell className="text-center">
                  {t('common.field.order')}
                  <ColumnResizeHandle column="order" width={widths.order} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.simulator.name')}
                  <ColumnResizeHandle column="name" width={widths.name} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.simulator.address')}
                  <ColumnResizeHandle column="address" width={widths.address} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.simulator.dataType')}
                  <ColumnResizeHandle
                    column="dataType"
                    width={widths.dataType}
                    onResize={resize}
                  />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.simulator.registerType')}
                  <ColumnResizeHandle
                    column="registerType"
                    width={widths.registerType}
                    onResize={resize}
                  />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.simulator.mode')}
                  <ColumnResizeHandle column="mode" width={widths.mode} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('common.field.actions')}
                  <ColumnResizeHandle column="actions" width={widths.actions} onResize={resize} />
                </DataTableHeadCell>
              </tr>
            </DataTableHead>
            <DataTableBody>
              {pageRows.map((row, index) => (
                <DataTableRow key={row.key}>
                  <DataTableCell className="text-center">{pageStart + index + 1}</DataTableCell>
                  <DataTableCell>
                    {row.saved ? (
                      row.name || '—'
                    ) : (
                      <Input
                        value={row.name}
                        aria-label={t('gateway.simulator.name')}
                        onChange={(event) => changeRow(row.key, { name: event.target.value })}
                      />
                    )}
                  </DataTableCell>
                  <DataTableCell>
                    {row.saved ? (
                      row.address
                    ) : (
                      <Input
                        type="number"
                        min={1}
                        max={65535}
                        value={row.address}
                        aria-label={t('gateway.simulator.address')}
                        onChange={(event) => changeRow(row.key, { address: event.target.value })}
                      />
                    )}
                  </DataTableCell>
                  <DataTableCell>
                    {row.saved ? (
                      row.dataType
                    ) : (
                      <SearchSelect
                        value={row.dataType}
                        options={dataTypes.map((item) => ({ value: item, label: item }))}
                        onChange={(dataType) =>
                          changeRow(row.key, {
                            dataType: dataType as SimulatorDataType,
                          })
                        }
                        ariaLabel={t('gateway.simulator.dataType')}
                        emptyText={t('common.state.empty')}
                        searchable={false}
                        className="w-full"
                      />
                    )}
                  </DataTableCell>
                  <DataTableCell>
                    {t('gateway.simulator.registerType.holding')}
                  </DataTableCell>
                  <DataTableCell>
                    {row.saved ? (
                      t(`gateway.simulator.mode.${row.mode}`)
                    ) : (
                      <SearchSelect
                        value={row.mode}
                        options={modes.map((item) => ({
                          value: item,
                          label: t(`gateway.simulator.mode.${item}`),
                        }))}
                        onChange={(mode) =>
                          changeRow(row.key, { mode: mode as SimulatorMode })
                        }
                        ariaLabel={t('gateway.simulator.mode')}
                        emptyText={t('common.state.empty')}
                        searchable={false}
                        className="w-full"
                      />
                    )}
                  </DataTableCell>
                  <DataTableCell>
                    <TableActions>
                      <TableActionButton
                        label={t('gateway.simulator.add')}
                        icon={<Plus />}
                        onClick={addRow}
                      />
                      <TableActionButton
                        label={t('gateway.simulator.delete')}
                        icon={<Trash2 />}
                        onClick={() => removeRow(row)}
                      />
                    </TableActions>
                  </DataTableCell>
                </DataTableRow>
              ))}
            </DataTableBody>
          </DataTable>
        )}
      </ListTableCard>
    </div>
  )
}
