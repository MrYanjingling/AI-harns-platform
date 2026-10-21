import { useRef, useState } from 'react'
import { Download, FileJson, Upload } from 'lucide-react'
import { toast } from 'sonner'
import { useImportThingModel } from '@/api/hooks'
import { exportThingModel } from '@/api/thing-models'
import type { ThingModelDetail, ThingModelDocument } from '@/api/types'
import { useI18n } from '@harns/core'
import { errorMessage } from '@harns/core'
import { Button } from '@harns/core'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@harns/core'
import { Dialog } from '@harns/core'
import { Field, Textarea } from '@harns/core'

const MAX_DOCUMENT_BYTES = 1024 * 1024

export function TslPanel({ model }: { model: ThingModelDetail }) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const [text, setText] = useState('')
  const [exporting, setExporting] = useState(false)
  const [loadingCurrent, setLoadingCurrent] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)
  const importMutation = useImportThingModel()

  const exportDocument = async () => {
    setExporting(true)
    try {
      const doc = await exportThingModel(model.id)
      const blob = new Blob([JSON.stringify(doc, null, 2)], {
        type: 'application/json;charset=utf-8',
      })
      const url = URL.createObjectURL(blob)
      const link = createDownloadLink()
      link.href = url
      link.download = `${model.thingModelKey || 'thing-model'}.json`
      link.click()
      URL.revokeObjectURL(url)
      toast.success(t('tsl.exported'))
    } catch (error) {
      toast.error(errorMessage(error))
    } finally {
      setExporting(false)
    }
  }

  const loadCurrent = async () => {
    setLoadingCurrent(true)
    try {
      const doc = await exportThingModel(model.id)
      setText(JSON.stringify(doc, null, 2))
    } catch (error) {
      toast.error(errorMessage(error))
    } finally {
      setLoadingCurrent(false)
    }
  }

  const openImport = () => {
    setText('')
    setOpen(true)
  }

  const onPickFile = async (file: File | undefined) => {
    if (!file) return
    if (file.size > MAX_DOCUMENT_BYTES) {
      toast.error(t('tsl.fileTooLarge'))
      return
    }
    try {
      setText(await file.text())
    } catch {
      toast.error(t('tsl.readFailed'))
    }
  }

  const submit = async () => {
    if (!text.trim()) {
      toast.error(t('tsl.pasteRequired'))
      return
    }
    let document: unknown
    try {
      document = JSON.parse(text)
    } catch {
      toast.error(t('tsl.jsonInvalid'))
      return
    }
    if (!document || typeof document !== 'object' || !Array.isArray((document as { properties?: unknown }).properties)) {
      toast.error(t('tsl.propertiesRequired'))
      return
    }
    try {
      await importMutation.mutateAsync({
        id: model.id,
        document: document as ThingModelDocument,
      })
      toast.success(t('tsl.imported'))
      setOpen(false)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('tsl.exportTitle')}</CardTitle>
            <CardDescription>{t('tsl.exportDescription')}</CardDescription>
          </div>
          <FileJson className="size-4 text-faint" />
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-xs leading-relaxed text-muted">
            {t('tsl.exportSummary', {
              properties: model.properties?.length ?? 0,
              features: model.features?.length ?? 0,
            })}
          </p>
          <Button
            variant="outline"
            icon={<Download className="size-4" />}
            loading={exporting}
            onClick={exportDocument}
          >
            {t('tsl.exportJson')}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{t('tsl.importTitle')}</CardTitle>
            <CardDescription>{t('tsl.importDescription')}</CardDescription>
          </div>
          <Upload className="size-4 text-faint" />
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-xs leading-relaxed text-muted">{t('tsl.importHint')}</p>
          <Button variant="primary" icon={<Upload className="size-4" />} onClick={openImport}>
            {t('tsl.importJson')}
          </Button>
        </CardContent>
      </Card>

      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        size="xl"
        title={t('tsl.dialogTitle')}
        description={t('tsl.dialogDescription')}
        footer={
          <>
            <Button variant="ghost" onClick={() => setOpen(false)} disabled={importMutation.isPending}>
              {t('common.action.cancel')}
            </Button>
            <Button
              variant="primary"
              onClick={submit}
              loading={importMutation.isPending}
              disabled={!text.trim()}
            >
              {t('tsl.importReplace')}
            </Button>
          </>
        }
      >
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              loading={loadingCurrent}
              onClick={loadCurrent}
            >
              {t('tsl.loadCurrent')}
            </Button>
            <Button size="sm" variant="outline" onClick={() => fileRef.current?.click()}>
              {t('tsl.pickFile')}
            </Button>
            <input
              ref={fileRef}
              type="file"
              accept="application/json,.json"
              className="hidden"
              onChange={(event) => {
                void onPickFile(event.target.files?.[0])
                event.target.value = ''
              }}
            />
            <span className="text-[11px] text-faint">{t('tsl.fileLimit')}</span>
          </div>
          <Field label={t('tsl.jsonField')}>
            <Textarea
              value={text}
              className="min-h-80 font-mono text-xs"
              placeholder={'{\n  "properties": []\n}'}
              spellCheck={false}
              onChange={(event) => setText(event.target.value)}
            />
          </Field>
        </div>
      </Dialog>
    </div>
  )
}

function createDownloadLink(): HTMLAnchorElement {
  const link = window.document.createElement('a')
  link.rel = 'noopener'
  return link
}
