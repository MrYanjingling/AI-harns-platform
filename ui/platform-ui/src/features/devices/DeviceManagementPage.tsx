import { useMemo, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { Download, ListFilter, Play, Plus, RefreshCw, Square, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteDevice, useDeviceDetails, useSwitchDeviceStatus } from '@/api/hooks'
import type { DeviceDetail, DeviceStatus, DeviceStatusAction, DeviceType } from '@/api/types'
import { useI18n } from '@/i18n'
import { DEVICE_STATUS, PROTOCOLS } from '@/lib/constants'
import { errorMessage } from '@/lib/http'
import { cn } from '@/lib/cn'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { Input, Select } from '@/components/ui/Field'
import { TableSkeleton } from '@/components/ui/Skeleton'
import { ErrorState } from '@/components/shared/ErrorState'
import { Pagination } from '@/components/shared/Pagination'
import { DeviceTreePanel } from './DeviceTreePanel'
import { DeviceTable } from './DeviceTable'
import {
  EMPTY_DEVICE_FILTERS,
  exportDevicesCsv,
  filterDevices,
  type DeviceFilters,
  type TreeMode,
} from './device-list'

interface FilterDraft {
  name: string
  code: string
  protocol: DeviceType | ''
  status: DeviceStatus | ''
}

const EMPTY_DRAFT: FilterDraft = { name: '', code: '', protocol: '', status: '' }
const DEFAULT_PAGE_SIZE = 20

function FilterItem({
  label,
  htmlFor,
  children,
}: {
  label: string
  htmlFor: string
  children: ReactNode
}) {
  return (
    <div className="flex items-center gap-2">
      <label htmlFor={htmlFor} className="text-sm whitespace-nowrap text-muted">
        {label}
      </label>
      {children}
    </div>
  )
}

export function DeviceManagementPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const devicesQuery = useDeviceDetails()
  const deleteMutation = useDeleteDevice()
  const statusMutation = useSwitchDeviceStatus()

  const [treeMode, setTreeMode] = useState<TreeMode>('protocol')
  const [draft, setDraft] = useState<FilterDraft>(EMPTY_DRAFT)
  const [applied, setApplied] = useState<DeviceFilters>(EMPTY_DEVICE_FILTERS)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [deleteTargets, setDeleteTargets] = useState<DeviceDetail[]>([])
  const [batchPending, setBatchPending] = useState(false)

  const devices = devicesQuery.data ?? []
  const filtered = useMemo(() => filterDevices(devices, applied), [devices, applied])
  const total = filtered.length
  const pageCount = Math.max(1, Math.ceil(total / pageSize))
  const currentPage = Math.min(page, pageCount)
  const rows = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize)
  const allSelected = rows.length > 0 && rows.every((device) => selected.has(device.id))

  const selectedDevices = useMemo(
    () => devices.filter((device) => selected.has(device.id)),
    [devices, selected],
  )

  const resetSelection = () => setSelected(new Set())

  const applySearch = () => {
    setApplied({ ...draft, model: '' })
    setPage(1)
    resetSelection()
  }

  const reset = () => {
    setDraft(EMPTY_DRAFT)
    setApplied(EMPTY_DEVICE_FILTERS)
    setPage(1)
    resetSelection()
  }

  const selectTree = (protocol: DeviceType | '', model: string) => {
    setDraft((current) => ({ ...current, protocol }))
    setApplied((current) => ({ ...current, protocol, model }))
    setPage(1)
    resetSelection()
  }

  const toggle = (id: string) => {
    setSelected((current) => {
      const next = new Set(current)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleAll = () => {
    setSelected((current) => {
      const next = new Set(current)
      if (allSelected) rows.forEach((device) => next.delete(device.id))
      else rows.forEach((device) => next.add(device.id))
      return next
    })
  }

  const confirmDelete = async () => {
    if (deleteTargets.length === 0) return
    setBatchPending(true)
    let deleted = 0
    let failed = 0
    for (const device of deleteTargets) {
      try {
        await deleteMutation.mutateAsync({ id: device.id, etag: device.eTag })
        deleted += 1
      } catch {
        failed += 1
      }
    }
    setBatchPending(false)
    setDeleteTargets([])
    resetSelection()
    if (deleted > 0) toast.success(t('devices.list.deleted', { count: deleted }))
    if (failed > 0) toast.error(t('devices.list.deleteFailed', { count: failed }))
  }

  const runStatusAction = async (action: DeviceStatusAction, targets: DeviceDetail[]) => {
    if (targets.length === 0) return
    setBatchPending(true)
    let done = 0
    let failed = 0
    for (const device of targets) {
      try {
        await statusMutation.mutateAsync({ id: device.id, action })
        done += 1
      } catch {
        failed += 1
      }
    }
    setBatchPending(false)
    const verb = t(
      action === 'stop'
        ? 'enums.deviceAction.stop'
        : action === 'restart'
          ? 'enums.deviceAction.restart'
          : 'enums.deviceAction.start',
    )
    if (done > 0) toast.success(t('devices.list.statusSubmitted', { count: done, verb }))
    if (failed > 0) toast.error(t('devices.list.statusFailed', { count: failed, verb }))
  }

  const handleRefresh = async () => {
    const result = await devicesQuery.refetch()
    if (result.error) toast.error(errorMessage(result.error))
    else toast.success(t('devices.list.refreshed'))
  }

  const handleExport = () => {
    if (filtered.length === 0) {
      toast.warning(t('devices.list.exportEmpty'))
      return
    }
    exportDevicesCsv(filtered)
    toast.success(t('devices.list.exportSuccess', { count: filtered.length }))
  }

  const hasFilters = Boolean(
    applied.name || applied.code || applied.protocol || applied.status || applied.model,
  )

  return (
    <div className="flex h-full min-h-0">
      <aside className="hidden w-64 shrink-0 border-r border-border bg-surface md:block">
        <DeviceTreePanel
          devices={devices}
          mode={treeMode}
          onModeChange={setTreeMode}
          activeProtocol={applied.protocol}
          activeModel={applied.model}
          totalCount={devices.length}
          loading={devicesQuery.isLoading}
          onSelect={selectTree}
        />
      </aside>

      <section className="flex min-w-0 flex-1 flex-col">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-border bg-surface p-3">
          <FilterItem label={t('devices.list.filterName')} htmlFor="device-name">
            <Input
              id="device-name"
              value={draft.name}
              onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))}
              onKeyDown={(event) => {
                if (event.key === 'Enter') applySearch()
              }}
              placeholder={t('devices.list.placeholder')}
              className="w-40"
            />
          </FilterItem>
          <FilterItem label={t('devices.list.filterCode')} htmlFor="device-code">
            <Input
              id="device-code"
              value={draft.code}
              onChange={(event) => setDraft((current) => ({ ...current, code: event.target.value }))}
              onKeyDown={(event) => {
                if (event.key === 'Enter') applySearch()
              }}
              placeholder={t('devices.list.placeholder')}
              className="w-44"
            />
          </FilterItem>
          <FilterItem label={t('devices.list.filterProtocol')} htmlFor="device-protocol">
            <Select
              id="device-protocol"
              value={draft.protocol}
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  protocol: event.target.value as DeviceType | '',
                }))
              }
              className="w-32"
            >
              <option value="">{t('common.state.all')}</option>
              {Object.values(PROTOCOLS).map((protocol) => (
                <option key={protocol.type} value={protocol.type}>
                  {protocol.label}
                </option>
              ))}
            </Select>
          </FilterItem>
          <FilterItem label={t('devices.list.filterStatus')} htmlFor="device-status">
            <Select
              id="device-status"
              value={draft.status}
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  status: event.target.value as DeviceStatus | '',
                }))
              }
              className="w-32"
            >
              <option value="">{t('common.state.all')}</option>
              {Object.entries(DEVICE_STATUS).map(([value, meta]) => (
                <option key={value} value={value}>
                  {t(meta.labelKey)}
                </option>
              ))}
            </Select>
          </FilterItem>
          <div className="ml-auto flex gap-2">
            <Button variant="primary" onClick={applySearch}>
              {t('common.action.query')}
            </Button>
            <Button variant="secondary" onClick={reset}>
              {t('common.action.reset')}
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border bg-surface px-3 py-2">
          <div className="flex flex-wrap gap-2">
            <Button
              variant="primary"
              icon={<Plus className="size-3.5" />}
              onClick={() => navigate('/devices/new')}
            >
              {t('common.action.create')}
            </Button>
            <Button
              variant="dangerSoft"
              icon={<Trash2 className="size-3.5" />}
              disabled={selectedDevices.length === 0 || batchPending}
              onClick={() => setDeleteTargets(selectedDevices)}
            >
              {t('common.action.delete')}
            </Button>
            <Button
              variant="secondary"
              icon={<Play className="size-3.5" />}
              disabled={selectedDevices.length === 0 || batchPending}
              onClick={() => void runStatusAction('start', selectedDevices)}
            >
              {t('common.action.start')}
            </Button>
            <Button
              variant="secondary"
              icon={<Square className="size-3.5" />}
              disabled={selectedDevices.length === 0 || batchPending}
              onClick={() => void runStatusAction('stop', selectedDevices)}
            >
              {t('common.action.stop')}
            </Button>
            <Button
              variant="secondary"
              icon={<Download className="size-3.5" />}
              disabled={batchPending}
              onClick={handleExport}
            >
              {t('common.action.export')}
            </Button>
          </div>

          <div className="flex items-center gap-3">
            <span className="text-xs text-muted">
              {t('devices.list.totalDevices', { count: total })}
              {selectedDevices.length > 0 ? (
                <span className="ml-2 text-primary">
                  {t('devices.list.selected', { count: selectedDevices.length })}
                </span>
              ) : null}
            </span>
            <Button
              variant="secondary"
              icon={<RefreshCw className={cn('size-3.5', devicesQuery.isFetching && 'animate-spin')} />}
              disabled={batchPending}
              onClick={() => void handleRefresh()}
            >
              {t('common.action.refresh')}
            </Button>
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-auto p-2">
          {devicesQuery.isLoading ? (
            <div className="border border-border bg-surface">
              <TableSkeleton rows={8} cols={8} />
            </div>
          ) : devicesQuery.isError ? (
            <div className="p-2">
              <ErrorState error={devicesQuery.error} onRetry={devicesQuery.refetch} compact />
            </div>
          ) : total === 0 ? (
            <EmptyState
              icon={<ListFilter className="size-5" />}
              title={
                hasFilters ? t('devices.list.noMatchTitle') : t('devices.list.emptyTitle')
              }
              description={
                hasFilters
                  ? t('devices.list.noMatchDescription')
                  : t('devices.list.emptyDescription')
              }
              action={
                hasFilters ? (
                  <Button variant="secondary" onClick={reset}>
                    {t('devices.list.clearFilters')}
                  </Button>
                ) : (
                  <Button
                    variant="primary"
                    icon={<Plus className="size-4" />}
                    onClick={() => navigate('/devices/new')}
                  >
                    {t('devices.create.title')}
                  </Button>
                )
              }
            />
          ) : (
            <DeviceTable
              rows={rows}
              page={currentPage}
              pageSize={pageSize}
              selected={selected}
              allSelected={allSelected}
              busy={batchPending}
              onToggle={toggle}
              onToggleAll={toggleAll}
              onView={(device) => navigate(`/devices/${device.id}`)}
              onEdit={(device) => navigate(`/devices/${device.id}/edit`)}
              onDelete={(device) => setDeleteTargets([device])}
              onStatusAction={(device, action) => void runStatusAction(action, [device])}
            />
          )}
        </div>

        {total > 0 ? (
          <Pagination
            page={currentPage}
            pageSize={pageSize}
            total={total}
            onPageChange={(next) => {
              setPage(next)
              resetSelection()
            }}
            onPageSizeChange={(size) => {
              setPageSize(size)
              setPage(1)
              resetSelection()
            }}
            disabled={devicesQuery.isFetching || batchPending}
          />
        ) : null}
      </section>

      <ConfirmDialog
        open={deleteTargets.length > 0}
        title={t('devices.delete.title')}
        description={t('devices.delete.description', { count: deleteTargets.length })}
        confirmText={t('common.action.delete')}
        danger
        loading={batchPending}
        onConfirm={() => void confirmDelete()}
        onClose={() => setDeleteTargets([])}
      />
    </div>
  )
}
