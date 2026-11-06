import { useParams, useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { useDevice, useUpdateDevice } from '@/api/hooks'
import { useI18n } from '@/i18n'
import { errorMessage } from '@/lib/http'
import { Card, CardContent } from '@/components/ui/Card'
import { PageLoading } from '@/components/ui/Spinner'
import { ErrorState } from '@/components/shared/ErrorState'
import { PageHeader } from '@/components/shared/PageHeader'
import { DeviceForm } from './DeviceForm'
import { toFormValues, toPayload, type DeviceFormValues } from './schema'

export function DeviceEditPage() {
  const { t } = useI18n()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const device = useDevice(id)
  const updateDevice = useUpdateDevice()

  if (device.isLoading) {
    return <PageLoading label={t('devices.loading')} />
  }

  if (device.isError || !device.data) {
    return (
      <div>
        <PageHeader
          backTo="/devices"
          backLabel={t('devices.backToList')}
          title={t('layout.title.deviceEdit')}
        />
        <ErrorState error={device.error} onRetry={device.refetch} />
      </div>
    )
  }

  const detail = device.data

  const submit = (values: DeviceFormValues) => {
    updateDevice.mutate(
      { id: detail.id, payload: toPayload(values), etag: detail.eTag },
      {
        onSuccess: () => {
          toast.success(t('devices.edit.saved', { name: values.name }))
          navigate(`/devices/${detail.id}`)
        },
        onError: (error) => toast.error(errorMessage(error)),
      },
    )
  }

  return (
    <div>
      <PageHeader
        backTo={`/devices/${detail.id}`}
        backLabel={t('devices.backToDetail')}
        title={t('devices.edit.titleWithName', { name: detail.name })}
        description={t('devices.edit.description')}
      />
      <Card className="mb-4">
        <CardContent className="flex flex-wrap items-center gap-x-6 gap-y-2 pt-4 text-xs text-muted">
          <span>
            ID <code className="ml-1 font-mono text-fg">{detail.id}</code>
          </span>
          <span>
            eTag <code className="ml-1 font-mono text-fg">{detail.eTag}</code>
          </span>
          <span>
            {t('devices.edit.lastModified')} <span className="text-fg">{detail.modTime}</span>
          </span>
        </CardContent>
      </Card>
      <DeviceForm
        mode="edit"
        type={detail.deviceType}
        initialValues={toFormValues(detail)}
        submitting={updateDevice.isPending}
        onSubmit={submit}
        onCancel={() => navigate(`/devices/${detail.id}`)}
      />
    </div>
  )
}
