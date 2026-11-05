import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { Boxes, Info } from 'lucide-react'
import { z } from 'zod'
import { useCreateThingModel } from '@/api/hooks'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Field, Input, Textarea } from '@/components/ui/Field'
import { PageHeader } from '@/components/shared/PageHeader'

const KEY_PATTERN = /^[a-zA-Z][a-zA-Z0-9_-]{3,63}$/

const schema = z.object({
  name: z.string().trim().min(1, '请输入物模型名称').max(64, '名称不能超过 64 个字符'),
  thingModelKey: z
    .string()
    .trim()
    .max(64, 'Key 不能超过 64 个字符')
    .refine(
      (value) => value === '' || KEY_PATTERN.test(value),
      'Key 需以字母开头,仅含字母、数字、下划线和连字符,长度 4~64',
    ),
  description: z.string().trim().max(255, '描述不能超过 255 个字符'),
})

type FormValues = z.infer<typeof schema>

export function ThingModelCreatePage() {
  const navigate = useNavigate()
  const mutation = useCreateThingModel()
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
      toast.success('物模型已创建,继续添加属性')
      navigate(`/platform/thing-models/${created.id}`, { replace: true })
    } catch (error) {
      toast.error(errorMessage(error))
    }
  })

  return (
    <div>
      <PageHeader
        title="新建物模型"
        description="物模型是设备属性与特征的模板,创建后可继续添加属性、导入 TSL。"
        backTo="/platform/thing-models"
        backLabel="返回物模型列表"
      />

      <div className="grid gap-4 lg:grid-cols-[1.5fr_1fr]">
        <Card>
          <CardHeader>
            <div>
              <CardTitle>基本信息</CardTitle>
              <CardDescription>名称全局唯一,Key 创建后不可修改。</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={onSubmit} noValidate>
              <Field label="物模型名称" htmlFor="name" required error={errors.name?.message}>
                <Input id="name" placeholder="例如 温控器" {...register('name')} />
              </Field>

              <Field
                label="物模型 Key"
                htmlFor="thingModelKey"
                error={errors.thingModelKey?.message}
                hint="可选,留空由后端生成;创建后不可修改。"
              >
                <Input
                  id="thingModelKey"
                  placeholder="例如 tm_thermostat"
                  className="font-mono"
                  {...register('thingModelKey')}
                />
              </Field>

              <Field
                label="描述"
                htmlFor="description"
                error={errors.description?.message}
              >
                <Textarea
                  id="description"
                  placeholder="可选,描述物模型的用途"
                  {...register('description')}
                />
              </Field>

              <div className="flex justify-end gap-2 pt-2">
                <Button variant="ghost" onClick={() => navigate('/platform/thing-models')}>
                  取消
                </Button>
                <Button type="submit" variant="primary" loading={mutation.isPending}>
                  创建并继续
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card className="h-fit">
          <CardHeader>
            <div>
              <CardTitle>创建后可以做什么</CardTitle>
              <CardDescription>物模型的属性与特征在详情页维护。</CardDescription>
            </div>
            <Boxes className="size-4 text-faint" />
          </CardHeader>
          <CardContent className="space-y-3 text-xs leading-relaxed text-muted">
            <p className="flex gap-2">
              <Info className="mt-0.5 size-3.5 shrink-0 text-primary" />
              添加属性:标识符、数据类型、读写标记与单位;同一物模型内标识符唯一。
            </p>
            <p className="flex gap-2">
              <Info className="mt-0.5 size-3.5 shrink-0 text-primary" />
              特征定义只随物模型创建或 TSL 导入维护,设备可为其填写静态值。
            </p>
            <p className="flex gap-2">
              <Info className="mt-0.5 size-3.5 shrink-0 text-primary" />
              导入/导出 TSL 可在详情页完成,导入会整体替换属性列表。
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
