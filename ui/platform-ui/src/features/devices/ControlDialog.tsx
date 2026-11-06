import { useEffect, useMemo, useState } from 'react'
import { Send } from 'lucide-react'
import { toast } from 'sonner'
import { useControlDevice } from '@/api/hooks'
import type { AccessMode, DataType, DeviceAction, DeviceDetail } from '@/api/types'
import { useI18n, type Translate } from '@/i18n'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select } from '@/components/ui/Field'

const NUMERIC_TYPES: DataType[] = ['int16', 'uint16', 'int32', 'int64', 'float32', 'float64', 'number']

function rawValue(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function parseValue(type: DataType, raw: string, t: Translate): unknown {
  if (type === 'string') return raw
  if (type === 'bool') {
    if (raw === 'true' || raw === '1') return true
    if (raw === 'false' || raw === '0') return false
    throw new Error(t('control.invalidBool'))
  }
  if (NUMERIC_TYPES.includes(type)) {
    const parsed = Number(raw)
    if (raw.trim() === '' || Number.isNaN(parsed)) {
      throw new Error(t('control.invalidNumber', { value: raw }))
    }
    return parsed
  }
  return raw
}

export function ControlDialog({
  open,
  device,
  only,
  onClose,
}: {
  open: boolean
  device: DeviceDetail
  /** 仅显示指定变量名(从变量行进入时使用) */
  only?: string
  onClose: () => void
}) {
  const { t } = useI18n()
  const control = useControlDevice()
  const variables = useMemo(() => {
    // 三种协议的变量结构不同,反控只关心公共字段。
    const all = device.variables as Array<{
      name: string
      dataType: DataType
      accessMode: AccessMode
      value?: unknown
    }>
    const writable = all.filter((variable) => variable.accessMode === 'rw')
    return only ? writable.filter((variable) => variable.name === only) : writable
  }, [device.variables, only])

  const [values, setValues] = useState<Record<string, string>>({})
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    const next: Record<string, string> = {}
    for (const variable of variables) {
      next[variable.name] = rawValue(variable.value)
    }
    setValues(next)
    setError(null)
  }, [open, variables])

  const submit = () => {
    setError(null)
    const actions: DeviceAction[] = []
    try {
      for (const variable of variables) {
        actions.push({
          [variable.name]: parseValue(variable.dataType, values[variable.name] ?? '', t),
        })
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t('control.invalidInput'))
      return
    }

    if (actions.length === 0) {
      setError(t('control.emptyActions'))
      return
    }

    control.mutate(
      { id: device.id, actions },
      {
        onSuccess: () => {
          toast.success(t('control.success', { name: device.name, count: actions.length }))
          onClose()
        },
        onError: (cause) => setError(errorMessage(cause)),
      },
    )
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('control.title')}
      description={t('control.description')}
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            {t('common.action.cancel')}
          </Button>
          <Button
            variant="primary"
            loading={control.isPending}
            icon={<Send className="size-4" />}
            onClick={submit}
            disabled={variables.length === 0}
          >
            {t('control.submit')}
          </Button>
        </>
      }
    >
      {variables.length === 0 ? (
        <p className="rounded-lg border border-border bg-surface-2 px-4 py-6 text-center text-sm text-muted">
          {t('control.noVariables')}
        </p>
      ) : (
        <div className="space-y-3">
          {variables.map((variable) => (
            <Field
              key={variable.name}
              label={
                <span className="flex items-center gap-2">
                  <span className="font-medium text-fg">{variable.name}</span>
                  <span className="rounded bg-surface-2 px-1.5 py-0.5 font-mono text-[10px] text-faint">
                    {variable.dataType}
                  </span>
                </span>
              }
              htmlFor={`control-${variable.name}`}
            >
              {variable.dataType === 'bool' ? (
                <Select
                  id={`control-${variable.name}`}
                  value={values[variable.name] ?? 'false'}
                  onChange={(event) =>
                    setValues((prev) => ({ ...prev, [variable.name]: event.target.value }))
                  }
                >
                  <option value="true">{t('control.boolTrue')}</option>
                  <option value="false">{t('control.boolFalse')}</option>
                </Select>
              ) : (
                <Input
                  id={`control-${variable.name}`}
                  inputMode={NUMERIC_TYPES.includes(variable.dataType) ? 'decimal' : 'text'}
                  value={values[variable.name] ?? ''}
                  onChange={(event) =>
                    setValues((prev) => ({ ...prev, [variable.name]: event.target.value }))
                  }
                  placeholder={
                    variable.dataType === 'string'
                      ? t('control.stringPlaceholder')
                      : t('control.numberPlaceholder')
                  }
                />
              )}
            </Field>
          ))}
          {error ? (
            <p className="rounded-lg border border-danger/30 bg-danger/5 px-3 py-2 text-xs text-danger">
              {error}
            </p>
          ) : null}
          <p className="text-[11px] leading-relaxed text-faint">{t('control.formatHint')}</p>
        </div>
      )}
    </Dialog>
  )
}
