import { useEffect, useMemo, useState } from 'react'
import { Boxes, Plus, Search } from 'lucide-react'
import { useNavigate, useSearchParams } from 'react-router-dom'
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
  SearchSelect,
  TableSkeleton,
  formatDateTime,
  showErrorToast,
  cn,
  useI18n,
  type SearchSelectOption,
} from '@harns/core'
import {
  useDeletePushGroup,
  usePushConnectors,
  usePushGroupAction,
  usePushGroups,
} from '@/api/hooks'
import type { PushGroup } from '@/api/types'
import {
  GatewayTBody,
  GatewayTD,
  GatewayTH,
  GatewayTHead,
  GatewayTR,
  GatewayTable,
} from '@/components/GatewayTable'
import { ConnectorStatusBadge } from '@/components/StatusBadge'
import { ColumnResizeHandle, useColumnResize } from '@/components/useColumnResize'
import { PushGroupActionsMenu } from './PushGroupActionsMenu'
import { PushGroupFormDialog } from './PushGroupFormDialog'
import { PushGroupPayloadDialog } from './PushGroupPayloadDialog'

type PendingAction = { kind: 'delete' | 'start' | 'stop' | 'restart'; group: PushGroup }

export function PushGroupsPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [selectedConnectorId, setSelectedConnectorId] = useState('')
  const focusedGroupId = searchParams.get('groupId') ?? ''
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<PushGroup | null>(null)
  const [payloadGroup, setPayloadGroup] = useState<PushGroup | null>(null)
  const [pending, setPending] = useState<PendingAction | null>(null)

  const connectors = usePushConnectors({ page: 1, pageSize: 100 })
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
  const activeConnectorId = useMemo(() => {
    if (connectorList.some((item) => item.id === selectedConnectorId)) return selectedConnectorId
    const fromQuery = searchParams.get('connectorId') ?? ''
    if (connectorList.some((item) => item.id === fromQuery)) return fromQuery
    return connectorList[0]?.id ?? ''
  }, [connectorList, searchParams, selectedConnectorId])
  const groups = usePushGroups({
    page,
    pageSize,
    keyword: keyword || undefined,
    connectorId: activeConnectorId || undefined,
  })
  const list = groups.data?.list ?? []
  const total = groups.data?.total ?? 0
  const noConnectors = !connectors.isLoading && connectorList.length === 0
  const { widths, totalWidth, resize } = useColumnResize({
    order: 80,
    name: 220,
    connector: 180,
    status: 120,
    dataTopic: 200,
    interval: 120,
    scope: 100,
    max: 140,
    lastPush: 160,
    actions: 180,
  })

  useEffect(() => {
    if (!focusedGroupId) return
    const element = document.querySelector<HTMLElement>(
      `[data-group-id="${focusedGroupId}"]`,
    )
    element?.scrollIntoView({ block: 'center' })
  }, [focusedGroupId, list])

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    setPage(1)
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
      <div className="mb-3 flex shrink-0 flex-wrap items-center gap-2">
        <label htmlFor="push-group-connector-filter" className="text-sm text-muted">
          {t('gateway.push.columns.connector')}
        </label>
        <SearchSelect
          id="push-group-connector-filter"
          value={activeConnectorId}
          options={connectorOptions}
          onChange={selectConnector}
          ariaLabel={t('gateway.push.columns.connector')}
          emptyText={t('common.state.empty')}
          className="w-[200px]"
          panelClassName="w-[200px]"
        />
        <div className="relative w-[200px]">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
          <Input
            value={keywordInput}
            placeholder={t('gateway.push.group.searchPlaceholder')}
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
          disabled={noConnectors || connectors.isLoading}
          icon={<Plus className="size-3.5" />}
          onClick={openCreate}
        >
          {t('gateway.push.group.create')}
        </Button>
      </div>

      <Card className="flex min-h-0 flex-1 flex-col overflow-hidden">
        <div className="min-h-0 flex-1 overflow-auto">
          {connectors.isLoading ? (
            <TableSkeleton rows={6} cols={10} />
          ) : connectors.isError ? (
            <div className="p-5">
              <ErrorState
                error={connectors.error}
                onRetry={() => void connectors.refetch()}
                compact
              />
            </div>
          ) : noConnectors ? (
            <EmptyState
              icon={<Boxes className="size-5" />}
              title={t('gateway.push.group.noConnectorTitle')}
              description={t('gateway.push.group.noConnectorDescription')}
              action={
                <Button variant="primary" onClick={() => navigate('/push')}>
                  {t('gateway.push.group.goConnectors')}
                </Button>
              }
            />
          ) : groups.isLoading ? (
            <TableSkeleton rows={6} cols={10} />
          ) : groups.isError ? (
            <div className="p-5">
              <ErrorState error={groups.error} onRetry={() => void groups.refetch()} compact />
            </div>
          ) : list.length === 0 ? (
            <div className="flex h-full min-h-40 items-center justify-center text-sm text-faint">
              {t('common.state.empty')}
            </div>
          ) : (
            <GatewayTable
              style={{
                tableLayout: 'fixed',
                width: `max(100%, ${totalWidth}px)`,
              }}
            >
              <colgroup>
                <col style={{ width: widths.order }} />
                <col style={{ width: widths.name }} />
                <col style={{ width: widths.connector }} />
                <col style={{ width: widths.status }} />
                <col style={{ width: widths.dataTopic }} />
                <col style={{ width: widths.interval }} />
                <col style={{ width: widths.scope }} />
                <col style={{ width: widths.max }} />
                <col style={{ width: widths.lastPush }} />
                <col style={{ width: widths.actions }} />
              </colgroup>
              <GatewayTHead>
                <tr>
                  <GatewayTH className="text-center">
                    {t('common.field.order')}
                    <ColumnResizeHandle column="order" width={widths.order} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('common.field.name')}
                    <ColumnResizeHandle column="name" width={widths.name} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.push.columns.connector')}
                    <ColumnResizeHandle
                      column="connector"
                      width={widths.connector}
                      onResize={resize}
                    />
                  </GatewayTH>
                  <GatewayTH>
                    {t('common.field.status')}
                    <ColumnResizeHandle column="status" width={widths.status} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.push.field.dataTopic')}
                    <ColumnResizeHandle
                      column="dataTopic"
                      width={widths.dataTopic}
                      onResize={resize}
                    />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.push.columns.interval')}
                    <ColumnResizeHandle
                      column="interval"
                      width={widths.interval}
                      onResize={resize}
                    />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.push.columns.scope')}
                    <ColumnResizeHandle column="scope" width={widths.scope} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.push.columns.maxPerMessage')}
                    <ColumnResizeHandle column="max" width={widths.max} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.push.payload.lastPushTime')}
                    <ColumnResizeHandle
                      column="lastPush"
                      width={widths.lastPush}
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
                {list.map((group, index) => {
                  return (
                    <GatewayTR
                      key={group.id}
                      data-group-id={group.id}
                      className={cn(
                        focusedGroupId === group.id && 'bg-primary-soft/40 ring-1 ring-primary/25',
                      )}
                    >
                      <GatewayTD className="text-center text-muted">
                        {(page - 1) * pageSize + index + 1}
                      </GatewayTD>
                      <GatewayTD>
                        <span className="truncate text-fg">{group.name}</span>
                      </GatewayTD>
                      <GatewayTD className="text-muted">
                        {connectorById.get(group.connectorId)?.name ?? '—'}
                      </GatewayTD>
                      <GatewayTD>
                        <ConnectorStatusBadge
                          status={group.status}
                          error={group.lastError}
                          className="border-[1.5px] px-2 py-1"
                        />
                      </GatewayTD>
                      <GatewayTD>
                        <span className="block truncate font-mono text-muted" title={group.dataTopic}>
                          {group.dataTopic || '—'}
                        </span>
                      </GatewayTD>
                      <GatewayTD className="whitespace-nowrap text-muted">
                        {group.pushIntervalMs} ms
                      </GatewayTD>
                      <GatewayTD>
                        {group.allCollections ? (
                          <Badge tone="info">{t('gateway.push.scope.all')}</Badge>
                        ) : (
                          <Badge tone="neutral">
                            {t('gateway.push.scope.selected', {
                              count: group.collectionIds?.length ?? 0,
                            })}
                          </Badge>
                        )}
                      </GatewayTD>
                      <GatewayTD className="text-muted">
                        {group.maxCollectionsPerPushMessage}
                      </GatewayTD>
                      <GatewayTD className="whitespace-nowrap text-muted">
                        {group.lastPushTime ? formatDateTime(group.lastPushTime) : '—'}
                      </GatewayTD>
                      <GatewayTD>
                        <div className="flex items-center gap-2">
                          <button
                            type="button"
                            onClick={() => openEdit(group)}
                            className="inline-flex h-7 items-center rounded text-sm text-primary transition-colors hover:text-primary-hover"
                          >
                            {t('gateway.collection.action.update')}
                          </button>
                          <button
                            type="button"
                            onClick={() => setPending({ kind: 'delete', group })}
                            className="inline-flex h-7 items-center rounded text-sm text-danger transition-colors hover:opacity-80"
                          >
                            {t('common.action.delete')}
                          </button>
                          <PushGroupActionsMenu
                            group={group}
                            connectorStatus={connectorById.get(group.connectorId)?.status}
                            disabled={action.isPending}
                            onStart={() => setPending({ kind: 'start', group })}
                            onStop={() => setPending({ kind: 'stop', group })}
                            onRestart={() => setPending({ kind: 'restart', group })}
                            onViewPayloads={() => setPayloadGroup(group)}
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
            disabled={groups.isFetching || groups.isLoading}
          />
        </div>
      </Card>

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
