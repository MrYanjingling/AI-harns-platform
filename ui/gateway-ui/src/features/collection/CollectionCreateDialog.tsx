import { useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react'
import { CircleHelp, Download, Upload } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  FormDialog,
  ErrorState,
  Field,
  Input,
  SearchSelect,
  Skeleton,
  Tabs,
  showErrorToast,
  useI18n,
  type SearchSelectOption,
} from '@harns/core'
import {
  useCreateCollection,
  useDriverList,
  useDriverSchema,
} from '@/api/hooks'
import { ProtocolFields } from '@/components/ProtocolFields'
import {
  createEmptyCalculatedVariable,
  createEmptyVariable,
  createFormValues,
  isBlankVariable,
  toVariableInputs,
  validateCollectionForm,
  validateVariableRows,
  type CollectionFormValues,
  type VariableFormValues,
} from '@/lib/collection-form'
import { driverModelLabel, driverTypeLabel } from '@/lib/constants'
import {
  exportCollectionVariablesTemplate,
  importCollectionVariablesFile,
} from './collection-variables-excel'
import { CalculatedVariablesEditor } from './CalculatedVariablesEditor'
import { CreateStepHeader } from './CreateStepHeader'
import { VariablesEditor } from './VariablesEditor'

interface CollectionCreateDialogProps {
  open: boolean
  onClose: () => void
}

type VariableTab = 'collection' | 'calculated'

function addCrossListDuplicateErrors(
  existing: VariableFormValues[],
  rows: VariableFormValues[],
  errors: Record<string, string>,
  message: string,
): void {
  const names = new Set(existing.map((variable) => variable.name.trim()).filter(Boolean))
  rows.forEach((variable, index) => {
    const name = variable.name.trim()
    if (name && names.has(name) && !errors[`variables.${index}.name`]) {
      errors[`variables.${index}.name`] = message
    }
  })
}

export function CollectionCreateDialog({ open, onClose }: CollectionCreateDialogProps) {
  const { t, locale } = useI18n()
  const drivers = useDriverList()
  const createCollection = useCreateCollection()
  const [step, setStep] = useState<1 | 2>(1)
  const [selectedModel, setSelectedModel] = useState('')
  const schema = useDriverSchema(selectedModel)
  const [form, setForm] = useState<CollectionFormValues | null>(null)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [variables, setVariables] = useState<VariableFormValues[]>([])
  const [calculated, setCalculated] = useState<VariableFormValues[]>([])
  const [variableErrors, setVariableErrors] = useState<Record<string, string>>({})
  const [calculatedErrors, setCalculatedErrors] = useState<Record<string, string>>({})
  const [tab, setTab] = useState<VariableTab>('collection')
  const [importing, setImporting] = useState(false)
  const [exporting, setExporting] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const preservedNameRef = useRef('')

  useEffect(() => {
    if (!open) return
    setStep(1)
    setSelectedModel('')
    setForm(null)
    setErrors({})
    setVariables([])
    setCalculated([])
    setVariableErrors({})
    setCalculatedErrors({})
    setTab('collection')
    setImporting(false)
    setExporting(false)
    preservedNameRef.current = ''
  }, [open])

  useEffect(() => {
    if (!open || selectedModel || !drivers.data?.length) return
    setSelectedModel(drivers.data[0].model)
  }, [open, selectedModel, drivers.data])

  useEffect(() => {
    if (!open || !selectedModel || !schema.data || form) return
    const next = createFormValues(
      schema.data.driverType,
      schema.data.driverModel,
      schema.data.schema,
    )
    next.name = preservedNameRef.current
    setForm(next)
  }, [open, selectedModel, schema.data, form])

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

  const driverSchema = schema.data?.schema
  const submitting = createCollection.isPending

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
    preservedNameRef.current = form?.name ?? preservedNameRef.current
    setSelectedModel(model)
    setForm(null)
    setErrors({})
    setVariables([])
    setCalculated([])
  }

  const validateBasics = (): boolean => {
    if (!form || !schema.data) return false
    const result = validateCollectionForm(form, schema.data.schema, t, false)
    setErrors(result.errors)
    return Boolean(result.payload)
  }

  const validateVariables = (
    nextVariables: VariableFormValues[] = variables,
    nextCalculated: VariableFormValues[] = calculated,
  ): boolean => {
    if (!driverSchema) return false
    const nextVariableErrors = validateVariableRows(nextVariables, driverSchema, t)
    const nextCalculatedErrors = validateVariableRows(nextCalculated, driverSchema, t)
    const duplicate = t('gateway.form.validation.variableNameDuplicate')
    addCrossListDuplicateErrors(nextVariables, nextCalculated, nextCalculatedErrors, duplicate)
    addCrossListDuplicateErrors(nextCalculated, nextVariables, nextVariableErrors, duplicate)
    setVariableErrors(nextVariableErrors)
    setCalculatedErrors(nextCalculatedErrors)
    return (
      Object.keys(nextVariableErrors).length === 0 &&
      Object.keys(nextCalculatedErrors).length === 0
    )
  }

  const exportTemplate = async () => {
    if (!driverSchema) return
    setExporting(true)
    try {
      await exportCollectionVariablesTemplate({ schema: driverSchema, locale, t })
    } catch (error) {
      showErrorToast(error)
    } finally {
      setExporting(false)
    }
  }

  const importVariables = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.currentTarget.files?.[0]
    event.currentTarget.value = ''
    if (!file || !driverSchema) return
    if (!file.name.toLowerCase().endsWith('.xlsx')) {
      showErrorToast(new Error(t('gateway.variable.importInvalidFile')))
      return
    }
    setImporting(true)
    try {
      const rows = await importCollectionVariablesFile({ file, schema: driverSchema, locale, t })
      const nextVariables = [
        ...variables.filter((variable) => !isBlankVariable(variable)),
        ...rows,
      ]
      setVariables(nextVariables)
      setTab('collection')
      if (validateVariables(nextVariables, calculated)) {
        toast.success(t('gateway.variable.imported', { count: rows.length }))
      } else {
        toast.error(t('gateway.variable.importWithErrors', { count: rows.length }))
      }
    } catch (error) {
      showErrorToast(error)
    } finally {
      setImporting(false)
    }
  }

  const next = () => {
    if (!validateBasics()) return
    if (variables.length === 0 && driverSchema) {
      setVariables(Array.from({ length: 5 }, () => createEmptyVariable(driverSchema)))
    }
    setStep(2)
  }

  const changeTab = (nextTab: VariableTab) => {
    if (nextTab === 'calculated' && calculated.length === 0 && driverSchema) {
      setCalculated(
        Array.from({ length: 5 }, () => createEmptyCalculatedVariable(driverSchema)),
      )
    }
    setTab(nextTab)
  }

  const save = async () => {
    if (!form || !schema.data) return
    const result = validateCollectionForm(form, schema.data.schema, t, false)
    setErrors(result.errors)
    if (!result.payload || !validateVariables()) return
    try {
      await createCollection.mutateAsync({
        ...result.payload,
        variables: toVariableInputs([...variables, ...calculated]),
      })
      toast.success(t('gateway.form.toast.created'))
      onClose()
    } catch (error) {
      showErrorToast(error)
    }
  }

  const footer =
    step === 1 ? (
      <>
        <Button variant="outline" onClick={onClose}>
          {t('common.action.cancel')}
        </Button>
        <Button
          variant="primary"
          disabled={!form || !schema.data}
          onClick={next}
        >
          {t('gateway.form.next')}
        </Button>
      </>
    ) : (
      <>
        <Button variant="outline" onClick={() => setStep(1)}>
          {t('gateway.form.previous')}
        </Button>
        <Button variant="ghost" onClick={onClose}>
          {t('common.action.cancel')}
        </Button>
        <Button variant="primary" loading={submitting} onClick={() => void save()}>
          {t('common.action.save')}
        </Button>
      </>
    )

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={t('gateway.form.createTitle')}
      footer={footer}
    >
      <CreateStepHeader current={step} />

      {schema.isError ? (
        <ErrorState error={schema.error} onRetry={() => void schema.refetch()} />
      ) : !form || !driverSchema ? (
        <div className="space-y-3">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-40 w-full" />
        </div>
      ) : step === 1 ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle>{t('gateway.form.section.basic')}</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-x-8 gap-y-4 sm:grid-cols-2">
              <Field
                label={t('gateway.form.field.name')}
                htmlFor="dialog-collection-name"
                labelClassName="text-sm"
                labelWidthClassName="w-[84px]"
                inline
                required
                error={errors.name}
              >
                <Input
                  id="dialog-collection-name"
                  value={form.name}
                  onChange={(event) => update({ name: event.target.value })}
                  placeholder={t('gateway.form.namePlaceholder')}
                  autoComplete="off"
                  className="w-full max-w-[200px]"
                />
              </Field>
              <Field
                label={t('gateway.form.field.cycle')}
                htmlFor="dialog-collection-cycle"
                labelClassName="text-sm"
                labelWidthClassName="w-[84px]"
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
                  id="dialog-collection-cycle"
                  value={form.collectorCycleMs}
                  inputMode="numeric"
                  className="w-full max-w-[200px]"
                  onChange={(event) => update({ collectorCycleMs: event.target.value })}
                />
              </Field>
              <Field
                label={t('gateway.form.field.driverType')}
                htmlFor="dialog-driver-type"
                labelClassName="text-sm"
                labelWidthClassName="w-[84px]"
                inline
              >
                <SearchSelect
                  id="dialog-driver-type"
                  value={selectedType}
                  options={driverTypeOptions}
                  onChange={(type) => {
                    const first = driverTypes.find((item) => item.type === type)
                    if (first) changeModel(first.models[0])
                  }}
                  ariaLabel={t('gateway.form.field.driverType')}
                  emptyText={t('common.state.empty')}
                  className="w-full max-w-[200px]"
                />
              </Field>
              <Field
                label={t('gateway.form.field.driverModel')}
                htmlFor="dialog-driver-model"
                labelClassName="text-sm"
                labelWidthClassName="w-[84px]"
                inline
              >
                <SearchSelect
                  id="dialog-driver-model"
                  value={form.driverModel}
                  options={driverModelOptions}
                  onChange={changeModel}
                  ariaLabel={t('gateway.form.field.driverModel')}
                  emptyText={t('common.state.empty')}
                  className="w-full max-w-[200px]"
                />
              </Field>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t('gateway.form.section.protocol')}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="app-dialog-error mb-2">{errors.form}</p>
              <ProtocolFields
                schema={driverSchema}
                driverModel={form.driverModel}
                values={form.protocol}
                errors={errors}
                onChange={updateProtocol}
                columns={2}
              />
            </CardContent>
          </Card>
        </>
      ) : (
        <>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <Tabs<VariableTab>
              items={[
                {
                  value: 'collection',
                  label: `${t('gateway.form.tab.collectionVariables')} (${variables.length})`,
                },
                {
                  value: 'calculated',
                  label: `${t('gateway.form.tab.calculatedVariables')} (${calculated.length})`,
                },
              ]}
              value={tab}
              onChange={changeTab}
            />
            {tab === 'collection' ? (
              <div className="flex items-center gap-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".xlsx"
                  className="hidden"
                  onChange={(event) => void importVariables(event)}
                />
                <Button
                  variant="outline"
                  size="sm"
                  loading={importing}
                  disabled={exporting}
                  icon={<Upload className="size-3.5" />}
                  onClick={() => fileInputRef.current?.click()}
                >
                  {t('gateway.variable.import')}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  loading={exporting}
                  disabled={importing}
                  icon={<Download className="size-3.5" />}
                  onClick={() => void exportTemplate()}
                >
                  {t('gateway.variable.exportTemplate')}
                </Button>
              </div>
            ) : null}
          </div>

          {tab === 'collection' ? (
            <VariablesEditor
              schema={driverSchema}
              variables={variables}
              errors={variableErrors}
              onAdd={() => setVariables((rows) => [...rows, createEmptyVariable(driverSchema)])}
              onRemove={(index) =>
                setVariables((rows) => rows.filter((_, rowIndex) => rowIndex !== index))
              }
              onChange={(index, patch) =>
                setVariables((rows) =>
                  rows.map((row, rowIndex) =>
                    rowIndex === index ? { ...row, ...patch } : row,
                  ),
                )
              }
            />
          ) : (
            <CalculatedVariablesEditor
              schema={driverSchema}
              variables={calculated}
              errors={calculatedErrors}
              onAdd={() =>
                setCalculated((rows) => [...rows, createEmptyCalculatedVariable(driverSchema)])
              }
              onRemove={(index) =>
                setCalculated((rows) => rows.filter((_, rowIndex) => rowIndex !== index))
              }
              onChange={(index, patch) =>
                setCalculated((rows) =>
                  rows.map((row, rowIndex) =>
                    rowIndex === index ? { ...row, ...patch } : row,
                  ),
                )
              }
            />
          )}
        </>
      )}
    </FormDialog>
  )
}
