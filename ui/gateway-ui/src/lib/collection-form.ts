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
  isCalculated?: boolean
  expression?: string
  precision?: string
  offset?: string
  value?: string
  isNew?: boolean
}

export interface CollectionFormValues {
  driverType: string
  driverModel: string
  name: string
  collectorCycleMs: string
  protocol: Record<string, string>
  variables: VariableFormValues[]
}

export interface ResolvedDriverField {
  labelZh: string
  labelEn: string
  regex: string
  placeholderZh?: string
  placeholderEn?: string
}

/** 按同组字段的当前值选择 variant,用于动态标签、占位符和校验规则。 */
export function resolveField(
  field: DriverField,
  values: Record<string, string>,
): ResolvedDriverField {
  const variant = field.variants?.find((item) => values[item.when.field] === item.when.equals)
  return {
    labelZh: variant?.labelZh ?? field.labelZh,
    labelEn: variant?.labelEn ?? field.labelEn,
    regex: variant?.regex ?? field.regex,
    placeholderZh: variant?.placeholderZh ?? field.placeholderZh,
    placeholderEn: variant?.placeholderEn ?? field.placeholderEn,
  }
}

const VARIABLE_NAME = /^[A-Za-z_][A-Za-z0-9_]*$/
const POSITIVE_INT = /^\d+$/

/** 常用协议字段的合理默认值;其余字段留空由用户填写。 */
function defaultProtocolValue(field: DriverField, model: string): string {
  const defaults: Record<string, string> = {
    baudRate: '9600',
    link: '0',
    dataBits: '8',
    parity: 'noParity',
    stopBits: '1',
    positionAddress: '0',
    concurrency: '1',
    rack: '0',
    slot: '1',
    pduSize: '480',
    connectionType: '1',
    localTsap: '0x1000',
    remoteTsap: '0x1000',
    type: '0',
    expires: '500',
    maxSize: '64',
    timeoutMs: '3000',
    amsPort: '851',
    localAmsPort: '851',
    intervalMs: '20',
    transport: 'tcp',
    maxReadWords: '30',
    port: model === 'opcUa' ? '4840'
      : model === 'omronCip' || model === 'ethernetIp' || model === 'EtherNet/IP-ControlLogic 55xx' || model === 'EtherNet/IP-MicroLogix 1400' ? '44818'
      : model === 'fins' ? '9600'
      : model === 'ethercat' ? '48898'
      : model === 'profinet' ? '102'
      : model === 'dnp3' ? '20000'
      : model === 'cjt188_2004' || model === 'cjt188_2018' ? '37121'
      : model === 'dlt645_2007' || model === 'dlt645_1997' ? '2010'
      : model === 'dlt698_45' ? '0'
      : model === 'ppi' ? '32502'
      : '102',
  }
  if (model === 'EtherNet/IP-ControlLogic 55xx' && field.name === 'slot') return '0'
  if (model === 'EtherNet/IP-ControlLogic 55xx' && field.name === 'interval') return '20'
  if (model === 'EtherNet/IP-MicroLogix 1400' && field.name === 'cipSize') return '500'
  if (model === 'EtherNet/IP-MicroLogix 1400' && field.name === 'logixArraySize') return '120'
  if (model === 'dnp3' && field.name === 'masterId') return '1'
  if (model === 'dnp3' && field.name === 'slaveId') return '2'
  if (model === 'dnp3' && field.name === 'class0123Interval') return '20000'
  if (model === 'dnp3' && field.name === 'class123Interval') return '1000'
  if (model === 'dnp3' && field.name === 'timeSync') return '0'
  if (model === 'mpi' && field.name === 'baudRate') return '38400'
  if (model === 'mpi' && field.name === 'parity') return 'oddParity'
  if (model === 'mpi' && field.name === 'station') return '2'
  if (model === 'ppi' && field.name === 'baudRate') return '9600'
  if (model === 'ppi' && field.name === 'parity') return 'evenParity'
  if (model === 'ppi' && field.name === 'station') return '2'
  if (model === 'KNXnet/IP' && field.name === 'discoveryIp') return '224.0.23.12'
  if (model === 'KNXnet/IP' && field.name === 'discoveryPort') return '3671'
  if (model === 'KND' && field.name === 'timeoutMs') return '5000'
  if (model === 'heidenhain' && field.name === 'port') return '19000'
  if (model === 'heidenhain' && field.name === 'timeoutMs') return '5000'
  if (model === 'fanuc focas' && field.name === 'port') return '8193'
  if (model === 'fanuc focas' && field.name === 'timeoutMs') return '3000'
  if (model === 'MTConnect' && field.name === 'port') return '5000'
  if (model === 'MTConnect' && field.name === 'timeoutMs') return '5000'
  if (model === 'Kuka' && field.name === 'port') return '54601'
  if (model === 'Kuka' && field.name === 'connectionMode') return '1'
  if (model === 'Kuka' && field.name === 'timeoutMs') return '3000'
  if (model === 'codesys-v3' && field.name === 'port') return '11740'
  if (model === 'codesys-v3' && field.name === 'timeoutMs') return '3000'
  if (model === 'ab5000' && field.name === 'port') return '44818'
  if (model === 'ab5000' && field.name === 'slot') return '0'
  if (model === 'ab5000' && field.name === 'compatibilityMode') return 'false'
  if (model === 'ab5000' && field.name === 'interval') return '20'
  if (model === 'DF1' && field.name === 'baudRate') return '9600'
  if (model === 'DF1' && field.name === 'parity') return 'evenParity'
  if (model === 'DF1' && field.name === 'timeoutMs') return '3000'
  if (model === 'DF1' && field.name === 'intervalMs') return '30'
  if (model === 'HJ212' && field.name === 'port') return '9000'
  if (model === 'HJ212' && field.name === 'timeoutMs') return '3000'
  if (model === 'HJ212' && field.name === 'intervalMs') return '30'
  if (model === 'COMLI' && field.name === 'baud') return '9600'
  if (model === 'COMLI' && field.name === 'data') return '8'
  if (model === 'COMLI' && field.name === 'parity') return '0'
  if (model === 'COMLI' && field.name === 'stop') return '1'
  if (model === 'COMLI' && field.name === 'timeout') return '3000'
  if (model === 'COMLI' && field.name === 'interval') return '20'
  if (model === 'COMLI' && field.name === 'retry') return '0'
  if ((model === 'MC-1E' || model === 'MC-3E' || model === 'MC-4E')) {
    if (field.name === 'transport_mode') return '0'
    if (field.name === 'port') return '2000'
    if (field.name === 'src-port') return '52001'
    if (field.name === 'timeout') return '15000'
    if (field.name === 'interval') return '20'
    if (field.name === 'retry') return '0'
  }
  if (model === 'MC-FX') {
    if (field.name === 'baud') return '9600'
    if (field.name === 'data') return '7'
    if (field.name === 'parity') return '2'
    if (field.name === 'stop') return '1'
    if (field.name === 'timeout') return '3000'
    if (field.name === 'interval') return '20'
    if (field.name === 'retry') return '0'
  }
  if (model === 'HJ212' && field.name === 'ST') return '32'
  if (model === 'BACnet' && field.name === 'port') return '47808'
  if (model === 'BACnet' && field.name === 'srcPort') return '0'
  if (model === 'BACnet' && field.name === 'bbmd') return '0'
  if (model === 'BACnet' && field.name === 'timeoutMs') return '3000'
  if (model === 'BACnet' && field.name === 'intervalMs') return '20'
  if (model === 'Beckhoff ADS' && field.name === 'port') return '48898'
  if (model === 'Beckhoff ADS' && field.name === 'srcAmsNetId') return '127.0.0.1.1.1'
  if (model === 'Beckhoff ADS' && field.name === 'srcAdsPort') return '851'
  if (model === 'Beckhoff ADS' && field.name === 'dstAdsPort') return '851'
  if (model === 'Beckhoff ADS' && field.name === 'timeoutMs') return '5000'
  if (model === 'Beckhoff ADS' && field.name === 'intervalMs') return '20'
  if (model === 'SRTP' && field.name === 'port') return '18245'
  if (model === 'SRTP' && field.name === 'timeoutMs') return '3000'
  if (model === 'SRTP' && field.name === 'intervalMs') return '20'
  if (model === 'SNMP-V2' && field.name === 'port') return '161'
  if (model === 'SNMP-V2' && field.name === 'localPort') return '0'
  if (model === 'SNMP-V2' && field.name === 'timeoutMs') return '3000'
  if (model === 'SNMP-V2' && field.name === 'intervalMs') return '20'
  if (model === 'IEC61850' && field.name === 'port') return '102'
  if (model === 'IEC61850' && field.name === 'defaultFC') return 'ST'
  if (model === 'IEC61850' && field.name === 'timeoutMs') return '5000'
  if (model === 'IEC61850' && field.name === 'intervalMs') return '20'
  if ((model === 'cjt188_2004' || model === 'cjt188_2018' || model === 'dlt698_45') && field.name === 'baudRate') return '2400'
  if ((model === 'cjt188_2004' || model === 'cjt188_2018' || model === 'dlt645_2007' || model === 'dlt645_1997' || model === 'dlt698_45') && field.name === 'parity') return 'evenParity'
  if ((model === 'cjt188_2004' || model === 'cjt188_2018' || model === 'dlt645_2007' || model === 'dlt645_1997' || model === 'dlt698_45') && field.name === 'intervalMs') return '30'
  if (defaults[field.name] !== undefined) return defaults[field.name]
  if (model === 'dlt698_45' && field.name === 'clientAddress') return '0'
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
    isCalculated: false,
    expression: '',
    precision: '',
    offset: '',
  }
}

export function createEmptyCalculatedVariable(
  schema: DriverSchema | undefined,
): VariableFormValues {
  return {
    ...createEmptyVariable(schema),
    address: '',
    accessMode: 'r',
    isCalculated: true,
  }
}

export function isBlankVariable(variable: VariableFormValues): boolean {
  return (
    !variable.name.trim() &&
    !variable.address.trim() &&
    !(variable.bits ?? '').trim() &&
    !(variable.expression ?? '').trim() &&
    !(variable.precision ?? '').trim() &&
    !(variable.rate ?? '').trim() &&
    !(variable.offset ?? '').trim()
  )
}

export const ENDPOINT_FIELD = '__endpoint'

export interface EndpointPlan {
  kind: 'host-port' | 'location-port'
  serial: boolean
  hostKey?: string
  locationKey: string
  portKey: string
  linkKey?: string
  field: DriverField
  defaultPort: string
}

export function buildEndpointPlan(
  schema: DriverSchema | undefined,
  model: string,
  values: Record<string, string>,
): EndpointPlan | null {
  if (!schema) return null
  const fields = new Map(schema.fields.map((field) => [field.name, field]))
  const port = fields.get('port')
  const host = fields.get('host')
  const location = fields.get('location')
  const link = fields.get('link')
  if (!port || (!host && !location)) return null
  const base = host ?? location
  if (!base) return null
  const serial = Boolean(link && (values[link.name] ?? '0') === '0')
  const locationKey = host ? host.name : location!.name
  const portHint = (port.placeholderZh ?? '').trim()
  const defaultPort = /^\d+$/.test(portHint)
    ? portHint
    : defaultProtocolValue(port, model) || port.placeholderZh || 'PORT'
  const basePlaceholderZh = base.placeholderZh ?? ''
  const basePlaceholderEn = base.placeholderEn ?? ''
  const endpointPlaceholder = (zh: boolean) => {
    if (serial) {
      const fallback = zh ? '/dev/ttyUSB0' : '/dev/ttyUSB0'
      return (zh ? location?.placeholderZh : location?.placeholderEn) ?? fallback
    }
    const source = zh ? basePlaceholderZh : basePlaceholderEn
    if (!source) return `IP:${defaultPort}`
    if (source.includes('://')) {
      return source.includes(`:${defaultPort}`) ? source : `${source.replace(/\/$/, '')}:${defaultPort}`
    }
    return source.includes(':') ? source : `${source}:${defaultPort}`
  }
  return {
    kind: host ? 'host-port' : 'location-port',
    serial,
    hostKey: host?.name,
    locationKey,
    portKey: port.name,
    linkKey: link?.name,
    defaultPort,
    field: {
      ...base,
      name: ENDPOINT_FIELD,
      labelZh: serial ? '串口设备' : '目标地址',
      labelEn: serial ? 'Serial Port' : 'Target Address',
      regex: '^.+$',
      required: true,
      placeholderZh: endpointPlaceholder(true),
      placeholderEn: endpointPlaceholder(false),
    },
  }
}

export function endpointValue(
  plan: EndpointPlan | null,
  values: Record<string, string>,
): string {
  if (!plan) return ''
  if (plan.serial) return values[plan.locationKey] ?? ''
  const location = (values[plan.locationKey] ?? '').trim()
  if (!location) return ''
  const port = (values[plan.portKey] ?? '').trim() || plan.defaultPort
  if (location.includes('://')) {
    return location.includes(`:${port}`) ? location : `${location.replace(/\/$/, '')}:${port}`
  }
  return `${location}:${port}`
}

export function splitEndpoint(
  plan: EndpointPlan | null,
  value: string,
  values: Record<string, string>,
): Record<string, string> {
  if (!plan) return { [ENDPOINT_FIELD]: value }
  const next = value.trim()
  if (plan.serial || !next) return { [plan.locationKey]: next }
  let location = next
  let port = (values[plan.portKey] ?? '').trim() || plan.defaultPort
  if (next.includes('://')) {
    const match = next.match(/:(\d+)\/?$/)
    if (match?.[1]) port = match[1]
    return { [plan.locationKey]: next, [plan.portKey]: port }
  }
  const separator = next.lastIndexOf(':')
  if (separator > 0 && /^\d+$/.test(next.slice(separator + 1))) {
    location = next.slice(0, separator)
    port = next.slice(separator + 1)
  }
  return { [plan.locationKey]: location, [plan.portKey]: port }
}

export function protocolDisplayFields(
  schema: DriverSchema | undefined,
  model: string,
  values: Record<string, string>,
): DriverField[] {
  const plan = buildEndpointPlan(schema, model, values)
  const fields = sortedFields(schema)
  if (!plan) return fields
  return fields
    .filter((field) => field.name !== plan.portKey && field.name !== plan.locationKey)
    .concat(plan.field)
    .sort((a, b) => a.sort - b.sort)
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
  const endpointPlan = buildEndpointPlan(schema, driverModel, protocol)
  if (endpointPlan) protocol[ENDPOINT_FIELD] = endpointValue(endpointPlan, protocol)
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
  if (
    !protocol.location &&
    protocol.host &&
    schema?.fields.some((field) => field.name === 'location') &&
    !schema.fields.some((field) => field.name === 'port')
  ) {
    protocol.location = `${protocol.host}:${protocol.port || '8000'}`
  }
  const endpointPlan = buildEndpointPlan(schema, detail.driverModel, protocol)
  if (endpointPlan) protocol[ENDPOINT_FIELD] = endpointValue(endpointPlan, protocol)
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
  for (const [key, raw] of Object.entries(values.protocol)) {
    if (!names.has(key) || raw.trim() === REDACTED_VALUE) continue
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
  return variables.filter((variable) => !isBlankVariable(variable)).map((variable) => {
    const calculated = Boolean(variable.isCalculated)
    const input: CollectionVariableInput = {
      name: variable.name.trim(),
      dataType: variable.dataType.trim(),
      address: calculated ? '' : variable.address.trim(),
      accessMode: calculated ? 'r' : variable.accessMode,
    }
    if (calculated) {
      input.isCalculated = true
      input.expression = (variable.expression ?? '').trim()
    } else {
      const bits = (variable.bits ?? '').trim() === '' ? undefined : Number(variable.bits)
      if (bits !== undefined && Number.isFinite(bits)) input.bits = bits
    }
    const rate = (variable.rate ?? '').trim() === '' ? undefined : Number(variable.rate)
    if (rate !== undefined && Number.isFinite(rate)) input.rate = rate
    const offset = (variable.offset ?? '').trim() === '' ? undefined : Number(variable.offset)
    if (offset !== undefined && Number.isFinite(offset)) input.offset = offset
    const precision =
      (variable.precision ?? '').trim() === '' ? undefined : Number(variable.precision)
    if (precision !== undefined && Number.isFinite(precision)) input.precision = precision
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

  const endpointPlan = buildEndpointPlan(schema, values.driverModel, values.protocol)
  for (const field of sortedFields(schema)) {
    if (
      endpointPlan &&
      (field.name === endpointPlan.portKey || field.name === endpointPlan.locationKey)
    ) {
      continue
    }
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
    const pattern = resolveField(field, values.protocol).regex
    if (pattern && !safeRegExp(pattern)?.test(value)) {
      setError(path, t('gateway.form.validation.protocolInvalid'))
    }
  }
  if (endpointPlan) {
    const value = endpointValue(endpointPlan, values.protocol).trim()
    if (!value && endpointPlan.field.required) {
      setError(`protocol.${ENDPOINT_FIELD}`, t('gateway.form.validation.protocolRequired'))
    } else if (value && !safeRegExp(endpointPlan.field.regex)?.test(value)) {
      setError(`protocol.${ENDPOINT_FIELD}`, t('gateway.form.validation.protocolInvalid'))
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

    if (variable.isCalculated) {
      if (!(variable.expression ?? '').trim()) {
        setError(`${path}.expression`, t('gateway.form.validation.variableExpressionRequired'))
      }
    } else {
      const address = variable.address.trim()
      if (!address) {
        setError(`${path}.address`, t('gateway.form.validation.variableAddressRequired'))
      } else if (type?.addressRegex && !safeRegExp(type.addressRegex)?.test(address)) {
        setError(`${path}.address`, t('gateway.form.validation.variableAddressInvalid'))
      }
      if ((variable.bits ?? '').trim() !== '') {
        const bits = Number(variable.bits)
        if (!POSITIVE_INT.test(variable.bits ?? '') || bits < 0 || bits > 16) {
          setError(`${path}.bits`, t('gateway.form.validation.variableBitsInvalid'))
        }
      }
    }

    if ((variable.rate ?? '').trim() !== '') {
      const rate = Number(variable.rate)
      if (!Number.isFinite(rate) || rate <= 0) {
        setError(`${path}.rate`, t('gateway.form.validation.variableRateInvalid'))
      }
    }
    if ((variable.precision ?? '').trim() !== '') {
      const precision = Number(variable.precision)
      if (!POSITIVE_INT.test(variable.precision ?? '') || precision < 0) {
        setError(`${path}.precision`, t('gateway.form.validation.variablePrecisionInvalid'))
      }
    }
    if ((variable.offset ?? '').trim() !== '') {
      const offset = Number(variable.offset)
      if (!Number.isFinite(offset)) {
        setError(`${path}.offset`, t('gateway.form.validation.variableOffsetInvalid'))
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
