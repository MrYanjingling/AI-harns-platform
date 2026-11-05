import { useRef, useState } from 'react'
import { Download, FileJson, Upload } from 'lucide-react'
import { toast } from 'sonner'
import { useImportThingModel } from '@/api/hooks'
import { exportThingModel } from '@/api/thing-models'
import type { ThingModelDetail, ThingModelDocument } from '@/api/types'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Textarea } from '@/components/ui/Field'

const MAX_DOCUMENT_BYTES = 1024 * 1024

export function TslPanel({ model }: { model: ThingModelDetail }) {
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
      toast.success('TSL 已导出')
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
      toast.error('文件不能超过 1 MiB')
      return
    }
    try {
      setText(await file.text())
    } catch {
      toast.error('读取文件失败')
    }
  }

  const submit = async () => {
    if (!text.trim()) {
      toast.error('请粘贴或选择 TSL JSON')
      return
    }
    let document: unknown
    try {
      document = JSON.parse(text)
    } catch {
      toast.error('JSON 语法错误,请检查后重试')
      return
    }
    if (!document || typeof document !== 'object' || !Array.isArray((document as { properties?: unknown }).properties)) {
      toast.error('导入内容必须包含 properties 数组')
      return
    }
    try {
      await importMutation.mutateAsync({
        id: model.id,
        document: document as ThingModelDocument,
      })
      toast.success('物模型已导入,属性已整体替换')
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
            <CardTitle>导出 TSL</CardTitle>
            <CardDescription>导出包含 Key、名称、属性与特征的 JSON 文档。</CardDescription>
          </div>
          <FileJson className="size-4 text-faint" />
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-xs leading-relaxed text-muted">
            当前物模型包含 <span className="font-medium text-fg tabular">{model.properties?.length ?? 0}</span>{' '}
            个属性、<span className="font-medium text-fg tabular">{model.features?.length ?? 0}</span>{' '}
            个特征定义。导出文件可直接用于备份或导入到其他物模型。
          </p>
          <Button
            variant="outline"
            icon={<Download className="size-4" />}
            loading={exporting}
            onClick={exportDocument}
          >
            导出 JSON
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>导入 TSL</CardTitle>
            <CardDescription>导入会整体替换属性;features 省略时保留原定义。</CardDescription>
          </div>
          <Upload className="size-4 text-faint" />
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-xs leading-relaxed text-muted">
            仅接受 JSON。文档中的 <code className="font-mono text-[11px]">thingModelKey</code>{' '}
            可省略;若提供则必须与当前 Key 一致。属性标识符在物模型内唯一。
          </p>
          <Button variant="primary" icon={<Upload className="size-4" />} onClick={openImport}>
            导入 JSON
          </Button>
        </CardContent>
      </Card>

      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        size="xl"
        title="导入物模型 TSL"
        description="导入成功后属性与特征会按文档整体替换,请先导出备份。"
        footer={
          <>
            <Button variant="ghost" onClick={() => setOpen(false)} disabled={importMutation.isPending}>
              取消
            </Button>
            <Button
              variant="primary"
              onClick={submit}
              loading={importMutation.isPending}
              disabled={!text.trim()}
            >
              导入并替换
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
              读取当前定义
            </Button>
            <Button size="sm" variant="outline" onClick={() => fileRef.current?.click()}>
              选择 JSON 文件
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
            <span className="text-[11px] text-faint">文件不超过 1 MiB</span>
          </div>
          <Field label="TSL JSON">
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
