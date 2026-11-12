import { Field, Input, Select, useI18n } from '@harns/core'
import type { DriverSchema } from '@/api/types'
import { sortedFields } from '@/lib/collection-form'
import { localized } from '@/lib/labels'

interface ProtocolFieldsProps {
  schema: DriverSchema
  values: Record<string, string>
  errors: Record<string, string>
  onChange: (name: string, value: string) => void
}

/** 由驱动 Schema 生成的协议连接字段;字段顺序、类型与校验规则全部来自后端。 */
export function ProtocolFields({ schema, values, errors, onChange }: ProtocolFieldsProps) {
  const { locale, t } = useI18n()

  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {sortedFields(schema).map((field) => {
        const label = localized(field.labelZh, field.labelEn, locale)
        const error = errors[`protocol.${field.name}`]
        return (
          <Field
            key={field.name}
            label={label}
            htmlFor={`protocol-${field.name}`}
            required={field.required}
            error={error}
          >
            {field.type === 'select' ? (
              <Select
                id={`protocol-${field.name}`}
                value={values[field.name] ?? ''}
                onChange={(event) => onChange(field.name, event.target.value)}
                aria-invalid={Boolean(error)}
              >
                <option value="">{t('gateway.form.selectPlaceholder')}</option>
                {(field.options ?? []).map((option) => (
                  <option key={option.value} value={option.value}>
                    {localized(option.labelZh, option.labelEn, locale)}
                  </option>
                ))}
              </Select>
            ) : (
              <Input
                id={`protocol-${field.name}`}
                value={values[field.name] ?? ''}
                placeholder={field.name === 'location' ? t('gateway.form.locationPlaceholder') : ''}
                aria-invalid={Boolean(error)}
                autoComplete="off"
                onChange={(event) => onChange(field.name, event.target.value)}
              />
            )}
          </Field>
        )
      })}
    </div>
  )
}
