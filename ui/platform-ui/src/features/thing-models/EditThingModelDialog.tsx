import { useEffect } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import { useUpdateThingModel } from '@/api/hooks'
import type { ThingModelInfo } from '@/api/types'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Textarea } from '@/components/ui/Field'

const schema = z.object({
  name: z.string().trim().min(1, '请输入物模型名称').max(64, '名称不能超过 64 个字符'),
  description: z.string().trim().max(255, '描述不能超过 255 个字符'),
})

type FormValues = z.infer<typeof schema>

export function EditThingModelDialog({
  open,
  onClose,
  model,
}: {
  open: boolean
  onClose: () => void
  model: ThingModelInfo
}) {
  const mutation = useUpdateThingModel()
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
      toast.success('物模型已更新')
      onClose()
    } catch (error) {
      toast.error(errorMessage(error))
    }
  })

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="编辑物模型"
      description={`Key ${model.thingModelKey} 不可修改。`}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={mutation.isPending}>
            取消
          </Button>
          <Button variant="primary" onClick={onSubmit} loading={mutation.isPending}>
            保存
          </Button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={onSubmit} noValidate>
        <Field label="名称" htmlFor="model-name" required error={errors.name?.message}>
          <Input id="model-name" {...register('name')} />
        </Field>
        <Field label="描述" htmlFor="model-description" error={errors.description?.message}>
          <Textarea id="model-description" {...register('description')} />
        </Field>
      </form>
    </Dialog>
  )
}
