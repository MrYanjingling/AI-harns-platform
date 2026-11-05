import { useEffect } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import { useUpdateThing } from '@/api/hooks'
import type { ThingDetail } from '@/api/types'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Textarea } from '@/components/ui/Field'

const schema = z.object({
  deviceName: z.string().trim().min(1, '请输入设备名称').max(64, '名称不能超过 64 个字符'),
  deviceDesc: z.string().trim().max(255, '描述不能超过 255 个字符'),
  mqttClientId: z.string().trim().max(64, '不能超过 64 个字符'),
  mqttUsername: z.string().trim().max(64, '不能超过 64 个字符'),
  mqttPassword: z.string().max(64, '不能超过 64 个字符'),
})

type FormValues = z.infer<typeof schema>

export function EditThingDialog({
  open,
  onClose,
  thing,
}: {
  open: boolean
  onClose: () => void
  thing: ThingDetail
}) {
  const mutation = useUpdateThing()
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
      toast.success('设备已更新')
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
      title="编辑设备"
      description="Device Key、Device Secret、Token 与 Topic 创建后不可修改。"
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
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="设备名称" htmlFor="thing-name" required error={errors.deviceName?.message}>
            <Input id="thing-name" {...register('deviceName')} />
          </Field>
          <Field
            label="Device Key"
            hint="不可修改"
          >
            <Input value={thing.deviceKey} readOnly disabled className="font-mono" />
          </Field>
          <Field label="MQTT Client ID" htmlFor="thing-client" error={errors.mqttClientId?.message}>
            <Input id="thing-client" className="font-mono" {...register('mqttClientId')} />
          </Field>
          <Field label="MQTT Username" htmlFor="thing-username" error={errors.mqttUsername?.message}>
            <Input id="thing-username" className="font-mono" {...register('mqttUsername')} />
          </Field>
          <Field label="MQTT Password" htmlFor="thing-password" error={errors.mqttPassword?.message}>
            <Input
              id="thing-password"
              type="text"
              className="font-mono"
              autoComplete="off"
              {...register('mqttPassword')}
            />
          </Field>
        </div>
        <Field label="描述" htmlFor="thing-desc" error={errors.deviceDesc?.message}>
          <Textarea id="thing-desc" {...register('deviceDesc')} />
        </Field>
      </form>
    </Dialog>
  )
}
