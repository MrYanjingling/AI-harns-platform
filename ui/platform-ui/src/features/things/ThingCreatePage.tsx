import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Boxes, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useCreateThings, useThingModel, useThingModels } from '@/api/hooks'
import type { ThingDetail } from '@/api/types'
import { useI18n } from '@harns/core'
import { errorMessage } from '@harns/core'
import { Button } from '@harns/core'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@harns/core'
import { Dialog } from '@harns/core'
import { EmptyState } from '@harns/core'
import { Field, Input, Select } from '@harns/core'
import { Skeleton } from '@harns/core'
import { PageHeader } from '@harns/core'
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
  const { t } = useI18n()
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
      toast.error(t('thingCreate.maxBatch', { count: MAX_BATCH }))
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
        rowErrors.deviceName = t('thingCreate.validation.nameRequired')
      } else if (name.length > 64) {
        rowErrors.deviceName = t('thingCreate.validation.nameMax')
      } else if (names.has(name)) {
        rowErrors.deviceName = t('thingCreate.validation.nameDuplicate')
      }
      if (name) names.set(name, index)

      if (key) {
        if (!KEY_PATTERN.test(key)) {
          rowErrors.deviceKey = t('thingCreate.validation.keyPattern')
        } else if (keys.has(key)) {
          rowErrors.deviceKey = t('thingCreate.validation.keyDuplicate')
        }
        keys.set(key, index)
      }

      if (row.deviceDesc.length > 255) {
        rowErrors.deviceDesc = t('thingCreate.validation.descMax')
      }

      if (Object.keys(rowErrors).length > 0) next[row.key] = rowErrors
    })

    setErrors(next)
    return Object.keys(next).length === 0
  }

  const submit = async () => {
    if (!modelId) {
      toast.error(t('thingCreate.selectModelFirst'))
      return
    }
    if (!validate()) {
      toast.error(t('thingCreate.fixErrors'))
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
      toast.success(t('thingCreate.createdToast', { count: result.length }))
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  if (created) {
    return (
      <div>
        <PageHeader
          title={t('thingCreate.doneTitle')}
          description={t('thingCreate.doneDescription')}
          backTo="/platform/things"
          backLabel={t('thingCreate.back')}
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
        title={t('thingCreate.title')}
        description={t('thingCreate.description')}
        backTo="/platform/things"
        backLabel={t('thingCreate.back')}
      />

      {models.isLoading ? (
        <Skeleton className="h-32 w-full" />
      ) : models.isError ? (
        <EmptyState
          title={t('thingCreate.modelsFailed')}
          description={errorMessage(models.error)}
          action={
            <Button variant="outline" onClick={() => void models.refetch()}>
              {t('common.action.retry')}
            </Button>
          }
        />
      ) : !modelsLoaded ? (
        <EmptyState
          icon={<Boxes className="size-5" />}
          title={t('thingCreate.noModels')}
          description={t('thingCreate.noModelsDescription')}
          action={
            <Button
              variant="primary"
              onClick={() => navigate('/platform/thing-models/new')}
            >
              {t('thingModels.create')}
            </Button>
          }
        />
      ) : (
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <div>
                <CardTitle>{t('thingCreate.selectModel')}</CardTitle>
                <CardDescription>{t('thingCreate.selectModelDescription')}</CardDescription>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              <Field label={t('thingCreate.modelLabel')} required>
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
                    {t('thingCreate.modelSummary', {
                      name: model.data.name,
                      key: model.data.thingModelKey,
                    })}
                  </p>
                  <p className="mt-1">
                    {t('thingCreate.modelCounts', {
                      properties: model.data.properties?.length ?? 0,
                      features: model.data.features?.length ?? 0,
                    })}
                  </p>
                </div>
              ) : null}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <div>
                <CardTitle>{t('thingCreate.deviceList')}</CardTitle>
                <CardDescription>
                  {t('thingCreate.deviceListDescription', {
                    count: rows.length,
                    max: MAX_BATCH,
                  })}
                </CardDescription>
              </div>
              <Button size="sm" variant="outline" icon={<Plus className="size-3.5" />} onClick={addRow}>
                {t('thingCreate.addRow')}
              </Button>
            </CardHeader>
            <CardContent className="space-y-3">
              {rows.map((row, index) => {
                const rowErrors = errors[row.key] ?? {}
                return (
                  <div key={row.key} className="rounded-xl border border-border p-3">
                    <div className="mb-2 flex items-center justify-between">
                      <p className="text-xs font-medium text-muted">
                        {t('thingCreate.rowTitle', { index: index + 1 })}
                      </p>
                      <div className="flex items-center gap-1">
                        <Button
                          size="sm"
                          variant="ghost"
                          disabled={features.length === 0}
                          title={
                            features.length === 0
                              ? t('thingCreate.featuresTitle')
                              : t('thingCreate.featuresHint')
                          }
                          onClick={() => openFeatures(row)}
                        >
                          {t('thingCreate.features')}
                          {Object.keys(row.features).length > 0 ? (
                            <span className="ml-1 rounded-full bg-primary-soft px-1.5 text-[10px] text-primary tabular">
                              {Object.keys(row.features).length}
                            </span>
                          ) : null}
                        </Button>
                        <Button
                          size="icon"
                          variant="ghost"
                          aria-label={t('thingCreate.deleteRowAria', { index: index + 1 })}
                          disabled={rows.length === 1}
                          onClick={() => removeRow(row.key)}
                        >
                          <Trash2 className="size-3.5 text-danger" />
                        </Button>
                      </div>
                    </div>
                    <div className="grid gap-3 lg:grid-cols-[1.2fr_1fr_1.4fr]">
                      <Field
                        label={t('thingCreate.deviceName')}
                        required
                        error={rowErrors.deviceName}
                        htmlFor={`name-${row.key}`}
                      >
                        <Input
                          id={`name-${row.key}`}
                          value={row.deviceName}
                          placeholder={t('thingCreate.deviceNamePlaceholder')}
                          onChange={(event) => updateRow(row.key, { deviceName: event.target.value })}
                        />
                      </Field>
                      <Field
                        label={t('thingCreate.deviceKey')}
                        error={rowErrors.deviceKey}
                        hint={t('thingCreate.optional')}
                        htmlFor={`key-${row.key}`}
                      >
                        <Input
                          id={`key-${row.key}`}
                          value={row.deviceKey}
                          placeholder={t('thingCreate.deviceKeyPlaceholder')}
                          className="font-mono"
                          onChange={(event) => updateRow(row.key, { deviceKey: event.target.value })}
                        />
                      </Field>
                      <Field
                        label={t('thingCreate.deviceDesc')}
                        error={rowErrors.deviceDesc}
                        htmlFor={`desc-${row.key}`}
                      >
                        <Input
                          id={`desc-${row.key}`}
                          value={row.deviceDesc}
                          placeholder={t('dataType.optional')}
                          onChange={(event) => updateRow(row.key, { deviceDesc: event.target.value })}
                        />
                      </Field>
                    </div>
                  </div>
                )
              })}

              <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
                <p className="text-[11px] text-faint">{t('thingCreate.keyNote')}</p>
                <div className="flex gap-2">
                  <Button variant="ghost" onClick={() => navigate('/platform/things')}>
                    {t('common.action.cancel')}
                  </Button>
                  <Button variant="primary" onClick={submit} loading={createMutation.isPending}>
                    {t('thingCreate.createCount', { count: rows.length })}
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
        title={t('thingCreate.featureDialogTitle', {
          name: featureRow?.deviceName || t('thingCreate.newDevice'),
        })}
        description={t('thingCreate.featureDialogDescription')}
        size="lg"
        footer={
          <>
            <Button variant="ghost" onClick={() => setFeatureRowKey(null)}>
              {t('common.action.cancel')}
            </Button>
            <Button variant="primary" onClick={saveFeatures}>
              {t('common.action.apply')}
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
