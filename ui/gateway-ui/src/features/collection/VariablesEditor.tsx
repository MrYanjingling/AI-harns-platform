import { useMemo } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import {
  Input,
  SearchSelect,
  Table,
  TBody,
  TD,
  TH,
  THead,
  TR,
  useI18n,
} from '@harns/core'
import type { DriverSchema } from '@/api/types'
import { ACCESS_MODES } from '@/lib/constants'
import type { VariableFormValues } from '@/lib/collection-form'
import { variableTypes } from '@/lib/collection-form'
import { localized } from '@/lib/labels'
import { ColumnResizeHandle, useColumnResize } from '@/components/useColumnResize'
import { VariableErrors } from './VariableErrors'

interface VariablesEditorProps {
  schema: DriverSchema
  variables: VariableFormValues[]
  errors: Record<string, string>
  onAdd: () => void
  onRemove: (index: number) => void
  onChange: (index: number, patch: Partial<VariableFormValues>) => void
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
  const accessOptions = useMemo(
    () => ACCESS_MODES.map((mode) => ({ value: mode.value, label: t(mode.labelKey) })),
    [t],
  )
  const typeOptions = useMemo(
    () =>
      types.map((type) => ({
        value: type.value,
        label: localized(type.labelZh, type.labelEn, locale) || type.value,
      })),
    [types, locale],
  )
  const { widths, totalWidth, resize } = useColumnResize({
    name: 160,
    dataType: 120,
    address: 158,
    bits: 80,
    precision: 80,
    rate: 80,
    offset: 80,
    accessMode: 96,
    actions: 80,
  })
  const address = schema.variableAddress
  const addressLabel = address ? localized(address.labelZh, address.labelEn, locale) : ''
  const placeholder = address
    ? localized(address.placeholderZh, address.placeholderEn, locale)
    : ''
  return (
    <div className="space-y-2">
      <div className="rounded border border-border">
        <Table style={{ tableLayout: 'fixed', width: totalWidth }}>
          <colgroup>
            <col style={{ width: widths.name }} />
            <col style={{ width: widths.dataType }} />
            <col style={{ width: widths.address }} />
            <col style={{ width: widths.bits }} />
            <col style={{ width: widths.precision }} />
            <col style={{ width: widths.rate }} />
            <col style={{ width: widths.offset }} />
            <col style={{ width: widths.accessMode }} />
            <col style={{ width: widths.actions }} />
          </colgroup>
          <THead className="text-sm font-medium text-fg">
            <TR>
              <TH className="relative">
                {t('gateway.variable.name')}
                <span className="ml-0.5 align-super text-[10px] leading-none text-danger">*</span>
                <ColumnResizeHandle column="name" width={widths.name} onResize={resize} />
              </TH>
              <TH className="relative">
                {t('gateway.variable.dataType')}
                <ColumnResizeHandle column="dataType" width={widths.dataType} onResize={resize} />
              </TH>
              <TH className="relative">
                {addressLabel || t('gateway.variable.address')}
                <span className="ml-0.5 align-super text-[10px] leading-none text-danger">*</span>
                <ColumnResizeHandle column="address" width={widths.address} onResize={resize} />
              </TH>
              <TH className="relative">
                {t('gateway.variable.bits')}
                <ColumnResizeHandle column="bits" width={widths.bits} onResize={resize} />
              </TH>
              <TH className="relative">
                {t('gateway.variable.precision')}
                <ColumnResizeHandle column="precision" width={widths.precision} onResize={resize} />
              </TH>
              <TH className="relative">
                {t('gateway.variable.rate')}
                <ColumnResizeHandle column="rate" width={widths.rate} onResize={resize} />
              </TH>
              <TH className="relative">
                {t('gateway.variable.offset')}
                <ColumnResizeHandle column="offset" width={widths.offset} onResize={resize} />
              </TH>
              <TH className="relative">
                {t('gateway.variable.accessMode')}
                <ColumnResizeHandle
                  column="accessMode"
                  width={widths.accessMode}
                  onResize={resize}
                />
              </TH>
              <TH className="relative">
                {t('common.field.actions')}
                <ColumnResizeHandle column="actions" width={widths.actions} onResize={resize} />
              </TH>
            </TR>
          </THead>
          <TBody>
            {variables.map((variable, index) => {
              return (
                <TR key={index}>
                  <TD>
                    <Input
                      value={variable.name}
                      aria-label={t('gateway.variable.name')}
                      aria-invalid={Boolean(errors[`variables.${index}.name`])}
                      placeholder={t('gateway.variable.namePlaceholder')}
                      className={
                        errors[`variables.${index}.name`]
                          ? 'border-danger hover:border-danger focus:border-danger focus:ring-danger/15'
                          : undefined
                      }
                      onChange={(event) => onChange(index, { name: event.target.value })}
                    />
                  </TD>
                  <TD>
                    {types.length > 0 ? (
                      <SearchSelect
                        value={variable.dataType}
                        options={typeOptions}
                        onChange={(next) => onChange(index, { dataType: next })}
                        ariaLabel={t('gateway.variable.dataType')}
                        emptyText={t('common.state.empty')}
                        searchable={false}
                        className="w-full"
                        panelClassName="w-full"
                      />
                    ) : (
                      <Input
                        value={variable.dataType}
                        aria-label={t('gateway.variable.dataType')}
                        onChange={(event) => onChange(index, { dataType: event.target.value })}
                      />
                    )}
                  </TD>
                  <TD>
                    <Input
                      value={variable.address}
                      aria-label={t('gateway.variable.address')}
                      aria-invalid={Boolean(errors[`variables.${index}.address`])}
                      placeholder={placeholder || t('gateway.variable.addressPlaceholder')}
                      autoComplete="off"
                      className={
                        errors[`variables.${index}.address`]
                          ? 'border-danger hover:border-danger focus:border-danger focus:ring-danger/15'
                          : undefined
                      }
                      onChange={(event) => onChange(index, { address: event.target.value })}
                    />
                  </TD>
                  <TD>
                    <Input
                      value={variable.bits}
                      aria-label={t('gateway.variable.bits')}
                      inputMode="numeric"
                      placeholder="—"
                      onChange={(event) => onChange(index, { bits: event.target.value })}
                    />
                  </TD>
                  <TD>
                    <Input
                      value={variable.precision ?? ''}
                      aria-label={t('gateway.variable.precision')}
                      inputMode="numeric"
                      placeholder="0"
                      onChange={(event) => onChange(index, { precision: event.target.value })}
                    />
                  </TD>
                  <TD>
                    <Input
                      value={variable.rate}
                      aria-label={t('gateway.variable.rate')}
                      inputMode="decimal"
                      placeholder="1"
                      onChange={(event) => onChange(index, { rate: event.target.value })}
                    />
                  </TD>
                  <TD>
                    <Input
                      value={variable.offset ?? ''}
                      aria-label={t('gateway.variable.offset')}
                      inputMode="decimal"
                      placeholder="0"
                      onChange={(event) => onChange(index, { offset: event.target.value })}
                    />
                  </TD>
                  <TD>
                    <SearchSelect
                      value={variable.accessMode}
                      options={accessOptions}
                      onChange={(next) =>
                        onChange(index, {
                          accessMode: next as VariableFormValues['accessMode'],
                        })
                      }
                      ariaLabel={t('gateway.variable.accessMode')}
                      emptyText={t('common.state.empty')}
                      searchable={false}
                      className="w-full"
                      panelClassName="w-full"
                    />
                  </TD>
                  <TD className="text-right">
                    <div className="flex items-center justify-end gap-0.5">
                      <button
                        type="button"
                        title={t('gateway.variable.addCollection')}
                        aria-label={t('gateway.variable.addCollection')}
                        onClick={onAdd}
                        className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
                      >
                        <Plus className="size-3.5" />
                      </button>
                      <button
                        type="button"
                        title={t('gateway.variable.remove')}
                        aria-label={t('gateway.variable.remove')}
                        onClick={() => onRemove(index)}
                        disabled={variables.length <= 1}
                        className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-danger/10 hover:text-danger disabled:pointer-events-none disabled:opacity-40"
                      >
                        <Trash2 className="size-3.5" />
                      </button>
                    </div>
                  </TD>
                </TR>
              )
            })}
          </TBody>
        </Table>
      </div>
      <VariableErrors errors={errors} />
    </div>
  )
}
