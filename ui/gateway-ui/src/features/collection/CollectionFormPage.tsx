import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Save } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  ErrorState,
  Field,
  Input,
  PageHeader,
  PageLoading,
  Select,
  Skeleton,
  errorMessage,
  useI18n,
} from '@harns/core'
import {
  useCollection,
  useCreateCollection,
  useDriverList,
  useDriverSchema,
  useUpdateCollection,
} from '@/api/hooks'
import type { DriverSchema } from '@/api/types'
import { ProtocolFields } from '@/components/ProtocolFields'
import { ProtocolLabel } from '@/components/ProtocolLabel'
import {
  createEmptyVariable,
  createFormValues,
  detailToForm,
  validateCollectionForm,
  type CollectionFormValues,
  type VariableFormValues,
} from '@/lib/collection-form'
import { driverModelLabel, driverTypeLabel } from '@/lib/constants'
import { VariablesEditor } from './VariablesEditor'

interface CollectionFormPageProps {
  mode: 'create' | 'edit'
}

export function CollectionFormPage({ mode }: CollectionFormPageProps) {
  const { t } = useI18n()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isCreate = mode === 'create'

  const detail = useCollection(isCreate ? undefined : id)
  const drivers = useDriverList()
  const [selectedModel, setSelectedModel] = useState('')
  const schema = useDriverSchema(
    isCreate ? selectedModel : detail.data?.driverModel,
  )
  const createCollection = useCreateCollection()
  const updateCollection = useUpdateCollection()

  const [form, setForm] = useState<CollectionFormValues | null>(null)
  const [errors, setErrors] = useState<Record<string, string>>({})

  // 新建:驱动列表就绪后选中第一个型号,再由 Schema 初始化表单。
  useEffect(() => {
    if (isCreate && !selectedModel && drivers.data?.length) {
      setSelectedModel(drivers.data[0].model)
    }
  }, [isCreate, selectedModel, drivers.data])

  useEffect(() => {
    if (!isCreate || !selectedModel || !schema.data || form) return
    setForm(
      createFormValues(schema.data.driverType, schema.data.driverModel, schema.data.schema),
    )
  }, [isCreate, selectedModel, schema.data, form])

  // 编辑:详情与 Schema 都就绪后用 protocolConfig 初始化。
  useEffect(() => {
    if (isCreate || form || !detail.data || !schema.data) return
    setForm(detailToForm(detail.data, schema.data.schema))
  }, [isCreate, form, detail.data, schema.data])

  const driverTypes = useMemo(() => {
    const map = new Map<string, Set<string>>()
    for (const driver of drivers.data ?? []) {
      const models = map.get(driver.driverType) ?? new Set<string>()
      models.add(driver.model)
      map.set(driver.driverType, models)
    }
    return [...map.entries()].map(([type, models]) => ({
      type,
      models: [...models],
    }))
  }, [drivers.data])

  const update = (patch: Partial<CollectionFormValues>) => {
    setForm((current) => (current ? { ...current, ...patch } : current))
  }

  const updateProtocol = (name: string, value: string) => {
    setForm((current) =>
      current
        ? { ...current, protocol: { ...current.protocol, [name]: value } }
        : current,
    )
  }

  const updateVariable = (index: number, patch: Partial<VariableFormValues>) => {
    setForm((current) => {
      if (!current) return current
      const variables = current.variables.map((item, itemIndex) =>
        itemIndex === index ? { ...item, ...patch } : item,
      )
      return { ...current, variables }
    })
  }

  const changeModel = (model: string) => {
    setSelectedModel(model)
    setForm(null)
    setErrors({})
  }

  const submit = async () => {
    if (!form || !schema.data) return
    const result = validateCollectionForm(form, schema.data.schema, t, isCreate)
    setErrors(result.errors)
    if (!result.payload) return
    try {
      if (isCreate) {
        const created = await createCollection.mutateAsync(result.payload)
        toast.success(t('gateway.form.toast.created'))
        navigate(`/collections/${created.id}`)
      } else if (id) {
        await updateCollection.mutateAsync({
          id,
          name: result.payload.name,
          payload: result.payload,
        })
        toast.success(t('gateway.form.toast.updated'))
        navigate(`/collections/${id}`)
      }
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  if (!isCreate && detail.isLoading) return <PageLoading />
  if (!isCreate && detail.isError) {
    return <ErrorState error={detail.error} onRetry={() => void detail.refetch()} />
  }

  const submitting = createCollection.isPending || updateCollection.isPending
  const selectedType = form?.driverType ?? ''
  const driverSchema = schema.data?.schema

  return (
    <div className="space-y-3">
      <PageHeader
        title={
          isCreate ? t('gateway.form.createTitle') : t('gateway.form.editTitle', {
            name: detail.data?.name ?? '',
          })
        }
        description={t('gateway.form.description')}
        backTo={isCreate ? '/collections' : `/collections/${id}`}
        backLabel={t('gateway.form.back')}
        actions={
          <>
            <Link
              to={isCreate ? '/collections' : `/collections/${id}`}
              className="inline-flex h-8 items-center gap-1.5 rounded border border-border-strong bg-surface px-3 text-sm text-fg transition-colors hover:bg-surface-2"
            >
              <ArrowLeft className="size-3.5" />
              {t('common.action.cancel')}
            </Link>
            <Button
              variant="primary"
              loading={submitting}
              disabled={!form || !schema.data}
              onClick={() => void submit()}
              icon={<Save className="size-3.5" />}
            >
              {isCreate ? t('gateway.form.submitCreate') : t('gateway.form.submitUpdate')}
            </Button>
          </>
        }
      />

      {schema.isError ? (
        <ErrorState error={schema.error} onRetry={() => void schema.refetch()} />
      ) : !form || !driverSchema ? (
        <Card className="p-4">
          <div className="space-y-3">
            <Skeleton className="h-8 w-64" />
            <Skeleton className="h-40 w-full" />
          </div>
        </Card>
      ) : (
        <>
          <Card>
            <CardHeader>
              <CardTitle>{t('gateway.form.section.basic')}</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {isCreate ? (
                <>
                  <Field label={t('gateway.form.field.driverType')} htmlFor="driver-type">
                    <Select
                      id="driver-type"
                      value={selectedType}
                      onChange={(event) => {
                        const type = event.target.value
                        const first = driverTypes.find((item) => item.type === type)
                        if (first) changeModel(first.models[0])
                      }}
                    >
                      {driverTypes.map((item) => (
                        <option key={item.type} value={item.type}>
                          {driverTypeLabel(item.type)}
                        </option>
                      ))}
                    </Select>
                  </Field>
                  <Field label={t('gateway.form.field.driverModel')} htmlFor="driver-model">
                    <Select
                      id="driver-model"
                      value={form.driverModel}
                      onChange={(event) => changeModel(event.target.value)}
                    >
                      {(driverTypes.find((item) => item.type === selectedType)?.models ?? []).map(
                        (model) => (
                          <option key={model} value={model}>
                            {driverModelLabel(model)}
                          </option>
                        ),
                      )}
                    </Select>
                  </Field>
                </>
              ) : (
                <Field label={t('gateway.form.field.protocol')}>
                  <div className="flex h-8 items-center">
                    <ProtocolLabel type={form.driverType} model={form.driverModel} />
                  </div>
                </Field>
              )}
              <Field
                label={t('gateway.form.field.name')}
                htmlFor="collection-name"
                required
                error={errors.name}
              >
                <Input
                  id="collection-name"
                  value={form.name}
                  onChange={(event) => update({ name: event.target.value })}
                  placeholder={t('gateway.form.namePlaceholder')}
                  autoComplete="off"
                />
              </Field>
              <Field
                label={t('gateway.form.field.cycle')}
                htmlFor="collection-cycle"
                required
                error={errors.collectorCycleMs}
                hint={t('gateway.form.field.cycleHint')}
              >
                <Input
                  id="collection-cycle"
                  value={form.collectorCycleMs}
                  inputMode="numeric"
                  onChange={(event) => update({ collectorCycleMs: event.target.value })}
                />
              </Field>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t('gateway.form.section.protocol')}</CardTitle>
            </CardHeader>
            <CardContent>
              {errors.form ? (
                <p className="mb-2 text-xs text-danger">{errors.form}</p>
              ) : null}
              <ProtocolFields
                schema={driverSchema}
                values={form.protocol}
                errors={errors}
                onChange={updateProtocol}
              />
            </CardContent>
          </Card>

          {isCreate ? (
            <Card>
              <CardHeader>
                <CardTitle>{t('gateway.form.section.variables')}</CardTitle>
              </CardHeader>
              <CardContent>
                <VariablesForm
                  form={form}
                  errors={errors}
                  schema={driverSchema}
                  updateVariable={updateVariable}
                  update={(variables) => update({ variables })}
                />
              </CardContent>
            </Card>
          ) : (
            <Card>
              <CardContent className="py-3">
                <p className="text-xs text-muted">{t('gateway.form.variablesHint')}</p>
              </CardContent>
            </Card>
          )}
        </>
      )}
    </div>
  )
}

function VariablesForm({
  form,
  schema,
  errors,
  update,
  updateVariable,
}: {
  form: CollectionFormValues
  schema: DriverSchema
  errors: Record<string, string>
  update: (variables: VariableFormValues[]) => void
  updateVariable: (index: number, patch: Partial<VariableFormValues>) => void
}) {
  return (
    <VariablesEditor
      schema={schema}
      variables={form.variables}
      errors={errors}
      onAdd={() => update([...form.variables, createEmptyVariable(schema)])}
      onRemove={(index) => update(form.variables.filter((_, itemIndex) => itemIndex !== index))}
      onChange={updateVariable}
    />
  )
}
