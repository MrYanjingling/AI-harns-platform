import { useEffect, useMemo } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import { useCreateOrg, useUpdateOrg } from '@/api/hooks'
import type { OrgInfo } from '@/api/types'
import { useI18n, type Translate } from '@harns/core'
import { errorMessage } from '@harns/core'
import { Button } from '@harns/core'
import { Dialog } from '@harns/core'
import { Field, Input, Select, Textarea } from '@harns/core'
import { flattenOrgs } from './org-utils'

type FormValues = {
  code: string
  name: string
  parentId: string
  sort: string
  description: string
  remark: string
  status: string
}

function createOrgFormSchema(t: Translate): z.ZodType<FormValues> {
  return z.object({
    code: z
      .string()
      .trim()
      .min(1, t('identity.orgForm.validation.codeRequired'))
      .max(64, t('identity.orgForm.validation.codeMax')),
    name: z
      .string()
      .trim()
      .min(1, t('identity.orgForm.validation.nameRequired'))
      .max(64, t('identity.orgForm.validation.nameMax')),
    parentId: z.string(),
    sort: z.string().regex(/^\d*$/, t('identity.orgForm.validation.order')),
    description: z
      .string()
      .trim()
      .max(255, t('identity.orgForm.validation.descriptionMax')),
    remark: z.string().trim().max(255, t('identity.orgForm.validation.remarkMax')),
    status: z.string(),
  })
}

function subtreeIds(node: OrgInfo): string[] {
  return [node.id, ...(node.children ?? []).flatMap(subtreeIds)]
}

function buildEmpty(parentId: string): FormValues {
  return {
    code: '',
    name: '',
    parentId,
    sort: '0',
    description: '',
    remark: '',
    status: 'enabled',
  }
}

export function OrgFormDialog({
  open,
  onClose,
  tree,
  org,
  defaultParentId = '',
}: {
  open: boolean
  onClose: () => void
  tree: OrgInfo[]
  org?: OrgInfo
  defaultParentId?: string
}) {
  const createMutation = useCreateOrg()
  const updateMutation = useUpdateOrg()
  const { t } = useI18n()
  const schema = useMemo(() => createOrgFormSchema(t), [t])
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: buildEmpty('') })

  useEffect(() => {
    if (!open) return
    reset(
      org
        ? {
            code: org.code,
            name: org.name,
            parentId: org.parentId ?? '',
            sort: String(org.sort ?? 0),
            description: org.description ?? '',
            remark: org.remark ?? '',
            status: org.status || 'enabled',
          }
        : buildEmpty(defaultParentId),
    )
  }, [open, org, defaultParentId, reset])

  const forbiddenParents = new Set(org ? subtreeIds(org) : [])
  const parentOptions = flattenOrgs(tree).filter((item) => !forbiddenParents.has(item.id))

  const pending = createMutation.isPending || updateMutation.isPending

  const onSubmit = handleSubmit(async (values) => {
    const payload = {
      code: values.code,
      name: values.name,
      parentId: values.parentId || undefined,
      sort: Number(values.sort || 0),
      description: values.description || undefined,
      remark: values.remark || undefined,
      status: values.status as 'enabled' | 'disabled',
    }
    try {
      if (org) {
        await updateMutation.mutateAsync({ id: org.id, payload })
        toast.success(t('identity.orgForm.updatedToast'))
      } else {
        await createMutation.mutateAsync(payload)
        toast.success(t('identity.orgForm.createdToast'))
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
      size="lg"
      title={org ? t('identity.orgForm.editTitle') : t('identity.orgForm.createTitle')}
      description={t('identity.orgForm.description')}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={pending}>
            {t('common.action.cancel')}
          </Button>
          <Button variant="primary" onClick={onSubmit} loading={pending}>
            {org ? t('common.action.save') : t('common.action.create')}
          </Button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={onSubmit} noValidate>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field
            label={t('identity.orgForm.name')}
            htmlFor="org-name"
            required
            error={errors.name?.message}
          >
            <Input id="org-name" {...register('name')} />
          </Field>
          <Field
            label={t('identity.orgForm.code')}
            htmlFor="org-code"
            required
            error={errors.code?.message}
          >
            <Input id="org-code" className="font-mono" {...register('code')} />
          </Field>
          <Field
            label={t('identity.orgForm.parent')}
            htmlFor="org-parent"
            hint={t('identity.orgForm.parentHint')}
          >
            <Select id="org-parent" {...register('parentId')}>
              <option value="">{t('identity.orgForm.root')}</option>
              {parentOptions.map((item) => (
                <option key={item.id} value={item.id}>
                  {'　'.repeat(item.depth)}
                  {item.name} ({item.code})
                </option>
              ))}
            </Select>
          </Field>
          <Field label={t('identity.orgForm.order')} htmlFor="org-sort" error={errors.sort?.message}>
            <Input id="org-sort" type="number" min={0} {...register('sort')} />
          </Field>
          <Field label={t('common.field.status')} htmlFor="org-status">
            <Select id="org-status" {...register('status')}>
              <option value="enabled">{t('common.field.enabled')}</option>
              <option value="disabled">{t('common.field.disabled')}</option>
            </Select>
          </Field>
        </div>
        <Field
          label={t('common.field.description')}
          htmlFor="org-desc"
          error={errors.description?.message}
        >
          <Textarea id="org-desc" {...register('description')} />
        </Field>
        <Field
          label={t('common.field.remark')}
          htmlFor="org-remark"
          error={errors.remark?.message}
        >
          <Textarea id="org-remark" {...register('remark')} />
        </Field>
      </form>
    </Dialog>
  )
}
