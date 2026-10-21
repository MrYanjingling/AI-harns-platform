import { useMemo, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Pencil, Play, RotateCw, SlidersHorizontal, Square, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  ConfirmDialog,
  CopyButton,
  Dialog,
  EmptyState,
  ErrorState,
  Field,
  Input,
  PageHeader,
  PageLoading,
  Table,
  TBody,
  TD,
  TH,
  THead,
  TR,
  errorMessage,
  formatDateTime,
  useI18n,
} from '@harns/core'
import {
  useControlDevice,
  useDeleteDevice,
  useDevice,
  useGatewayMeta,
  useSwitchDeviceStatus,
} from '@/api/hooks'
import type { DeviceOperator, GatewayVariable } from '@/api/types'
import { DeviceStatusBadge } from '@/components/DeviceStatusBadge'
import { ProtocolLabel } from '@/components/ProtocolLabel'
import { defaultTopic, usesSerial } from '@/lib/constants'

type PendingAction = 'delete' | DeviceOperator

function InfoItem({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="min-w-0">
      <p className="text-xs text-faint">{label}</p>
      <div className="mt-1 text-sm text-fg">{children}</div>
    </div>
  )
}

function parseControlValue(variable: GatewayVariable, raw: string): unknown {
  const value = raw.trim()
  if (variable.dataType === 'bool') return value === 'true' || value === '1'
  if (variable.dataType === 'string') return value
  const numeric = Number(value)
  return Number.isFinite(numeric) ? numeric : value
}

export function DeviceDetailPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const device = useDevice(id)
  const meta = useGatewayMeta()
  const deleteDevice = useDeleteDevice()
  const switchStatus = useSwitchDeviceStatus()
  const controlDevice = useControlDevice()
  const [pending, setPending] = useState<PendingAction | null>(null)
  const [controlOpen, setControlOpen] = useState(false)
  const [controlValues, setControlValues] = useState<Record<string, string>>({})

  const detail = device.data
  const writableVariables = useMemo(
    () => detail?.variables?.filter((item) => item.accessMode === 'rw') ?? [],
    [detail],
  )

  const topic = detail
    ? detail.topic || (meta.data?.id ? defaultTopic(meta.data.id, detail.id) : '')
    : ''

  const confirm = async () => {
    if (!pending || !detail) return
    try {
      if (pending === 'delete') {
        await deleteDevice.mutateAsync({ id: detail.id, eTag: detail.eTag, name: detail.name })
        toast.success(t('gateway.collection.toast.deleted'))
        navigate('/gateway/collection')
        return
      }
      await switchStatus.mutateAsync({ id: detail.id, name: detail.name, status: pending })
      toast.success(
        pending === 'start'
          ? t('gateway.collection.toast.started')
          : pending === 'stop'
            ? t('gateway.collection.toast.stopped')
            : t('gateway.collection.toast.restarted'),
      )
      setPending(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const openControl = () => {
    setControlValues({})
    setControlOpen(true)
  }

  const submitControl = async () => {
    if (!detail) return
    const actions = writableVariables
      .filter((variable) => (controlValues[variable.name] ?? '').trim() !== '')
      .map((variable) => ({
        [variable.name]: parseControlValue(variable, controlValues[variable.name]),
      }))
    if (actions.length === 0) return
    try {
      await controlDevice.mutateAsync({ id: detail.id, name: detail.name, actions })
      toast.success(t('gateway.detail.control.toast'))
      setControlOpen(false)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  if (device.isLoading) return <PageLoading />
  if (device.isError) {
    return <ErrorState error={device.error} onRetry={() => void device.refetch()} />
  }
  if (!detail) {
    return (
      <EmptyState
        title={t('gateway.detail.notFound')}
        action={
          <Link
            to="/gateway/collection"
            className="inline-flex h-8 items-center rounded bg-primary px-3 text-sm font-medium text-primary-fg"
          >
            {t('gateway.form.backToCollection')}
          </Link>
        }
      />
    )
  }

  const option = detail.address?.option
  const serial = usesSerial(detail.deviceModel)
  const pendingLoading = deleteDevice.isPending || switchStatus.isPending

  return (
    <div className="space-y-4">
      <PageHeader
        title={detail.name}
        description={detail.deviceCode || detail.id}
        backTo="/gateway/collection"
        backLabel={t('gateway.detail.back')}
        actions={
          <>
            <Link
              to={`/gateway/collection/${detail.id}/edit`}
              className="inline-flex h-8 items-center gap-1.5 rounded border border-border-strong bg-surface px-3 text-sm text-fg transition-colors hover:bg-surface-2"
            >
              <Pencil className="size-4" />
              {t('gateway.collection.action.edit')}
            </Link>
            {writableVariables.length > 0 ? (
              <Button
                variant="outline"
                onClick={openControl}
                icon={<SlidersHorizontal className="size-4" />}
              >
                {t('gateway.collection.action.control')}
              </Button>
            ) : null}
            {detail.collectStatus === 'collecting' ? (
              <Button
                variant="outline"
                onClick={() => setPending('stop')}
                icon={<Square className="size-4" />}
              >
                {t('gateway.collection.action.stop')}
              </Button>
            ) : (
              <Button
                variant="outline"
                onClick={() => setPending('start')}
                icon={<Play className="size-4" />}
              >
                {t('gateway.collection.action.start')}
              </Button>
            )}
            <Button
              variant="outline"
              onClick={() => setPending('restart')}
              icon={<RotateCw className="size-4" />}
            >
              {t('gateway.collection.action.restart')}
            </Button>
            <Button
              variant="dangerSoft"
              onClick={() => setPending('delete')}
              icon={<Trash2 className="size-4" />}
            >
              {t('gateway.collection.action.delete')}
            </Button>
          </>
        }
      />

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('gateway.detail.section.identity')}</CardTitle>
            <CardDescription>{t('gateway.detail.title')}</CardDescription>
          </div>
          <DeviceStatusBadge status={detail.collectStatus} />
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <InfoItem label={t('gateway.detail.field.id')}>
            <span className="flex items-center gap-1 font-mono text-xs">
              {detail.id}
              <CopyButton value={detail.id} />
            </span>
          </InfoItem>
          <InfoItem label={t('gateway.collection.columns.protocol')}>
            <ProtocolLabel type={detail.deviceType} model={detail.deviceModel} />
          </InfoItem>
          <InfoItem label={t('gateway.detail.field.collectorCycle')}>
            {t('gateway.common.cycle', { seconds: detail.collectorCycle })}
          </InfoItem>
          <InfoItem label={t('gateway.detail.field.variableInterval')}>
            {detail.variableInterval
              ? `${detail.variableInterval} ms`
              : t('gateway.common.notAvailable')}
          </InfoItem>
          <InfoItem label={t('gateway.detail.field.topic')}>
            <span className="flex items-center gap-1">
              <span className="min-w-0 truncate font-mono text-xs">
                {topic || t('gateway.collection.noTopic')}
              </span>
              {topic ? (
                <CopyButton value={topic} label={t('gateway.common.topicCopied')} />
              ) : null}
            </span>
          </InfoItem>
          <InfoItem label={t('gateway.detail.field.version')}>
            <span className="font-mono text-xs">{detail.eTag}</span>
          </InfoItem>
          <InfoItem label={t('gateway.detail.field.modTime')}>
            {formatDateTime(detail.modTime)}
          </InfoItem>
          <InfoItem label={t('gateway.detail.field.variableCount')}>
            <Badge tone="neutral">
              {t('gateway.detail.varCount', { count: detail.variables?.length ?? 0 })}
            </Badge>
          </InfoItem>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('gateway.detail.section.connection')}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <InfoItem label={t('gateway.detail.field.location')}>
            <span className="font-mono text-xs">{detail.address?.location || '—'}</span>
          </InfoItem>
          {serial ? (
            <>
              <InfoItem label={t('gateway.form.field.baudRate')}>
                {option?.baudRate ?? '—'}
              </InfoItem>
              <InfoItem label={t('gateway.form.field.dataBits')}>
                {option?.dataBits ?? '—'}
              </InfoItem>
              <InfoItem label={t('gateway.form.field.parity')}>{option?.parity ?? '—'}</InfoItem>
              <InfoItem label={t('gateway.form.field.stopBits')}>
                {option?.stopBits ?? '—'}
              </InfoItem>
            </>
          ) : (
            <InfoItem label={t('gateway.form.field.port')}>{option?.port ?? '—'}</InfoItem>
          )}
          {detail.deviceType === 'modbus' ? (
            <>
              <InfoItem label={t('gateway.detail.field.slave')}>
                {detail.slave ?? '—'}
              </InfoItem>
              <InfoItem label={t('gateway.detail.field.memoryLayout')}>
                {detail.memoryLayout ?? '—'}
              </InfoItem>
              <InfoItem label={t('gateway.detail.field.positionAddress')}>
                {detail.positionAddress ?? 0}
              </InfoItem>
            </>
          ) : null}
          {detail.deviceType === 'opcUa' ? (
            <InfoItem label={t('gateway.form.field.username')}>
              {option?.username || '—'}
            </InfoItem>
          ) : null}
          {detail.deviceType === 's7' ? (
            <>
              <InfoItem label={t('gateway.detail.field.rack')}>{option?.rack ?? 0}</InfoItem>
              <InfoItem label={t('gateway.detail.field.slot')}>{option?.slot ?? 0}</InfoItem>
            </>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('gateway.detail.section.variables')}</CardTitle>
            <CardDescription>
              {t('gateway.detail.varCount', { count: detail.variables?.length ?? 0 })}
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent className="px-0 pb-0">
          {!detail.variables || detail.variables.length === 0 ? (
            <EmptyState title={t('gateway.detail.variables.empty')} className="py-10" />
          ) : (
            <Table>
              <THead>
                <TR>
                  <TH>{t('gateway.form.field.variableName')}</TH>
                  <TH>{t('gateway.form.field.dataType')}</TH>
                  <TH>{t('gateway.form.field.address')}</TH>
                  {detail.deviceType === 'modbus' ? (
                    <>
                      <TH>{t('gateway.form.field.functionCode')}</TH>
                      <TH>{t('gateway.form.field.bits')}</TH>
                      <TH>{t('gateway.form.field.amount')}</TH>
                    </>
                  ) : null}
                  {detail.deviceType === 'opcUa' ? (
                    <TH>{t('gateway.form.field.namespace')}</TH>
                  ) : null}
                  {detail.deviceType !== 'opcUa' ? (
                    <TH>{t('gateway.form.field.rate')}</TH>
                  ) : null}
                  <TH>{t('gateway.form.field.defaultValue')}</TH>
                  <TH>{t('gateway.form.field.accessMode')}</TH>
                </TR>
              </THead>
              <TBody>
                {detail.variables.map((variable) => (
                  <TR key={variable.name}>
                    <TD className="font-medium">{variable.name}</TD>
                    <TD className="font-mono text-xs text-muted">{variable.dataType}</TD>
                    <TD className="font-mono text-xs text-muted">{String(variable.address)}</TD>
                    {detail.deviceType === 'modbus' ? (
                      <>
                        <TD className="tabular">{variable.functionCode ?? '—'}</TD>
                        <TD className="tabular">{variable.bits ?? '—'}</TD>
                        <TD className="tabular">{variable.amount ?? '—'}</TD>
                      </>
                    ) : null}
                    {detail.deviceType === 'opcUa' ? (
                      <TD className="tabular">{variable.namespace ?? '—'}</TD>
                    ) : null}
                    {detail.deviceType !== 'opcUa' ? (
                      <TD className="tabular">{variable.rate ?? '—'}</TD>
                    ) : null}
                    <TD className="text-xs text-muted">
                      {variable.defaultValue === undefined || variable.defaultValue === null
                        ? '—'
                        : String(variable.defaultValue)}
                    </TD>
                    <TD>
                      <Badge tone={variable.accessMode === 'rw' ? 'primary' : 'neutral'}>
                        {variable.accessMode}
                      </Badge>
                    </TD>
                  </TR>
                ))}
              </TBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <ConfirmDialog
        open={Boolean(pending)}
        onClose={() => setPending(null)}
        onConfirm={() => void confirm()}
        loading={pendingLoading}
        danger={pending === 'delete'}
        title={
          pending === 'delete'
            ? t('gateway.collection.confirm.deleteTitle')
            : pending === 'start'
              ? t('gateway.collection.confirm.startTitle')
              : pending === 'stop'
                ? t('gateway.collection.confirm.stopTitle')
                : t('gateway.collection.confirm.restartTitle')
        }
        description={
          pending === 'delete'
            ? t('gateway.collection.confirm.deleteDescription', { name: detail.name })
            : pending === 'start'
              ? t('gateway.collection.confirm.startDescription')
              : pending === 'stop'
                ? t('gateway.collection.confirm.stopDescription')
                : t('gateway.collection.confirm.restartDescription')
        }
        confirmText={
          pending === 'delete'
            ? t('common.action.delete')
            : pending === 'start'
              ? t('common.action.start')
              : pending === 'stop'
                ? t('common.action.stop')
                : t('common.action.restart')
        }
      />

      <Dialog
        open={controlOpen}
        onClose={() => setControlOpen(false)}
        title={t('gateway.detail.control.title')}
        description={t('gateway.detail.control.description')}
        footer={
          <>
            <Button variant="ghost" onClick={() => setControlOpen(false)}>
              {t('common.action.cancel')}
            </Button>
            <Button
              variant="primary"
              loading={controlDevice.isPending}
              disabled={
                writableVariables.every(
                  (variable) => (controlValues[variable.name] ?? '').trim() === '',
                )
              }
              onClick={() => void submitControl()}
            >
              {t('gateway.detail.control.submit')}
            </Button>
          </>
        }
      >
        {writableVariables.length === 0 ? (
          <p className="text-sm text-muted">{t('gateway.detail.control.empty')}</p>
        ) : (
          <div className="space-y-3">
            {writableVariables.map((variable) => (
              <Field
                key={variable.name}
                label={`${variable.name} · ${variable.dataType}`}
                htmlFor={`control-${variable.name}`}
                hint={
                  variable.defaultValue === undefined || variable.defaultValue === null
                    ? undefined
                    : t('gateway.detail.control.defaultValue', {
                        value: String(variable.defaultValue),
                      })
                }
              >
                <Input
                  id={`control-${variable.name}`}
                  value={controlValues[variable.name] ?? ''}
                  placeholder={t('gateway.detail.control.valuePlaceholder')}
                  onChange={(event) =>
                    setControlValues((current) => ({
                      ...current,
                      [variable.name]: event.target.value,
                    }))
                  }
                />
              </Field>
            ))}
          </div>
        )}
      </Dialog>
    </div>
  )
}
