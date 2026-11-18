import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  FormDialog,
  Field,
  Input,
  Textarea,
  showErrorToast,
  useI18n,
} from '@harns/core'
import { useCreateDriver } from '@/api/hooks'
import { EMPTY_DRIVER_SCHEMA_TEXT, parseDriverSchema } from './driver-utils'

interface DriverForm {
  driverType: string
  model: string
  pluginName: string
  schemaText: string
}

const EMPTY_DRIVER: DriverForm = {
  driverType: '',
  model: '',
  pluginName: '',
  schemaText: EMPTY_DRIVER_SCHEMA_TEXT,
}

export function DriverFormDialog({
  open,
  onClose,
}: {
  open: boolean
  onClose: () => void
}) {
  const { t } = useI18n()
  const create = useCreateDriver()
  const [form, setForm] = useState<DriverForm>(EMPTY_DRIVER)
  const [errors, setErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!open) return
    setForm(EMPTY_DRIVER)
    setErrors({})
  }, [open])

  const submit = async () => {
    const next: Record<string, string> = {}
    if (!form.driverType.trim()) next.driverType = t('gateway.drivers.validation.identityRequired')
    if (!form.model.trim()) next.model = t('gateway.drivers.validation.identityRequired')
    if (!form.pluginName.trim()) next.pluginName = t('gateway.drivers.validation.pluginNameRequired')
    const schema = parseDriverSchema(form.schemaText)
    if (!schema) next.schema = t('gateway.drivers.validation.schemaInvalid')
    setErrors(next)
    if (Object.keys(next).length > 0 || !schema) return
    try {
      await create.mutateAsync({
        driverType: form.driverType.trim(),
        model: form.model.trim(),
        pluginName: form.pluginName.trim(),
        schema,
      })
      toast.success(t('gateway.drivers.toast.created'))
      onClose()
    } catch (error) {
      showErrorToast(error)
    }
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={t('gateway.drivers.createTitle')}
      footer={
        <>
          <Button variant="outline" onClick={onClose}>
            {t('common.action.cancel')}
          </Button>
          <Button variant="primary" loading={create.isPending} onClick={() => void submit()}>
            {t('gateway.drivers.create')}
          </Button>
        </>
      }
    >
      <Card>
        <CardHeader>
          <CardTitle>{t('gateway.form.section.basic')}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-x-8 gap-y-4 sm:grid-cols-2">
          <Field
            label={t('gateway.drivers.field.driverType')}
            htmlFor="driver-create-type"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
            required
            error={errors.driverType}
          >
            <Input
              id="driver-create-type"
              value={form.driverType}
              placeholder="modbus"
              autoComplete="off"
              onChange={(event) => setForm({ ...form, driverType: event.target.value })}
            />
          </Field>
          <Field
            label={t('gateway.drivers.field.model')}
            htmlFor="driver-create-model"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
            required
            error={errors.model}
          >
            <Input
              id="driver-create-model"
              value={form.model}
              placeholder="myModbusTcp"
              autoComplete="off"
              onChange={(event) => setForm({ ...form, model: event.target.value })}
            />
          </Field>
          <Field
            label={t('gateway.drivers.field.pluginName')}
            htmlFor="driver-create-plugin"
            labelClassName="text-sm"
            labelWidthClassName="w-[96px]"
            inline
            required
            error={errors.pluginName}
            className="sm:col-span-2"
          >
            <Input
              id="driver-create-plugin"
              value={form.pluginName}
              placeholder="my-driver-plugin"
              autoComplete="off"
              onChange={(event) => setForm({ ...form, pluginName: event.target.value })}
            />
          </Field>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('gateway.drivers.field.schema')}</CardTitle>
        </CardHeader>
        <CardContent>
          <Textarea
            value={form.schemaText}
            spellCheck={false}
            className="min-h-80 font-mono text-xs"
            aria-invalid={Boolean(errors.schema)}
            onChange={(event) => setForm({ ...form, schemaText: event.target.value })}
          />
          <p className="mt-1 min-h-4 text-xs leading-4 text-danger">{errors.schema}</p>
        </CardContent>
      </Card>
    </FormDialog>
  )
}
