import { useEffect, useMemo } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import { useUpdateThing } from '@/api/hooks'
import type { ThingDetail } from '@/api/types'
import { useI18n, type Translate } from '@/i18n'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Textarea } from '@/components/ui/Field'

type FormValues = {
  deviceName: string
  deviceDesc: string
  mqttClientId: string
  mqttUsername: string
  mqttPassword: string
}

function createEditThingSchema(t: Translate): z.ZodType<FormValues> {
  return z.object({
    deviceName: z
      .string()
      .trim()
      .min(1, t('editThing.validation.nameRequired'))
      .max(64, t('editThing.validation.nameMax')),
    deviceDesc: z
      .string()
      .trim()
      .max(255, t('editThing.validation.descMax')),
    mqttClientId: z
      .string()
      .trim()
      .max(64, t('editThing.validation.fieldMax')),
    mqttUsername: z
      .string()
      .trim()
      .max(64, t('editThing.validation.fieldMax')),
    mqttPassword: z.string().max(64, t('editThing.validation.fieldMax')),
  })
}

export function EditThingDialog({
  open,
  onClose,
  thing,
}: {
  open: boolean
  onClose: () => void
  thing: ThingDetail
}) {
  const { t } = useI18n()
  const mutation = useUpdateThing()
  const schema = useMemo(() => createEditThingSchema(t), [t])
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      deviceName: thing.deviceName,
      deviceDesc: thing.deviceDesc,
      mqttClientId: thing.mqttClientId,
      mqttUsername: thing.mqttUsername,
      mqttPassword: thing.mqttPassword,
    },
  })

  useEffect(() => {
    if (!open) return
    reset({
      deviceName: thing.deviceName,
      deviceDesc: thing.deviceDesc,
      mqttClientId: thing.mqttClientId,
      mqttUsername: thing.mqttUsername,
      mqttPassword: thing.mqttPassword,
    })
  }, [open, thing, reset])

  const onSubmit = handleSubmit(async (values) => {
    try {
      await mutation.mutateAsync({
        id: thing.id,
        payload: {
          deviceName: values.deviceName,
          deviceDesc: values.deviceDesc,
          ...(values.mqttClientId ? { mqttClientId: values.mqttClientId } : {}),
          ...(values.mqttUsername ? { mqttUsername: values.mqttUsername } : {}),
          ...(values.mqttPassword ? { mqttPassword: values.mqttPassword } : {}),
        },
      })
      toast.success(t('editThing.updatedToast'))
      onClose()
    } catch (error) {
      toast.error(errorMessage(error))
    }
  })

  return (
    <Dialog
      open={open}
      onClose={onClose}
      size="lg"
      title={t('editThing.title')}
      description={t('editThing.description')}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            {t('common.action.cancel')}
          </Button>
          <Button variant="primary" onClick={onSubmit} loading={mutation.isPending}>
            {t('common.action.save')}
          </Button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={onSubmit} noValidate>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field
            label={t('editThing.deviceName')}
            htmlFor="thing-name"
            required
            error={errors.deviceName?.message}
          >
            <Input id="thing-name" {...register('deviceName')} />
          </Field>
          <Field label={t('editThing.deviceKey')} hint={t('editThing.immutable')}>
            <Input value={thing.deviceKey} readOnly disabled className="font-mono" />
          </Field>
          <Field
            label="MQTT Client ID"
            htmlFor="thing-client"
            error={errors.mqttClientId?.message}
          >
            <Input id="thing-client" className="font-mono" {...register('mqttClientId')} />
          </Field>
          <Field
            label="MQTT Username"
            htmlFor="thing-username"
            error={errors.mqttUsername?.message}
          >
            <Input id="thing-username" className="font-mono" {...register('mqttUsername')} />
          </Field>
          <Field
            label="MQTT Password"
            htmlFor="thing-password"
            error={errors.mqttPassword?.message}
          >
            <Input
              id="thing-password"
              type="text"
              className="font-mono"
              autoComplete="off"
              {...register('mqttPassword')}
            />
          </Field>
        </div>
        <Field
          label={t('editThing.descriptionField')}
          htmlFor="thing-desc"
          error={errors.deviceDesc?.message}
        >
          <Textarea id="thing-desc" {...register('deviceDesc')} />
        </Field>
      </form>
    </Dialog>
  )
}
