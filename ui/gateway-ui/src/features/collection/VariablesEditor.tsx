import { Plus, Trash2 } from 'lucide-react'
import { Button, Input, Select, Table, TBody, TD, TH, THead, TR, useI18n } from '@harns/core'
import type { DriverSchema } from '@/api/types'
import { ACCESS_MODES } from '@/lib/constants'
import type { VariableFormValues } from '@/lib/collection-form'
import { variableTypes } from '@/lib/collection-form'
import { localized } from '@/lib/labels'

interface VariablesEditorProps {
  schema: DriverSchema
  variables: VariableFormValues[]
  errors: Record<string, string>
  onAdd: () => void
  onRemove: (index: number) => void
  onChange: (index: number, patch: Partial<VariableFormValues>) => void
}

function FieldError({ message }: { message?: string }) {
  if (!message) return null
  return <p className="mt-1 text-[11px] leading-tight text-danger">{message}</p>
}

/** 变量定义编辑器;数据类型与地址规则来自驱动 Schema。 */
export function VariablesEditor({
  schema,
  variables,
  errors,
  onAdd,
  onRemove,
  onChange,
}: VariablesEditorProps) {
  const { t, locale } = useI18n()
  const types = variableTypes(schema)
  const address = schema.variableAddress
  const addressLabel = address ? localized(address.labelZh, address.labelEn, locale) : ''
  const placeholder = address
    ? localized(address.placeholderZh, address.placeholderEn, locale)
    : ''
  const description = address
    ? localized(address.descriptionZh, address.descriptionEn, locale)
    : ''

  return (
    <div className="space-y-2">
      <div className="overflow-hidden rounded border border-border">
        <Table>
          <THead>
            <TR>
              <TH className="min-w-32">{t('gateway.variable.name')}</TH>
              <TH className="min-w-28">{t('gateway.variable.dataType')}</TH>
              <TH className="min-w-36">{addressLabel || t('gateway.variable.address')}</TH>
              <TH className="min-w-20">{t('gateway.variable.bits')}</TH>
              <TH className="min-w-20">{t('gateway.variable.rate')}</TH>
              <TH className="min-w-24">{t('gateway.variable.accessMode')}</TH>
              <TH className="w-10 text-right">{t('common.field.actions')}</TH>
            </TR>
          </THead>
          <TBody>
            {variables.map((variable, index) => {
              const prefix = `variables.${index}`
              return (
                <TR key={index}>
                  <TD>
                    <Input
                      value={variable.name}
                      aria-label={t('gateway.variable.name')}
                      placeholder={t('gateway.variable.namePlaceholder')}
                      onChange={(event) => onChange(index, { name: event.target.value })}
                    />
                    <FieldError message={errors[`${prefix}.name`]} />
                  </TD>
                  <TD>
                    {types.length > 0 ? (
                      <Select
                        value={variable.dataType}
                        aria-label={t('gateway.variable.dataType')}
                        onChange={(event) => onChange(index, { dataType: event.target.value })}
                      >
                        {types.map((type) => (
                          <option key={type.value} value={type.value}>
                            {localized(type.labelZh, type.labelEn, locale) || type.value}
                          </option>
                        ))}
                      </Select>
                    ) : (
                      <Input
                        value={variable.dataType}
                        aria-label={t('gateway.variable.dataType')}
                        onChange={(event) => onChange(index, { dataType: event.target.value })}
                      />
                    )}
                    <FieldError message={errors[`${prefix}.dataType`]} />
                  </TD>
                  <TD>
                    <Input
                      value={variable.address}
                      aria-label={t('gateway.variable.address')}
                      placeholder={placeholder || t('gateway.variable.addressPlaceholder')}
                      autoComplete="off"
                      onChange={(event) => onChange(index, { address: event.target.value })}
                    />
                    <FieldError message={errors[`${prefix}.address`]} />
                  </TD>
                  <TD>
                    <Input
                      value={variable.bits}
                      aria-label={t('gateway.variable.bits')}
                      inputMode="numeric"
                      placeholder="—"
                      onChange={(event) => onChange(index, { bits: event.target.value })}
                    />
                    <FieldError message={errors[`${prefix}.bits`]} />
                  </TD>
                  <TD>
                    <Input
                      value={variable.rate}
                      aria-label={t('gateway.variable.rate')}
                      inputMode="decimal"
                      placeholder="1"
                      onChange={(event) => onChange(index, { rate: event.target.value })}
                    />
                    <FieldError message={errors[`${prefix}.rate`]} />
                  </TD>
                  <TD>
                    <Select
                      value={variable.accessMode}
                      aria-label={t('gateway.variable.accessMode')}
                      onChange={(event) =>
                        onChange(index, {
                          accessMode: event.target.value as VariableFormValues['accessMode'],
                        })
                      }
                    >
                      {ACCESS_MODES.map((mode) => (
                        <option key={mode.value} value={mode.value}>
                          {t(mode.labelKey)}
                        </option>
                      ))}
                    </Select>
                  </TD>
                  <TD className="text-right">
                    <Button
                      variant="ghost"
                      size="icon"
                      title={t('gateway.variable.remove')}
                      aria-label={t('gateway.variable.remove')}
                      onClick={() => onRemove(index)}
                      disabled={variables.length <= 1}
                    >
                      <Trash2 className="size-3.5 text-danger" />
                    </Button>
                  </TD>
                </TR>
              )
            })}
          </TBody>
        </Table>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <Button variant="outline" size="sm" onClick={onAdd} icon={<Plus className="size-3.5" />}>
          {t('gateway.variable.add')}
        </Button>
        {description ? (
          <p className="text-[11px] text-faint">{description}</p>
        ) : null}
      </div>
      {errors.variables ? <p className="text-xs text-danger">{errors.variables}</p> : null}
    </div>
  )
}
