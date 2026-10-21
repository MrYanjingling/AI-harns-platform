import { z } from 'zod'
import type { Translate } from '@harns/core'
import type {
  AccessMode,
  DeviceModel,
  DevicePayload,
  DeviceType,
  GatewayDeviceDetail,
  GatewayVariable,
} from '@/api/types'
import {
  DEVICE_TYPES,
  MODBUS_DATA_TYPES,
  OPCUA_DATA_TYPES,
  S7_DATA_TYPES,
  usesSerial,
} from './constants'

export interface VariableFormValues {
  name: string
  dataType: string
  address: string
  bits: string
  functionCode: string
  rate: string
  amount: string
  namespace: string
  defaultValue: string
  accessMode: AccessMode
}

export interface DeviceFormValues {
  deviceType: DeviceType
  deviceModel: string
  name: string
  deviceCode: string
  collectorCycle: string
  variableInterval: string
  topic: string
  location: string
  port: string
  baudRate: string
  dataBits: string
  parity: string
  stopBits: string
  slave: string
  memoryLayout: string
  positionAddress: string
  username: string
  password: string
  rack: string
  slot: string
  variables: VariableFormValues[]
}

const NUMERIC = /^\d+$/
const DECIMAL = /^\d+(\.\d+)?$/

export function createEmptyVariable(deviceType: DeviceType): VariableFormValues {
  return {
    name: '',
    dataType: deviceType === 'opcUa' ? 'string' : 'int16',
    address: deviceType === 's7' ? '' : '0',
    bits: '',
    functionCode: '3',
    rate: '',
    amount: '',
    namespace: '1',
    defaultValue: '',
    accessMode: 'r',
  }
}

export function createDefaultFormValues(): DeviceFormValues {
  return {
    deviceType: 'modbus',
    deviceModel: 'modbusTcp',
    name: '',
    deviceCode: '',
    collectorCycle: '1',
    variableInterval: '',
    topic: '',
    location: '127.0.0.1',
    port: '502',
    baudRate: '9600',
    dataBits: '8',
    parity: 'noParity',
    stopBits: '1',
    slave: '1',
    memoryLayout: 'ABCD',
    positionAddress: '',
    username: '',
    password: '',
    rack: '0',
    slot: '1',
    variables: [createEmptyVariable('modbus')],
  }
}

export function detailToForm(device: GatewayDeviceDetail): DeviceFormValues {
  const deviceType = isDeviceType(device.deviceType) ? device.deviceType : 'modbus'
  const option = device.address?.option ?? {}
  return {
    deviceType,
    deviceModel: device.deviceModel || DEVICE_TYPES[0].models[0].value,
    name: device.name,
    deviceCode: device.deviceCode ?? '',
    collectorCycle: String(device.collectorCycle ?? 1),
    variableInterval:
      device.variableInterval === undefined ? '' : String(device.variableInterval),
    topic: device.topic ?? '',
    location: device.address?.location ?? '',
    port: option.port === undefined ? '502' : String(option.port),
    baudRate: option.baudRate === undefined ? '9600' : String(option.baudRate),
    dataBits: option.dataBits === undefined ? '8' : String(option.dataBits),
    parity: option.parity ?? 'noParity',
    stopBits: option.stopBits ?? '1',
    slave: device.slave === undefined ? '1' : String(device.slave),
    memoryLayout: device.memoryLayout ?? 'ABCD',
    positionAddress:
      device.positionAddress === undefined ? '' : String(device.positionAddress),
    username: option.username ?? '',
    password: option.password ?? '',
    rack: option.rack === undefined ? '0' : String(option.rack),
    slot: option.slot === undefined ? '1' : String(option.slot),
    variables:
      device.variables && device.variables.length > 0
        ? device.variables.map(toVariableForm)
        : [createEmptyVariable(deviceType)],
  }
}

function toVariableForm(variable: GatewayVariable): VariableFormValues {
  return {
    name: variable.name,
    dataType: variable.dataType,
    address: String(variable.address ?? ''),
    bits: variable.bits === undefined ? '' : String(variable.bits),
    functionCode: variable.functionCode === undefined ? '3' : String(variable.functionCode),
    rate: variable.rate === undefined ? '' : String(variable.rate),
    amount: variable.amount === undefined ? '' : String(variable.amount),
    namespace: variable.namespace === undefined ? '1' : String(variable.namespace),
    defaultValue: stringifyDefault(variable.defaultValue),
    accessMode: variable.accessMode,
  }
}

function stringifyDefault(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

export function isDeviceType(value: string): value is DeviceType {
  return DEVICE_TYPES.some((item) => item.value === value)
}

function createSchema(t: Translate) {
  const required = (message: string) => z.string().trim().min(1, message)

  return z
    .object({
      deviceType: z.enum(['modbus', 'opcUa', 's7']),
      deviceModel: z.string().min(1),
      name: required(t('gateway.form.validation.nameRequired')).max(
        64,
        t('gateway.form.validation.nameMax'),
      ),
      deviceCode: required(t('gateway.form.validation.deviceCodeRequired')).max(
        32,
        t('gateway.form.validation.deviceCodeMax'),
      ),
      collectorCycle: required(t('gateway.form.validation.collectorCycleRequired')),
      variableInterval: z.string(),
      topic: z.string().max(256, t('gateway.form.validation.topicMax')),
      location: z.string(),
      port: z.string(),
      baudRate: z.string(),
      dataBits: z.string(),
      parity: z.string(),
      stopBits: z.string(),
      slave: z.string(),
      memoryLayout: z.string(),
      positionAddress: z.string(),
      username: z.string(),
      password: z.string(),
      rack: z.string(),
      slot: z.string(),
      variables: z.array(
        z.object({
          name: z.string(),
          dataType: z.string(),
          address: z.string(),
          bits: z.string(),
          functionCode: z.string(),
          rate: z.string(),
          amount: z.string(),
          namespace: z.string(),
          defaultValue: z.string(),
          accessMode: z.enum(['r', 'rw']),
        }),
      ),
    })
    .superRefine((value, ctx) => {
      const issue = (path: (string | number)[], message: string) => {
        ctx.addIssue({ code: z.ZodIssueCode.custom, path, message })
      }

      if (!NUMERIC.test(value.collectorCycle)) {
        issue(['collectorCycle'], t('gateway.form.validation.collectorCycleRequired'))
      } else {
        const cycle = Number(value.collectorCycle)
        if (cycle < 1 || cycle > 86400) {
          issue(['collectorCycle'], t('gateway.form.validation.collectorCycleRange'))
        }
      }

      if (value.variableInterval && !NUMERIC.test(value.variableInterval)) {
        issue(['variableInterval'], t('gateway.form.validation.numberInvalid'))
      }

      if (!value.location.trim()) {
        issue(['location'], t('gateway.form.validation.locationRequired'))
      }

      // 后端 binding 禁止名称包含斜杠,提前拦截避免 400。
      if (/[\\/]/.test(value.name)) {
        issue(['name'], t('gateway.form.validation.invalidChars'))
      }
      if (/[\\/]/.test(value.deviceCode)) {
        issue(['deviceCode'], t('gateway.form.validation.invalidChars'))
      }

      if (value.deviceType === 'modbus') {
        if (!['modbusTcp', 'modbusRtu', 'modbusRtuOverTcp'].includes(value.deviceModel)) {
          issue(['deviceModel'], t('gateway.form.validation.numberInvalid'))
        }
        if (!usesSerial(value.deviceModel) && !inRange(value.port, 1, 65535)) {
          issue(['port'], t('gateway.form.validation.portRange'))
        }
        if (!inRange(value.slave, 1, 247)) {
          issue(['slave'], t('gateway.form.validation.slaveRange'))
        }
        if (value.positionAddress && !NUMERIC.test(value.positionAddress)) {
          issue(['positionAddress'], t('gateway.form.validation.numberInvalid'))
        }
      }

      if (value.deviceType === 'opcUa') {
        if (value.deviceModel !== 'opcUa') {
          issue(['deviceModel'], t('gateway.form.validation.numberInvalid'))
        }
        if (!inRange(value.port, 1, 65535)) {
          issue(['port'], t('gateway.form.validation.portRange'))
        }
      }

      if (value.deviceType === 's7') {
        if (value.deviceModel !== 's71500') {
          issue(['deviceModel'], t('gateway.form.validation.numberInvalid'))
        }
        if (!inRange(value.port, 1, 65535)) {
          issue(['port'], t('gateway.form.validation.portRange'))
        }
        if (value.rack && !inRange(value.rack, 0, 255)) {
          issue(['rack'], t('gateway.form.validation.numberInvalid'))
        }
        if (value.slot && !inRange(value.slot, 0, 255)) {
          issue(['slot'], t('gateway.form.validation.numberInvalid'))
        }
      }

      if (value.variables.length === 0) {
        issue(['variables'], t('gateway.form.validation.variablesRequired'))
      }

      const seen = new Set<string>()
      value.variables.forEach((variable, index) => {
        const path = ['variables', index]
        const name = variable.name.trim()
        if (!name) {
          issue([...path, 'name'], t('gateway.form.validation.variableNameRequired'))
        } else if (seen.has(name)) {
          issue([...path, 'name'], t('gateway.form.validation.variableNameDuplicate'))
        } else {
          seen.add(name)
          if (name.length > 64) {
            issue([...path, 'name'], t('gateway.form.validation.variableNameMax'))
          }
          if (/[\\/]/.test(name)) {
            issue([...path, 'name'], t('gateway.form.validation.invalidChars'))
          }
        }

        const allowedTypes = dataTypesForType(value.deviceType)
        if (!allowedTypes.includes(variable.dataType)) {
          issue([...path, 'dataType'], t('gateway.form.validation.numberInvalid'))
        }

        if (value.deviceType === 'modbus') {
          if (!NUMERIC.test(variable.address)) {
            issue([...path, 'address'], t('gateway.form.validation.addressNumber'))
          }
          if (!inRange(variable.functionCode, 1, 4)) {
            issue([...path, 'functionCode'], t('gateway.form.validation.functionCodeRange'))
          }
          if (variable.bits && !inRange(variable.bits, 0, 7)) {
            issue([...path, 'bits'], t('gateway.form.validation.numberInvalid'))
          }
          if (variable.amount && !inRange(variable.amount, 1, 65535)) {
            issue([...path, 'amount'], t('gateway.form.validation.numberInvalid'))
          }
          if (variable.rate && !DECIMAL.test(variable.rate)) {
            issue([...path, 'rate'], t('gateway.form.validation.numberInvalid'))
          }
        }

        if (value.deviceType === 'opcUa') {
          if (!variable.address.trim()) {
            issue([...path, 'address'], t('gateway.form.validation.addressRequired'))
          }
          if (!inRange(variable.namespace, 1, 65535)) {
            issue([...path, 'namespace'], t('gateway.form.validation.namespaceRange'))
          }
        }

        if (value.deviceType === 's7') {
          if (!variable.address.trim()) {
            issue([...path, 'address'], t('gateway.form.validation.addressRequired'))
          }
          if (variable.rate && !DECIMAL.test(variable.rate)) {
            issue([...path, 'rate'], t('gateway.form.validation.numberInvalid'))
          }
        }
      })
    })
}

function inRange(raw: string, min: number, max: number): boolean {
  if (!NUMERIC.test(raw)) return false
  const value = Number(raw)
  return value >= min && value <= max
}

function dataTypesForType(deviceType: DeviceType): readonly string[] {
  if (deviceType === 'opcUa') return OPCUA_DATA_TYPES
  if (deviceType === 's7') return S7_DATA_TYPES
  return MODBUS_DATA_TYPES
}

export interface DeviceFormValidation {
  errors: Record<string, string>
  payload?: DevicePayload
}

/** 动态协议字段较多,使用受控状态 + zod 校验,错误路径与后端字段对齐。 */
export function validateDeviceForm(
  values: DeviceFormValues,
  t: Translate,
): DeviceFormValidation {
  const result = createSchema(t).safeParse(values)
  if (!result.success) {
    const errors: Record<string, string> = {}
    for (const issue of result.error.issues) {
      const path = issue.path.join('.')
      if (!errors[path]) errors[path] = issue.message
    }
    return { errors }
  }
  return { errors: {}, payload: formToPayload(values) }
}

function optionalNumber(raw: string): number | undefined {
  return raw.trim() === '' ? undefined : Number(raw)
}

function parseDefaultValue(raw: string): unknown {
  const value = raw.trim()
  if (!value) return undefined
  if (value === 'true') return true
  if (value === 'false') return false
  if (DECIMAL.test(value) || /^-?\d+(\.\d+)?$/.test(value)) return Number(value)
  return value
}

export function formToPayload(values: DeviceFormValues): DevicePayload {
  const payload: DevicePayload = {
    name: values.name.trim(),
    deviceCode: values.deviceCode.trim(),
    deviceType: values.deviceType,
    deviceModel: values.deviceModel as DeviceModel,
    collectorCycle: Number(values.collectorCycle),
    address: { location: values.location.trim(), option: {} },
    variables: [],
  }

  const variableInterval = optionalNumber(values.variableInterval)
  if (variableInterval !== undefined && variableInterval > 0) {
    payload.variableInterval = variableInterval
  }
  const topic = values.topic.trim()
  if (topic) payload.topic = topic

  if (values.deviceType === 'modbus') {
    payload.slave = Number(values.slave)
    payload.memoryLayout = values.memoryLayout
    const positionAddress = optionalNumber(values.positionAddress)
    if (positionAddress !== undefined) payload.positionAddress = positionAddress
    if (usesSerial(values.deviceModel)) {
      payload.address.option = {
        baudRate: Number(values.baudRate),
        dataBits: Number(values.dataBits),
        parity: values.parity,
        stopBits: values.stopBits,
      }
    } else {
      payload.address.option = { port: Number(values.port) }
    }
    payload.variables = values.variables.map((variable) => {
      const result: GatewayVariable = {
        name: variable.name.trim(),
        dataType: variable.dataType,
        address: Number(variable.address),
        functionCode: Number(variable.functionCode),
        accessMode: variable.accessMode,
      }
      const bits = optionalNumber(variable.bits)
      if (bits !== undefined) result.bits = bits
      const rate = optionalNumber(variable.rate)
      if (rate !== undefined) result.rate = rate
      const amount = optionalNumber(variable.amount)
      if (amount !== undefined) result.amount = amount
      const defaultValue = parseDefaultValue(variable.defaultValue)
      if (defaultValue !== undefined) result.defaultValue = defaultValue
      return result
    })
    return payload
  }

  if (values.deviceType === 'opcUa') {
    payload.address.option = { port: Number(values.port) }
    if (values.username.trim()) payload.address.option.username = values.username.trim()
    if (values.password) payload.address.option.password = values.password
    payload.variables = values.variables.map((variable) => {
      const result: GatewayVariable = {
        name: variable.name.trim(),
        dataType: variable.dataType,
        address: variable.address.trim(),
        namespace: Number(variable.namespace),
        accessMode: variable.accessMode,
      }
      const defaultValue = parseDefaultValue(variable.defaultValue)
      if (defaultValue !== undefined) result.defaultValue = defaultValue
      return result
    })
    return payload
  }

  payload.address.option = {
    port: Number(values.port),
    rack: Number(values.rack || 0),
    slot: Number(values.slot || 0),
  }
  payload.variables = values.variables.map((variable) => {
    const result: GatewayVariable = {
      name: variable.name.trim(),
      dataType: variable.dataType,
      address: variable.address.trim(),
      accessMode: variable.accessMode,
    }
    const rate = optionalNumber(variable.rate)
    if (rate !== undefined) result.rate = rate
    const defaultValue = parseDefaultValue(variable.defaultValue)
    if (defaultValue !== undefined) result.defaultValue = defaultValue
    return result
  })
  return payload
}
