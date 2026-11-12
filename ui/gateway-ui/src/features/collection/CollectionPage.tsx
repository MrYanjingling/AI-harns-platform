import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Eye,
  Pencil,
  Play,
  Plus,
  RefreshCw,
  RotateCw,
  Search,
  Square,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  ConfirmDialog,
  EmptyState,
  ErrorState,
  Input,
  PageHeader,
  Pagination,
  Select,
  StatCard,
  Switch,
  Table,
  TableSkeleton,
  TBody,
  TD,
  TH,
  THead,
  TR,
  errorMessage,
  formatRelative,
  useI18n,
} from '@harns/core'
import {
  useCollections,
  useCollectionRuntimeAction,
  useDeleteCollection,
  useDriverList,
  useSetCollectionEnable,
} from '@/api/hooks'
import type { Collection } from '@/api/types'
import { CollectStatusBadge } from '@/components/StatusBadge'
import { ProtocolLabel } from '@/components/ProtocolLabel'
import {
  COLLECT_STATUS,
  driverTypeLabel,
  formatCycle,
  isAbnormalStatus,
} from '@/lib/constants'

type PendingAction = { kind: 'delete' | 'start' | 'stop' | 'restart'; collection: Collection }

const STATUS_OPTIONS = Object.keys(COLLECT_STATUS) as Array<keyof typeof COLLECT_STATUS>

export function CollectionPage() {
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [driverType, setDriverType] = useState('all')
  const [statusFilter, setStatusFilter] = useState('all')
  const [pending, setPending] = useState<PendingAction | null>(null)

  const collections = useCollections({
    page,
    pageSize,
    keyword: keyword || undefined,
    driverType: driverType === 'all' ? undefined : driverType,
  })
  const summary = useCollections({ page: 1, pageSize: 100 })
  const drivers = useDriverList()
  const deleteCollection = useDeleteCollection()
  const runtimeAction = useCollectionRuntimeAction()
  const setEnable = useSetCollectionEnable()

  const list = collections.data?.list ?? []
  const total = collections.data?.total ?? 0

  const driverTypes = useMemo(() => {
    const types = new Set((drivers.data ?? []).map((driver) => driver.driverType))
    return [...types]
  }, [drivers.data])

  const stats = useMemo(() => {
    const items = summary.data?.list ?? []
    return {
      total: summary.data?.total ?? items.length,
      collecting: items.filter((item) => item.collectStatus === 'collecting').length,
      abnormal: items.filter((item) => isAbnormalStatus(item.collectStatus)).length,
      stopped: items.filter((item) => item.collectStatus === 'stopped').length,
    }
  }, [summary.data])

  const filtered = useMemo(() => {
    if (statusFilter === 'all') return list
    return list.filter((item) => item.collectStatus === statusFilter)
  }, [list, statusFilter])

  const submitSearch = () => {
    setPage(1)
    setKeyword(keywordInput.trim())
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
      toast.error(errorMessage(error))
    }
  }

  const toggleEnable = async (collection: Collection, enable: boolean) => {
    try {
      await setEnable.mutateAsync({ id: collection.id, name: collection.name, enable })
      toast.success(
        enable ? t('gateway.collection.toast.enabled') : t('gateway.collection.toast.disabled'),
      )
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const loading = collections.isLoading
  const error = collections.error

  return (
    <div className="space-y-3">
      <PageHeader
        title={t('gateway.collection.title')}
        description={t('gateway.collection.description')}
        actions={
          <>
            <Button
              variant="outline"
              loading={collections.isFetching}
              onClick={() => void collections.refetch()}
              icon={<RefreshCw className="size-3.5" />}
            >
              {t('common.action.refresh')}
            </Button>
            <Link
              to="/collections/new"
              className="inline-flex h-8 items-center gap-1.5 rounded bg-primary px-3 text-sm font-medium text-primary-fg transition-colors hover:bg-primary-hover"
            >
              <Plus className="size-3.5" />
              {t('gateway.collection.create')}
            </Link>
          </>
        }
      />

      <div className="grid gap-2.5 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard label={t('gateway.collection.stat.total')} value={stats.total} />
        <StatCard label={t('gateway.collection.stat.collecting')} value={stats.collecting} />
        <StatCard label={t('gateway.collection.stat.abnormal')} value={stats.abnormal} />
        <StatCard label={t('gateway.collection.stat.stopped')} value={stats.stopped} />
      </div>

      <Card className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-2 border-b border-border p-2.5">
          <div className="relative min-w-52 flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-faint" />
            <Input
              value={keywordInput}
              onChange={(event) => setKeywordInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') submitSearch()
              }}
              placeholder={t('gateway.collection.searchPlaceholder')}
              aria-label={t('common.action.search')}
              className="pl-8"
            />
          </div>
          <Select
            value={driverType}
            onChange={(event) => {
              setDriverType(event.target.value)
              setPage(1)
            }}
            aria-label={t('gateway.collection.filter.driverType')}
            className="w-40"
          >
            <option value="all">{t('gateway.collection.filter.allDriverTypes')}</option>
            {driverTypes.map((type) => (
              <option key={type} value={type}>
                {driverTypeLabel(type)}
              </option>
            ))}
          </Select>
          <Select
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value)}
            aria-label={t('gateway.collection.filter.status')}
            className="w-36"
          >
            <option value="all">{t('gateway.collection.filter.allStatuses')}</option>
            {STATUS_OPTIONS.map((status) => (
              <option key={status} value={status}>
                {t(COLLECT_STATUS[status].labelKey)}
              </option>
            ))}
          </Select>
          <Button variant="outline" onClick={submitSearch} icon={<Search className="size-3.5" />}>
            {t('common.action.search')}
          </Button>
        </div>

        {loading ? (
          <TableSkeleton rows={6} cols={7} />
        ) : error ? (
          <div className="p-3">
            <ErrorState error={error} onRetry={() => void collections.refetch()} />
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            title={
              keyword || driverType !== 'all' || statusFilter !== 'all'
                ? t('gateway.collection.empty.filteredTitle')
                : t('gateway.collection.empty.title')
            }
            description={
              keyword || driverType !== 'all' || statusFilter !== 'all'
                ? t('gateway.collection.empty.filteredDescription')
                : t('gateway.collection.empty.description')
            }
          />
        ) : (
          <Table>
            <THead>
              <TR>
                <TH>{t('gateway.collection.columns.name')}</TH>
                <TH>{t('gateway.collection.columns.protocol')}</TH>
                <TH>{t('gateway.collection.columns.status')}</TH>
                <TH>{t('gateway.collection.columns.cycle')}</TH>
                <TH>{t('gateway.collection.columns.enable')}</TH>
                <TH>{t('gateway.collection.columns.modTime')}</TH>
                <TH className="text-right">{t('common.field.actions')}</TH>
              </TR>
            </THead>
            <TBody>
              {filtered.map((collection) => {
                const disableToggle =
                  setEnable.isPending ||
                  (collection.enable && collection.collectStatus !== 'stopped')
                return (
                  <TR key={collection.id}>
                    <TD>
                      <div className="flex items-center gap-2">
                        <div className="min-w-0">
                          <Link
                            to={`/collections/${collection.id}`}
                            className="block truncate text-[13px] font-medium text-fg transition-colors hover:text-primary"
                          >
                            {collection.name}
                          </Link>
                          <p className="truncate font-mono text-[10px] text-faint">
                            {collection.id}
                          </p>
                        </div>
                      </div>
                    </TD>
                    <TD>
                      <ProtocolLabel type={collection.driverType} model={collection.driverModel} />
                    </TD>
                    <TD>
                      <CollectStatusBadge status={collection.collectStatus} />
                    </TD>
                    <TD className="whitespace-nowrap text-xs text-muted tabular">
                      {formatCycle(collection.collectorCycleMs)}
                    </TD>
                    <TD>
                      <Switch
                        checked={collection.enable}
                        disabled={disableToggle}
                        label={t('gateway.collection.columns.enable')}
                        onChange={(enable) => void toggleEnable(collection, enable)}
                      />
                    </TD>
                    <TD className="whitespace-nowrap text-xs text-muted">
                      {formatRelative(collection.modTime)}
                    </TD>
                    <TD>
                      <div className="flex items-center justify-end gap-0.5">
                        <Link
                          to={`/collections/${collection.id}`}
                          className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
                          title={t('gateway.collection.action.detail')}
                          aria-label={t('gateway.collection.action.detail')}
                        >
                          <Eye className="size-3.5" />
                        </Link>
                        <Link
                          to={`/collections/${collection.id}/edit`}
                          className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
                          title={t('common.action.edit')}
                          aria-label={t('common.action.edit')}
                        >
                          <Pencil className="size-3.5" />
                        </Link>
                        {collection.collectStatus === 'collecting' ? (
                          <button
                            type="button"
                            onClick={() => setPending({ kind: 'stop', collection })}
                            className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
                            title={t('common.action.stop')}
                            aria-label={t('common.action.stop')}
                          >
                            <Square className="size-3.5" />
                          </button>
                        ) : (
                          <button
                            type="button"
                            onClick={() => setPending({ kind: 'start', collection })}
                            className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
                            title={t('common.action.start')}
                            aria-label={t('common.action.start')}
                          >
                            <Play className="size-3.5" />
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={() => setPending({ kind: 'restart', collection })}
                          className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
                          title={t('common.action.restart')}
                          aria-label={t('common.action.restart')}
                        >
                          <RotateCw className="size-3.5" />
                        </button>
                        <button
                          type="button"
                          onClick={() => setPending({ kind: 'delete', collection })}
                          className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-danger/10 hover:text-danger"
                          title={t('common.action.delete')}
                          aria-label={t('common.action.delete')}
                        >
                          <Trash2 className="size-3.5" />
                        </button>
                      </div>
                    </TD>
                  </TR>
                )
              })}
            </TBody>
          </Table>
        )}

        <Pagination
          page={page}
          pageSize={pageSize}
          total={total}
          onPageChange={setPage}
          onPageSizeChange={(size) => {
            setPageSize(size)
            setPage(1)
          }}
          disabled={loading}
        />
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
            ? t('gateway.collection.confirm.deleteDescription', {
                name: pending.collection.name,
              })
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
