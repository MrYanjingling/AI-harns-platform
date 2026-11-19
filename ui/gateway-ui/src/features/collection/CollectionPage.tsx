import { useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Pencil, Plus, Pointer, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  ConfirmDialog,
  ErrorState,
  ListTableCard,
  Pagination,
  SearchInput,
  TableActions,
  SearchSelect,
  Switch,
  TableSkeleton,
  showErrorToast,
  useI18n,
  type SearchSelectOption,
} from '@harns/core'
import {
  useCollections,
  useCollectionRuntimeAction,
  useDeleteCollection,
  useDriverList,
  useSetCollectionEnable,
} from '@/api/hooks'
import type { Collection } from '@/api/types'
import {
  DataTable,
  DataTableBody,
  DataTableCell,
  DataTableHead,
  DataTableHeadCell,
  DataTableRow,
  TableActionButton,
} from '@harns/core'
import { CollectStatusBadge } from '@/components/StatusBadge'
import { canToggleCollectionEnable } from '@/lib/collection-actions'
import { driverModelLabel, driverTypeLabel } from '@/lib/constants'
import { ColumnResizeHandle, useColumnResize } from '@harns/core'
import { CollectionActionsMenu } from './CollectionActionsMenu'
import { CollectionCreateDialog } from './CollectionCreateDialog'
import { CollectionEditDialog } from './CollectionEditDialog'
import { CollectionVariablesDialog } from './CollectionVariablesDialog'

type PendingAction = { kind: 'delete' | 'start' | 'stop' | 'restart'; collection: Collection }

interface CollectionFilters {
  keyword: string
  driverType: string
  driverModel: string
}

const emptyFilters: CollectionFilters = {
  keyword: '',
  driverType: 'all',
  driverModel: 'all',
}

export function CollectionPage() {
  const { t } = useI18n()
  const [searchParams, setSearchParams] = useSearchParams()
  const [createOpen, setCreateOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [variablesCollection, setVariablesCollection] = useState<Collection | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [filters, setFilters] = useState<CollectionFilters>(emptyFilters)
  const [applied, setApplied] = useState<CollectionFilters>(emptyFilters)
  const [pending, setPending] = useState<PendingAction | null>(null)

  const collections = useCollections({
    page,
    pageSize,
    keyword: applied.keyword || undefined,
    driverType: applied.driverType === 'all' ? undefined : applied.driverType,
    driverModel: applied.driverModel === 'all' ? undefined : applied.driverModel,
  })
  const drivers = useDriverList()
  const deleteCollection = useDeleteCollection()
  const runtimeAction = useCollectionRuntimeAction()
  const setEnable = useSetCollectionEnable()

  const list = collections.data?.list ?? []
  const total = collections.data?.total ?? 0
  const { widths, totalWidth, resize } = useColumnResize({
    order: 80,
    name: 220,
    status: 130,
    driverType: 120,
    model: 180,
    cycle: 120,
    enable: 100,
    variableCount: 120,
    actions: 140,
  })

  useEffect(() => {
    if (searchParams.get('create') === '1') setCreateOpen(true)
  }, [searchParams])

  const closeCreate = () => {
    setCreateOpen(false)
    if (searchParams.has('create')) {
      const next = new URLSearchParams(searchParams)
      next.delete('create')
      setSearchParams(next, { replace: true })
    }
  }

  const driverTypes = useMemo(
    () => [...new Set((drivers.data ?? []).map((driver) => driver.driverType))],
    [drivers.data],
  )
  const driverModels = useMemo(() => {
    const source = drivers.data ?? []
    const filtered = filters.driverType === 'all'
      ? source
      : source.filter((driver) => driver.driverType === filters.driverType)
    return [...new Set(filtered.map((driver) => driver.model))]
  }, [drivers.data, filters.driverType])

  const driverTypeOptions = useMemo<SearchSelectOption[]>(
    () => [
      { value: 'all', label: t('gateway.collection.filter.allDriverTypes') },
      ...driverTypes.map((type) => ({ value: type, label: driverTypeLabel(type) || type })),
    ],
    [driverTypes, t],
  )
  const driverModelOptions = useMemo<SearchSelectOption[]>(
    () => [
      { value: 'all', label: t('gateway.collection.filter.allDriverModels') },
      ...driverModels.map((model) => ({ value: model, label: driverModelLabel(model) || model })),
    ],
    [driverModels, t],
  )

  const applyFilters = () => {
    const next = { ...filters, keyword: filters.keyword.trim() }
    const unchanged =
      page === 1 &&
      next.keyword === applied.keyword &&
      next.driverType === applied.driverType &&
      next.driverModel === applied.driverModel
    setApplied(next)
    setPage(1)
    if (unchanged) void collections.refetch()
  }

  const resetFilters = () => {
    setFilters(emptyFilters)
    setApplied(emptyFilters)
    setPage(1)
  }

  const confirm = async () => {
    if (!pending) return
    const { kind, collection } = pending
    try {
      if (kind === 'delete') {
        await deleteCollection.mutateAsync({ id: collection.id, name: collection.name })
        toast.success(t('gateway.collection.toast.deleted'))
        setPending(null)
        return
      }
      await runtimeAction.mutateAsync({ id: collection.id, name: collection.name, action: kind })
      toast.success(
        kind === 'start'
          ? t('gateway.collection.toast.started')
          : kind === 'stop'
            ? t('gateway.collection.toast.stopped')
            : t('gateway.collection.toast.restarted'),
      )
      setPending(null)
    } catch (error) {
      showErrorToast(error)
    }
  }

  const toggleEnable = async (collection: Collection, enable: boolean) => {
    try {
      await setEnable.mutateAsync({ id: collection.id, name: collection.name, enable })
      toast.success(
        enable ? t('gateway.collection.toast.enabled') : t('gateway.collection.toast.disabled'),
      )
    } catch (error) {
      showErrorToast(error)
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="app-page-toolbar">
        <SearchInput
          wrapperClassName="w-[200px]"
          value={filters.keyword}
          placeholder={t('gateway.collection.searchPlaceholder')}
          onChange={(event) => setFilters((current) => ({ ...current, keyword: event.target.value }))}
          onKeyDown={(event) => {
            if (event.key === 'Enter') applyFilters()
          }}
        />
        <SearchSelect
          value={filters.driverType}
          options={driverTypeOptions}
          onChange={(driverType) =>
            setFilters((current) => ({ ...current, driverType, driverModel: 'all' }))
          }
          ariaLabel={t('gateway.collection.filter.driverType')}
          emptyText={t('common.state.empty')}
          placeholderValue="all"
          className="w-[140px]"
        />
        <SearchSelect
          value={filters.driverModel}
          options={driverModelOptions}
          onChange={(driverModel) =>
            setFilters((current) => ({ ...current, driverModel }))
          }
          ariaLabel={t('gateway.collection.filter.driverModel')}
          emptyText={t('common.state.empty')}
          placeholderValue="all"
          className="w-[140px]"
        />
        <Button variant="primary" className="h-9" onClick={applyFilters}>
          {t('common.action.search')}
        </Button>
        <Button variant="ghost" className="h-9" onClick={resetFilters}>
          {t('common.action.reset')}
        </Button>
        <Button
          variant="primary"
          className="ml-auto h-9"
          icon={<Plus className="size-3.5" />}
          onClick={() => setCreateOpen(true)}
        >
          {t('gateway.collection.create')}
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
            disabled={collections.isFetching || collections.isLoading}
          />
        }
      >
        {collections.isLoading ? (
          <TableSkeleton rows={6} cols={9} />
        ) : collections.isError ? (
          <div className="p-5">
            <ErrorState
              error={collections.error}
              onRetry={() => void collections.refetch()}
              compact
            />
          </div>
        ) : list.length === 0 ? null : (
          <DataTable
            style={{
              tableLayout: 'fixed',
              width: `max(100%, ${totalWidth}px)`,
            }}
          >
            <colgroup>
              <col style={{ width: widths.order }} />
              <col style={{ width: widths.name }} />
              <col style={{ width: widths.status }} />
              <col style={{ width: widths.driverType }} />
              <col style={{ width: widths.model }} />
              <col style={{ width: widths.cycle }} />
              <col style={{ width: widths.enable }} />
              <col style={{ width: widths.variableCount }} />
              <col style={{ width: widths.actions }} />
            </colgroup>
            <DataTableHead>
              <tr>
                <DataTableHeadCell style={{ textAlign: 'center' }}>
                  {t('common.field.order')}
                  <ColumnResizeHandle column="order" width={widths.order} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.collection.columns.name')}
                  <ColumnResizeHandle column="name" width={widths.name} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.collection.columns.status')}
                  <ColumnResizeHandle column="status" width={widths.status} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.collection.columns.driverType')}
                  <ColumnResizeHandle
                    column="driverType"
                    width={widths.driverType}
                    onResize={resize}
                  />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.collection.columns.model')}
                  <ColumnResizeHandle column="model" width={widths.model} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.collection.columns.cycle')}
                  <ColumnResizeHandle column="cycle" width={widths.cycle} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.collection.columns.enable')}
                  <ColumnResizeHandle column="enable" width={widths.enable} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.collection.columns.variableCount')}
                  <ColumnResizeHandle
                    column="variableCount"
                    width={widths.variableCount}
                    onResize={resize}
                  />
                </DataTableHeadCell>
                <DataTableHeadCell style={{ textAlign: 'left' }}>
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
              {list.map((collection, index) => {
                const disableToggle =
                  setEnable.isPending || !canToggleCollectionEnable(collection)
                return (
                  <DataTableRow key={collection.id}>
                    <DataTableCell style={{ textAlign: 'center' }}>
                      {(page - 1) * pageSize + index + 1}
                    </DataTableCell>
                    <DataTableCell>
                      <Link
                        to={`/collections/${collection.id}`}
                        className="transition-colors hover:text-primary"
                      >
                        {collection.name}
                      </Link>
                    </DataTableCell>
                    <DataTableCell>
                      <CollectStatusBadge
                        status={collection.collectStatus}
                        error={collection.lastError}
                      />
                    </DataTableCell>
                    <DataTableCell>{driverTypeLabel(collection.driverType) || '—'}</DataTableCell>
                    <DataTableCell>{driverModelLabel(collection.driverModel) || '—'}</DataTableCell>
                    <DataTableCell className="whitespace-nowrap">
                      {collection.collectorCycleMs > 0 ? collection.collectorCycleMs : '—'}
                    </DataTableCell>
                    <DataTableCell>
                      <Switch
                        checked={collection.enable}
                        disabled={disableToggle}
                        label={t('gateway.collection.columns.enable')}
                        onChange={(enable) => void toggleEnable(collection, enable)}
                      />
                    </DataTableCell>
                    <DataTableCell
                      role="button"
                      tabIndex={0}
                      onClick={() => setVariablesCollection(collection)}
                      onKeyDown={(event) => {
                        if (event.key === 'Enter' || event.key === ' ') {
                          event.preventDefault()
                          setVariablesCollection(collection)
                        }
                      }}
                      className="cursor-pointer transition-colors hover:text-primary-hover"
                    >
                      <span className="inline-flex items-center gap-1">
                        {collection.variableCount}
                        <Pointer className="size-3.5 text-faint" aria-hidden="true" />
                      </span>
                    </DataTableCell>
                    <DataTableCell>
                      <TableActions>
                        <TableActionButton
                          label={t('gateway.collection.action.update')}
                          icon={<Pencil />}
                          onClick={() => setEditingId(collection.id)}
                        />
                        <TableActionButton
                          label={t('common.action.delete')}
                          icon={<Trash2 />}
                          onClick={() => setPending({ kind: 'delete', collection })}
                        />
                        <CollectionActionsMenu
                          collection={collection}
                          onAction={(kind) => setPending({ kind, collection })}
                        />
                      </TableActions>
                    </DataTableCell>
                  </DataTableRow>
                )
              })}
            </DataTableBody>
          </DataTable>
        )}
      </ListTableCard>

      <CollectionCreateDialog open={createOpen} onClose={closeCreate} />
      <CollectionEditDialog collectionId={editingId} onClose={() => setEditingId(null)} />
      <CollectionVariablesDialog
        collection={variablesCollection}
        onClose={() => setVariablesCollection(null)}
      />

      <ConfirmDialog
        open={Boolean(pending)}
        onClose={() => setPending(null)}
        onConfirm={() => void confirm()}
        loading={deleteCollection.isPending || runtimeAction.isPending}
        danger={pending?.kind === 'delete'}
        title={
          pending?.kind === 'delete'
            ? t('gateway.collection.confirm.deleteTitle')
            : pending?.kind === 'start'
              ? t('gateway.collection.confirm.startTitle')
              : pending?.kind === 'stop'
                ? t('gateway.collection.confirm.stopTitle')
                : t('gateway.collection.confirm.restartTitle')
        }
        description={
          pending?.kind === 'delete'
            ? t('gateway.collection.confirm.deleteDescription', { name: pending.collection.name })
            : pending?.kind === 'start'
              ? t('gateway.collection.confirm.startDescription')
              : pending?.kind === 'stop'
                ? t('gateway.collection.confirm.stopDescription')
                : t('gateway.collection.confirm.restartDescription')
        }
        confirmText={
          pending?.kind === 'delete'
            ? t('common.action.delete')
            : pending?.kind === 'start'
              ? t('common.action.start')
              : pending?.kind === 'stop'
                ? t('common.action.stop')
                : t('common.action.restart')
        }
      />
    </div>
  )
}
