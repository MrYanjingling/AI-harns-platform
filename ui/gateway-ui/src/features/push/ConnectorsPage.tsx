import { useState } from 'react'
import { Pencil, Plus, Pointer, Search, Trash2 } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import {
  Button,
  Card,
  ConfirmDialog,
  ErrorState,
  Input,
  Pagination,
  SearchSelect,
  TableSkeleton,
  formatDateTime,
  showErrorToast,
  useI18n,
} from '@harns/core'
import {
  useDeletePushConnector,
  usePushConnectorAction,
  usePushConnectors,
} from '@/api/hooks'
import type { PushConnector } from '@/api/types'
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
import { TableActionButton } from '@/components/TableActionButton'
import { ConnectorActionsMenu, type ConnectorOperation } from './ConnectorActionsMenu'
import { ConnectorFormDialog } from './ConnectorFormDialog'
import { PushGroupFormDialog } from './PushGroupFormDialog'

type PendingAction = {
  kind: ConnectorOperation | 'delete'
  connector: PushConnector
}

export function ConnectorsPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [connectorType, setConnectorType] = useState('all')
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<PushConnector | null>(null)
  const [groupConnector, setGroupConnector] = useState<PushConnector | null>(null)
  const [pending, setPending] = useState<PendingAction | null>(null)
  const [cascade, setCascade] = useState(false)

  const connectors = usePushConnectors({
    page,
    pageSize,
    keyword: keyword || undefined,
    type: connectorType === 'all' ? undefined : connectorType,
  })
  const allConnectors = usePushConnectors({ page: 1, pageSize: 100 })
  const action = usePushConnectorAction()
  const remove = useDeletePushConnector()
  const list = connectors.data?.list ?? []
  const total = connectors.data?.total ?? 0
  const { widths, totalWidth, resize } = useColumnResize({
    order: 80,
    name: 220,
    status: 130,
    type: 100,
    broker: 240,
    groups: 100,
    updatedAt: 180,
    actions: 140,
  })

  const typeOptions = [
    { value: 'all', label: t('gateway.push.filter.allTypes') },
    { value: 'mqtt', label: 'MQTT' },
    { value: 'kafka', label: 'KAFKA' },
  ]

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    setPage(1)
  }

  const resetSearch = () => {
    setKeywordInput('')
    setKeyword('')
    setConnectorType('all')
    setPage(1)
  }

  const openCreate = () => {
    setEditing(null)
    setFormOpen(true)
  }

  const openEdit = (connector: PushConnector) => {
    setEditing(connector)
    setFormOpen(true)
  }

  const openGroups = (connector: PushConnector) => {
    navigate(`/push/groups?connectorId=${encodeURIComponent(connector.id)}`)
  }

  const confirm = async () => {
    if (!pending) return
    const { kind, connector } = pending
    try {
      if (kind === 'delete') {
        await remove.mutateAsync({ id: connector.id, name: connector.name, cascade })
        toast.success(t('gateway.push.toast.connectorDeleted'))
      } else {
        await action.mutateAsync({ id: connector.id, name: connector.name, action: kind })
        toast.success(
          kind === 'start'
            ? t('gateway.push.toast.started')
            : kind === 'stop'
              ? t('gateway.push.toast.stopped')
              : t('gateway.push.toast.restarted'),
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
        <SearchSelect
          id="push-connector-type-filter"
          value={connectorType}
          options={typeOptions}
          onChange={(next) => {
            setConnectorType(next)
            setPage(1)
          }}
          ariaLabel={t('gateway.push.filter.type')}
          emptyText={t('common.state.empty')}
          className="w-[140px]"
        />
        <div className="relative w-[200px]">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
          <Input
            value={keywordInput}
            placeholder={t('gateway.push.searchPlaceholder')}
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
          onClick={openCreate}
        >
          {t('gateway.push.connector.create')}
        </Button>
      </div>

      <Card className="flex min-h-0 flex-1 flex-col overflow-hidden">
        <div className="min-h-0 flex-1 overflow-auto">
          {connectors.isLoading ? (
            <TableSkeleton rows={6} cols={8} />
          ) : connectors.isError ? (
            <div className="p-5">
              <ErrorState
                error={connectors.error}
                onRetry={() => void connectors.refetch()}
                compact
              />
            </div>
          ) : list.length === 0 ? null : (
            <GatewayTable
              style={{
                tableLayout: 'fixed',
                width: `max(100%, ${totalWidth}px)`,
              }}
            >
              <colgroup>
                <col style={{ width: widths.order }} />
                <col style={{ width: widths.name }} />
                <col style={{ width: widths.status }} />
                <col style={{ width: widths.type }} />
                <col style={{ width: widths.broker }} />
                <col style={{ width: widths.groups }} />
                <col style={{ width: widths.updatedAt }} />
                <col style={{ width: widths.actions }} />
              </colgroup>
              <GatewayTHead>
                <tr>
                  <GatewayTH className="text-center">
                    {t('common.field.order')}
                    <ColumnResizeHandle column="order" width={widths.order} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.push.columns.name')}
                    <ColumnResizeHandle column="name" width={widths.name} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.push.columns.connectionStatus')}
                    <ColumnResizeHandle column="status" width={widths.status} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.push.columns.type')}
                    <ColumnResizeHandle column="type" width={widths.type} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.push.columns.broker')}
                    <ColumnResizeHandle column="broker" width={widths.broker} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.push.columns.groups')}
                    <ColumnResizeHandle column="groups" width={widths.groups} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('common.field.updatedAt')}
                    <ColumnResizeHandle
                      column="updatedAt"
                      width={widths.updatedAt}
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
                {list.map((connector, index) => (
                  <GatewayTR key={connector.id}>
                    <GatewayTD className="text-center">
                      {(page - 1) * pageSize + index + 1}
                    </GatewayTD>
                    <GatewayTD>
                      <span className="truncate">{connector.name}</span>
                    </GatewayTD>
                    <GatewayTD>
                      <ConnectorStatusBadge
                        status={connector.status}
                        error={connector.lastError}
                      />
                    </GatewayTD>
                    <GatewayTD>
                      {connector.type.toUpperCase()}
                    </GatewayTD>
                    <GatewayTD>
                      <span
                        className="block truncate"
                        title={connector.pushConfig?.brokerUrl ?? undefined}
                      >
                        {connector.pushConfig?.brokerUrl || '—'}
                      </span>
                    </GatewayTD>
                    <GatewayTD
                      role="button"
                      tabIndex={0}
                      onClick={() => openGroups(connector)}
                      onKeyDown={(event) => {
                        if (event.key === 'Enter' || event.key === ' ') {
                          event.preventDefault()
                          openGroups(connector)
                        }
                      }}
                      className="cursor-pointer transition-colors hover:bg-surface-2 hover:text-primary-hover"
                    >
                      <span className="inline-flex items-center gap-1">
                        {connector.groupCount ?? 0}
                        <Pointer className="size-3.5 text-faint" aria-hidden="true" />
                      </span>
                    </GatewayTD>
                    <GatewayTD className="whitespace-nowrap">
                      {formatDateTime(connector.updatedTime)}
                    </GatewayTD>
                    <GatewayTD>
                      <div className="flex items-center gap-2">
                        <TableActionButton
                          label={t('gateway.collection.action.update')}
                          icon={<Pencil />}
                          onClick={() => openEdit(connector)}
                        />
                        <TableActionButton
                          label={t('common.action.delete')}
                          icon={<Trash2 />}
                          onClick={() => {
                            setCascade(false)
                            setPending({ kind: 'delete', connector })
                          }}
                        />
                        <ConnectorActionsMenu
                          connector={connector}
                          disabled={action.isPending}
                          onAction={(kind) => setPending({ kind, connector })}
                          onCreateGroup={() => setGroupConnector(connector)}
                        />
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
            disabled={connectors.isFetching || connectors.isLoading}
          />
        </div>
      </Card>

      <ConnectorFormDialog
        open={formOpen}
        connector={editing}
        onClose={() => setFormOpen(false)}
      />

      <PushGroupFormDialog
        open={Boolean(groupConnector)}
        group={null}
        connectors={allConnectors.data?.list ?? list}
        defaultConnectorId={groupConnector?.id}
        onClose={() => setGroupConnector(null)}
      />

      <ConfirmDialog
        open={Boolean(pending)}
        onClose={() => setPending(null)}
        onConfirm={() => void confirm()}
        loading={action.isPending || remove.isPending}
        danger={pending?.kind === 'delete'}
        title={
          pending?.kind === 'delete'
            ? t('gateway.push.connector.deleteTitle')
            : pending?.kind === 'start'
              ? t('gateway.push.connector.confirm.startTitle')
              : pending?.kind === 'stop'
                ? t('gateway.push.connector.confirm.stopTitle')
                : t('gateway.push.connector.confirm.restartTitle')
        }
        description={
          pending?.kind === 'delete' ? (
            <div className="space-y-2">
              <p>
                {t('gateway.push.connector.deleteDescription', {
                  name: pending.connector.name,
                })}
              </p>
              <label className="flex items-center gap-2 text-xs text-muted">
                <input
                  type="checkbox"
                  checked={cascade}
                  onChange={(event) => setCascade(event.target.checked)}
                />
                {t('gateway.push.connector.cascade')}
              </label>
            </div>
          ) : pending?.kind === 'start' ? (
            t('gateway.push.connector.confirm.startDescription')
          ) : pending?.kind === 'stop' ? (
            t('gateway.push.connector.confirm.stopDescription')
          ) : (
            t('gateway.push.connector.confirm.restartDescription')
          )
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
