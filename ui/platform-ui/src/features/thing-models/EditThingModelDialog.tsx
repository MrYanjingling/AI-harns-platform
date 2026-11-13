import { useEffect, useMemo } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import { useUpdateThingModel } from '@/api/hooks'
import type { ThingModelInfo } from '@/api/types'
import { useI18n, type Translate } from '@harns/core'
import { errorMessage } from '@harns/core'
import { Button } from '@harns/core'
import { Dialog } from '@harns/core'
import { Field, Input, Textarea } from '@harns/core'

type FormValues = { name: string; description: string }

function createEditThingModelSchema(t: Translate): z.ZodType<FormValues> {
  return z.object({
    name: z
      .string()
      .trim()
      .min(1, t('thingModelCreate.validation.nameRequired'))
      .max(64, t('thingModelCreate.validation.nameMax')),
    description: z
      .string()
      .trim()
      .max(255, t('thingModelCreate.validation.descriptionMax')),
  })
}

export function EditThingModelDialog({
  open,
  onClose,
  model,
}: {
  open: boolean
  onClose: () => void
  model: ThingModelInfo
}) {
  const { t } = useI18n()
  const mutation = useUpdateThingModel()
  const schema = useMemo(() => createEditThingModelSchema(t), [t])
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: model.name, description: model.description },
  })

  useEffect(() => {
    if (open) reset({ name: model.name, description: model.description })
  }, [open, model.name, model.description, reset])

  const onSubmit = handleSubmit(async (values) => {
    try {
      await mutation.mutateAsync({ id: model.id, payload: values })
      toast.success(t('editThingModel.updatedToast'))
      onClose()
    } catch (error) {
      toast.error(errorMessage(error))
    }
  })

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('editThingModel.title')}
      description={t('editThingModel.description', { key: model.thingModelKey })}
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
        <Field
          label={t('editThingModel.name')}
          htmlFor="model-name"
          required
          error={errors.name?.message}
        >
          <Input id="model-name" {...register('name')} />
        </Field>
        <Field
          label={t('editThingModel.descriptionField')}
          htmlFor="model-description"
          error={errors.description?.message}
        >
          <Textarea id="model-description" {...register('description')} />
        </Field>
      </form>
    </Dialog>
  )
}
