import { useState } from 'react'
import { Download, X } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  CardContent,
  ConfirmDialog,
  DataTable,
  DataTableBody,
  DataTableCell,
  DataTableHead,
  DataTableHeadCell,
  DataTableRow,
  EnumBadge,
  ErrorState,
  Field,
  Input,
  ListTableCard,
  Skeleton,
  Switch,
  TableActions,
  TableActionButton,
  ColumnResizeHandle,
  formatDateTime,
  showErrorToast,
  useColumnResize,
  useI18n,
} from '@harns/core'
import {
  useApplySystemUpdate,
  useRejectSystemUpdate,
  useSystemUpdate,
  useUpdateSystemUpdateSettings,
} from '@/api/hooks'
import { OtaStatusBadge } from '@/components/StatusBadge'

type PendingAction = 'apply' | 'reject'

export function SystemUpdatePage() {
  const { t } = useI18n()
  const update = useSystemUpdate()
  const apply = useApplySystemUpdate()
  const reject = useRejectSystemUpdate()
  const saveSettings = useUpdateSystemUpdateSettings()
  const [pending, setPending] = useState<PendingAction | null>(null)
  const [draftOpen, setDraftOpen] = useState(false)
  const [brokerUrl, setBrokerUrl] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [settingsError, setSettingsError] = useState('')
  const task = update.data?.task
  const settings = update.data?.settings
  const { widths, totalWidth, resize } = useColumnResize({
    order: 80,
    version: 140,
    status: 140,
    backend: 140,
    frontend: 140,
    restart: 100,
    updatedAt: 180,
    actions: 140,
  })

  const closePending = () => setPending(null)

  const openDraft = () => {
    setBrokerUrl(settings?.brokerUrl ?? '')
    setUsername(settings?.username ?? '')
    setPassword('')
    setSettingsError('')
    setDraftOpen(true)
  }

  const closeDraft = () => {
    if (saveSettings.isPending) return
    setDraftOpen(false)
    setSettingsError('')
  }

  const submitSettings = () => {
    const broker = brokerUrl.trim()
    if (!broker) {
      setSettingsError(t('gateway.update.settings.brokerRequired'))
      return
    }
    saveSettings.mutate(
      { enabled: true, brokerUrl: broker, username: username.trim(), password },
      {
        onSuccess: () => {
          setDraftOpen(false)
          setSettingsError('')
          toast.success(t('gateway.update.settings.saved'))
        },
        onError: (error) => showErrorToast(error),
      },
    )
  }

  const toggleSettings = (enabled: boolean) => {
    if (enabled) {
      openDraft()
      return
    }
    if (!settings?.enabled) {
      closeDraft()
      return
    }
    saveSettings.mutate(
      {
        enabled: false,
        brokerUrl: settings.brokerUrl,
        username: settings.username,
        password: '',
      },
      {
        onSuccess: () => toast.success(t('gateway.update.settings.disabled')),
        onError: (error) => showErrorToast(error),
      },
    )
  }

  const confirm = async () => {
    if (!pending || !task) return
    try {
      if (pending === 'apply') {
        await apply.mutateAsync()
        toast.success(t('gateway.update.toast.applied'))
      } else {
        await reject.mutateAsync()
        toast.success(t('gateway.update.toast.rejected'))
      }
      closePending()
    } catch (error) {
      showErrorToast(error)
    }
  }

  const canApply = task?.state === 'ready'
  const confirming = pending === 'apply' ? apply.isPending : reject.isPending
  const confirmDescription =
    pending === 'apply'
      ? task?.restartRequired
        ? t('gateway.update.confirm.applyRestart', { version: task?.releaseVersion ?? '—' })
        : t('gateway.update.confirm.apply', { version: task?.releaseVersion ?? '—' })
      : task
        ? t('gateway.update.confirm.reject', { version: task.releaseVersion ?? '—' })
        : ''
  const settingsVisible = Boolean(settings?.enabled) || draftOpen
  const switchChecked = Boolean(settings?.enabled) || draftOpen

  if (update.isLoading) {
    return (
      <div className="mx-auto flex min-h-0 w-full max-w-[1600px] flex-1 flex-col">
        <Skeleton className="h-44 w-full" />
      </div>
    )
  }

  if (update.isError) {
    return (
      <div className="mx-auto flex min-h-0 w-full max-w-[1600px] flex-1 flex-col">
        <ErrorState error={update.error} onRetry={update.refetch} compact />
      </div>
    )
  }

  return (
    <div className="mx-auto flex min-h-0 w-full max-w-[1600px] flex-1 flex-col gap-4">
      <Card>
        <CardContent className="space-y-4 p-4">
          <div className="flex items-center justify-between gap-4">
            <p className="text-sm font-medium text-muted">{t('gateway.update.settings.enable')}</p>
            <Switch
              checked={switchChecked}
              onChange={toggleSettings}
              disabled={update.isLoading || saveSettings.isPending}
              label={t('gateway.update.settings.enable')}
            />
          </div>

          {settingsVisible ? (
            <div className="space-y-4 border-t border-border pt-4">
              <Field
                label={t('gateway.update.settings.broker')}
                htmlFor="ota-mqtt-broker"
                labelClassName="text-sm"
                required
                error={settingsError}
              >
                <Input
                  id="ota-mqtt-broker"
                  value={brokerUrl}
                  placeholder={t('gateway.update.settings.brokerPlaceholder')}
                  onChange={(event) => {
                    setBrokerUrl(event.target.value)
                    setSettingsError('')
                  }}
                  autoComplete="off"
                  className="w-full"
                />
              </Field>
              <Field
                label={t('gateway.update.settings.username')}
                htmlFor="ota-mqtt-username"
                labelClassName="text-sm"
              >
                <Input
                  id="ota-mqtt-username"
                  value={username}
                  placeholder={t('gateway.update.settings.usernamePlaceholder')}
                  onChange={(event) => setUsername(event.target.value)}
                  autoComplete="off"
                  className="w-full"
                />
              </Field>
              <Field
                label={t('gateway.update.settings.password')}
                htmlFor="ota-mqtt-password"
                labelClassName="text-sm"
              >
                <Input
                  id="ota-mqtt-password"
                  type="password"
                  value={password}
                  placeholder={
                    settings?.hasPassword
                      ? t('gateway.update.settings.passwordKeep')
                      : t('gateway.update.settings.passwordPlaceholder')
                  }
                  onChange={(event) => setPassword(event.target.value)}
                  autoComplete="new-password"
                  className="w-full"
                />
              </Field>
              <div className="flex items-center justify-end gap-2 border-t border-border pt-3">
                {draftOpen ? (
                  <Button variant="outline" onClick={closeDraft} disabled={saveSettings.isPending}>
                    {t('common.action.cancel')}
                  </Button>
                ) : null}
                <Button variant="primary" loading={saveSettings.isPending} onClick={submitSettings}>
                  {t('common.action.save')}
                </Button>
              </div>
            </div>
          ) : null}
        </CardContent>
      </Card>

      {task ? (
        <ListTableCard>
          <DataTable
            style={{
              tableLayout: 'fixed',
              width: `max(100%, ${totalWidth}px)`,
            }}
          >
            <colgroup>
              <col style={{ width: widths.order }} />
              <col style={{ width: widths.version }} />
              <col style={{ width: widths.status }} />
              <col style={{ width: widths.backend }} />
              <col style={{ width: widths.frontend }} />
              <col style={{ width: widths.restart }} />
              <col style={{ width: widths.updatedAt }} />
              <col style={{ width: widths.actions }} />
            </colgroup>
            <DataTableHead>
              <tr>
                <DataTableHeadCell className="text-center">
                  {t('gateway.update.columns.order')}
                  <ColumnResizeHandle column="order" width={widths.order} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.update.columns.version')}
                  <ColumnResizeHandle column="version" width={widths.version} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.update.columns.status')}
                  <ColumnResizeHandle column="status" width={widths.status} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.update.columns.backend')}
                  <ColumnResizeHandle column="backend" width={widths.backend} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.update.columns.frontend')}
                  <ColumnResizeHandle column="frontend" width={widths.frontend} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.update.columns.restart')}
                  <ColumnResizeHandle column="restart" width={widths.restart} onResize={resize} />
                </DataTableHeadCell>
                <DataTableHeadCell>
                  {t('gateway.update.columns.updatedAt')}
                  <ColumnResizeHandle
                    column="updatedAt"
                    width={widths.updatedAt}
                    onResize={resize}
                  />
                </DataTableHeadCell>
                <DataTableHeadCell className="text-left">
                  {t('gateway.update.columns.actions')}
                  <ColumnResizeHandle column="actions" width={widths.actions} onResize={resize} />
                </DataTableHeadCell>
              </tr>
            </DataTableHead>
            <DataTableBody>
              <DataTableRow key={task.taskId}>
                <DataTableCell className="text-center tabular">1</DataTableCell>
                <DataTableCell>{task.releaseVersion || '—'}</DataTableCell>
                <DataTableCell>
                  <OtaStatusBadge status={task.state} error={task.message} />
                </DataTableCell>
                <DataTableCell>{task.backendVersion || '—'}</DataTableCell>
                <DataTableCell>{task.frontendVersion || '—'}</DataTableCell>
                <DataTableCell>
                  <EnumBadge tone={task.restartRequired ? 'warning' : 'neutral'}>
                    {task.restartRequired
                      ? t('gateway.update.restart.yes')
                      : t('gateway.update.restart.no')}
                  </EnumBadge>
                </DataTableCell>
                <DataTableCell>{task.updatedAt ? formatDateTime(task.updatedAt) : '—'}</DataTableCell>
                <DataTableCell>
                  <TableActions>
                    <TableActionButton
                      label={t('gateway.update.action.apply')}
                      icon={<Download className="size-4" />}
                      disabled={!canApply || apply.isPending || reject.isPending}
                      disabledReason={!canApply ? t('gateway.update.status.readyHint') : undefined}
                      onClick={() => setPending('apply')}
                    />
                    <TableActionButton
                      label={t('gateway.update.action.reject')}
                      icon={<X className="size-4" />}
                      disabled={!canApply || apply.isPending || reject.isPending}
                      disabledReason={!canApply ? t('gateway.update.status.readyHint') : undefined}
                      onClick={() => setPending('reject')}
                    />
                  </TableActions>
                </DataTableCell>
              </DataTableRow>
            </DataTableBody>
          </DataTable>
        </ListTableCard>
      ) : null}

      <ConfirmDialog
        open={pending !== null}
        title={
          pending === 'apply'
            ? t('gateway.update.action.apply')
            : t('gateway.update.action.reject')
        }
        description={confirmDescription}
        confirmText={
          pending === 'apply'
            ? t('gateway.update.action.apply')
            : t('gateway.update.action.reject')
        }
        cancelText={t('common.action.cancel')}
        danger={pending === 'reject'}
        loading={confirming}
        onConfirm={() => void confirm()}
        onClose={closePending}
      />
    </div>
  )
}
