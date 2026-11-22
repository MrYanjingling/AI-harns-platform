import { Field, Input, SearchSelect, type SearchSelectOption, useI18n } from '@harns/core'
import type { DriverSchema } from '@/api/types'
import { resolveField, sortedFields } from '@/lib/collection-form'
import { localized } from '@/lib/labels'

interface ProtocolFieldsProps {
  schema: DriverSchema
  values: Record<string, string>
  errors: Record<string, string>
  onChange: (name: string, value: string) => void
  columns?: 2 | 3
}

/** 由驱动 Schema 生成的协议连接字段;字段顺序、类型与校验规则全部来自后端。 */
export function ProtocolFields({
  schema,
  values,
  errors,
  onChange,
  columns = 3,
}: ProtocolFieldsProps) {
  const { locale, t } = useI18n()

  return (
    <div
      className={
        columns === 2
          ? 'grid gap-x-8 gap-y-4 sm:grid-cols-2'
          : 'grid gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3'
      }
    >
      {sortedFields(schema).map((field) => {
        const resolved = resolveField(field, values)
        const label = localized(resolved.labelZh, resolved.labelEn, locale)
        const error = errors[`protocol.${field.name}`]
        const placeholder = resolved.placeholderZh || resolved.placeholderEn
          ? localized(resolved.placeholderZh ?? '', resolved.placeholderEn ?? '', locale)
          : field.name === 'location'
            ? t('gateway.form.locationPlaceholder')
            : ''
        return (
          <Field
            key={field.name}
            label={label}
            htmlFor={`protocol-${field.name}`}
            labelClassName="text-sm"
            labelWidthClassName="w-[70px]"
            inline
            required={field.required}
            error={error}
          >
            <div className="w-full max-w-[200px]">
              {field.type === 'select' ? (
                <SearchSelect
                  id={`protocol-${field.name}`}
                  value={values[field.name] ?? ''}
                  options={(field.options ?? []).map<SearchSelectOption>((option) => ({
                    value: option.value,
                    label: localized(option.labelZh, option.labelEn, locale),
                  }))}
                  onChange={(next) => onChange(field.name, next)}
                  ariaLabel={label}
                  emptyText={t('common.state.empty')}
                  searchable={false}
                  className="w-full"
                />
              ) : (
                <Input
                  id={`protocol-${field.name}`}
                  value={values[field.name] ?? ''}
                  placeholder={placeholder}
                  aria-invalid={Boolean(error)}
                  autoComplete="off"
                  onChange={(event) => onChange(field.name, event.target.value)}
                />
              )}
            </div>
          </Field>
        )
      })}
    </div>
  )
}
