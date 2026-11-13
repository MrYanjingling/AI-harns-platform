import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { useAddThingModelProperties, useUpdateThingModelProperty } from '@/api/hooks'
import type { RwFlag, ThingDataTypeDetail, ThingModelProperty } from '@/api/types'
import { useI18n } from '@harns/core'
import { RW_FLAGS } from '@/lib/constants'
import { errorMessage } from '@harns/core'
import { Button } from '@harns/core'
import { Dialog } from '@harns/core'
import { Field, Input, Select, Textarea } from '@harns/core'
import { Switch } from '@harns/core'
import { DataTypeEditor } from './DataTypeEditor'
import { emptyDataType, normalizeDataType, validatePropertyInput, type FieldIssue } from './data-type'

const DEFAULT_SORT = 0

export function PropertyDialog({
  open,
  onClose,
  modelId,
  property,
}: {
  open: boolean
  onClose: () => void
  modelId: string
  property?: ThingModelProperty
}) {
  const isEdit = Boolean(property)
  const { t } = useI18n()
  const addMutation = useAddThingModelProperties()
  const updateMutation = useUpdateThingModelProperty()

  const [identifier, setIdentifier] = useState('')
  const [name, setName] = useState('')
  const [rwFlag, setRwFlag] = useState<RwFlag>('r')
  const [required, setRequired] = useState(false)
  const [description, setDescription] = useState('')
  const [sort, setSort] = useState(DEFAULT_SORT)
  const [dataType, setDataType] = useState<ThingDataTypeDetail>(() => emptyDataType('double'))
  const [issues, setIssues] = useState<FieldIssue[]>([])

  useEffect(() => {
    if (!open) return
    setIssues([])
    if (property) {
      setIdentifier(property.identifier)
      setName(property.name)
      setRwFlag(property.rwFlag ?? 'r')
      setRequired(Boolean(property.required))
      setDescription(property.description ?? '')
      setSort(property.sort ?? DEFAULT_SORT)
      setDataType(property.dataType)
    } else {
      setIdentifier('')
      setName('')
      setRwFlag('r')
      setRequired(false)
      setDescription('')
      setSort(DEFAULT_SORT)
      setDataType(emptyDataType('double'))
    }
  }, [open, property])

  const pending = addMutation.isPending || updateMutation.isPending
  const fieldError = (key: string) => issues.find((issue) => issue.field === key)?.message
  const dataTypeIssues = issues.filter((issue) => issue.field.startsWith('dataType'))

  const submit = async () => {
    const payload = {
      identifier: identifier.trim(),
      name: name.trim(),
      dataType: normalizeDataType(dataType),
      rwFlag,
      required,
      description: description.trim() || undefined,
      sort,
    }
    const nextIssues = validatePropertyInput(payload)
    if (nextIssues.length > 0) {
      setIssues(nextIssues)
      return
    }
    try {
      if (property) {
        await updateMutation.mutateAsync({
          id: modelId,
          identifier: property.identifier,
          property: payload,
        })
        toast.success(t('propertyDialog.updatedToast'))
      } else {
        await addMutation.mutateAsync({ id: modelId, properties: [payload] })
        toast.success(t('propertyDialog.addedToast'))
      }
      onClose()
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      size="xl"
      title={
        isEdit ? t('propertyDialog.editTitle') : t('propertyDialog.addTitle')
      }
      description={
        isEdit
          ? t('propertyDialog.editDescription')
          : t('propertyDialog.addDescription')
      }
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={pending}>
            {t('common.action.cancel')}
          </Button>
          <Button variant="primary" onClick={submit} loading={pending}>
            {isEdit ? t('common.action.save') : t('common.action.add')}
          </Button>
        </>
      }
    >
      <div className="max-h-[68vh] space-y-4 overflow-y-auto pr-1">
        <div className="grid gap-3 sm:grid-cols-2">
          <Field
            label={t('propertyDialog.identifier')}
            htmlFor="property-identifier"
            required
            error={fieldError('identifier')}
            hint={
              isEdit
                ? t('propertyDialog.identifierHintEdit')
                : t('propertyDialog.identifierHintAdd')
            }
          >
            <Input
              id="property-identifier"
              value={identifier}
              disabled={isEdit}
              placeholder={t('propertyDialog.identifierPlaceholder')}
              onChange={(event) => setIdentifier(event.target.value)}
            />
          </Field>
          <Field
            label={t('propertyDialog.name')}
            htmlFor="property-name"
            required
            error={fieldError('name')}
            hint={t('propertyDialog.nameHint')}
          >
            <Input
              id="property-name"
              value={name}
              placeholder={t('propertyDialog.namePlaceholder')}
              onChange={(event) => setName(event.target.value)}
            />
          </Field>
          <Field label={t('propertyDialog.rwFlag')} error={fieldError('rwFlag')}>
            <Select value={rwFlag} onChange={(event) => setRwFlag(event.target.value as RwFlag)}>
              {RW_FLAGS.map((item) => (
                <option key={item.value} value={item.value}>
                  {t(item.labelKey)} · {t(item.descriptionKey)}
                </option>
              ))}
            </Select>
          </Field>
          <Field label={t('propertyDialog.sort')} error={fieldError('sort')}>
            <Input
              type="number"
              min={0}
              value={sort}
              onChange={(event) => setSort(Number(event.target.value) || 0)}
            />
          </Field>
        </div>

        <div className="flex items-center justify-between rounded-xl border border-border bg-surface-2/50 px-3 py-2.5">
          <div>
            <p className="text-xs font-medium">{t('propertyDialog.required')}</p>
            <p className="text-[11px] text-faint">{t('propertyDialog.requiredHint')}</p>
          </div>
          <Switch checked={required} onChange={setRequired} label={t('propertyDialog.required')} />
        </div>

        <Field label={t('propertyDialog.description')} error={fieldError('description')}>
          <Textarea
            value={description}
            placeholder={t('dataType.optional')}
            onChange={(event) => setDescription(event.target.value)}
          />
        </Field>

        <div>
          <p className="mb-2 text-xs font-medium text-muted">{t('propertyDialog.dataType')}</p>
          <DataTypeEditor value={dataType} onChange={setDataType} />
          {dataTypeIssues.length > 0 ? (
            <ul className="mt-2 space-y-1 rounded-lg border border-danger/25 bg-danger/5 px-3 py-2 text-xs text-danger">
              {dataTypeIssues.map((issue) => (
                <li key={`${issue.field}-${issue.message}`}>
                  <span className="font-mono text-[10px] opacity-70">{issue.field}</span>{' '}
                  {issue.message}
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      </div>
    </Dialog>
  )
}
