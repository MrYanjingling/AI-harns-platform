import { useMemo, useState } from 'react'
import {
  Pencil,
  Play,
  Plus,
  RadioTower,
  RefreshCw,
  Search,
  Square,
  Trash2,
  Waypoints,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Badge,
  Button,
  Card,
  ConfirmDialog,
  Dialog,
  EmptyState,
  ErrorState,
  Field,
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
  Tabs,
  errorMessage,
  formatDateTime,
  useI18n,
} from '@harns/core'
import {
  useCollections,
  useCreatePushConnector,
  useCreatePushGroup,
  useDeletePushConnector,
  useDeletePushGroup,
  usePushConnectorAction,
  usePushConnectors,
  usePushGroups,
  useUpdatePushConnector,
  useUpdatePushGroup,
} from '@/api/hooks'
import type {
  MQTTConfig,
  PushConnector,
  PushConnectorPayload,
  PushGroup,
  PushGroupPayload,
} from '@/api/types'
import { ConnectorStatusBadge } from '@/components/StatusBadge'

type TabValue = 'connectors' | 'groups'

interface ConnectorForm {
  name: string
  brokerUrl: string
  username: string
  password: string
  dataTopic: string
  commandTopic: string
  clientId: string
  qos: string
  retain: boolean
  keepAlive: string
}

interface GroupForm {
  name: string
  connectorId: string
  pushIntervalMs: string
  maxCollectionsPerPushMessage: string
  allCollections: boolean
  collectionIds: string[]
}

const EMPTY_CONNECTOR: ConnectorForm = {
  name: '',
  brokerUrl: '',
  username: '',
  password: '',
  dataTopic: '',
  commandTopic: '',
  clientId: '',
  qos: '0',
  retain: false,
  keepAlive: '60',
}

const EMPTY_GROUP: GroupForm = {
  name: '',
  connectorId: '',
  pushIntervalMs: '5000',
  maxCollectionsPerPushMessage: '50',
  allCollections: true,
  collectionIds: [],
}

function connectorToForm(connector: PushConnector): ConnectorForm {
  const config = connector.pushConfig ?? ({} as MQTTConfig)
  return {
    name: connector.name,
    brokerUrl: config.brokerUrl ?? '',
    username: config.username ?? '',
    password: config.password ?? '',
    dataTopic: config.dataTopic ?? '',
    commandTopic: config.commandTopic ?? '',
    clientId: config.clientId ?? '',
    qos: String(config.qos ?? 0),
    retain: Boolean(config.retain),
    keepAlive: String(config.keepAlive ?? 60),
  }
}

function connectorPayload(form: ConnectorForm): PushConnectorPayload {
  const config: MQTTConfig = {
    brokerUrl: form.brokerUrl.trim(),
    dataTopic: form.dataTopic.trim(),
  }
  if (form.username.trim()) config.username = form.username.trim()
  if (form.password.trim()) config.password = form.password.trim()
  if (form.commandTopic.trim()) config.commandTopic = form.commandTopic.trim()
  if (form.clientId.trim()) config.clientId = form.clientId.trim()
  config.qos = Number(form.qos)
  config.retain = form.retain
  const keepAlive = Number(form.keepAlive)
  if (Number.isFinite(keepAlive) && keepAlive > 0) config.keepAlive = keepAlive
  return { type: 'mqtt', name: form.name.trim(), pushConfig: config }
}

function groupToForm(group: PushGroup): GroupForm {
  return {
    name: group.name,
    connectorId: group.connectorId,
    pushIntervalMs: String(group.pushIntervalMs || 5000),
    maxCollectionsPerPushMessage: String(group.maxCollectionsPerPushMessage || 50),
    allCollections: group.allCollections,
    collectionIds: [...(group.collectionIds ?? [])],
  }
}

function groupPayload(form: GroupForm): PushGroupPayload {
  const payload: PushGroupPayload = {
    name: form.name.trim(),
    connectorId: form.connectorId,
    pushIntervalMs: Number(form.pushIntervalMs),
    allCollections: form.allCollections,
    collectionIds: form.allCollections ? [] : form.collectionIds,
  }
  const max = Number(form.maxCollectionsPerPushMessage)
  if (Number.isFinite(max) && max > 0) payload.maxCollectionsPerPushMessage = max
  return payload
}

function validateConnector(form: ConnectorForm, t: ReturnType<typeof useI18n>['t']) {
  const errors: Record<string, string> = {}
  if (!form.name.trim()) errors.name = t('gateway.form.validation.nameRequired')
  if (!form.brokerUrl.trim()) errors.brokerUrl = t('gateway.push.validation.brokerRequired')
  else if (!/^(mqtt|ws|wss|ssl|tcp):\/\/.+/i.test(form.brokerUrl.trim())) {
    errors.brokerUrl = t('gateway.push.validation.brokerFormat')
  }
  if (!form.dataTopic.trim()) errors.dataTopic = t('gateway.push.validation.dataTopicRequired')
  const keepAlive = Number(form.keepAlive)
  if (form.keepAlive.trim() !== '' && (!Number.isFinite(keepAlive) || keepAlive <= 0)) {
    errors.keepAlive = t('gateway.push.validation.keepAliveInvalid')
  }
  return errors
}

function validateGroup(
  form: GroupForm,
  t: ReturnType<typeof useI18n>['t'],
): Record<string, string> {
  const errors: Record<string, string> = {}
  if (!form.name.trim()) errors.name = t('gateway.form.validation.nameRequired')
  if (!form.connectorId) errors.connectorId = t('gateway.push.validation.connectorRequired')
  const interval = Number(form.pushIntervalMs)
  if (!Number.isFinite(interval) || interval <= 0) {
    errors.pushIntervalMs = t('gateway.push.validation.intervalInvalid')
  }
  const max = Number(form.maxCollectionsPerPushMessage)
  if (
    form.maxCollectionsPerPushMessage.trim() !== '' &&
    (!Number.isFinite(max) || max <= 0)
  ) {
    errors.maxCollectionsPerPushMessage = t('gateway.push.validation.maxInvalid')
  }
  if (!form.allCollections && form.collectionIds.length === 0) {
    errors.collectionIds = t('gateway.push.validation.collectionsRequired')
  }
  return errors
}

export function PushPage() {
  const { t } = useI18n()
  const [tab, setTab] = useState<TabValue>('connectors')

  return (
    <div className="space-y-3">
      <PageHeader
        title={t('gateway.push.title')}
        description={t('gateway.push.description')}
        actions={
          <Tabs
            items={[
              { value: 'connectors', label: t('gateway.push.tab.connectors') },
              { value: 'groups', label: t('gateway.push.tab.groups') },
            ]}
            value={tab}
            onChange={setTab}
          />
        }
      />
      {tab === 'connectors' ? <ConnectorsTab /> : <GroupsTab />}
    </div>
  )
}

function ConnectorsTab() {
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const connectors = usePushConnectors({ page, pageSize, keyword: keyword || undefined })
  const action = usePushConnectorAction()
  const create = useCreatePushConnector()
  const update = useUpdatePushConnector()
  const remove = useDeletePushConnector()

  const [editing, setEditing] = useState<PushConnector | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [form, setForm] = useState<ConnectorForm>(EMPTY_CONNECTOR)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [deleteTarget, setDeleteTarget] = useState<PushConnector | null>(null)
  const [cascade, setCascade] = useState(false)

  const list = connectors.data?.list ?? []

  const openCreate = () => {
    setEditing(null)
    setForm(EMPTY_CONNECTOR)
    setErrors({})
    setDialogOpen(true)
  }

  const openEdit = (connector: PushConnector) => {
    setEditing(connector)
    setForm(connectorToForm(connector))
    setErrors({})
    setDialogOpen(true)
  }

  const submit = async () => {
    const nextErrors = validateConnector(form, t)
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return
    try {
      if (editing) {
        await update.mutateAsync({ id: editing.id, payload: connectorPayload(form) })
        toast.success(t('gateway.push.toast.connectorUpdated'))
      } else {
        await create.mutateAsync(connectorPayload(form))
        toast.success(t('gateway.push.toast.connectorCreated'))
      }
      setDialogOpen(false)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const toggleStatus = async (connector: PushConnector) => {
    const nextAction = connector.status === 'running' ? 'stop' : 'start'
    try {
      await action.mutateAsync({ id: connector.id, name: connector.name, action: nextAction })
      toast.success(
        nextAction === 'start'
          ? t('gateway.push.toast.started')
          : t('gateway.push.toast.stopped'),
      )
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await remove.mutateAsync({
        id: deleteTarget.id,
        name: deleteTarget.name,
        cascade,
      })
      toast.success(t('gateway.push.toast.connectorDeleted'))
      setDeleteTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <>
      <div className="grid gap-2.5 sm:grid-cols-3">
        <StatCard
          label={t('gateway.push.stat.connectors')}
          value={connectors.data?.total ?? list.length}
          icon={<RadioTower className="size-3.5" />}
        />
        <StatCard
          label={t('gateway.push.stat.running')}
          value={list.filter((item) => item.status === 'running').length}
          icon={<Play className="size-3.5" />}
        />
        <StatCard
          label={t('gateway.push.stat.stopped')}
          value={list.filter((item) => item.status === 'stopped').length}
          icon={<Square className="size-3.5" />}
        />
      </div>

      <Card className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-2 border-b border-border p-2.5">
          <div className="relative min-w-52 flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-faint" />
            <Input
              value={keyword}
              onChange={(event) => {
                setKeyword(event.target.value)
                setPage(1)
              }}
              placeholder={t('gateway.push.searchPlaceholder')}
              aria-label={t('common.action.search')}
              className="pl-8"
            />
          </div>
          <Button
            variant="outline"
            loading={connectors.isFetching}
            onClick={() => void connectors.refetch()}
            icon={<RefreshCw className="size-3.5" />}
          >
            {t('common.action.refresh')}
          </Button>
          <Button variant="primary" onClick={openCreate} icon={<Plus className="size-3.5" />}>
            {t('gateway.push.connector.create')}
          </Button>
        </div>

        {connectors.isLoading ? (
          <TableSkeleton rows={5} cols={6} />
        ) : connectors.isError ? (
          <div className="p-3">
            <ErrorState error={connectors.error} onRetry={() => void connectors.refetch()} />
          </div>
        ) : list.length === 0 ? (
          <EmptyState
            icon={<RadioTower className="size-5" />}
            title={t('gateway.push.connector.emptyTitle')}
            description={t('gateway.push.connector.emptyDescription')}
            action={
              <Button variant="primary" onClick={openCreate} icon={<Plus className="size-3.5" />}>
                {t('gateway.push.connector.create')}
              </Button>
            }
          />
        ) : (
          <Table>
            <THead>
              <TR>
                <TH>{t('common.field.name')}</TH>
                <TH>{t('gateway.push.columns.broker')}</TH>
                <TH>{t('gateway.push.columns.dataTopic')}</TH>
                <TH>{t('gateway.push.columns.qos')}</TH>
                <TH>{t('common.field.status')}</TH>
                <TH>{t('common.field.updatedAt')}</TH>
                <TH className="text-right">{t('common.field.actions')}</TH>
              </TR>
            </THead>
            <TBody>
              {list.map((connector) => (
                <TR key={connector.id}>
                  <TD>
                    <p className="text-[13px] font-medium">{connector.name}</p>
                    <p className="font-mono text-[10px] text-faint">
                      {connector.type} · {connector.id}
                    </p>
                  </TD>
                  <TD className="max-w-56 truncate font-mono text-[11px] text-muted">
                    {connector.pushConfig?.brokerUrl ?? '—'}
                    {connector.pushConfig?.username ? (
                      <span className="ml-1 text-faint">
                        ({connector.pushConfig.username})
                      </span>
                    ) : null}
                  </TD>
                  <TD className="max-w-48 truncate font-mono text-[11px] text-muted">
                    {connector.pushConfig?.dataTopic ?? '—'}
                  </TD>
                  <TD className="tabular text-xs">{connector.pushConfig?.qos ?? 0}</TD>
                  <TD>
                    <ConnectorStatusBadge status={connector.status} />
                  </TD>
                  <TD className="whitespace-nowrap text-xs text-muted">
                    {formatDateTime(connector.updatedTime)}
                  </TD>
                  <TD>
                    <div className="flex items-center justify-end gap-0.5">
                      <button
                        type="button"
                        onClick={() => void toggleStatus(connector)}
                        disabled={action.isPending}
                        className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg disabled:opacity-50"
                        title={
                          connector.status === 'running'
                            ? t('common.action.stop')
                            : t('common.action.start')
                        }
                        aria-label={
                          connector.status === 'running'
                            ? t('common.action.stop')
                            : t('common.action.start')
                        }
                      >
                        {connector.status === 'running' ? (
                          <Square className="size-3.5" />
                        ) : (
                          <Play className="size-3.5" />
                        )}
                      </button>
                      <button
                        type="button"
                        onClick={() => openEdit(connector)}
                        className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
                        title={t('common.action.edit')}
                        aria-label={t('common.action.edit')}
                      >
                        <Pencil className="size-3.5" />
                      </button>
                      <button
                        type="button"
                        onClick={() => {
                          setCascade(false)
                          setDeleteTarget(connector)
                        }}
                        className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-danger/10 hover:text-danger"
                        title={t('common.action.delete')}
                        aria-label={t('common.action.delete')}
                      >
                        <Trash2 className="size-3.5" />
                      </button>
                    </div>
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
        )}

        <Pagination
          page={page}
          pageSize={pageSize}
          total={connectors.data?.total ?? 0}
          onPageChange={setPage}
          onPageSizeChange={(size) => {
            setPageSize(size)
            setPage(1)
          }}
          disabled={connectors.isLoading}
        />
      </Card>

      <Dialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        title={
          editing
            ? t('gateway.push.connector.editTitle')
            : t('gateway.push.connector.createTitle')
        }
        description={t('gateway.push.connector.dialogDescription')}
        size="lg"
        footer={
          <>
            <Button variant="ghost" onClick={() => setDialogOpen(false)}>
              {t('common.action.cancel')}
            </Button>
            <Button
              variant="primary"
              loading={create.isPending || update.isPending}
              onClick={() => void submit()}
            >
              {t('common.action.save')}
            </Button>
          </>
        }
      >
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label={t('common.field.name')} required error={errors.name}>
            <Input
              value={form.name}
              autoFocus
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>
          <Field label={t('gateway.push.field.type')}>
            <Input value="MQTT" readOnly disabled />
          </Field>
          <Field
            label={t('gateway.push.field.brokerUrl')}
            required
            error={errors.brokerUrl}
            hint={t('gateway.push.field.brokerHint')}
            className="sm:col-span-2"
          >
            <Input
              value={form.brokerUrl}
              placeholder="tcp://broker:1883"
              autoComplete="off"
              onChange={(event) => setForm({ ...form, brokerUrl: event.target.value })}
            />
          </Field>
          <Field label={t('gateway.push.field.username')}>
            <Input
              value={form.username}
              autoComplete="off"
              onChange={(event) => setForm({ ...form, username: event.target.value })}
            />
          </Field>
          <Field label={t('gateway.push.field.password')}>
            <Input
              type="password"
              value={form.password}
              autoComplete="new-password"
              placeholder={t('gateway.push.field.passwordPlaceholder')}
              onChange={(event) => setForm({ ...form, password: event.target.value })}
            />
          </Field>
          <Field label={t('gateway.push.field.dataTopic')} required error={errors.dataTopic}>
            <Input
              value={form.dataTopic}
              placeholder="data/{gatewayId}/v1/#"
              autoComplete="off"
              onChange={(event) => setForm({ ...form, dataTopic: event.target.value })}
            />
          </Field>
          <Field label={t('gateway.push.field.commandTopic')}>
            <Input
              value={form.commandTopic}
              placeholder="command/{gatewayId}/#"
              autoComplete="off"
              onChange={(event) => setForm({ ...form, commandTopic: event.target.value })}
            />
          </Field>
          <Field label={t('gateway.push.field.clientId')}>
            <Input
              value={form.clientId}
              autoComplete="off"
              onChange={(event) => setForm({ ...form, clientId: event.target.value })}
            />
          </Field>
          <Field label={t('gateway.push.field.qos')}>
            <Select
              value={form.qos}
              onChange={(event) => setForm({ ...form, qos: event.target.value })}
            >
              <option value="0">0</option>
              <option value="1">1</option>
              <option value="2">2</option>
            </Select>
          </Field>
          <Field
            label={t('gateway.push.field.keepAlive')}
            error={errors.keepAlive}
            hint={t('gateway.push.field.keepAliveHint')}
          >
            <Input
              value={form.keepAlive}
              inputMode="numeric"
              onChange={(event) => setForm({ ...form, keepAlive: event.target.value })}
            />
          </Field>
          <div className="flex items-center gap-2 self-end pb-1">
            <Switch
              checked={form.retain}
              label={t('gateway.push.field.retain')}
              onChange={(retain) => setForm({ ...form, retain })}
            />
            <div>
              <p className="text-xs font-medium text-muted">{t('gateway.push.field.retain')}</p>
              <p className="text-[11px] text-faint">{t('gateway.push.field.retainHint')}</p>
            </div>
          </div>
        </div>
      </Dialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => void confirmDelete()}
        loading={remove.isPending}
        danger
        title={t('gateway.push.connector.deleteTitle')}
        description={
          <div className="space-y-2">
            <p>{t('gateway.push.connector.deleteDescription', { name: deleteTarget?.name ?? '' })}</p>
            <label className="flex items-center gap-2 text-xs text-muted">
              <input
                type="checkbox"
                checked={cascade}
                onChange={(event) => setCascade(event.target.checked)}
              />
              {t('gateway.push.connector.cascade')}
            </label>
          </div>
        }
        confirmText={t('common.action.delete')}
      />
    </>
  )
}

function GroupsTab() {
  const { t } = useI18n()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [keyword, setKeyword] = useState('')
  const groups = usePushGroups({ page, pageSize, keyword: keyword || undefined })
  const connectors = usePushConnectors({ page: 1, pageSize: 100 })
  const collections = useCollections({ page: 1, pageSize: 100 })
  const create = useCreatePushGroup()
  const update = useUpdatePushGroup()
  const remove = useDeletePushGroup()

  const [editing, setEditing] = useState<PushGroup | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [form, setForm] = useState<GroupForm>(EMPTY_GROUP)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [deleteTarget, setDeleteTarget] = useState<PushGroup | null>(null)
  const [collectionQuery, setCollectionQuery] = useState('')

  const connectorMap = useMemo(
    () => new Map((connectors.data?.list ?? []).map((item) => [item.id, item.name])),
    [connectors.data],
  )
  const collectionList = useMemo(() => {
    const query = collectionQuery.trim().toLowerCase()
    const items = collections.data?.list ?? []
    if (!query) return items
    return items.filter((item) => item.name.toLowerCase().includes(query))
  }, [collections.data, collectionQuery])

  const openCreate = () => {
    setEditing(null)
    setForm({
      ...EMPTY_GROUP,
      connectorId: connectors.data?.list[0]?.id ?? '',
    })
    setErrors({})
    setCollectionQuery('')
    setDialogOpen(true)
  }

  const openEdit = (group: PushGroup) => {
    setEditing(group)
    setForm(groupToForm(group))
    setErrors({})
    setCollectionQuery('')
    setDialogOpen(true)
  }

  const submit = async () => {
    const nextErrors = validateGroup(form, t)
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return
    try {
      if (editing) {
        await update.mutateAsync({ id: editing.id, payload: groupPayload(form) })
        toast.success(t('gateway.push.toast.groupUpdated'))
      } else {
        await create.mutateAsync(groupPayload(form))
        toast.success(t('gateway.push.toast.groupCreated'))
      }
      setDialogOpen(false)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await remove.mutateAsync({ id: deleteTarget.id, name: deleteTarget.name })
      toast.success(t('gateway.push.toast.groupDeleted'))
      setDeleteTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const toggleCollection = (id: string) => {
    setForm((current) => ({
      ...current,
      collectionIds: current.collectionIds.includes(id)
        ? current.collectionIds.filter((item) => item !== id)
        : [...current.collectionIds, id],
    }))
  }

  const list = groups.data?.list ?? []

  return (
    <>
      <div className="grid gap-2.5 sm:grid-cols-3">
        <StatCard
          label={t('gateway.push.stat.groups')}
          value={groups.data?.total ?? list.length}
          icon={<Waypoints className="size-3.5" />}
        />
        <StatCard
          label={t('gateway.push.stat.allCollections')}
          value={list.filter((item) => item.allCollections).length}
        />
        <StatCard
          label={t('gateway.push.stat.connectors')}
          value={connectors.data?.total ?? 0}
          icon={<RadioTower className="size-3.5" />}
        />
      </div>

      <Card className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-2 border-b border-border p-2.5">
          <div className="relative min-w-52 flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-faint" />
            <Input
              value={keyword}
              onChange={(event) => {
                setKeyword(event.target.value)
                setPage(1)
              }}
              placeholder={t('gateway.push.group.searchPlaceholder')}
              aria-label={t('common.action.search')}
              className="pl-8"
            />
          </div>
          <Button
            variant="outline"
            loading={groups.isFetching}
            onClick={() => void groups.refetch()}
            icon={<RefreshCw className="size-3.5" />}
          >
            {t('common.action.refresh')}
          </Button>
          <Button
            variant="primary"
            onClick={openCreate}
            disabled={connectors.data?.total === 0}
            icon={<Plus className="size-3.5" />}
          >
            {t('gateway.push.group.create')}
          </Button>
        </div>

        {connectors.data?.total === 0 ? (
          <EmptyState
            icon={<RadioTower className="size-5" />}
            title={t('gateway.push.group.noConnectorTitle')}
            description={t('gateway.push.group.noConnectorDescription')}
          />
        ) : groups.isLoading ? (
          <TableSkeleton rows={5} cols={7} />
        ) : groups.isError ? (
          <div className="p-3">
            <ErrorState error={groups.error} onRetry={() => void groups.refetch()} />
          </div>
        ) : list.length === 0 ? (
          <EmptyState
            icon={<Waypoints className="size-5" />}
            title={t('gateway.push.group.emptyTitle')}
            description={t('gateway.push.group.emptyDescription')}
            action={
              <Button variant="primary" onClick={openCreate} icon={<Plus className="size-3.5" />}>
                {t('gateway.push.group.create')}
              </Button>
            }
          />
        ) : (
          <Table>
            <THead>
              <TR>
                <TH>{t('common.field.name')}</TH>
                <TH>{t('gateway.push.columns.connector')}</TH>
                <TH>{t('gateway.push.columns.interval')}</TH>
                <TH>{t('gateway.push.columns.scope')}</TH>
                <TH>{t('gateway.push.columns.maxPerMessage')}</TH>
                <TH>{t('common.field.updatedAt')}</TH>
                <TH className="text-right">{t('common.field.actions')}</TH>
              </TR>
            </THead>
            <TBody>
              {list.map((group) => (
                <TR key={group.id}>
                  <TD>
                    <p className="text-[13px] font-medium">{group.name}</p>
                    <p className="font-mono text-[10px] text-faint">{group.id}</p>
                  </TD>
                  <TD className="text-xs text-muted">
                    {connectorMap.get(group.connectorId) ?? (
                      <span className="font-mono text-[11px]">{group.connectorId}</span>
                    )}
                  </TD>
                  <TD className="tabular text-xs">{group.pushIntervalMs} ms</TD>
                  <TD>
                    {group.allCollections ? (
                      <Badge tone="info">{t('gateway.push.scope.all')}</Badge>
                    ) : (
                      <Badge tone="neutral">
                        {t('gateway.push.scope.selected', {
                          count: group.collectionIds?.length ?? 0,
                        })}
                      </Badge>
                    )}
                  </TD>
                  <TD className="tabular text-xs">{group.maxCollectionsPerPushMessage}</TD>
                  <TD className="whitespace-nowrap text-xs text-muted">
                    {formatDateTime(group.updatedTime)}
                  </TD>
                  <TD>
                    <div className="flex items-center justify-end gap-0.5">
                      <button
                        type="button"
                        onClick={() => openEdit(group)}
                        className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
                        title={t('common.action.edit')}
                        aria-label={t('common.action.edit')}
                      >
                        <Pencil className="size-3.5" />
                      </button>
                      <button
                        type="button"
                        onClick={() => setDeleteTarget(group)}
                        className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-danger/10 hover:text-danger"
                        title={t('common.action.delete')}
                        aria-label={t('common.action.delete')}
                      >
                        <Trash2 className="size-3.5" />
                      </button>
                    </div>
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
        )}

        <Pagination
          page={page}
          pageSize={pageSize}
          total={groups.data?.total ?? 0}
          onPageChange={setPage}
          onPageSizeChange={(size) => {
            setPageSize(size)
            setPage(1)
          }}
          disabled={groups.isLoading}
        />
      </Card>

      <Dialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        title={editing ? t('gateway.push.group.editTitle') : t('gateway.push.group.createTitle')}
        description={t('gateway.push.group.dialogDescription')}
        size="lg"
        footer={
          <>
            <Button variant="ghost" onClick={() => setDialogOpen(false)}>
              {t('common.action.cancel')}
            </Button>
            <Button
              variant="primary"
              loading={create.isPending || update.isPending}
              onClick={() => void submit()}
            >
              {t('common.action.save')}
            </Button>
          </>
        }
      >
        <div className="space-y-3">
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label={t('common.field.name')} required error={errors.name}>
              <Input
                value={form.name}
                autoFocus
                onChange={(event) => setForm({ ...form, name: event.target.value })}
              />
            </Field>
            <Field
              label={t('gateway.push.columns.connector')}
              required
              error={errors.connectorId}
            >
              <Select
                value={form.connectorId}
                onChange={(event) => setForm({ ...form, connectorId: event.target.value })}
              >
                <option value="">{t('gateway.form.selectPlaceholder')}</option>
                {(connectors.data?.list ?? []).map((connector) => (
                  <option key={connector.id} value={connector.id}>
                    {connector.name} · {connector.pushConfig?.brokerUrl ?? ''}
                  </option>
                ))}
              </Select>
            </Field>
            <Field
              label={t('gateway.push.columns.interval')}
              error={errors.pushIntervalMs}
              hint={t('gateway.push.group.intervalHint')}
            >
              <Input
                value={form.pushIntervalMs}
                inputMode="numeric"
                onChange={(event) => setForm({ ...form, pushIntervalMs: event.target.value })}
              />
            </Field>
            <Field
              label={t('gateway.push.columns.maxPerMessage')}
              error={errors.maxCollectionsPerPushMessage}
              hint={t('gateway.push.group.maxHint')}
            >
              <Input
                value={form.maxCollectionsPerPushMessage}
                inputMode="numeric"
                onChange={(event) =>
                  setForm({ ...form, maxCollectionsPerPushMessage: event.target.value })
                }
              />
            </Field>
          </div>

          <div className="flex items-center gap-2 rounded border border-border bg-surface-2 px-3 py-2">
            <Switch
              checked={form.allCollections}
              label={t('gateway.push.scope.all')}
              onChange={(allCollections) => setForm({ ...form, allCollections })}
            />
            <div>
              <p className="text-xs font-medium text-muted">{t('gateway.push.group.allCollections')}</p>
              <p className="text-[11px] text-faint">{t('gateway.push.group.allCollectionsHint')}</p>
            </div>
          </div>

          {!form.allCollections ? (
            <div className="space-y-2">
              <div className="flex items-center justify-between gap-2">
                <p className="text-xs font-medium text-muted">
                  {t('gateway.push.group.selectCollections', {
                    count: form.collectionIds.length,
                  })}
                </p>
                <Input
                  value={collectionQuery}
                  onChange={(event) => setCollectionQuery(event.target.value)}
                  placeholder={t('gateway.push.group.filterCollections')}
                  aria-label={t('gateway.push.group.filterCollections')}
                  className="w-44"
                />
              </div>
              <div className="max-h-56 overflow-y-auto rounded border border-border">
                {collectionList.length === 0 ? (
                  <p className="px-3 py-4 text-center text-xs text-faint">
                    {t('gateway.push.group.noCollections')}
                  </p>
                ) : (
                  collectionList.map((collection) => {
                    const checked = form.collectionIds.includes(collection.id)
                    return (
                      <label
                        key={collection.id}
                        className="flex cursor-pointer items-center gap-2 border-b border-border px-3 py-1.5 text-xs last:border-b-0 hover:bg-surface-2"
                      >
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => toggleCollection(collection.id)}
                        />
                        <span className="min-w-0 flex-1 truncate text-fg">{collection.name}</span>
                        <span className="font-mono text-[10px] text-faint">{collection.id}</span>
                      </label>
                    )
                  })
                )}
              </div>
              {errors.collectionIds ? (
                <p className="text-xs text-danger">{errors.collectionIds}</p>
              ) : null}
            </div>
          ) : null}
        </div>
      </Dialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => void confirmDelete()}
        loading={remove.isPending}
        danger
        title={t('gateway.push.group.deleteTitle')}
        description={t('gateway.push.group.deleteDescription', {
          name: deleteTarget?.name ?? '',
        })}
        confirmText={t('common.action.delete')}
      />
    </>
  )
}
