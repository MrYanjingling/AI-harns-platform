import { useState } from 'react'
import type { ThingModelFeature } from '@/api/types'
import { isNumericDataType } from '@/features/thing-models/data-type'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'

function hasValue(value: unknown): boolean {
  return value !== undefined && value !== null && value !== ''
}

/** 按特征 dataType 渲染输入控件,复杂类型使用 JSON 文本。 */
export function FeatureValuesEditor({
  features,
  value,
  onChange,
}: {
  features: ThingModelFeature[]
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
}) {
  const set = (identifier: string, next: unknown) => {
    const copy = { ...value }
    if (!hasValue(next)) delete copy[identifier]
    else copy[identifier] = next
    onChange(copy)
  }

  if (features.length === 0) {
    return (
      <p className="rounded-lg border border-border bg-surface-2/50 px-3 py-2 text-xs text-muted">
        当前物模型没有定义特征,无法填写静态特征值。
      </p>
    )
  }

  return (
    <div className="space-y-3">
      {features.map((feature) => (
        <FeatureField
          key={feature.identifier}
          feature={feature}
          value={value[feature.identifier]}
          onChange={(next) => set(feature.identifier, next)}
        />
      ))}
    </div>
  )
}

function FeatureField({
  feature,
  value,
  onChange,
}: {
  feature: ThingModelFeature
  value: unknown
  onChange: (next: unknown) => void
}) {
  const type = feature.dataType.type
  const unit = feature.dataType.unit ? ` (${feature.dataType.unit})` : ''

  if (isNumericDataType(type)) {
    return (
      <Field label={`${feature.name}${unit}`} hint={feature.identifier}>
        <Input
          type="number"
          step="any"
          value={typeof value === 'number' || typeof value === 'string' ? String(value) : ''}
          placeholder="未设置"
          onChange={(event) => {
            const raw = event.target.value
            if (raw === '') return onChange(undefined)
            const parsed = Number(raw)
            onChange(Number.isFinite(parsed) ? parsed : raw)
          }}
        />
      </Field>
    )
  }

  if (type === 'bool') {
    return (
      <Field label={feature.name} hint={feature.identifier}>
        <Select
          value={value === true ? 'true' : value === false ? 'false' : ''}
          onChange={(event) => {
            const raw = event.target.value
            onChange(raw === '' ? undefined : raw === 'true')
          }}
        >
          <option value="">未设置</option>
          <option value="true">{feature.dataType.trueValue || 'true'}</option>
          <option value="false">{feature.dataType.falseValue || 'false'}</option>
        </Select>
      </Field>
    )
  }

  if (type === 'enum') {
    return (
      <Field label={feature.name} hint={feature.identifier}>
        <Select
          value={hasValue(value) ? String(value) : ''}
          onChange={(event) => {
            const raw = event.target.value
            onChange(raw === '' ? undefined : Number(raw))
          }}
        >
          <option value="">未设置</option>
          {(feature.dataType.enumSpec?.values ?? []).map((item) => (
            <option key={item.value} value={item.value}>
              {item.desc || String(item.value)} ({item.value})
            </option>
          ))}
        </Select>
      </Field>
    )
  }

  if (type === 'array' || type === 'struct') {
    return (
      <JsonFeatureInput
        key={`${feature.identifier}-${JSON.stringify(value ?? null)}`}
        label={feature.name}
        hint={`${feature.identifier} · JSON`}
        value={value}
        onChange={onChange}
      />
    )
  }

  return (
    <Field label={feature.name} hint={feature.identifier}>
      <Input
        value={typeof value === 'string' || typeof value === 'number' ? String(value) : ''}
        placeholder="未设置"
        onChange={(event) => onChange(event.target.value || undefined)}
      />
    </Field>
  )
}

function JsonFeatureInput({
  label,
  hint,
  value,
  onChange,
}: {
  label: string
  hint: string
  value: unknown
  onChange: (next: unknown) => void
}) {
  const [text, setText] = useState(() => (value === undefined ? '' : JSON.stringify(value)))
  const [error, setError] = useState<string | null>(null)

  const commit = (next: string) => {
    setText(next)
    if (!next.trim()) {
      setError(null)
      onChange(undefined)
      return
    }
    try {
      onChange(JSON.parse(next))
      setError(null)
    } catch {
      setError('JSON 格式错误')
    }
  }

  return (
    <Field label={label} hint={hint} error={error ?? undefined}>
      <Textarea
        value={text}
        className="min-h-20 font-mono text-xs"
        placeholder="JSON 值,例如 [1,2,3]"
        spellCheck={false}
        onChange={(event) => commit(event.target.value)}
      />
    </Field>
  )
}
