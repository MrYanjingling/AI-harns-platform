import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Pencil,
  Play,
  RotateCw,
  SlidersHorizontal,
  Square,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteDevice, useDevice, useSwitchDeviceStatus } from '@/api/hooks'
import type { DeviceStatusAction } from '@/api/types'
import { useI18n } from '@/i18n'
import { DEVICE_STATUS, PROTOCOLS } from '@/lib/constants'
import { errorMessage } from '@/lib/http'
import { formatDateTime } from '@/lib/format'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { CopyButton } from '@/components/ui/CopyButton'
import { EmptyState } from '@/components/ui/EmptyState'
import { PageLoading } from '@/components/ui/Spinner'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'
import { DeviceTypeBadge } from '@/components/shared/DeviceTypeBadge'
import { ErrorState } from '@/components/shared/ErrorState'
import { PageHeader } from '@/components/shared/PageHeader'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { ControlDialog } from './ControlDialog'

function displayValue(value: unknown): string {
  if (value === undefined || value === null || value === '') return '—'
  if (typeof value === 'boolean') return value ? 'true' : 'false'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

export function DeviceDetailPage() {
  const { t } = useI18n()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const device = useDevice(id)
  const switchStatus = useSwitchDeviceStatus()
  const deleteDevice = useDeleteDevice()
  const [controlOpen, setControlOpen] = useState(false)
  const [controlTarget, setControlTarget] = useState<string | undefined>(undefined)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const detail = device.data

  const rows = useMemo(() => {
    if (!detail) return []
    type Row = {
      name: string
      dataType: string
      address: string
      detail: string
      accessMode: string
      value: unknown
    }
    if (detail.deviceType === 'modbus') {
      return detail.variables.map<Row>((variable) => ({
        name: variable.name,
        dataType: variable.dataType,
        address: String(variable.address),
        detail: `FC${variable.functionCode}${variable.dataType === 'bool' ? ` · bit ${variable.bits}` : ''} · ×${variable.rate} · ${t('devices.detail.amount', { value: variable.amount })}`,
        accessMode: variable.accessMode,
        value: variable.value,
      }))
    }
    if (detail.deviceType === 'opcUa') {
      return detail.variables.map<Row>((variable) => ({
        name: variable.name,
        dataType: variable.dataType,
        address: displayValue(variable.address),
        detail: `ns=${variable.namespace}`,
        accessMode: variable.accessMode,
        value: variable.value,
      }))
    }
    return detail.variables.map<Row>((variable) => ({
      name: variable.name,
      dataType: variable.dataType,
      address: variable.address,
      detail: `×${variable.rate}`,
      accessMode: variable.accessMode,
      value: variable.value,
    }))
  }, [detail, t])

  if (device.isLoading) {
    return <PageLoading label={t('devices.detail.loading')} />
  }

  if (device.isError || !detail) {
    return (
      <div>
        <PageHeader
          backTo="/devices"
          backLabel={t('devices.backToList')}
          title={t('layout.title.deviceDetail')}
        />
        <ErrorState error={device.error} onRetry={device.refetch} />
      </div>
    )
  }

  const protocol = PROTOCOLS[detail.deviceType]
  const running =
    detail.collectStatus === 'collecting' || detail.collectStatus === 'collectingError'
  const writableCount = rows.filter((row) => row.accessMode === 'rw').length

  const runAction = (action: DeviceStatusAction) => {
    const label = t(
      action === 'stop'
        ? 'enums.deviceAction.stop'
        : action === 'restart'
          ? 'enums.deviceAction.restart'
          : 'enums.deviceAction.start',
    )
    switchStatus.mutate(
      { id: detail.id, action },
      {
        onSuccess: () =>
          toast.success(t('devices.detail.actionDone', { name: detail.name, action: label })),
        onError: (error) => toast.error(errorMessage(error)),
      },
    )
  }

  const openControl = (name?: string) => {
    setControlTarget(name)
    setControlOpen(true)
  }

  const confirmRemove = () => {
    deleteDevice.mutate(
      { id: detail.id, etag: detail.eTag },
      {
        onSuccess: () => {
          toast.success(t('devices.detail.deleted', { name: detail.name }))
          navigate('/devices')
        },
        onError: (error) => toast.error(errorMessage(error)),
      },
    )
  }

  return (
    <div>
      <PageHeader
        backTo="/devices"
        backLabel={t('devices.backToList')}
        title={
          <span className="flex flex-wrap items-center gap-3">
            {detail.name}
            <StatusBadge status={detail.collectStatus} />
          </span>
        }
        description={
          <span className="flex flex-wrap items-center gap-2">
            <DeviceTypeBadge type={detail.deviceType} />
            <span className="font-mono text-xs">{detail.deviceModel}</span>
            <span className="text-faint">·</span>
            <span className="font-mono text-xs">{detail.deviceCode}</span>
            <span className="text-faint">·</span>
            <span>
              {DEVICE_STATUS[detail.collectStatus]
                ? t(DEVICE_STATUS[detail.collectStatus].descriptionKey)
                : ''}
            </span>
          </span>
        }
        actions={
          <>
            {writableCount > 0 ? (
              <Button
                variant="secondary"
                icon={<SlidersHorizontal className="size-4" />}
                onClick={() => openControl()}
              >
                {t('devices.detail.control')}
              </Button>
            ) : null}
            <Button
              variant="outline"
              icon={<Pencil className="size-4" />}
              onClick={() => navigate(`/devices/${detail.id}/edit`)}
            >
              {t('common.action.edit')}
            </Button>
            {running ? (
              <>
                <Button
                  variant="outline"
                  icon={<RotateCw className="size-4" />}
                  disabled={switchStatus.isPending}
                  onClick={() => runAction('restart')}
                >
                  {t('common.action.restart')}
                </Button>
                <Button
                  variant="outline"
                  icon={<Square className="size-4" />}
                  disabled={switchStatus.isPending}
                  onClick={() => runAction('stop')}
                >
                  {t('common.action.stop')}
                </Button>
              </>
            ) : (
              <Button
                variant="primary"
                icon={<Play className="size-4" />}
                disabled={switchStatus.isPending}
                onClick={() => runAction('start')}
              >
                {t('common.action.start')}
              </Button>
            )}
            <Button
              variant="outline"
              className="text-danger hover:bg-danger/10 hover:text-danger"
              icon={<Trash2 className="size-4" />}
              disabled={deleteDevice.isPending}
              onClick={() => setConfirmDelete(true)}
            >
              {t('common.action.delete')}
            </Button>
          </>
        }
      />

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <div>
              <CardTitle>{t('devices.detail.info')}</CardTitle>
              <CardDescription>{t('devices.detail.infoDescription')}</CardDescription>
            </div>
            {device.isFetching ? (
              <span className="flex items-center gap-1.5 text-[11px] text-faint">
                <span className="size-1.5 animate-pulse rounded-full bg-primary" />
                {t('devices.detail.syncing')}
              </span>
            ) : null}
          </CardHeader>
          <CardContent>
            <dl className="grid gap-x-6 gap-y-4 sm:grid-cols-2">
              <InfoItem label={t('devices.detail.deviceCode')} value={detail.deviceCode} mono />
              <InfoItem
                label={t('devices.detail.protocolModel')}
                value={`${protocol.label} / ${detail.deviceModel}`}
              />
              <InfoItem
                label={t('devices.detail.collectorCycle')}
                value={t('devices.detail.seconds', { value: detail.collectorCycle })}
              />
              <InfoItem
                label={t('devices.detail.variableInterval')}
                value={
                  detail.variableInterval > 0
                    ? t('devices.detail.milliseconds', { value: detail.variableInterval })
                    : t('devices.detail.noWait')
                }
              />
              <InfoItem
                label={t('devices.detail.topic')}
                value={detail.topic || t('devices.defaultTopic')}
                mono
              />
              <InfoItem
                label={t('devices.detail.variableQuantity')}
                value={t('devices.detail.variableCount', { count: rows.length })}
              />
              <InfoItem
                label={t('devices.detail.deviceId')}
                value={detail.id}
                mono
                copyValue={detail.id}
              />
              <InfoItem label="eTag" value={detail.eTag} mono copyValue={detail.eTag} />
              <InfoItem
                label={t('devices.detail.lastModified')}
                value={formatDateTime(detail.modTime)}
              />
            </dl>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('devices.detail.connection')}</CardTitle>
              <CardDescription>{t(protocol.descriptionKey)}</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            <dl className="space-y-4">
              {detail.deviceType === 'modbus' ? (
                <>
                  <InfoItem
                    label={
                      detail.deviceModel === 'modbusRtu'
                        ? t('devices.detail.serialDevice')
                        : t('devices.detail.deviceAddress')
                    }
                    value={detail.address.location}
                    mono
                  />
                  {detail.deviceModel === 'modbusRtu' ? (
                    <InfoItem
                      label={t('devices.detail.serialParams')}
                      value={t('devices.detail.serialValue', {
                        baud: detail.address.option?.baudRate ?? '—',
                        dataBits: detail.address.option?.dataBits ?? '—',
                        parity: detail.address.option?.parity ?? '—',
                        stopBits: detail.address.option?.stopBits ?? '—',
                      })}
                    />
                  ) : (
                    <InfoItem
                      label={t('devices.detail.port')}
                      value={String(detail.address.option?.port ?? '—')}
                      mono
                    />
                  )}
                  <InfoItem label={t('devices.detail.slave')} value={String(detail.slave)} />
                  <InfoItem
                    label={t('devices.detail.memoryLayout')}
                    value={detail.memoryLayout}
                    mono
                  />
                  <InfoItem
                    label={t('devices.detail.startOffset')}
                    value={String(detail.positionAddress)}
                  />
                </>
              ) : null}

              {detail.deviceType === 'opcUa' ? (
                <>
                  <InfoItem label="Endpoint" value={detail.address.location} mono />
                  <InfoItem
                    label={t('devices.detail.port')}
                    value={String(detail.address.option?.port ?? '—')}
                    mono
                  />
                  <InfoItem
                    label={t('devices.detail.username')}
                    value={detail.address.option?.username || t('devices.detail.anonymous')}
                    mono
                  />
                  <InfoItem
                    label={t('devices.detail.password')}
                    value={
                      detail.address.option?.password
                        ? '••••••••'
                        : t('common.state.unset')
                    }
                    mono
                  />
                </>
              ) : null}

              {detail.deviceType === 's7' ? (
                <>
                  <InfoItem
                    label={t('devices.detail.plcAddress')}
                    value={detail.address.location}
                    mono
                  />
                  <InfoItem
                    label={t('devices.detail.port')}
                    value={String(detail.address.option?.port ?? '—')}
                    mono
                  />
                  <InfoItem
                    label="Rack / Slot"
                    value={`${detail.address.option?.rack ?? 0} / ${detail.address.option?.slot ?? 1}`}
                    mono
                  />
                </>
              ) : null}
            </dl>
          </CardContent>
        </Card>
      </div>

      <Card className="mt-4 overflow-hidden">
        <CardHeader>
          <div>
            <CardTitle>{t('devices.detail.variables')}</CardTitle>
            <CardDescription>{t('devices.detail.variablesDescription')}</CardDescription>
          </div>
          <Badge tone="neutral">
            {t('devices.detail.variablesBadge', { count: rows.length })}
          </Badge>
        </CardHeader>

        {rows.length === 0 ? (
          <EmptyState
            title={t('devices.detail.noVariables')}
            description={t('devices.detail.noVariablesDescription')}
            action={
              <Button
                variant="outline"
                onClick={() => navigate(`/devices/${detail.id}/edit`)}
                icon={<Pencil className="size-4" />}
              >
                {t('devices.detail.addVariable')}
              </Button>
            }
          />
        ) : (
          <Table>
            <THead>
              <tr>
                <TH>{t('devices.detail.variableName')}</TH>
                <TH>{t('devices.detail.variableType')}</TH>
                <TH>{t('devices.detail.variableAddress')}</TH>
                <TH>{t('devices.detail.variableParams')}</TH>
                <TH>{t('devices.detail.currentValue')}</TH>
                <TH>{t('devices.detail.access')}</TH>
                <TH className="text-right">{t('common.field.actions')}</TH>
              </tr>
            </THead>
            <TBody>
              {rows.map((row) => (
                <TR key={row.name}>
                  <TD className="font-medium">{row.name}</TD>
                  <TD>
                    <code className="rounded bg-surface-2 px-1.5 py-0.5 font-mono text-[11px] text-muted">
                      {row.dataType}
                    </code>
                  </TD>
                  <TD className="font-mono text-xs text-muted">{row.address}</TD>
                  <TD className="text-xs text-muted">{row.detail}</TD>
                  <TD>
                    <span className="font-mono text-xs text-fg">{displayValue(row.value)}</span>
                  </TD>
                  <TD>
                    {row.accessMode === 'rw' ? (
                      <Badge tone="warning">rw</Badge>
                    ) : (
                      <Badge tone="neutral">r</Badge>
                    )}
                  </TD>
                  <TD>
                    <div className="flex justify-end">
                      {row.accessMode === 'rw' ? (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => openControl(row.name)}
                          icon={<SlidersHorizontal className="size-3.5" />}
                        >
                          {t('devices.detail.write')}
                        </Button>
                      ) : (
                        <span className="text-[11px] text-faint">
                          {t('devices.detail.readOnly')}
                        </span>
                      )}
                    </div>
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <p className="mt-3 text-[11px] text-faint">{t('devices.detail.footer')}</p>

      <ControlDialog
        open={controlOpen}
        device={detail}
        only={controlTarget}
        onClose={() => setControlOpen(false)}
      />

      <ConfirmDialog
        open={confirmDelete}
        title={t('devices.detail.deleteTitle')}
        description={t('devices.detail.deleteDescription', { name: detail.name })}
        confirmText={t('common.action.delete')}
        danger
        loading={deleteDevice.isPending}
        onConfirm={confirmRemove}
        onClose={() => setConfirmDelete(false)}
      />
    </div>
  )
}

function InfoItem({
  label,
  value,
  mono = false,
  copyValue,
}: {
  label: string
  value: string
  mono?: boolean
  copyValue?: string
}) {
  return (
    <div className="min-w-0">
      <dt className="text-[11px] text-faint">{label}</dt>
      <dd className="mt-1 flex items-center gap-1">
        <span
          className={
            mono
              ? 'min-w-0 truncate font-mono text-xs text-fg'
              : 'min-w-0 truncate text-sm text-fg'
          }
          title={value}
        >
          {value}
        </span>
        {copyValue ? <CopyButton value={copyValue} /> : null}
      </dd>
    </div>
  )
}
