import { useEffect } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import { useCreateOrg, useUpdateOrg } from '@/api/hooks'
import type { OrgInfo } from '@/api/types'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { flattenOrgs } from './org-utils'

const schema = z.object({
  code: z.string().trim().min(1, '请输入组织编码').max(64, '编码不能超过 64 个字符'),
  name: z.string().trim().min(1, '请输入组织名称').max(64, '名称不能超过 64 个字符'),
  parentId: z.string(),
  sort: z.string().regex(/^\d*$/, '排序必须是非负整数'),
  description: z.string().trim().max(255, '描述不能超过 255 个字符'),
  remark: z.string().trim().max(255, '备注不能超过 255 个字符'),
  status: z.string(),
})

type FormValues = z.infer<typeof schema>

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
        toast.success('组织已更新')
      } else {
        await createMutation.mutateAsync(payload)
        toast.success('组织已创建')
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
      title={org ? '编辑组织' : '新建组织'}
      description="组织以树状结构维护,上级为空时作为根节点。"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={pending}>
            取消
          </Button>
          <Button variant="primary" onClick={onSubmit} loading={pending}>
            {org ? '保存' : '创建'}
          </Button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={onSubmit} noValidate>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="组织名称" htmlFor="org-name" required error={errors.name?.message}>
            <Input id="org-name" {...register('name')} />
          </Field>
          <Field label="组织编码" htmlFor="org-code" required error={errors.code?.message}>
            <Input id="org-code" className="font-mono" {...register('code')} />
          </Field>
          <Field label="上级组织" htmlFor="org-parent" hint="不选则作为根组织">
            <Select id="org-parent" {...register('parentId')}>
              <option value="">根组织</option>
              {parentOptions.map((item) => (
                <option key={item.id} value={item.id}>
                  {'　'.repeat(item.depth)}
                  {item.name} ({item.code})
                </option>
              ))}
            </Select>
          </Field>
          <Field label="排序" htmlFor="org-sort" error={errors.sort?.message}>
            <Input id="org-sort" type="number" min={0} {...register('sort')} />
          </Field>
          <Field label="状态" htmlFor="org-status">
            <Select id="org-status" {...register('status')}>
              <option value="enabled">启用</option>
              <option value="disabled">禁用</option>
            </Select>
          </Field>
        </div>
        <Field label="描述" htmlFor="org-desc" error={errors.description?.message}>
          <Textarea id="org-desc" {...register('description')} />
        </Field>
        <Field label="备注" htmlFor="org-remark" error={errors.remark?.message}>
          <Textarea id="org-remark" {...register('remark')} />
        </Field>
      </form>
    </Dialog>
  )
}
