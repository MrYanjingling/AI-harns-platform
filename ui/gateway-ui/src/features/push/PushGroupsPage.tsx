import { useEffect, useMemo, useState } from 'react'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import {
  Button,
  ConfirmDialog,
  Dialog,
  EnumBadge,
  ErrorState,
  ListTableCard,
  SearchInput,
  Pagination,
  SearchSelect,
  Table,
  TableSkeleton,
  TBody,
  TD,
  TH,
  THead,
  TR,
  formatDateTime,
  showErrorToast,
  cn,
  useI18n,
  type SearchSelectOption,
} from '@harns/core'
import { TableActions } from '@harns/core'
import {
  useCollections,
  useDeletePushGroup,
  usePushConnectors,
  usePushGroupAction,
  usePushGroups,
} from '@/api/hooks'
import type { PushGroup } from '@/api/types'
import {
  DataTableBody,
  DataTableCell,
  DataTableHeadCell,
  DataTableHead,
  DataTableRow,
  DataTable,
} from '@harns/core'
import { CollectStatusBadge, ConnectorStatusBadge } from '@/components/StatusBadge'
import { ColumnResizeHandle, useColumnResize } from '@harns/core'
import { driverModelLabel, driverTypeLabel } from '@/lib/constants'
import { TableActionButton } from '@harns/core'
import { PushGroupActionsMenu } from './PushGroupActionsMenu'
import { PushGroupFormDialog } from './PushGroupFormDialog'
import { PushGroupPayloadDialog } from './PushGroupPayloadDialog'
import {
  listViewKeys,
  useListViewRestore,
  useListPagination,
  useListPaginationClamp,
  usePersistedState,
} from '@/lib/list-view-state'

type PendingAction = { kind: 'delete' | 'start' | 'stop' | 'restart'; group: PushGroup }

export function PushGroupsPage() {
  const { t } = useI18n()
  const restore = useListViewRestore()
  const [searchParams, setSearchParams] = useSearchParams()
  const { page, pageSize, setPage, setPageSize } = useListPagination(
    listViewKeys.pushGroups,
    restore,
  )
  const [keywordInput, setKeywordInput] = usePersistedState(
    `${listViewKeys.pushGroups}:keywordInput`,
    '',
    restore,
  )
  const [keyword, setKeyword] = usePersistedState(
    `${listViewKeys.pushGroups}:keyword`,
    '',
    restore,
  )
  const [selectedConnectorId, setSelectedConnectorId] = usePersistedState(
    `${listViewKeys.pushGroups}:connectorId`,
    '',
    restore,
  )
  const focusedGroupId = searchParams.get('groupId') ?? ''
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<PushGroup | null>(null)
  const [payloadGroup, setPayloadGroup] = useState<PushGroup | null>(null)
  const [scopeGroup, setScopeGroup] = useState<PushGroup | null>(null)
  const [pending, setPending] = useState<PendingAction | null>(null)

  const connectors = usePushConnectors({ page: 1, pageSize: 100 })
  const collections = useCollections({ page: 1, pageSize: 100 })
  const remove = useDeletePushGroup()
  const action = usePushGroupAction()

  const connectorList = connectors.data?.list ?? []
  const connectorOptions = useMemo<SearchSelectOption[]>(
    () => connectorList.map((item) => ({ value: item.id, label: item.name })),
    [connectorList],
  )
  const connectorById = useMemo(
    () => new Map(connectorList.map((item) => [item.id, item])),
    [connectorList],
  )
  const collectionList = collections.data?.list ?? []
  const collectionById = useMemo(
    () => new Map(collectionList.map((item) => [item.id, item])),
    [collectionList],
  )
  const scopeCollections = useMemo(() => {
    if (!scopeGroup) return []
    if (scopeGroup.allCollections) return collectionList
    return (scopeGroup.collectionIds ?? [])
      .map((id) => collectionById.get(id))
      .filter((item): item is NonNullable<typeof item> => Boolean(item))
  }, [collectionById, collectionList, scopeGroup])
  const activeConnectorId = useMemo(() => {
    if (connectorList.some((item) => item.id === selectedConnectorId)) return selectedConnectorId
    const fromQuery = searchParams.get('connectorId') ?? ''
    if (connectorList.some((item) => item.id === fromQuery)) return fromQuery
    return connectorList[0]?.id ?? ''
  }, [connectorList, searchParams, selectedConnectorId])
  const groups = usePushGroups(
    {
      page,
      pageSize,
      keyword: keyword || undefined,
      connectorId: activeConnectorId || undefined,
    },
    Boolean(activeConnectorId),
  )
  const list = groups.data?.list ?? []
  const total = groups.data?.total ?? 0
  useListPaginationClamp(page, pageSize, total, setPage)
  const noConnectors = !connectors.isLoading && connectorList.length === 0
  const { widths, totalWidth, resize } = useColumnResize({
    order: 80,
    connector: 180,
    name: 220,
    status: 120,
    dataTopic: 200,
    commandTopic: 200,
    interval: 120,
    scope: 100,
    lastPush: 160,
    pending: 130,
    actions: 140,
  })

  useEffect(() => {
    if (!focusedGroupId) return
    const element = document.querySelector<HTMLElement>(
      `[data-group-id="${focusedGroupId}"]`,
    )
    element?.scrollIntoView({ block: 'center' })
  }, [focusedGroupId, list])

  const applySearch = () => {
    const next = keywordInput.trim()
    const unchanged = page === 1 && next === keyword
    setKeyword(next)
    setPage(1)
    if (unchanged) void groups.refetch()
  }

  const selectConnector = (id: string) => {
    setSelectedConnectorId(id)
    setPage(1)
    const next = new URLSearchParams(searchParams)
    next.set('connectorId', id)
    setSearchParams(next, { replace: true })
  }

  const resetSearch = () => {
    setKeywordInput('')
    setKeyword('')
    setPage(1)
  }

  const openCreate = () => {
    setEditing(null)
    setFormOpen(true)
  }

  const openEdit = (group: PushGroup) => {
    setEditing(group)
    setFormOpen(true)
  }

  const confirm = async () => {
    if (!pending) return
    const { kind, group } = pending
    try {
      if (kind === 'delete') {
        await remove.mutateAsync({ id: group.id, name: group.name })
        toast.success(t('gateway.push.toast.groupDeleted'))
      } else {
        await action.mutateAsync({ id: group.id, name: group.name, action: kind })
        toast.success(
          kind === 'start'
            ? t('gateway.push.group.toast.started')
            : kind === 'stop'
              ? t('gateway.push.group.toast.stopped')
              : t('gateway.push.group.toast.restarted'),
        )
      }
      setPending(null)
    } catch (error) {
      showErrorToast(error)
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="app-page-toolbar">
        <SearchSelect
          id="push-group-connector-filter"
          value={activeConnectorId}
          options={connectorOptions}
          onChange={selectConnector}
          ariaLabel={t('gateway.push.group.selectConnector')}
          emptyText={t('common.state.empty')}
          className="w-[200px]"
        />
        <SearchInput
          wrapperClassName="w-[200px]"
          value={keywordInput}
          placeholder={t('gateway.push.group.searchPlaceholder')}
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
          disabled={!activeConnectorId || noConnectors || connectors.isLoading}
          icon={<Plus className="size-3.5" />}
          onClick={openCreate}
        >
          {t('gateway.push.group.create')}
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
            disabled={groups.isFetching || groups.isLoading}
          />
        }
      >
          {connectors.isLoading ? (
            <TableSkeleton rows={6} cols={11} />
          ) : connectors.isError ? (
            <div className="p-5">
              <ErrorState
                error={connectors.error}
                onRetry={() => void connectors.refetch()}
                compact
              />
            </div>
          ) : noConnectors || !activeConnectorId ? null : groups.isLoading ? (
            <TableSkeleton rows={6} cols={11} />
          ) : groups.isError ? (
            <div className="p-5">
              <ErrorState error={groups.error} onRetry={() => void groups.refetch()} compact />
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
                <col style={{ width: widths.connector }} />
                <col style={{ width: widths.name }} />
                <col style={{ width: widths.status }} />
                <col style={{ width: widths.dataTopic }} />
                <col style={{ width: widths.commandTopic }} />
                <col style={{ width: widths.interval }} />
                <col style={{ width: widths.scope }} />
                <col style={{ width: widths.lastPush }} />
                <col style={{ width: widths.pending }} />
                <col style={{ width: widths.actions }} />
              </colgroup>
              <DataTableHead>
                <tr>
                  <DataTableHeadCell className="text-center">
                    {t('common.field.order')}
                    <ColumnResizeHandle column="order" width={widths.order} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.push.columns.connector')}
                    <ColumnResizeHandle
                      column="connector"
                      width={widths.connector}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('common.field.name')}
                    <ColumnResizeHandle column="name" width={widths.name} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.push.columns.pushStatus')}
                    <ColumnResizeHandle column="status" width={widths.status} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.push.field.dataTopic')}
                    <ColumnResizeHandle
                      column="dataTopic"
                      width={widths.dataTopic}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.push.field.commandTopic')}
                    <ColumnResizeHandle
                      column="commandTopic"
                      width={widths.commandTopic}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.push.columns.interval')}
                    <ColumnResizeHandle
                      column="interval"
                      width={widths.interval}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.push.columns.scope')}
                    <ColumnResizeHandle column="scope" width={widths.scope} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.push.payload.lastPushTime')}
                    <ColumnResizeHandle
                      column="lastPush"
                      width={widths.lastPush}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.push.columns.pending')}
                    <ColumnResizeHandle
                      column="pending"
                      width={widths.pending}
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
                {list.map((group, index) => {
                  return (
                    <DataTableRow
                      key={group.id}
                      data-group-id={group.id}
                      className={cn(
                        focusedGroupId === group.id && 'bg-primary-soft/40 ring-1 ring-primary/25',
                      )}
                    >
                      <DataTableCell className="text-center">
                        {(page - 1) * pageSize + index + 1}
                      </DataTableCell>
                      <DataTableCell>
                        {connectorById.get(group.connectorId)?.name ?? '—'}
                      </DataTableCell>
                      <DataTableCell>
                        <span className="truncate">{group.name}</span>
                      </DataTableCell>
                      <DataTableCell>
                        <ConnectorStatusBadge
                          status={group.status}
                          error={
                            [
                              group.pushError
                                ? t('gateway.push.error.push', { message: group.pushError })
                                : '',
                              group.subscriptionError
                                ? t('gateway.push.error.subscription', {
                                    message: group.subscriptionError,
                                  })
                                : '',
                            ]
                              .filter(Boolean)
                              .join('；') || group.lastError
                          }
                        />
                      </DataTableCell>
                      <DataTableCell>
                        <span className="block truncate" title={group.dataTopic}>
                          {group.dataTopic || '—'}
                        </span>
                      </DataTableCell>
                      <DataTableCell>
                        <span
                          className="block truncate"
                          title={group.commandTopic}
                        >
                          {group.commandTopic || '—'}
                        </span>
                      </DataTableCell>
                      <DataTableCell className="whitespace-nowrap">
                        {group.pushIntervalMs}
                      </DataTableCell>
                      <DataTableCell
                        role="button"
                        tabIndex={0}
                        onClick={() => setScopeGroup(group)}
                        onKeyDown={(event) => {
                          if (event.key === 'Enter' || event.key === ' ') {
                            event.preventDefault()
                            setScopeGroup(group)
                          }
                        }}
                        className="cursor-pointer transition-colors hover:bg-surface-2"
                      >
                        {group.allCollections ? (
                          <EnumBadge tone="info">{t('gateway.push.scope.all')}</EnumBadge>
                        ) : (
                          <EnumBadge tone="neutral">
                            {t('gateway.push.scope.selected', {
                              count: group.collectionIds?.length ?? 0,
                            })}
                          </EnumBadge>
                        )}
                      </DataTableCell>
                      <DataTableCell className="whitespace-nowrap">
                        {group.lastPushTime ? formatDateTime(group.lastPushTime) : '—'}
                      </DataTableCell>
                      <DataTableCell className="whitespace-nowrap">
                        {group.pendingMessages > 0 ? (
                          <span
                            title={`${t('gateway.push.pending.batches', { count: group.pendingBatches })}${
                              group.oldestPendingTime
                                ? ` · ${t('gateway.push.pending.oldest', {
                                    time: formatDateTime(group.oldestPendingTime),
                                  })}`
                                : ''
                            }`}
                          >
                            {group.pendingMessages}
                          </span>
                        ) : (
                          '—'
                        )}
                      </DataTableCell>
                      <DataTableCell>
                        <TableActions>
                          <TableActionButton
                            label={t('gateway.collection.action.update')}
                            icon={<Pencil />}
                            disabled={group.runState !== 'stopped'}
                            disabledReason={t('gateway.push.action.editRequiresStopped')}
                            onClick={() => openEdit(group)}
                          />
                          <TableActionButton
                            label={t('common.action.delete')}
                            icon={<Trash2 />}
                            onClick={() => setPending({ kind: 'delete', group })}
                          />
                          <PushGroupActionsMenu
                            group={group}
                            connectorStatus={connectorById.get(group.connectorId)?.status}
                            disabled={action.isPending}
                            onStart={() => setPending({ kind: 'start', group })}
                            onStop={() => setPending({ kind: 'stop', group })}
                            onRestart={() => setPending({ kind: 'restart', group })}
                            onViewPayloads={() => setPayloadGroup(group)}
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

      <PushGroupFormDialog
        open={formOpen}
        group={editing}
        connectors={connectorList}
        defaultConnectorId={activeConnectorId || undefined}
        onClose={() => setFormOpen(false)}
      />

      <PushGroupPayloadDialog
        open={Boolean(payloadGroup)}
        group={payloadGroup}
        connector={
          payloadGroup ? connectorById.get(payloadGroup.connectorId) : undefined
        }
        onClose={() => setPayloadGroup(null)}
      />

      <Dialog
        open={Boolean(scopeGroup)}
        onClose={() => setScopeGroup(null)}
        title={t('gateway.push.group.scopeTitle')}
        className="w-[760px] max-w-[calc(100vw-2rem)] rounded-lg"
        footer={
          <Button variant="outline" onClick={() => setScopeGroup(null)}>
            {t('common.action.close')}
          </Button>
        }
      >
        {scopeCollections.length === 0 ? (
          <p className="py-8 text-center text-sm text-faint">{t('gateway.push.scope.empty')}</p>
        ) : (
          <div className="max-h-[50dvh] overflow-auto">
            <Table>
              <THead>
                <TR>
                  <TH className="w-10">
                    <input
                      type="checkbox"
                      aria-label={t('common.action.selectAll')}
                      checked
                      readOnly
                    />
                  </TH>
                  <TH>{t('gateway.push.group.collectionName')}</TH>
                  <TH>{t('gateway.collection.columns.status')}</TH>
                  <TH>{t('gateway.collection.columns.driverType')}</TH>
                  <TH>{t('gateway.push.group.protocolType')}</TH>
                </TR>
              </THead>
              <TBody>
                {scopeCollections.map((collection) => (
                  <TR key={collection.id}>
                    <TD>
                      <input
                        type="checkbox"
                        aria-label={collection.name}
                        checked
                        readOnly
                      />
                    </TD>
                    <TD className="max-w-[220px] truncate text-fg">{collection.name}</TD>
                    <TD>
                      <CollectStatusBadge
                        status={collection.collectStatus}
                        error={collection.lastError}
                      />
                    </TD>
                    <TD className="text-muted">
                      {driverTypeLabel(collection.driverType) || '—'}
                    </TD>
                    <TD className="text-muted">
                      {driverModelLabel(collection.driverModel) || '—'}
                    </TD>
                  </TR>
                ))}
              </TBody>
            </Table>
          </div>
        )}
      </Dialog>

      <ConfirmDialog
        open={Boolean(pending)}
        onClose={() => setPending(null)}
        onConfirm={() => void confirm()}
        loading={remove.isPending || action.isPending}
        danger={pending?.kind === 'delete'}
        title={
          pending?.kind === 'delete'
            ? t('gateway.push.group.deleteTitle')
            : pending?.kind === 'start'
              ? t('gateway.push.group.confirm.startTitle')
              : pending?.kind === 'stop'
                ? t('gateway.push.group.confirm.stopTitle')
                : t('gateway.push.group.confirm.restartTitle')
        }
        description={
          pending?.kind === 'delete'
            ? t('gateway.push.group.deleteDescription', { name: pending.group.name })
            : pending?.kind === 'start'
              ? t('gateway.push.group.confirm.startDescription')
              : pending?.kind === 'stop'
                ? t('gateway.push.group.confirm.stopDescription')
                : t('gateway.push.group.confirm.restartDescription')
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
