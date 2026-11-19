import { useEffect, useState, type ChangeEvent } from 'react'
import { Download, Upload } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Dialog,
  Input,
  showErrorToast,
  useI18n,
} from '@harns/core'
import { useUploadDriverPlugin } from '@/api/hooks'
import { downloadDriverPluginTemplate } from '@/api/gateway'
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
  const [downloading, setDownloading] = useState(false)

  useEffect(() => {
    setFile(null)
    setError('')
    setDownloading(false)
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

  const downloadTemplate = async () => {
    if (!driver) return
    setDownloading(true)
    try {
      const data = await downloadDriverPluginTemplate(driver.id)
      const url = URL.createObjectURL(data)
      const link = document.createElement('a')
      link.href = url
      link.download = `${driver.pluginName || driver.model}.zip`
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(url)
    } catch (err) {
      showErrorToast(err)
    } finally {
      setDownloading(false)
    }
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
      headerActions={
        <Button
          variant="outline"
          size="sm"
          loading={downloading}
          icon={<Download className="size-3.5" />}
          onClick={() => void downloadTemplate()}
        >
          {t('gateway.drivers.downloadTemplate')}
        </Button>
      }
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
      <div>
        <label className="mt-2.5 flex min-h-[120px] cursor-pointer items-center justify-center gap-2 rounded border border-dashed border-border-strong bg-surface-2 px-4 py-0 text-sm text-muted transition-colors hover:border-primary hover:text-fg">
          <Upload className="size-4" />
          {file ? file.name : t('gateway.drivers.pluginFile')}
          <Input
            type="file"
            accept=".so"
            className="hidden"
            onChange={selectFile}
          />
        </label>
        {error ? <p className="app-dialog-error mt-2">{error}</p> : null}
      </div>
    </Dialog>
  )
}
