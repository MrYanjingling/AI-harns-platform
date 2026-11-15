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
import { variableTypes, type VariableFormValues } from '@/lib/collection-form'
import { localized } from '@/lib/labels'
import { ColumnResizeHandle, useColumnResize } from '@/components/useColumnResize'
import { VariableErrors } from './VariableErrors'

interface CalculatedVariablesEditorProps {
  schema: DriverSchema
  variables: VariableFormValues[]
  errors: Record<string, string>
  onAdd: () => void
  onRemove: (index: number) => void
  onChange: (index: number, patch: Partial<VariableFormValues>) => void
}

export function CalculatedVariablesEditor({
  schema,
  variables,
  errors,
  onAdd,
  onRemove,
  onChange,
}: CalculatedVariablesEditorProps) {
  const { t, locale } = useI18n()
  const types = variableTypes(schema)
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
    expression: 494,
    precision: 80,
    actions: 80,
  })

  return (
    <div className="space-y-2">
      <div className="rounded border border-border">
        <Table style={{ tableLayout: 'fixed', width: totalWidth }}>
          <colgroup>
            <col style={{ width: widths.name }} />
            <col style={{ width: widths.dataType }} />
            <col style={{ width: widths.expression }} />
            <col style={{ width: widths.precision }} />
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
                {t('gateway.variable.expression')}
                <span className="ml-0.5 align-super text-[10px] leading-none text-danger">*</span>
                <ColumnResizeHandle
                  column="expression"
                  width={widths.expression}
                  onResize={resize}
                />
              </TH>
              <TH className="relative">
                {t('gateway.variable.precision')}
                <ColumnResizeHandle column="precision" width={widths.precision} onResize={resize} />
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
                      value={variable.expression ?? ''}
                      aria-label={t('gateway.variable.expression')}
                      aria-invalid={Boolean(errors[`variables.${index}.expression`])}
                      placeholder={t('gateway.variable.expressionPlaceholder')}
                      autoComplete="off"
                      className={
                        errors[`variables.${index}.expression`]
                          ? 'border-danger hover:border-danger focus:border-danger focus:ring-danger/15'
                          : undefined
                      }
                      onChange={(event) => onChange(index, { expression: event.target.value })}
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
                  <TD className="text-right">
                    <div className="flex items-center justify-end gap-0.5">
                      <button
                        type="button"
                        title={t('gateway.variable.addCalculated')}
                        aria-label={t('gateway.variable.addCalculated')}
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
