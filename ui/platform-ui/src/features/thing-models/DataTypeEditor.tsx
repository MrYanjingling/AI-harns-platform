import { Plus, Trash2 } from 'lucide-react'
import type { ThingDataType, ThingDataTypeDetail } from '@/api/types'
import { useI18n } from '@harns/core'
import { THING_DATA_TYPES } from '@/lib/constants'
import { Button } from '@harns/core'
import { Field, Input, Select } from '@harns/core'
import { cn } from '@harns/core'
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
  const { t } = useI18n()
  const allowNested = depth < MAX_NESTING_DEPTH
  const types = allowNested ? THING_DATA_TYPES : SCALAR_TYPES

  const changeType = (type: ThingDataType) => {
    if (type === value.type) return
    onChange(emptyDataType(type))
  }

  return (
    <div className={cn('space-y-3', depth > 0 && 'rounded-xl border border-border bg-surface-2/40 p-3')}>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={depth === 0 ? t('dataType.label') : t('dataType.elementLabel')}>
          <Select
            value={value.type}
            onChange={(event) => changeType(event.target.value as ThingDataType)}
          >
            {types.map((item) => (
              <option key={item.value} value={item.value}>
                {item.label} · {t(item.descriptionKey)}
              </option>
            ))}
          </Select>
        </Field>

        {isNumericDataType(value.type) ? (
          <>
            <Field label={t('dataType.unit')} hint={t('dataType.unitHint')}>
              <Input
                value={value.unit ?? ''}
                placeholder={t('dataType.optional')}
                onChange={(event) =>
                  onChange({ ...value, unit: event.target.value || undefined })
                }
              />
            </Field>
            <Field label={t('dataType.unitName')}>
              <Input
                value={value.unitName ?? ''}
                placeholder={t('dataType.unitNamePlaceholder')}
                onChange={(event) =>
                  onChange({ ...value, unitName: event.target.value || undefined })
                }
              />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label={t('dataType.min')}>
                <Input
                  value={value.min ?? ''}
                  placeholder={t('dataType.optional')}
                  onChange={(event) =>
                    onChange({ ...value, min: event.target.value || undefined })
                  }
                />
              </Field>
              <Field label={t('dataType.max')}>
                <Input
                  value={value.max ?? ''}
                  placeholder={t('dataType.optional')}
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
            <Field label={t('dataType.trueValue')}>
              <Input
                value={value.trueValue ?? ''}
                placeholder={t('dataType.trueValuePlaceholder')}
                onChange={(event) =>
                  onChange({ ...value, trueValue: event.target.value || undefined })
                }
              />
            </Field>
            <Field label={t('dataType.falseValue')}>
              <Input
                value={value.falseValue ?? ''}
                placeholder={t('dataType.falseValuePlaceholder')}
                onChange={(event) =>
                  onChange({ ...value, falseValue: event.target.value || undefined })
                }
              />
            </Field>
          </>
        ) : null}

        {value.type === 'text' ? (
          <Field label={t('dataType.textLengthLabel')} required hint={t('dataType.textLengthHint')}>
            <Input
              value={value.length ?? ''}
              placeholder={t('dataType.textLengthPlaceholder')}
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
            <p className="text-xs font-medium text-muted">{t('dataType.enumValues')}</p>
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
              {t('dataType.addValue')}
            </Button>
          </div>
          {(value.enumSpec?.values ?? []).map((item, index) => (
            <div key={index} className="flex items-center gap-2">
              <Input
                aria-label={t('dataType.enumValueAria', { index: index + 1 })}
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
                aria-label={t('dataType.enumDescAria', { index: index + 1 })}
                value={item.desc}
                placeholder={t('dataType.enumDescPlaceholder')}
                onChange={(event) => {
                  const values = [...(value.enumSpec?.values ?? [])]
                  values[index] = { ...item, desc: event.target.value }
                  onChange({ ...value, enumSpec: { values } })
                }}
              />
              <Button
                size="icon"
                variant="ghost"
                aria-label={t('dataType.removeValue')}
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
          <Field label={t('dataType.arraySize')} hint={t('dataType.arraySizeHint')}>
            <Input
              value={value.arraySpec?.size ?? ''}
              placeholder={t('dataType.arraySizePlaceholder')}
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
            <p className="mb-1.5 text-xs font-medium text-muted">{t('dataType.elementType')}</p>
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
            <p className="text-xs font-medium text-muted">{t('dataType.structFields')}</p>
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
              {t('dataType.addField')}
            </Button>
          </div>
          {(value.structSpec?.fields ?? []).map((field, index) => (
            <div key={index} className="space-y-2 rounded-xl border border-border p-3">
              <div className="flex items-center gap-2">
                <Input
                  aria-label={t('dataType.fieldIdentifierAria', { index: index + 1 })}
                  value={field.identifier}
                  placeholder={t('dataType.fieldIdentifierPlaceholder')}
                  onChange={(event) => {
                    const fields = [...(value.structSpec?.fields ?? [])]
                    fields[index] = { ...field, identifier: event.target.value }
                    onChange({ ...value, structSpec: { fields } })
                  }}
                />
                <Input
                  aria-label={t('dataType.fieldNameAria', { index: index + 1 })}
                  value={field.name}
                  placeholder={t('dataType.fieldNamePlaceholder')}
                  onChange={(event) => {
                    const fields = [...(value.structSpec?.fields ?? [])]
                    fields[index] = { ...field, name: event.target.value }
                    onChange({ ...value, structSpec: { fields } })
                  }}
                />
                <Button
                  size="icon"
                  variant="ghost"
                  aria-label={t('dataType.removeField')}
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
