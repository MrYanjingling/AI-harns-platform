import { Plus, Trash2 } from 'lucide-react'
import type { ThingDataType, ThingDataTypeDetail } from '@/api/types'
import { THING_DATA_TYPES } from '@/lib/constants'
import { Button } from '@/components/ui/Button'
import { Field, Input, Select } from '@/components/ui/Field'
import { cn } from '@/lib/cn'
import { emptyDataType, isNumericDataType } from './data-type'

const MAX_NESTING_DEPTH = 2

const SCALAR_TYPES = THING_DATA_TYPES.filter(
  (item) => item.value !== 'array' && item.value !== 'struct',
)

export function DataTypeEditor({
  value,
  onChange,
  depth = 0,
}: {
  value: ThingDataTypeDetail
  onChange: (next: ThingDataTypeDetail) => void
  depth?: number
}) {
  const allowNested = depth < MAX_NESTING_DEPTH
  const types = allowNested ? THING_DATA_TYPES : SCALAR_TYPES

  const changeType = (type: ThingDataType) => {
    if (type === value.type) return
    onChange(emptyDataType(type))
  }

  return (
    <div className={cn('space-y-3', depth > 0 && 'rounded-xl border border-border bg-surface-2/40 p-3')}>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={depth === 0 ? '数据类型' : '元素/字段类型'}>
          <Select
            value={value.type}
            onChange={(event) => changeType(event.target.value as ThingDataType)}
          >
            {types.map((item) => (
              <option key={item.value} value={item.value}>
                {item.label} · {item.description}
              </option>
            ))}
          </Select>
        </Field>

        {isNumericDataType(value.type) ? (
          <>
            <Field label="单位" hint="例如 ℃、V、kW">
              <Input
                value={value.unit ?? ''}
                placeholder="可选"
                onChange={(event) =>
                  onChange({ ...value, unit: event.target.value || undefined })
                }
              />
            </Field>
            <Field label="单位名称">
              <Input
                value={value.unitName ?? ''}
                placeholder="例如 摄氏度"
                onChange={(event) =>
                  onChange({ ...value, unitName: event.target.value || undefined })
                }
              />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label="最小值">
                <Input
                  value={value.min ?? ''}
                  placeholder="可选"
                  onChange={(event) =>
                    onChange({ ...value, min: event.target.value || undefined })
                  }
                />
              </Field>
              <Field label="最大值">
                <Input
                  value={value.max ?? ''}
                  placeholder="可选"
                  onChange={(event) =>
                    onChange({ ...value, max: event.target.value || undefined })
                  }
                />
              </Field>
            </div>
          </>
        ) : null}

        {value.type === 'bool' ? (
          <>
            <Field label="真值文案">
              <Input
                value={value.trueValue ?? ''}
                placeholder="例如 运行"
                onChange={(event) =>
                  onChange({ ...value, trueValue: event.target.value || undefined })
                }
              />
            </Field>
            <Field label="假值文案">
              <Input
                value={value.falseValue ?? ''}
                placeholder="例如 停止"
                onChange={(event) =>
                  onChange({ ...value, falseValue: event.target.value || undefined })
                }
              />
            </Field>
          </>
        ) : null}

        {value.type === 'text' ? (
          <Field label="文本长度" required hint="正整数">
            <Input
              value={value.length ?? ''}
              placeholder="例如 64"
              onChange={(event) =>
                onChange({ ...value, length: event.target.value || undefined })
              }
            />
          </Field>
        ) : null}
      </div>

      {value.type === 'enum' ? (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <p className="text-xs font-medium text-muted">枚举取值</p>
            <Button
              size="sm"
              variant="ghost"
              icon={<Plus className="size-3.5" />}
              onClick={() =>
                onChange({
                  ...value,
                  enumSpec: {
                    values: [
                      ...(value.enumSpec?.values ?? []),
                      {
                        value: Math.max(-1, ...(value.enumSpec?.values ?? []).map((v) => v.value)) + 1,
                        desc: '',
                      },
                    ],
                  },
                })
              }
            >
              添加取值
            </Button>
          </div>
          {(value.enumSpec?.values ?? []).map((item, index) => (
            <div key={index} className="flex items-center gap-2">
              <Input
                aria-label={`枚举值 ${index + 1}`}
                type="number"
                className="w-28"
                value={item.value}
                onChange={(event) => {
                  const values = [...(value.enumSpec?.values ?? [])]
                  values[index] = { ...item, value: Number(event.target.value) }
                  onChange({ ...value, enumSpec: { values } })
                }}
              />
              <Input
                aria-label={`枚举描述 ${index + 1}`}
                value={item.desc}
                placeholder="取值描述"
                onChange={(event) => {
                  const values = [...(value.enumSpec?.values ?? [])]
                  values[index] = { ...item, desc: event.target.value }
                  onChange({ ...value, enumSpec: { values } })
                }}
              />
              <Button
                size="icon"
                variant="ghost"
                aria-label="删除枚举值"
                onClick={() =>
                  onChange({
                    ...value,
                    enumSpec: {
                      values: (value.enumSpec?.values ?? []).filter((_, i) => i !== index),
                    },
                  })
                }
              >
                <Trash2 className="size-4" />
              </Button>
            </div>
          ))}
        </div>
      ) : null}

      {value.type === 'array' ? (
        <div className="space-y-3">
          <Field label="元素个数" hint="可选,1~512">
            <Input
              value={value.arraySpec?.size ?? ''}
              placeholder="例如 10"
              className="w-40"
              onChange={(event) =>
                onChange({
                  ...value,
                  arraySpec: {
                    element: value.arraySpec?.element ?? { type: 'double' },
                    size: event.target.value || undefined,
                  },
                })
              }
            />
          </Field>
          <div>
            <p className="mb-1.5 text-xs font-medium text-muted">元素类型</p>
            <DataTypeEditor
              value={value.arraySpec?.element ?? { type: 'double' }}
              depth={depth + 1}
              onChange={(element) =>
                onChange({
                  ...value,
                  arraySpec: { element, size: value.arraySpec?.size },
                })
              }
            />
          </div>
        </div>
      ) : null}

      {value.type === 'struct' ? (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <p className="text-xs font-medium text-muted">结构体字段</p>
            <Button
              size="sm"
              variant="ghost"
              icon={<Plus className="size-3.5" />}
              onClick={() =>
                onChange({
                  ...value,
                  structSpec: {
                    fields: [
                      ...(value.structSpec?.fields ?? []),
                      { identifier: '', name: '', dataType: { type: 'double' } },
                    ],
                  },
                })
              }
            >
              添加字段
            </Button>
          </div>
          {(value.structSpec?.fields ?? []).map((field, index) => (
            <div key={index} className="space-y-2 rounded-xl border border-border p-3">
              <div className="flex items-center gap-2">
                <Input
                  aria-label={`字段标识符 ${index + 1}`}
                  value={field.identifier}
                  placeholder="字段标识符"
                  onChange={(event) => {
                    const fields = [...(value.structSpec?.fields ?? [])]
                    fields[index] = { ...field, identifier: event.target.value }
                    onChange({ ...value, structSpec: { fields } })
                  }}
                />
                <Input
                  aria-label={`字段名称 ${index + 1}`}
                  value={field.name}
                  placeholder="字段名称"
                  onChange={(event) => {
                    const fields = [...(value.structSpec?.fields ?? [])]
                    fields[index] = { ...field, name: event.target.value }
                    onChange({ ...value, structSpec: { fields } })
                  }}
                />
                <Button
                  size="icon"
                  variant="ghost"
                  aria-label="删除字段"
                  onClick={() =>
                    onChange({
                      ...value,
                      structSpec: {
                        fields: (value.structSpec?.fields ?? []).filter((_, i) => i !== index),
                      },
                    })
                  }
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>
              <DataTypeEditor
                value={field.dataType}
                depth={depth + 1}
                onChange={(dataType) => {
                  const fields = [...(value.structSpec?.fields ?? [])]
                  fields[index] = { ...field, dataType }
                  onChange({ ...value, structSpec: { fields } })
                }}
              />
            </div>
          ))}
        </div>
      ) : null}
    </div>
  )
}
