import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router-dom'
import { useMemo } from 'react'
import { toast } from 'sonner'
import { Boxes, Info } from 'lucide-react'
import { z } from 'zod'
import { useCreateThingModel } from '@/api/hooks'
import { useI18n, type Translate } from '@/i18n'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Field, Input, Textarea } from '@/components/ui/Field'
import { PageHeader } from '@/components/shared/PageHeader'

const KEY_PATTERN = /^[a-zA-Z][a-zA-Z0-9_-]{3,63}$/

type FormValues = { name: string; thingModelKey: string; description: string }

function createThingModelSchema(t: Translate): z.ZodType<FormValues> {
  return z.object({
    name: z
      .string()
      .trim()
      .min(1, t('thingModelCreate.validation.nameRequired'))
      .max(64, t('thingModelCreate.validation.nameMax')),
    thingModelKey: z
      .string()
      .trim()
      .max(64, t('thingModelCreate.validation.keyMax'))
      .refine(
        (value) => value === '' || KEY_PATTERN.test(value),
        t('thingModelCreate.validation.keyPattern'),
      ),
    description: z
      .string()
      .trim()
      .max(255, t('thingModelCreate.validation.descriptionMax')),
  })
}

export function ThingModelCreatePage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const mutation = useCreateThingModel()
  const schema = useMemo(() => createThingModelSchema(t), [t])
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: '', thingModelKey: '', description: '' },
  })

  const onSubmit = handleSubmit(async (values) => {
    try {
      const created = await mutation.mutateAsync({
        name: values.name,
        thingModelKey: values.thingModelKey || undefined,
        description: values.description || undefined,
      })
      toast.success(t('thingModelCreate.createdToast'))
      navigate(`/platform/thing-models/${created.id}`, { replace: true })
    } catch (error) {
      toast.error(errorMessage(error))
    }
  })

  return (
    <div>
      <PageHeader
        title={t('thingModelCreate.title')}
        description={t('thingModelCreate.description')}
        backTo="/platform/thing-models"
        backLabel={t('thingModelCreate.back')}
      />

      <div className="grid gap-4 lg:grid-cols-[1.5fr_1fr]">
        <Card>
          <CardHeader>
            <div>
              <CardTitle>{t('thingModelCreate.basic')}</CardTitle>
              <CardDescription>{t('thingModelCreate.basicDescription')}</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={onSubmit} noValidate>
              <Field
                label={t('thingModelCreate.name')}
                htmlFor="name"
                required
                error={errors.name?.message}
              >
                <Input
                  id="name"
                  placeholder={t('thingModelCreate.namePlaceholder')}
                  {...register('name')}
                />
              </Field>

              <Field
                label={t('thingModelCreate.key')}
                htmlFor="thingModelKey"
                error={errors.thingModelKey?.message}
                hint={t('thingModelCreate.keyHint')}
              >
                <Input
                  id="thingModelKey"
                  placeholder={t('thingModelCreate.keyPlaceholder')}
                  className="font-mono"
                  {...register('thingModelKey')}
                />
              </Field>

              <Field
                label={t('thingModelCreate.descriptionField')}
                htmlFor="description"
                error={errors.description?.message}
              >
                <Textarea
                  id="description"
                  placeholder={t('thingModelCreate.descriptionPlaceholder')}
                  {...register('description')}
                />
              </Field>

              <div className="flex justify-end gap-2 pt-2">
                <Button variant="ghost" onClick={() => navigate('/platform/thing-models')}>
                  {t('common.action.cancel')}
                </Button>
                <Button type="submit" variant="primary" loading={mutation.isPending}>
                  {t('thingModelCreate.createAndContinue')}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card className="h-fit">
          <CardHeader>
            <div>
              <CardTitle>{t('thingModelCreate.afterTitle')}</CardTitle>
              <CardDescription>{t('thingModelCreate.afterDescription')}</CardDescription>
            </div>
            <Boxes className="size-4 text-faint" />
          </CardHeader>
          <CardContent className="space-y-3 text-xs leading-relaxed text-muted">
            <p className="flex gap-2">
              <Info className="mt-0.5 size-3.5 shrink-0 text-primary" />
              {t('thingModelCreate.after1')}
            </p>
            <p className="flex gap-2">
              <Info className="mt-0.5 size-3.5 shrink-0 text-primary" />
              {t('thingModelCreate.after2')}
            </p>
            <p className="flex gap-2">
              <Info className="mt-0.5 size-3.5 shrink-0 text-primary" />
              {t('thingModelCreate.after3')}
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
