import { useEffect, useMemo } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import { useCreateRole, useUpdateRole } from '@/api/hooks'
import type { RoleInfo } from '@/api/types'
import { useI18n, type Translate } from '@harns/core'
import { errorMessage } from '@harns/core'
import { Button } from '@harns/core'
import { Dialog } from '@harns/core'
import { Field, Input, Select, Textarea } from '@harns/core'

type FormValues = {
  name: string
  code: string
  description: string
  status: string
}

function createRoleFormSchema(t: Translate): z.ZodType<FormValues> {
  return z.object({
    name: z
      .string()
      .trim()
      .min(1, t('identity.roleForm.validation.nameRequired'))
      .max(64, t('identity.roleForm.validation.nameMax')),
    code: z
      .string()
      .trim()
      .min(1, t('identity.roleForm.validation.codeRequired'))
      .max(64, t('identity.roleForm.validation.codeMax')),
    description: z
      .string()
      .trim()
      .max(255, t('identity.roleForm.validation.descriptionMax')),
    status: z.string(),
  })
}

const EMPTY: FormValues = { name: '', code: '', description: '', status: 'enabled' }

export function RoleFormDialog({
  open,
  onClose,
  role,
}: {
  open: boolean
  onClose: () => void
  role?: RoleInfo
}) {
  const createMutation = useCreateRole()
  const updateMutation = useUpdateRole()
  const { t } = useI18n()
  const schema = useMemo(() => createRoleFormSchema(t), [t])
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: EMPTY })

  useEffect(() => {
    if (!open) return
    reset(
      role
        ? {
            name: role.name,
            code: role.code,
            description: role.description ?? '',
            status: role.status || 'enabled',
          }
        : EMPTY,
    )
  }, [open, role, reset])

  const pending = createMutation.isPending || updateMutation.isPending

  const onSubmit = handleSubmit(async (values) => {
    const payload = {
      name: values.name,
      code: values.code,
      description: values.description || undefined,
      status: values.status as 'enabled' | 'disabled',
    }
    try {
      if (role) {
        await updateMutation.mutateAsync({ id: role.id, payload })
        toast.success(t('identity.roleForm.updatedToast'))
      } else {
        await createMutation.mutateAsync(payload)
        toast.success(t('identity.roleForm.createdToast'))
      }
      onClose()
    } catch (error) {
      toast.error(errorMessage(error))
    }
  })

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={role ? t('identity.roleForm.editTitle') : t('identity.roleForm.createTitle')}
      description={t('identity.roleForm.description')}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={pending}>
            {t('common.action.cancel')}
          </Button>
          <Button variant="primary" onClick={onSubmit} loading={pending}>
            {role ? t('common.action.save') : t('common.action.create')}
          </Button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={onSubmit} noValidate>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field
            label={t('identity.roleForm.name')}
            htmlFor="role-name"
            required
            error={errors.name?.message}
          >
            <Input id="role-name" {...register('name')} />
          </Field>
          <Field
            label={t('identity.roleForm.code')}
            htmlFor="role-code"
            required
            error={errors.code?.message}
          >
            <Input id="role-code" className="font-mono" {...register('code')} />
          </Field>
          <Field label={t('common.field.status')} htmlFor="role-status">
            <Select id="role-status" {...register('status')}>
              <option value="enabled">{t('common.field.enabled')}</option>
              <option value="disabled">{t('common.field.disabled')}</option>
            </Select>
          </Field>
        </div>
        <Field
          label={t('identity.roleForm.descriptionField')}
          htmlFor="role-desc"
          error={errors.description?.message}
        >
          <Textarea id="role-desc" {...register('description')} />
        </Field>
      </form>
    </Dialog>
  )
}
