import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, CircleHelp, Save } from 'lucide-react'
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
  SearchSelect,
  Skeleton,
  showErrorToast,
  useI18n,
  type SearchSelectOption,
} from '@harns/core'
import {
  useCollection,
  useCreateCollection,
  useDriverList,
  useDriverSchema,
  useUpdateCollection,
} from '@/api/hooks'
import { ProtocolFields } from '@/components/ProtocolFields'
import { ProtocolLabel } from '@/components/ProtocolLabel'
import {
  createFormValues,
  detailToForm,
  validateCollectionForm,
  type CollectionFormValues,
} from '@/lib/collection-form'
import { driverModelLabel, driverTypeLabel } from '@/lib/constants'

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

  const selectedType = form?.driverType ?? ''
  const driverTypeOptions = useMemo<SearchSelectOption[]>(
    () => driverTypes.map((item) => ({ value: item.type, label: driverTypeLabel(item.type) })),
    [driverTypes],
  )
  const driverModelOptions = useMemo<SearchSelectOption[]>(
    () =>
      (driverTypes.find((item) => item.type === selectedType)?.models ?? []).map((model) => ({
        value: model,
        label: driverModelLabel(model),
      })),
    [driverTypes, selectedType],
  )

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

  const changeModel = (model: string) => {
    setSelectedModel(model)
    setForm(null)
    setErrors({})
  }

  const submit = async () => {
    if (!form || !schema.data) return
    const result = validateCollectionForm(form, schema.data.schema, t, false)
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
      showErrorToast(error)
    }
  }

  if (!isCreate && detail.isLoading) return <PageLoading />
  if (!isCreate && detail.isError) {
    return <ErrorState error={detail.error} onRetry={() => void detail.refetch()} />
  }

  const submitting = createCollection.isPending || updateCollection.isPending
  const driverSchema = schema.data?.schema

  return (
    <div className="space-y-3">
      {isCreate ? null : (
        <PageHeader
          title={t('gateway.form.editTitle', { name: detail.data?.name ?? '' })}
          description={t('gateway.form.description')}
          backTo={`/collections/${id}`}
          backLabel={t('gateway.form.back')}
          actions={
            <>
              <Link
                to={`/collections/${id}`}
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
                {t('gateway.form.submitUpdate')}
              </Button>
            </>
          }
        />
      )}

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
            <CardContent className="grid gap-x-4 gap-y-3 sm:grid-cols-2 xl:grid-cols-4">
              <Field
                label={t('gateway.form.field.name')}
                htmlFor="collection-name"
                labelClassName="text-sm"
                inline
                required
                error={errors.name}
              >
                <Input
                  id="collection-name"
                  value={form.name}
                  onChange={(event) => update({ name: event.target.value })}
                  placeholder={t('gateway.form.namePlaceholder')}
                  autoComplete="off"
                  className="w-full max-w-[200px]"
                />
              </Field>
              <Field
                label={t('gateway.form.field.cycle')}
                htmlFor="collection-cycle"
                labelClassName="text-sm"
                inline
                required
                error={errors.collectorCycleMs}
                labelAccessory={
                  <span className="group relative inline-flex">
                    <CircleHelp
                      aria-hidden="true"
                      className="size-3 cursor-help self-start text-faint transition-colors group-hover:text-fg"
                    />
                    <span
                      role="tooltip"
                      className="pointer-events-none absolute top-full right-0 z-20 mt-1.5 hidden w-max max-w-56 rounded border border-border bg-surface px-2 py-1.5 text-[11px] leading-relaxed text-fg shadow-lg group-hover:block"
                    >
                      {t('gateway.form.field.cycleHint')}
                    </span>
                  </span>
                }
              >
                <Input
                  id="collection-cycle"
                  value={form.collectorCycleMs}
                  inputMode="numeric"
                  className="w-full max-w-[200px]"
                  onChange={(event) => update({ collectorCycleMs: event.target.value })}
                />
              </Field>
              {isCreate ? (
                <>
                  <Field
                    label={t('gateway.form.field.driverType')}
                    htmlFor="driver-type"
                    labelClassName="text-sm"
                        inline
                  >
                    <SearchSelect
                      id="driver-type"
                      value={selectedType}
                      options={driverTypeOptions}
                      onChange={(type) => {
                        const first = driverTypes.find((item) => item.type === type)
                        if (first) changeModel(first.models[0])
                      }}
                      ariaLabel={t('gateway.form.field.driverType')}
                      emptyText={t('common.state.empty')}
                      className="w-full max-w-[200px]"
                      panelClassName="w-[200px]"
                    />
                  </Field>
                  <Field
                    label={t('gateway.form.field.driverModel')}
                    htmlFor="driver-model"
                    labelClassName="text-sm"
                        inline
                  >
                    <SearchSelect
                      id="driver-model"
                      value={form.driverModel}
                      options={driverModelOptions}
                      onChange={changeModel}
                      ariaLabel={t('gateway.form.field.driverModel')}
                      emptyText={t('common.state.empty')}
                      className="w-full max-w-[200px]"
                      panelClassName="w-[200px]"
                    />
                  </Field>
                </>
              ) : (
                <Field
                  label={t('gateway.form.field.protocol')}
                  labelClassName="text-sm"
                    inline
                >
                  <div className="flex h-8 items-center">
                    <ProtocolLabel type={form.driverType} model={form.driverModel} />
                  </div>
                </Field>
              )}
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

          {isCreate ? null : (
            <Card>
              <CardContent className="py-3">
                <p className="text-xs text-muted">{t('gateway.form.variablesHint')}</p>
              </CardContent>
            </Card>
          )}
        </>
      )}

      {isCreate ? (
        <div className="flex items-center justify-end gap-2 pb-1">
          <Button variant="outline" onClick={() => navigate('/collections')}>
            {t('common.action.cancel')}
          </Button>
          <Button
            variant="primary"
            loading={submitting}
            disabled={!form || !schema.data}
            onClick={() => void submit()}
          >
            {t('gateway.form.next')}
          </Button>
        </div>
      ) : null}
    </div>
  )
}
