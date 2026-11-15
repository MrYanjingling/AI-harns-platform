import { useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Plus, Search } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  ConfirmDialog,
  ErrorState,
  Input,
  Pagination,
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
import { GatewayTBody, GatewayTD, GatewayTH, GatewayTHead, GatewayTR, GatewayTable } from '@/components/GatewayTable'
import { CollectStatusBadge } from '@/components/StatusBadge'
import { driverModelLabel, driverTypeLabel, formatCycle } from '@/lib/constants'
import { CollectionActionsMenu } from './CollectionActionsMenu'

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
  const navigate = useNavigate()
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
    setApplied({ ...filters, keyword: filters.keyword.trim() })
    setPage(1)
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
    <div>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative w-[200px]">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
          <Input
            value={filters.keyword}
            placeholder={t('gateway.collection.searchPlaceholder')}
            className="h-9 pl-9 text-sm"
            onChange={(event) => setFilters((current) => ({ ...current, keyword: event.target.value }))}
            onKeyDown={(event) => {
              if (event.key === 'Enter') applyFilters()
            }}
          />
        </div>
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
          panelClassName="w-[140px]"
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
          panelClassName="w-[140px]"
        />
        <Button variant="primary" className="h-9" onClick={applyFilters}>
          {t('common.action.search')}
        </Button>
        <Button variant="ghost" className="h-9" onClick={resetFilters}>
          {t('common.action.reset')}
        </Button>
        <Link
          to="/collections/new"
          className="ml-auto inline-flex h-9 items-center gap-1.5 rounded bg-primary px-3 text-sm font-medium text-primary-fg transition-colors hover:bg-primary-hover"
        >
          <Plus className="size-3.5" />
          {t('gateway.collection.create')}
        </Link>
      </div>

      <Card className="flex h-[calc(100dvh-212px)] min-h-[420px] flex-col overflow-hidden">
        <div className="min-h-0 flex-1 overflow-auto">
          {collections.isLoading ? (
            <TableSkeleton rows={6} cols={8} />
          ) : collections.isError ? (
            <div className="p-5">
              <ErrorState
                error={collections.error}
                onRetry={() => void collections.refetch()}
                compact
              />
            </div>
          ) : list.length === 0 ? null : (
            <GatewayTable>
              <GatewayTHead>
                <tr>
                  <GatewayTH className="w-16 text-center">{t('common.field.order')}</GatewayTH>
                  <GatewayTH>{t('gateway.collection.columns.name')}</GatewayTH>
                  <GatewayTH>{t('gateway.collection.columns.status')}</GatewayTH>
                  <GatewayTH>{t('gateway.collection.columns.model')}</GatewayTH>
                  <GatewayTH>{t('gateway.collection.columns.cycle')}</GatewayTH>
                  <GatewayTH>{t('gateway.collection.columns.enable')}</GatewayTH>
                  <GatewayTH>{t('gateway.collection.columns.variableCount')}</GatewayTH>
                  <GatewayTH className="text-right">{t('common.field.actions')}</GatewayTH>
                </tr>
              </GatewayTHead>
              <GatewayTBody>
                {list.map((collection, index) => {
                  const disableToggle =
                    setEnable.isPending ||
                    (collection.enable && collection.collectStatus !== 'stopped')
                  return (
                    <GatewayTR key={collection.id}>
                      <GatewayTD className="text-center text-muted">
                        {(page - 1) * pageSize + index + 1}
                      </GatewayTD>
                      <GatewayTD>
                        <Link
                          to={`/collections/${collection.id}`}
                          className="text-fg transition-colors hover:text-primary"
                        >
                          {collection.name}
                        </Link>
                      </GatewayTD>
                      <GatewayTD>
                        <CollectStatusBadge status={collection.collectStatus} />
                      </GatewayTD>
                      <GatewayTD>{driverModelLabel(collection.driverModel) || '—'}</GatewayTD>
                      <GatewayTD className="whitespace-nowrap text-muted">
                        {formatCycle(collection.collectorCycleMs)}
                      </GatewayTD>
                      <GatewayTD>
                        <Switch
                          checked={collection.enable}
                          disabled={disableToggle}
                          label={t('gateway.collection.columns.enable')}
                          onChange={(enable) => void toggleEnable(collection, enable)}
                        />
                      </GatewayTD>
                      <GatewayTD>{collection.variableCount}</GatewayTD>
                      <GatewayTD className="text-right">
                        <div className="flex justify-end">
                          <CollectionActionsMenu
                            collection={collection}
                            onView={() => navigate(`/collections/${collection.id}`)}
                            onEdit={() => navigate(`/collections/${collection.id}/edit`)}
                            onAction={(kind) => setPending({ kind, collection })}
                          />
                        </div>
                      </GatewayTD>
                    </GatewayTR>
                  )
                })}
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
            disabled={collections.isFetching || collections.isLoading}
          />
        </div>
      </Card>

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
