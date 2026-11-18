import { useEffect, useState, type ChangeEvent } from 'react'
import { Upload } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Dialog,
  Input,
  showErrorToast,
  useI18n,
} from '@harns/core'
import { useUploadDriverPlugin } from '@/api/hooks'
import type { Driver } from '@/api/types'
import { driverModelLabel } from '@/lib/constants'

export function DriverPluginFileDialog({
  driver,
  onClose,
}: {
  driver: Driver | null
  onClose: () => void
}) {
  const { t } = useI18n()
  const upload = useUploadDriverPlugin()
  const [file, setFile] = useState<File | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    setFile(null)
    setError('')
  }, [driver])

  const selectFile = (event: ChangeEvent<HTMLInputElement>) => {
    const next = event.currentTarget.files?.[0]
    event.currentTarget.value = ''
    if (!next) return
    if (!next.name.toLowerCase().endsWith('.so')) {
      setFile(null)
      setError(t('gateway.drivers.pluginFileInvalid'))
      return
    }
    setFile(next)
    setError('')
  }

  const close = () => {
    setFile(null)
    setError('')
    onClose()
  }

  const submit = async () => {
    if (!driver || !file) return
    try {
      await upload.mutateAsync({
        id: driver.id,
        title: `${driver.driverType}/${driver.model}`,
        file,
      })
      toast.success(t('gateway.drivers.pluginUploadSuccess'))
      close()
    } catch (err) {
      showErrorToast(err)
    }
  }

  return (
    <Dialog
      open={driver !== null}
      onClose={close}
      title={
        driver
          ? `${t('gateway.drivers.uploadPluginTitle')} · ${driverModelLabel(driver.model)}`
          : t('gateway.drivers.uploadPluginTitle')
      }
      size="lg"
      className="w-[400px] max-w-[calc(100vw-2rem)]"
      footer={
        <>
          <Button variant="outline" onClick={close}>
            {t('common.action.cancel')}
          </Button>
          <Button
            variant="primary"
            disabled={!file}
            loading={upload.isPending}
            onClick={() => void submit()}
          >
            {t('gateway.drivers.upload')}
          </Button>
        </>
      }
    >
      <div className="space-y-3">
        <label className="flex cursor-pointer items-center justify-center gap-2 rounded border border-dashed border-border-strong bg-surface-2 px-4 py-8 text-sm text-muted transition-colors hover:border-primary hover:text-fg">
          <Upload className="size-4" />
          {file ? file.name : t('gateway.drivers.pluginFile')}
          <Input
            type="file"
            accept=".so"
            className="hidden"
            onChange={selectFile}
          />
        </label>
        <p className="min-h-4 text-xs leading-4 text-danger">{error}</p>
      </div>
    </Dialog>
  )
}
