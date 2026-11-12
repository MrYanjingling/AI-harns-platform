import type { Translate } from '@harns/core'
import type {
  AccessMode,
  CollectionDetail,
  CollectionPayload,
  CollectionVariableInput,
  DriverField,
  DriverSchema,
  VariableDataType,
} from '@/api/types'
import { REDACTED_VALUE, toProtocolValue } from './constants'

export interface VariableFormValues {
  name: string
  dataType: string
  address: string
  bits: string
  rate: string
  accessMode: AccessMode
}

export interface CollectionFormValues {
  driverType: string
  driverModel: string
  name: string
  collectorCycleMs: string
  protocol: Record<string, string>
  variables: VariableFormValues[]
}

const VARIABLE_NAME = /^[A-Za-z_][A-Za-z0-9_]*$/
const POSITIVE_INT = /^\d+$/

/** 常用协议字段的合理默认值;其余字段留空由用户填写。 */
function defaultProtocolValue(field: DriverField, model: string): string {
  const defaults: Record<string, string> = {
    baudRate: '9600',
    dataBits: '8',
    parity: 'noParity',
    stopBits: '1',
    memoryLayout: 'ABCD',
    tcpCount: '1',
    rack: '0',
    slot: '1',
    pduSize: '480',
    connectionType: '1',
    localTsap: '0x1000',
    remoteTsap: '0x1000',
    port: model === 'opcUa' ? '4840' : '102',
  }
  if (defaults[field.name] !== undefined) return defaults[field.name]
  if (field.type === 'select') return field.options?.[0]?.value ?? ''
  return ''
}

export function sortedFields(schema: DriverSchema | undefined): DriverField[] {
  return [...(schema?.fields ?? [])].sort((a, b) => a.sort - b.sort)
}

export function variableTypes(schema: DriverSchema | undefined): VariableDataType[] {
  return schema?.variableDataTypes ?? []
}

export function createEmptyVariable(schema: DriverSchema | undefined): VariableFormValues {
  return {
    name: '',
    dataType: variableTypes(schema)[0]?.value ?? '',
    address: '',
    bits: '',
    rate: '',
    accessMode: 'r',
  }
}

export function createFormValues(
  driverType: string,
  driverModel: string,
  schema: DriverSchema | undefined,
): CollectionFormValues {
  const protocol: Record<string, string> = {}
  for (const field of sortedFields(schema)) {
    protocol[field.name] = defaultProtocolValue(field, driverModel)
  }
  return {
    driverType,
    driverModel,
    name: '',
    collectorCycleMs: '1000',
    protocol,
    variables: [createEmptyVariable(schema)],
  }
}

export function detailToForm(
  detail: CollectionDetail,
  schema: DriverSchema | undefined,
): CollectionFormValues {
  const protocol: Record<string, string> = {}
  for (const field of sortedFields(schema)) {
    const raw = detail.protocolConfig?.[field.name]
    protocol[field.name] = raw === undefined || raw === null ? '' : String(raw)
  }
  // schema 尚未加载时先把已有配置放进表单,避免切换 schema 后丢值。
  for (const [key, value] of Object.entries(detail.protocolConfig ?? {})) {
    if (!(key in protocol)) protocol[key] = value === null ? '' : String(value)
  }
  return {
    driverType: detail.driverType,
    driverModel: detail.driverModel,
    name: detail.name,
    collectorCycleMs: String(detail.collectorCycleMs || 1000),
    protocol,
    variables: [],
  }
}

function protocolConfigFromForm(
  values: CollectionFormValues,
  schema: DriverSchema | undefined,
): Record<string, unknown> {
  const config: Record<string, unknown> = {}
  const names = new Set(sortedFields(schema).map((field) => field.name))
  // 保留 schema 之外的自定义字段(自定义驱动)。
  for (const [key, raw] of Object.entries(values.protocol)) {
    if (!names.has(key)) continue
    if (raw.trim() === REDACTED_VALUE) continue
    const parsed = toProtocolValue(key, raw)
    if (parsed !== undefined) config[key] = parsed
  }
  return config
}

function variableInputsFromForm(
  values: CollectionFormValues,
): CollectionVariableInput[] {
  return toVariableInputs(values.variables)
}

export function toVariableInputs(
  variables: VariableFormValues[],
): CollectionVariableInput[] {
  return variables.map((variable) => {
    const input: CollectionVariableInput = {
      name: variable.name.trim(),
      dataType: variable.dataType.trim(),
      address: variable.address.trim(),
      accessMode: variable.accessMode,
    }
    const bits = variable.bits.trim() === '' ? undefined : Number(variable.bits)
    if (bits !== undefined && Number.isFinite(bits)) input.bits = bits
    const rate = variable.rate.trim() === '' ? undefined : Number(variable.rate)
    if (rate !== undefined && Number.isFinite(rate)) input.rate = rate
    return input
  })
}

export interface CollectionFormValidation {
  errors: Record<string, string>
  payload?: CollectionPayload
}

/**
 * 协议字段由驱动 Schema 决定,校验规则(required / regex / select)同样来自 Schema。
 * includeVariables 为 false 时表示编辑已有采集器;变量通过独立接口维护。
 */
export function validateCollectionForm(
  values: CollectionFormValues,
  schema: DriverSchema | undefined,
  t: Translate,
  includeVariables: boolean,
): CollectionFormValidation {
  const errors: Record<string, string> = {}
  const setError = (path: string, message: string) => {
    if (!errors[path]) errors[path] = message
  }

  if (!schema) {
    return { errors: { form: t('gateway.form.validation.schemaMissing') } }
  }

  const name = values.name.trim()
  if (!name) setError('name', t('gateway.form.validation.nameRequired'))
  else if (name.length > 128) setError('name', t('gateway.form.validation.nameMax'))

  if (!POSITIVE_INT.test(values.collectorCycleMs)) {
    setError('collectorCycleMs', t('gateway.form.validation.cycleInvalid'))
  } else {
    const cycle = Number(values.collectorCycleMs)
    if (cycle < 1) setError('collectorCycleMs', t('gateway.form.validation.cycleInvalid'))
  }

  for (const field of sortedFields(schema)) {
    const value = (values.protocol[field.name] ?? '').trim()
    const path = `protocol.${field.name}`
    if (!value) {
      if (field.required) setError(path, t('gateway.form.validation.protocolRequired'))
      continue
    }
    if (field.type === 'select' && !field.options?.some((option) => option.value === value)) {
      setError(path, t('gateway.form.validation.protocolInvalid'))
      continue
    }
    if (field.regex && !safeRegExp(field.regex)?.test(value)) {
      setError(path, t('gateway.form.validation.protocolInvalid'))
    }
  }

  if (includeVariables) {
    if (values.variables.length === 0) {
      setError('variables', t('gateway.form.validation.variablesRequired'))
    }
    for (const [path, message] of Object.entries(
      validateVariableRows(values.variables, schema, t),
    )) {
      setError(path, message)
    }
  }

  if (Object.keys(errors).length > 0) return { errors }

  const payload: CollectionPayload = {
    driverType: values.driverType,
    driverModel: values.driverModel,
    name,
    collectorCycleMs: Number(values.collectorCycleMs),
    protocolConfig: protocolConfigFromForm(values, schema),
  }
  if (includeVariables) payload.variables = variableInputsFromForm(values)
  return { errors: {}, payload }
}

/** 变量行校验;新增变量对话框与创建表单共用同一套规则。 */
export function validateVariableRows(
  variables: VariableFormValues[],
  schema: DriverSchema | undefined,
  t: Translate,
): Record<string, string> {
  const errors: Record<string, string> = {}
  const setError = (path: string, message: string) => {
    if (!errors[path]) errors[path] = message
  }
  const types = new Map(variableTypes(schema).map((item) => [item.value, item]))
  const seen = new Set<string>()
  variables.forEach((variable, index) => {
    const path = `variables.${index}`
    const variableName = variable.name.trim()
    if (!variableName) {
      setError(`${path}.name`, t('gateway.form.validation.variableNameRequired'))
    } else if (!VARIABLE_NAME.test(variableName)) {
      setError(`${path}.name`, t('gateway.form.validation.variableNameFormat'))
    } else if (seen.has(variableName)) {
      setError(`${path}.name`, t('gateway.form.validation.variableNameDuplicate'))
    }
    seen.add(variableName)

    const type = types.get(variable.dataType)
    if (!typeExists(types, variable.dataType)) {
      setError(`${path}.dataType`, t('gateway.form.validation.variableTypeRequired'))
    }

    const address = variable.address.trim()
    if (!address) {
      setError(`${path}.address`, t('gateway.form.validation.variableAddressRequired'))
    } else if (type?.addressRegex && !safeRegExp(type.addressRegex)?.test(address)) {
      setError(`${path}.address`, t('gateway.form.validation.variableAddressInvalid'))
    }

    if (variable.bits.trim() !== '') {
      const bits = Number(variable.bits)
      if (!POSITIVE_INT.test(variable.bits) || bits < 0 || bits > 16) {
        setError(`${path}.bits`, t('gateway.form.validation.variableBitsInvalid'))
      }
    }
    if (variable.rate.trim() !== '') {
      const rate = Number(variable.rate)
      if (!Number.isFinite(rate) || rate <= 0) {
        setError(`${path}.rate`, t('gateway.form.validation.variableRateInvalid'))
      }
    }
  })
  return errors
}

function typeExists(types: Map<string, VariableDataType>, value: string): boolean {
  return types.size === 0 || types.has(value)
}

function safeRegExp(pattern: string): RegExp | undefined {
  try {
    return new RegExp(pattern)
  } catch {
    return undefined
  }
}
