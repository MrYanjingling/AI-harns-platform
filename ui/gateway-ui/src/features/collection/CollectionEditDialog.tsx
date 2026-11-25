import { useEffect, useState } from 'react'
import { CircleHelp } from 'lucide-react'
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
  showErrorToast,
  useI18n,
} from '@harns/core'
import { useCollection, useDriverSchema, useUpdateCollection } from '@/api/hooks'
import { ProtocolFields } from '@/components/ProtocolFields'
import {
  detailToForm,
  validateCollectionForm,
  type CollectionFormValues,
} from '@/lib/collection-form'
import { driverModelLabel, driverTypeLabel } from '@/lib/constants'

interface CollectionEditDialogProps {
  collectionId: string | null
  onClose: () => void
}

export function CollectionEditDialog({ collectionId, onClose }: CollectionEditDialogProps) {
  const { t } = useI18n()
  const detail = useCollection(collectionId ?? undefined)
  const schema = useDriverSchema(detail.data?.driverModel)
  const updateCollection = useUpdateCollection()
  const [form, setForm] = useState<CollectionFormValues | null>(null)
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!collectionId) return
    setForm(null)
    setErrors({})
  }, [collectionId])

  useEffect(() => {
    if (!collectionId || form || !detail.data || !schema.data) return
    setForm(detailToForm(detail.data, schema.data.schema))
  }, [collectionId, form, detail.data, schema.data])

  const driverSchema = schema.data?.schema
  const submitting = updateCollection.isPending

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

  const save = async () => {
    if (!collectionId || !form || !schema.data) return
    const result = validateCollectionForm(form, schema.data.schema, t, false)
    setErrors(result.errors)
    if (!result.payload) return
    try {
      await updateCollection.mutateAsync({
        id: collectionId,
        name: result.payload.name,
        payload: result.payload,
      })
      toast.success(t('gateway.form.toast.updated'))
      onClose()
    } catch (error) {
      showErrorToast(error)
    }
  }

  return (
    <FormDialog
      open={Boolean(collectionId)}
      onClose={onClose}
      title={t('gateway.collection.updateTitle')}
      footer={
        <>
          <Button variant="outline" onClick={onClose}>
            {t('common.action.cancel')}
          </Button>
          <Button
            variant="primary"
            loading={submitting}
            disabled={!form || !schema.data}
            onClick={() => void save()}
          >
            {t('common.action.save')}
          </Button>
        </>
      }
    >
      {detail.isError ? (
        <ErrorState error={detail.error} onRetry={() => void detail.refetch()} />
      ) : schema.isError ? (
        <ErrorState error={schema.error} onRetry={() => void schema.refetch()} />
      ) : !form || !driverSchema ? (
        <div className="space-y-3">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-40 w-full" />
        </div>
      ) : (
        <>
          <Card>
            <CardHeader>
              <CardTitle>{t('gateway.form.section.basic')}</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-x-8 gap-y-4 sm:grid-cols-2">
              <Field
                label={t('gateway.form.field.name')}
                htmlFor="edit-collection-name"
                labelClassName="text-sm"
                labelWidthClassName="w-[84px]"
                inline
                required
                error={errors.name}
              >
                <Input
                  id="edit-collection-name"
                  value={form.name}
                  onChange={(event) => update({ name: event.target.value })}
                  placeholder={t('gateway.form.namePlaceholder')}
                  autoComplete="off"
                  className="w-full max-w-[200px]"
                />
              </Field>
              <Field
                label={t('gateway.form.field.cycle')}
                htmlFor="edit-collection-cycle"
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
                  id="edit-collection-cycle"
                  value={form.collectorCycleMs}
                  inputMode="numeric"
                  className="w-full max-w-[200px]"
                  onChange={(event) => update({ collectorCycleMs: event.target.value })}
                />
              </Field>
              <Field
                label={t('gateway.form.field.driverType')}
                htmlFor="edit-driver-type"
                labelClassName="text-sm"
                labelWidthClassName="w-[84px]"
                inline
              >
                <SearchSelect
                  id="edit-driver-type"
                  value={form.driverType}
                  options={[
                    {
                      value: form.driverType,
                      label: driverTypeLabel(form.driverType) || form.driverType,
                    },
                  ]}
                  onChange={() => undefined}
                  ariaLabel={t('gateway.form.field.driverType')}
                  emptyText={t('common.state.empty')}
                  searchable={false}
                  disabled
                  className="w-full max-w-[200px]"
                />
              </Field>
              <Field
                label={t('gateway.form.field.driverModel')}
                htmlFor="edit-driver-model"
                labelClassName="text-sm"
                labelWidthClassName="w-[84px]"
                inline
              >
                <SearchSelect
                  id="edit-driver-model"
                  value={form.driverModel}
                  options={[
                    {
                      value: form.driverModel,
                      label: driverModelLabel(form.driverModel) || form.driverModel,
                    },
                  ]}
                  onChange={() => undefined}
                  ariaLabel={t('gateway.form.field.driverModel')}
                  emptyText={t('common.state.empty')}
                  searchable={false}
                  disabled
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
      )}
    </FormDialog>
  )
}
