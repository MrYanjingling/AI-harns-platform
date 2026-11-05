import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Boxes, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useCreateThings, useThingModel, useThingModels } from '@/api/hooks'
import type { ThingDetail } from '@/api/types'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { Field, Input, Select } from '@/components/ui/Field'
import { Skeleton } from '@/components/ui/Skeleton'
import { PageHeader } from '@/components/shared/PageHeader'
import { CreatedThingsPanel } from './CreatedThingsPanel'
import { FeatureValuesEditor } from './FeatureValuesEditor'

const MAX_BATCH = 100
const KEY_PATTERN = /^[a-zA-Z][a-zA-Z0-9_-]{3,63}$/
const MODEL_LOOKUP_SIZE = 100

interface Row {
  key: string
  deviceName: string
  deviceKey: string
  deviceDesc: string
  features: Record<string, unknown>
}

interface RowErrors {
  deviceName?: string
  deviceKey?: string
  deviceDesc?: string
}

let rowSequence = 0

function newRow(): Row {
  rowSequence += 1
  return {
    key: `row-${rowSequence}`,
    deviceName: '',
    deviceKey: '',
    deviceDesc: '',
    features: {},
  }
}

export function ThingCreatePage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const initialModelId = searchParams.get('thingModelId') ?? ''

  const models = useThingModels({ page: 1, pageSize: MODEL_LOOKUP_SIZE })
  const [modelId, setModelId] = useState(initialModelId)
  const model = useThingModel(modelId || undefined)

  const [rows, setRows] = useState<Row[]>([newRow()])
  const [errors, setErrors] = useState<Record<string, RowErrors>>({})
  const [featureRowKey, setFeatureRowKey] = useState<string | null>(null)
  const [featureDraft, setFeatureDraft] = useState<Record<string, unknown>>({})
  const [created, setCreated] = useState<ThingDetail[] | null>(null)
  const createMutation = useCreateThings()

  useEffect(() => {
    if (!modelId && models.data?.list && models.data.list.length > 0) {
      setModelId(models.data.list[0].id)
    }
  }, [modelId, models.data])

  // 切换物模型后已有特征值不再有效,清空避免提交未定义的特征。
  useEffect(() => {
    if (!modelId) return
    setRows((current) => current.map((row) => ({ ...row, features: {} })))
  }, [modelId])

  const features = model.data?.features ?? []
  const featureRow = rows.find((row) => row.key === featureRowKey)

  const updateRow = (key: string, patch: Partial<Row>) => {
    setRows((current) => current.map((row) => (row.key === key ? { ...row, ...patch } : row)))
    setErrors((current) => {
      if (!current[key]) return current
      const next = { ...current }
      delete next[key]
      return next
    })
  }

  const addRow = () => {
    if (rows.length >= MAX_BATCH) {
      toast.error(`单批最多创建 ${MAX_BATCH} 台设备`)
      return
    }
    setRows((current) => [...current, newRow()])
  }

  const removeRow = (key: string) => {
    setRows((current) => (current.length === 1 ? current : current.filter((row) => row.key !== key)))
    setErrors((current) => {
      const next = { ...current }
      delete next[key]
      return next
    })
  }

  const openFeatures = (row: Row) => {
    setFeatureRowKey(row.key)
    setFeatureDraft(row.features)
  }

  const saveFeatures = () => {
    if (featureRowKey) updateRow(featureRowKey, { features: featureDraft })
    setFeatureRowKey(null)
  }

  const validate = (): boolean => {
    const next: Record<string, RowErrors> = {}
    const names = new Map<string, number>()
    const keys = new Map<string, number>()

    rows.forEach((row, index) => {
      const rowErrors: RowErrors = {}
      const name = row.deviceName.trim()
      const key = row.deviceKey.trim()

      if (!name) {
        rowErrors.deviceName = '请输入设备名称'
      } else if (name.length > 64) {
        rowErrors.deviceName = '设备名称不能超过 64 个字符'
      } else if (names.has(name)) {
        rowErrors.deviceName = '批内设备名称不能重复'
      }
      if (name) names.set(name, index)

      if (key) {
        if (!KEY_PATTERN.test(key)) {
          rowErrors.deviceKey = 'Key 需以字母开头,仅含字母、数字、下划线、连字符,长度 4~64'
        } else if (keys.has(key)) {
          rowErrors.deviceKey = '批内 Device Key 不能重复'
        }
        keys.set(key, index)
      }

      if (row.deviceDesc.length > 255) {
        rowErrors.deviceDesc = '描述不能超过 255 个字符'
      }

      if (Object.keys(rowErrors).length > 0) next[row.key] = rowErrors
    })

    setErrors(next)
    return Object.keys(next).length === 0
  }

  const submit = async () => {
    if (!modelId) {
      toast.error('请先选择物模型')
      return
    }
    if (!validate()) {
      toast.error('请修正表单中的错误')
      return
    }
    try {
      const result = await createMutation.mutateAsync({
        thingModelId: modelId,
        things: rows.map((row) => ({
          deviceName: row.deviceName.trim(),
          ...(row.deviceKey.trim() ? { deviceKey: row.deviceKey.trim() } : {}),
          ...(row.deviceDesc.trim() ? { deviceDesc: row.deviceDesc.trim() } : {}),
          ...(Object.keys(row.features).length > 0 ? { features: row.features } : {}),
        })),
      })
      setCreated(result)
      toast.success(`已创建 ${result.length} 台设备`)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  if (created) {
    return (
      <div>
        <PageHeader
          title="创建设备完成"
          description="请保存好设备凭证,并配置 MQTT 或 HTTP 上报。"
          backTo="/platform/things"
          backLabel="返回设备列表"
        />
        <CreatedThingsPanel
          things={created}
          onContinue={() => {
            setCreated(null)
            setRows([newRow()])
          }}
        />
      </div>
    )
  }

  const modelsLoaded = models.isSuccess && (models.data?.list?.length ?? 0) > 0

  return (
    <div>
      <PageHeader
        title="批量新建设备"
        description="一次最多创建 100 台设备;Device Key 留空时由后端生成。"
        backTo="/platform/things"
        backLabel="返回设备列表"
      />

      {models.isLoading ? (
        <Skeleton className="h-32 w-full" />
      ) : models.isError ? (
        <EmptyState
          title="物模型加载失败"
          description={errorMessage(models.error)}
          action={
            <Button variant="outline" onClick={() => void models.refetch()}>
              重试
            </Button>
          }
        />
      ) : !modelsLoaded ? (
        <EmptyState
          icon={<Boxes className="size-5" />}
          title="还没有可用的物模型"
          description="设备必须归属一个物模型,请先创建物模型。"
          action={
            <Button
              variant="primary"
              onClick={() => navigate('/platform/thing-models/new')}
            >
              新建物模型
            </Button>
          }
        />
      ) : (
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <div>
                <CardTitle>选择物模型</CardTitle>
                <CardDescription>同一批设备必须归属同一个物模型。</CardDescription>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              <Field label="物模型" required>
                <Select value={modelId} onChange={(event) => setModelId(event.target.value)}>
                  {(models.data?.list ?? []).map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name} · {item.thingModelKey}
                    </option>
                  ))}
                </Select>
              </Field>
              {model.data ? (
                <div className="rounded-xl border border-border bg-surface-2/50 px-3 py-2.5 text-xs text-muted">
                  <p>
                    物模型 <span className="font-medium text-fg">{model.data.name}</span> ·{' '}
                    <span className="font-mono text-[11px]">{model.data.thingModelKey}</span>
                  </p>
                  <p className="mt-1">
                    {model.data.properties?.length ?? 0} 个属性 ·{' '}
                    {model.data.features?.length ?? 0} 个特征定义
                  </p>
                </div>
              ) : null}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <div>
                <CardTitle>设备列表</CardTitle>
                <CardDescription>
                  已添加 {rows.length} / {MAX_BATCH} 台。图片可在设备详情上传。
                </CardDescription>
              </div>
              <Button size="sm" variant="outline" icon={<Plus className="size-3.5" />} onClick={addRow}>
                添加一行
              </Button>
            </CardHeader>
            <CardContent className="space-y-3">
              {rows.map((row, index) => {
                const rowErrors = errors[row.key] ?? {}
                return (
                  <div key={row.key} className="rounded-xl border border-border p-3">
                    <div className="mb-2 flex items-center justify-between">
                      <p className="text-xs font-medium text-muted">设备 {index + 1}</p>
                      <div className="flex items-center gap-1">
                        <Button
                          size="sm"
                          variant="ghost"
                          disabled={features.length === 0}
                          title={
                            features.length === 0 ? '当前物模型没有特征定义' : '填写静态特征值'
                          }
                          onClick={() => openFeatures(row)}
                        >
                          特征值
                          {Object.keys(row.features).length > 0 ? (
                            <span className="ml-1 rounded-full bg-primary-soft px-1.5 text-[10px] text-primary tabular">
                              {Object.keys(row.features).length}
                            </span>
                          ) : null}
                        </Button>
                        <Button
                          size="icon"
                          variant="ghost"
                          aria-label={`删除第 ${index + 1} 行`}
                          disabled={rows.length === 1}
                          onClick={() => removeRow(row.key)}
                        >
                          <Trash2 className="size-3.5 text-danger" />
                        </Button>
                      </div>
                    </div>
                    <div className="grid gap-3 lg:grid-cols-[1.2fr_1fr_1.4fr]">
                      <Field
                        label="设备名称"
                        required
                        error={rowErrors.deviceName}
                        htmlFor={`name-${row.key}`}
                      >
                        <Input
                          id={`name-${row.key}`}
                          value={row.deviceName}
                          placeholder="例如 1# 温控器"
                          onChange={(event) => updateRow(row.key, { deviceName: event.target.value })}
                        />
                      </Field>
                      <Field
                        label="Device Key"
                        error={rowErrors.deviceKey}
                        hint="可选"
                        htmlFor={`key-${row.key}`}
                      >
                        <Input
                          id={`key-${row.key}`}
                          value={row.deviceKey}
                          placeholder="留空自动生成"
                          className="font-mono"
                          onChange={(event) => updateRow(row.key, { deviceKey: event.target.value })}
                        />
                      </Field>
                      <Field
                        label="描述"
                        error={rowErrors.deviceDesc}
                        htmlFor={`desc-${row.key}`}
                      >
                        <Input
                          id={`desc-${row.key}`}
                          value={row.deviceDesc}
                          placeholder="可选"
                          onChange={(event) => updateRow(row.key, { deviceDesc: event.target.value })}
                        />
                      </Field>
                    </div>
                  </div>
                )
              })}

              <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
                <p className="text-[11px] text-faint">
                  未填写 Device Key 时后端生成 <code className="font-mono">dk_</code> 前缀的 Key。
                </p>
                <div className="flex gap-2">
                  <Button variant="ghost" onClick={() => navigate('/platform/things')}>
                    取消
                  </Button>
                  <Button variant="primary" onClick={submit} loading={createMutation.isPending}>
                    创建 {rows.length} 台设备
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      <Dialog
        open={Boolean(featureRow)}
        onClose={() => setFeatureRowKey(null)}
        title={`特征值 · ${featureRow?.deviceName || '新设备'}`}
        description="特征值是设备的静态属性,可留空;之后也可在设备详情修改。"
        size="lg"
        footer={
          <>
            <Button variant="ghost" onClick={() => setFeatureRowKey(null)}>
              取消
            </Button>
            <Button variant="primary" onClick={saveFeatures}>
              应用
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
    </div>
  )
}
