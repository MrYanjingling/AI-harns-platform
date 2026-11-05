import { useEffect } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import { useCreateRole, useUpdateRole } from '@/api/hooks'
import type { RoleInfo } from '@/api/types'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'

const schema = z.object({
  name: z.string().trim().min(1, '请输入角色名称').max(64, '名称不能超过 64 个字符'),
  code: z.string().trim().min(1, '请输入角色编码').max(64, '编码不能超过 64 个字符'),
  description: z.string().trim().max(255, '描述不能超过 255 个字符'),
  status: z.string(),
})

type FormValues = z.infer<typeof schema>

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
        toast.success('角色已更新')
      } else {
        await createMutation.mutateAsync(payload)
        toast.success('角色已创建')
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
      title={role ? '编辑角色' : '新建角色'}
      description="角色编码用于标识权限集合,建议保持唯一且稳定。"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={pending}>
            取消
          </Button>
          <Button variant="primary" onClick={onSubmit} loading={pending}>
            {role ? '保存' : '创建'}
          </Button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={onSubmit} noValidate>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="角色名称" htmlFor="role-name" required error={errors.name?.message}>
            <Input id="role-name" {...register('name')} />
          </Field>
          <Field label="角色编码" htmlFor="role-code" required error={errors.code?.message}>
            <Input id="role-code" className="font-mono" {...register('code')} />
          </Field>
          <Field label="状态" htmlFor="role-status">
            <Select id="role-status" {...register('status')}>
              <option value="enabled">启用</option>
              <option value="disabled">禁用</option>
            </Select>
          </Field>
        </div>
        <Field label="描述" htmlFor="role-desc" error={errors.description?.message}>
          <Textarea id="role-desc" {...register('description')} />
        </Field>
      </form>
    </Dialog>
  )
}
