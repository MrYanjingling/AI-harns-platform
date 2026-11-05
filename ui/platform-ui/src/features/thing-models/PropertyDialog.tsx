import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { useAddThingModelProperties, useUpdateThingModelProperty } from '@/api/hooks'
import type { RwFlag, ThingDataTypeDetail, ThingModelProperty } from '@/api/types'
import { RW_FLAGS } from '@/lib/constants'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Switch } from '@/components/ui/Switch'
import { DataTypeEditor } from './DataTypeEditor'
import { emptyDataType, normalizeDataType, validatePropertyInput, type FieldIssue } from './data-type'

const DEFAULT_SORT = 0

export function PropertyDialog({
  open,
  onClose,
  modelId,
  property,
}: {
  open: boolean
  onClose: () => void
  modelId: string
  property?: ThingModelProperty
}) {
  const isEdit = Boolean(property)
  const addMutation = useAddThingModelProperties()
  const updateMutation = useUpdateThingModelProperty()

  const [identifier, setIdentifier] = useState('')
  const [name, setName] = useState('')
  const [rwFlag, setRwFlag] = useState<RwFlag>('r')
  const [required, setRequired] = useState(false)
  const [description, setDescription] = useState('')
  const [sort, setSort] = useState(DEFAULT_SORT)
  const [dataType, setDataType] = useState<ThingDataTypeDetail>(() => emptyDataType('double'))
  const [issues, setIssues] = useState<FieldIssue[]>([])

  useEffect(() => {
    if (!open) return
    setIssues([])
    if (property) {
      setIdentifier(property.identifier)
      setName(property.name)
      setRwFlag(property.rwFlag ?? 'r')
      setRequired(Boolean(property.required))
      setDescription(property.description ?? '')
      setSort(property.sort ?? DEFAULT_SORT)
      setDataType(property.dataType)
    } else {
      setIdentifier('')
      setName('')
      setRwFlag('r')
      setRequired(false)
      setDescription('')
      setSort(DEFAULT_SORT)
      setDataType(emptyDataType('double'))
    }
  }, [open, property])

  const pending = addMutation.isPending || updateMutation.isPending
  const fieldError = (key: string) => issues.find((issue) => issue.field === key)?.message
  const dataTypeIssues = issues.filter((issue) => issue.field.startsWith('dataType'))

  const submit = async () => {
    const payload = {
      identifier: identifier.trim(),
      name: name.trim(),
      dataType: normalizeDataType(dataType),
      rwFlag,
      required,
      description: description.trim() || undefined,
      sort,
    }
    const nextIssues = validatePropertyInput(payload)
    if (nextIssues.length > 0) {
      setIssues(nextIssues)
      return
    }
    try {
      if (property) {
        await updateMutation.mutateAsync({
          id: modelId,
          identifier: property.identifier,
          property: payload,
        })
        toast.success('属性已更新')
      } else {
        await addMutation.mutateAsync({ id: modelId, properties: [payload] })
        toast.success('属性已添加')
      }
      onClose()
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      size="xl"
      title={isEdit ? '编辑属性' : '添加属性'}
      description={
        isEdit
          ? '标识符创建后不可修改;属性更新会整体替换数据类型定义。'
          : '属性标识符在物模型内唯一,创建后不可修改。'
      }
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={pending}>
            取消
          </Button>
          <Button variant="primary" onClick={submit} loading={pending}>
            {isEdit ? '保存' : '添加'}
          </Button>
        </>
      }
    >
      <div className="max-h-[68vh] space-y-4 overflow-y-auto pr-1">
        <div className="grid gap-3 sm:grid-cols-2">
          <Field
            label="标识符"
            htmlFor="property-identifier"
            required
            error={fieldError('identifier')}
            hint={isEdit ? '标识符不可修改' : '字母开头,可含数字与下划线'}
          >
            <Input
              id="property-identifier"
              value={identifier}
              disabled={isEdit}
              placeholder="例如 temperature"
              onChange={(event) => setIdentifier(event.target.value)}
            />
          </Field>
          <Field
            label="名称"
            htmlFor="property-name"
            required
            error={fieldError('name')}
            hint="字母开头,模型内唯一"
          >
            <Input
              id="property-name"
              value={name}
              placeholder="例如 温度"
              onChange={(event) => setName(event.target.value)}
            />
          </Field>
          <Field label="读写标记" error={fieldError('rwFlag')}>
            <Select value={rwFlag} onChange={(event) => setRwFlag(event.target.value as RwFlag)}>
              {RW_FLAGS.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label} · {item.description}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="排序" error={fieldError('sort')}>
            <Input
              type="number"
              min={0}
              value={sort}
              onChange={(event) => setSort(Number(event.target.value) || 0)}
            />
          </Field>
        </div>

        <div className="flex items-center justify-between rounded-xl border border-border bg-surface-2/50 px-3 py-2.5">
          <div>
            <p className="text-xs font-medium">必填属性</p>
            <p className="text-[11px] text-faint">设备上报时必须携带该属性</p>
          </div>
          <Switch checked={required} onChange={setRequired} label="必填属性" />
        </div>

        <Field label="描述" error={fieldError('description')}>
          <Textarea
            value={description}
            placeholder="可选"
            onChange={(event) => setDescription(event.target.value)}
          />
        </Field>

        <div>
          <p className="mb-2 text-xs font-medium text-muted">数据类型定义</p>
          <DataTypeEditor value={dataType} onChange={setDataType} />
          {dataTypeIssues.length > 0 ? (
            <ul className="mt-2 space-y-1 rounded-lg border border-danger/25 bg-danger/5 px-3 py-2 text-xs text-danger">
              {dataTypeIssues.map((issue) => (
                <li key={`${issue.field}-${issue.message}`}>
                  <span className="font-mono text-[10px] opacity-70">{issue.field}</span>{' '}
                  {issue.message}
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      </div>
    </Dialog>
  )
}
