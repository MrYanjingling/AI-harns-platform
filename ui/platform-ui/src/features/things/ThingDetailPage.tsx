import { useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Eye, EyeOff, ImageIcon, Pencil, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteThings, useThing, useThingModel, useUpdateThing } from '@/api/hooks'
import type { ThingDetail, ThingModelFeature, UpdateThingRequest } from '@/api/types'
import { useI18n } from '@/i18n'
import { formatDateTime, formatRelative, maskSecret } from '@/lib/format'
import { errorMessage } from '@/lib/http'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { CopyButton } from '@/components/ui/CopyButton'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { Switch } from '@/components/ui/Switch'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'
import { PageLoading } from '@/components/ui/Spinner'
import { ErrorState } from '@/components/shared/ErrorState'
import { PageHeader } from '@/components/shared/PageHeader'
import { describeDataType } from '@/features/thing-models/data-type'
import { EditThingDialog } from './EditThingDialog'
import { FeatureValuesEditor } from './FeatureValuesEditor'
import { ImageUploadField } from './ImageUploadField'

export function ThingDetailPage() {
  const { t } = useI18n()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const thingQuery = useThing(id)
  const modelQuery = useThingModel(thingQuery.data?.thingModelId)
  const updateMutation = useUpdateThing()
  const deleteMutation = useDeleteThings()

  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [featureOpen, setFeatureOpen] = useState(false)
  const [featureDraft, setFeatureDraft] = useState<Record<string, unknown>>({})

  if (thingQuery.isLoading) return <PageLoading label={t('thingDetail.loading')} />
  if (thingQuery.isError) {
    return <ErrorState error={thingQuery.error} onRetry={thingQuery.refetch} />
  }
  if (!thingQuery.data) {
    return <ErrorState error={new Error(t('thingDetail.notFound'))} onRetry={thingQuery.refetch} />
  }

  const thing = thingQuery.data
  const online = thing.enable === 1 && thing.onlineStatus === 1
  const features = modelQuery.data?.features ?? []

  const update = async (payload: UpdateThingRequest) => {
    try {
      await updateMutation.mutateAsync({ id: thing.id, payload })
      return true
    } catch (error) {
      toast.error(errorMessage(error))
      return false
    }
  }

  const toggleEnable = async (next: boolean) => {
    const ok = await update({ enable: next ? 1 : 0 })
    if (ok) toast.success(next ? t('thingDetail.toggleEnabled') : t('thingDetail.toggleDisabled'))
  }

  const changeImage = async (key: string) => {
    const ok = await update({ image: key })
    if (ok) toast.success(key ? t('thingDetail.imageUpdated') : t('thingDetail.imageRemoved'))
  }

  const openFeatures = () => {
    setFeatureDraft({ ...(thing.features ?? {}) })
    setFeatureOpen(true)
  }

  const saveFeatures = async () => {
    const ok = await update({ features: featureDraft })
    if (ok) {
      toast.success(t('thingDetail.featuresUpdated'))
      setFeatureOpen(false)
    }
  }

  const confirmDelete = async () => {
    try {
      await deleteMutation.mutateAsync([thing.id])
      toast.success(t('thingDetail.deleted'))
      navigate('/platform/things', { replace: true })
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div>
      <PageHeader
        title={thing.deviceName}
        description={thing.deviceDesc || t('thingDetail.noDescription')}
        backTo="/platform/things"
        backLabel={t('thingDetail.back')}
        actions={
          <>
            <Button
              variant="outline"
              icon={<Pencil className="size-4" />}
              onClick={() => setEditOpen(true)}
            >
              {t('common.action.edit')}
            </Button>
            <Button
              variant="ghost"
              icon={<Trash2 className="size-4 text-danger" />}
              onClick={() => setDeleteOpen(true)}
            >
              {t('common.action.delete')}
            </Button>
          </>
        }
      />

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="p-5 lg:col-span-3">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex flex-wrap items-center gap-3">
              {thing.enable === 1 ? (
                <Badge tone="success" dot>
                  {t('thingDetail.enabled')}
                </Badge>
              ) : (
                <Badge tone="neutral" dot>
                  {t('thingDetail.disabled')}
                </Badge>
              )}
              {online ? (
                <Badge tone="info" dot>
                  {t('thingDetail.online')}
                </Badge>
              ) : (
                <Badge tone="neutral" dot>
                  {t('thingDetail.offline')}
                </Badge>
              )}
              <span className="text-xs text-muted">
                {t('thingDetail.lastActive', {
                  time: thing.lastActiveTime ? formatRelative(thing.lastActiveTime) : '—',
                })}
              </span>
            </div>
            <label className="flex items-center gap-2 text-xs text-muted">
              <span>{t('thingDetail.enableDevice')}</span>
              <Switch
                checked={thing.enable === 1}
                onChange={(next) => void toggleEnable(next)}
                disabled={updateMutation.isPending}
                label={t('thingDetail.enableDevice')}
              />
            </label>
          </div>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('thingDetail.basic')}</CardTitle>
              <CardDescription>{t('thingDetail.basicDescription')}</CardDescription>
            </div>
          </CardHeader>
          <CardContent className="space-y-3 text-xs">
            <InfoRow label={t('thingDetail.model')}>
              {modelQuery.data ? (
                <Link
                  to={`/platform/thing-models/${modelQuery.data.id}`}
                  className="text-primary hover:underline"
                >
                  {modelQuery.data.name}
                </Link>
              ) : (
                <span className="font-mono text-[11px]">{thing.thingModelId}</span>
              )}
            </InfoRow>
            <InfoRow label={t('thingDetail.deviceKey')}>
              <CodeValue value={thing.deviceKey} />
            </InfoRow>
            <InfoRow label={t('thingDetail.upTopic')}>
              <CodeValue value={thing.upTopic} />
            </InfoRow>
            <InfoRow label={t('thingDetail.downTopic')}>
              <CodeValue value={thing.downTopic} />
            </InfoRow>
            <InfoRow label={t('common.field.createdAt')}>{formatDateTime(thing.createdTime)}</InfoRow>
            <InfoRow label={t('common.field.updatedAt')}>{formatDateTime(thing.updatedTime)}</InfoRow>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <div>
              <CardTitle>{t('thingDetail.credentials')}</CardTitle>
              <CardDescription>{t('thingDetail.credentialsDescription')}</CardDescription>
            </div>
          </CardHeader>
          <CardContent className="space-y-1">
            <SecretRow label="MQTT Client ID" value={thing.mqttClientId} secret={false} />
            <SecretRow label="MQTT Username" value={thing.mqttUsername} secret={false} />
            <SecretRow label="MQTT Password" value={thing.mqttPassword} secret />
            <SecretRow label="Device Secret" value={thing.deviceSecret} secret />
            <SecretRow label="HTTP Token" value={thing.token} secret />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('thingDetail.image')}</CardTitle>
              <CardDescription>{t('thingDetail.imageDescription')}</CardDescription>
            </div>
            <ImageIcon className="size-4 text-faint" />
          </CardHeader>
          <CardContent>
            <ImageUploadField value={thing.image} onChange={(key) => void changeImage(key)} />
          </CardContent>
        </Card>

        <Card className="lg:col-span-3">
          <CardHeader>
            <div>
              <CardTitle>{t('thingDetail.features')}</CardTitle>
              <CardDescription>
                {t('thingDetail.featuresDescription', {
                  name: `「${modelQuery.data ? modelQuery.data.name : '—'}」`,
                })}
              </CardDescription>
            </div>
            <Button
              size="sm"
              variant="outline"
              disabled={features.length === 0}
              onClick={openFeatures}
            >
              {t('thingDetail.editFeatures')}
            </Button>
          </CardHeader>
          <CardContent>
            <FeatureTable thing={thing} modelFeatures={features} />
          </CardContent>
        </Card>
      </div>

      <EditThingDialog open={editOpen} onClose={() => setEditOpen(false)} thing={thing} />

      <Dialog
        open={featureOpen}
        onClose={() => setFeatureOpen(false)}
        title={t('thingDetail.editFeatures')}
        description={t('thingDetail.featureDialogDescription')}
        size="lg"
        footer={
          <>
            <Button variant="ghost" onClick={() => setFeatureOpen(false)}>
              {t('common.action.cancel')}
            </Button>
            <Button variant="primary" onClick={saveFeatures} loading={updateMutation.isPending}>
              {t('common.action.save')}
            </Button>
          </>
        }
      >
        <div className="max-h-[60vh] overflow-y-auto pr-1">
          <FeatureValuesEditor
            features={features}
            value={featureDraft}
            onChange={setFeatureDraft}
          />
        </div>
      </Dialog>

      <ConfirmDialog
        open={deleteOpen}
        title={t('thingDetail.deleteTitle')}
        description={t('thingDetail.deleteDescription', { name: thing.deviceName })}
        confirmText={t('common.action.delete')}
        danger
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
        onClose={() => setDeleteOpen(false)}
      />
    </div>
  )
}

function InfoRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3">
      <span className="shrink-0 text-faint">{label}</span>
      <span className="min-w-0 text-right">{children}</span>
    </div>
  )
}

function CodeValue({ value }: { value: string }) {
  return (
    <span className="inline-flex max-w-64 items-center gap-1">
      <span className="truncate font-mono text-[11px]" title={value}>
        {value}
      </span>
      <CopyButton value={value} />
    </span>
  )
}

function SecretRow({
  label,
  value,
  secret,
}: {
  label: string
  value: string
  secret: boolean
}) {
  const { t } = useI18n()
  const [visible, setVisible] = useState(false)
  return (
    <div className="flex items-center justify-between gap-3 border-b border-border/60 py-2 last:border-0">
      <span className="shrink-0 text-xs text-faint">{label}</span>
      <span className="flex min-w-0 items-center gap-1">
        <span className="truncate font-mono text-[11px]" title={visible ? value : undefined}>
          {secret && !visible ? maskSecret(value) : value || '—'}
        </span>
        {secret ? (
          <Button
            size="icon"
            variant="ghost"
            aria-label={visible ? t('thingDetail.hide') : t('thingDetail.show')}
            onClick={() => setVisible((current) => !current)}
          >
            {visible ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
          </Button>
        ) : null}
        <CopyButton value={value} />
      </span>
    </div>
  )
}

function FeatureTable({
  thing,
  modelFeatures,
}: {
  thing: ThingDetail
  modelFeatures: ThingModelFeature[]
}) {
  const { t } = useI18n()
  const values = thing.features ?? {}
  const defined = new Set(modelFeatures.map((feature) => feature.identifier))
  const extras = Object.keys(values).filter((key) => !defined.has(key))

  if (modelFeatures.length === 0 && extras.length === 0) {
    return (
      <EmptyState
        title={t('thingDetail.noFeatures')}
        description={t('thingDetail.noFeaturesDescription')}
      />
    )
  }

  return (
    <Table>
      <THead>
        <tr>
          <TH>{t('thingDetail.table.feature')}</TH>
          <TH>{t('thingDetail.table.identifier')}</TH>
          <TH>{t('thingDetail.table.dataType')}</TH>
          <TH>{t('thingDetail.table.currentValue')}</TH>
        </tr>
      </THead>
      <TBody>
        {modelFeatures.map((feature) => (
          <TR key={feature.identifier}>
            <TD className="font-medium">{feature.name}</TD>
            <TD className="font-mono text-xs text-muted">{feature.identifier}</TD>
            <TD className="text-xs text-muted">{describeDataType(feature.dataType)}</TD>
            <TD className="max-w-72 truncate font-mono text-[11px]" title={renderValue(values[feature.identifier])}>
              {renderValue(values[feature.identifier])}
            </TD>
          </TR>
        ))}
        {extras.map((key) => (
          <TR key={key}>
            <TD className="text-muted">{t('thingDetail.undefinedFeature')}</TD>
            <TD className="font-mono text-xs text-muted">{key}</TD>
            <TD className="text-xs text-faint">—</TD>
            <TD className="max-w-72 truncate font-mono text-[11px]" title={renderValue(values[key])}>
              {renderValue(values[key])}
            </TD>
          </TR>
        ))}
      </TBody>
    </Table>
  )
}

function renderValue(value: unknown): string {
  if (value === undefined || value === null || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
