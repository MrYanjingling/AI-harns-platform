import { Plus, Trash2 } from 'lucide-react'
import { Button, Input, Select, Table, TBody, TD, TH, THead, TR, useI18n } from '@harns/core'
import type { DeviceType } from '@/api/types'
import { ACCESS_MODES, dataTypesFor } from '@/lib/constants'
import type { VariableFormValues } from '@/lib/device-form'

interface VariablesEditorProps {
  deviceType: DeviceType
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

export function VariablesEditor({
  deviceType,
  variables,
  errors,
  onAdd,
  onRemove,
  onChange,
}: VariablesEditorProps) {
  const { t } = useI18n()
  const dataTypes = dataTypesFor(deviceType)

  return (
    <div className="space-y-2">
      <div className="overflow-hidden rounded border border-border">
        <Table>
          <THead>
            <TR>
              <TH className="min-w-36">{t('gateway.form.field.variableName')}</TH>
              <TH className="min-w-28">{t('gateway.form.field.dataType')}</TH>
              <TH className="min-w-28">{t('gateway.form.field.address')}</TH>
              {deviceType === 'modbus' ? (
                <>
                  <TH className="min-w-24">{t('gateway.form.field.functionCode')}</TH>
                  <TH className="min-w-20">{t('gateway.form.field.bits')}</TH>
                  <TH className="min-w-24">{t('gateway.form.field.amount')}</TH>
                </>
              ) : null}
              {deviceType === 'opcUa' ? (
                <TH className="min-w-28">{t('gateway.form.field.namespace')}</TH>
              ) : null}
              {deviceType !== 'opcUa' ? (
                <TH className="min-w-24">{t('gateway.form.field.rate')}</TH>
              ) : null}
              <TH className="min-w-32">{t('gateway.form.field.defaultValue')}</TH>
              <TH className="min-w-24">{t('gateway.form.field.accessMode')}</TH>
              <TH className="w-12 text-right">{t('common.field.actions')}</TH>
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
                      aria-label={t('gateway.form.field.variableName')}
                      placeholder={t('gateway.form.variableRow', { index: index + 1 })}
                      onChange={(event) => onChange(index, { name: event.target.value })}
                    />
                    <FieldError message={errors[`${prefix}.name`]} />
                  </TD>
                  <TD>
                    <Select
                      value={variable.dataType}
                      aria-label={t('gateway.form.field.dataType')}
                      onChange={(event) => onChange(index, { dataType: event.target.value })}
                    >
                      {dataTypes.map((type) => (
                        <option key={type} value={type}>
                          {type}
                        </option>
                      ))}
                    </Select>
                  </TD>
                  <TD>
                    <Input
                      value={variable.address}
                      aria-label={t('gateway.form.field.address')}
                      inputMode={deviceType === 'modbus' ? 'numeric' : 'text'}
                      placeholder={deviceType === 'modbus' ? '0' : 'ns=2;s=Tag'}
                      onChange={(event) => onChange(index, { address: event.target.value })}
                    />
                    <FieldError message={errors[`${prefix}.address`]} />
                  </TD>
                  {deviceType === 'modbus' ? (
                    <>
                      <TD>
                        <Select
                          value={variable.functionCode}
                          aria-label={t('gateway.form.field.functionCode')}
                          onChange={(event) =>
                            onChange(index, { functionCode: event.target.value })
                          }
                        >
                          {[1, 2, 3, 4].map((code) => (
                            <option key={code} value={String(code)}>
                              {code}
                            </option>
                          ))}
                        </Select>
                        <FieldError message={errors[`${prefix}.functionCode`]} />
                      </TD>
                      <TD>
                        <Input
                          value={variable.bits}
                          aria-label={t('gateway.form.field.bits')}
                          inputMode="numeric"
                          placeholder="0"
                          onChange={(event) => onChange(index, { bits: event.target.value })}
                        />
                        <FieldError message={errors[`${prefix}.bits`]} />
                      </TD>
                      <TD>
                        <Input
                          value={variable.amount}
                          aria-label={t('gateway.form.field.amount')}
                          inputMode="numeric"
                          placeholder="1"
                          onChange={(event) => onChange(index, { amount: event.target.value })}
                        />
                        <FieldError message={errors[`${prefix}.amount`]} />
                      </TD>
                    </>
                  ) : null}
                  {deviceType === 'opcUa' ? (
                    <TD>
                      <Input
                        value={variable.namespace}
                        aria-label={t('gateway.form.field.namespace')}
                        inputMode="numeric"
                        placeholder="1"
                        onChange={(event) => onChange(index, { namespace: event.target.value })}
                      />
                      <FieldError message={errors[`${prefix}.namespace`]} />
                    </TD>
                  ) : null}
                  {deviceType !== 'opcUa' ? (
                    <TD>
                      <Input
                        value={variable.rate}
                        aria-label={t('gateway.form.field.rate')}
                        inputMode="decimal"
                        placeholder="1"
                        onChange={(event) => onChange(index, { rate: event.target.value })}
                      />
                      <FieldError message={errors[`${prefix}.rate`]} />
                    </TD>
                  ) : null}
                  <TD>
                    <Input
                      value={variable.defaultValue}
                      aria-label={t('gateway.form.field.defaultValue')}
                      onChange={(event) => onChange(index, { defaultValue: event.target.value })}
                    />
                  </TD>
                  <TD>
                    <Select
                      value={variable.accessMode}
                      aria-label={t('gateway.form.field.accessMode')}
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
                      title={t('gateway.form.removeVariable')}
                      aria-label={t('gateway.form.removeVariable')}
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
      <div className="flex items-center justify-between gap-2">
        <Button variant="outline" size="sm" onClick={onAdd} icon={<Plus className="size-3.5" />}>
          {t('gateway.form.addVariable')}
        </Button>
        {variables.length === 0 ? (
          <p className="text-xs text-danger">{t('gateway.form.emptyVariables')}</p>
        ) : null}
      </div>
      {errors.variables ? <p className="text-xs text-danger">{errors.variables}</p> : null}
    </div>
  )
}
