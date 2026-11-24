import { useLayoutEffect, useRef, useState, type CSSProperties } from 'react'
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
  const containerRef = useRef<HTMLDivElement>(null)
  const [labelWidth, setLabelWidth] = useState(70)
  const fields = sortedFields(schema)
  const labelSignature = fields
    .map((field) => {
      const resolved = resolveField(field, values)
      return localized(resolved.labelZh, resolved.labelEn, locale)
    })
    .join('\u0000')

  useLayoutEffect(() => {
    const labels = containerRef.current?.querySelectorAll<HTMLLabelElement>(
      'label[for^="protocol-"]',
    )
    if (!labels?.length) return
    let maxWidth = 0
    labels.forEach((label) => {
      maxWidth = Math.max(maxWidth, label.scrollWidth)
    })
    setLabelWidth(Math.min(240, Math.max(70, Math.ceil(maxWidth) + 8)))
  }, [labelSignature])

  return (
    <div
      ref={containerRef}
      className={
        columns === 2
          ? 'grid gap-x-8 gap-y-4 sm:grid-cols-2'
          : 'grid gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3'
      }
      style={{ '--protocol-label-width': `${labelWidth}px` } as CSSProperties}
    >
      {fields.map((field) => {
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
            labelWidthClassName="w-[var(--protocol-label-width)]"
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
