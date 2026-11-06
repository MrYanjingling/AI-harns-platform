import { z } from 'zod'
import type { Translate } from '@/i18n'
import type {
  AccessMode,
  DataType,
  DeviceDetail,
  DevicePayload,
  DeviceType,
  MemoryLayout,
  ModbusDevice,
  OpcUaDevice,
  S7Device,
} from '@/api/types'
import { PROTOCOLS } from '@/lib/constants'

/**
 * 表单采用扁平模型(而非按 protocol 的判别联合):
 * 三种协议共享同一组 RHF 字段,协议差异在提交时映射。
 * 条件校验集中在 superRefine,便于按 deviceType 给出精确到行的错误。
 */

export const DATA_TYPE_VALUES = [
  'bool',
  'int16',
  'uint16',
  'int32',
  'int64',
  'float32',
  'float64',
  'number',
  'string',
] as const

const MODBUS_TYPES: DataType[] = ['bool', 'int16', 'uint16', 'int32', 'int64', 'float32', 'float64']
const S7_TYPES: DataType[] = ['bool', 'int16', 'uint16', 'int32', 'int64', 'float32', 'float64']
const OPCUA_TYPES: DataType[] = ['number', 'string']

/** 校验消息跟随当前语言,因此在表单内用 useMemo 按 t 构建。 */
export function createDeviceFormSchema(t: Translate) {
  const variableSchema = z.object({
    name: z
      .string()
      .trim()
      .min(1, t('devices.validation.variableNameRequired'))
      .max(64, t('devices.validation.max64'))
      .regex(/^[^/\\]+$/, t('devices.validation.noSlash')),
    dataType: z.enum(DATA_TYPE_VALUES),
    address: z.string().trim().min(1, t('devices.validation.addressRequired')),
    bits: z.number({ invalid_type_error: t('devices.validation.number') }).int().min(0).max(7),
    functionCode: z
      .number({ invalid_type_error: t('devices.validation.number') })
      .int()
      .min(1)
      .max(4),
    rate: z
      .number({ invalid_type_error: t('devices.validation.number') })
      .min(-1000000)
      .max(1000000),
    amount: z
      .number({ invalid_type_error: t('devices.validation.number') })
      .int()
      .min(0)
      .max(65535),
    namespace: z
      .number({ invalid_type_error: t('devices.validation.number') })
      .int()
      .min(0)
      .max(65535),
    defaultValue: z.string(),
    accessMode: z.enum(['r', 'rw']),
  })

  const numberField = (message = t('devices.validation.number')) =>
    z.number({ invalid_type_error: message }).int(message)

  return z
    .object({
      name: z
        .string()
        .trim()
        .min(1, t('devices.validation.deviceNameRequired'))
        .max(64, t('devices.validation.max64'))
        .regex(/^[\p{L}_0-9_. ]+$/u, t('devices.validation.nameCharset')),
      deviceCode: z
        .string()
        .trim()
        .min(1, t('devices.validation.deviceCodeRequired'))
        .max(32, t('devices.validation.max32'))
        .regex(/^[^/\\]+$/, t('devices.validation.noSlash')),
      deviceType: z.enum(['modbus', 'opcUa', 's7']),
      deviceModel: z.string().min(1, t('devices.validation.modelRequired')),
      collectorCycle: numberField()
        .min(1, t('devices.validation.collectorMin'))
        .max(86400, t('devices.validation.max86400')),
      variableInterval: numberField().min(0).max(86400, t('devices.validation.max86400')),
      topic: z.string().trim().max(256, t('devices.validation.max256')),

      // Modbus
      modbusLocation: z.string().trim(),
      modbusPort: numberField()
        .min(1, t('devices.validation.portRange'))
        .max(65535, t('devices.validation.portRange')),
      modbusBaudRate: numberField(),
      modbusDataBits: numberField(),
      modbusParity: z.string(),
      modbusStopBits: z.string(),
      slave: numberField()
        .min(1, t('devices.validation.slaveMin'))
        .max(65535, t('devices.validation.slaveRange')),
      memoryLayout: z.enum(['ABCD', 'BADC', 'CDAB', 'DCBA']),
      positionAddress: numberField().min(0).max(65535),

      // OPC UA
      opcLocation: z.string().trim(),
      opcPort: numberField()
        .min(1, t('devices.validation.portRange'))
        .max(65535, t('devices.validation.portRange')),
      opcUsername: z.string().trim(),
      opcPassword: z.string(),

      // S7
      s7Location: z.string().trim(),
      s7Port: numberField()
        .min(1, t('devices.validation.portRange'))
        .max(65535, t('devices.validation.portRange')),
      s7Rack: numberField().min(0).max(7),
      s7Slot: numberField().min(0).max(31),

      variables: z.array(variableSchema).min(1, t('devices.validation.atLeastOneVariable')),
    })
    .superRefine((values, ctx) => {
      const type = values.deviceType

      if (type === 'modbus') {
        if (!values.modbusLocation) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['modbusLocation'],
            message: t('devices.validation.modbusLocationRequired'),
          })
        }
        if (values.deviceModel === 'modbusRtu') {
          if (
            ![1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200].includes(
              values.modbusBaudRate,
            )
          ) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['modbusBaudRate'],
              message: t('devices.validation.unsupportedBaud'),
            })
          }
          if (![5, 6, 7, 8].includes(values.modbusDataBits)) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['modbusDataBits'],
              message: t('devices.validation.dataBitsRange'),
            })
          }
        }
        values.variables.forEach((variable, index) => {
          if (!MODBUS_TYPES.includes(variable.dataType)) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['variables', index, 'dataType'],
              message: t('devices.validation.modbusTypeUnsupported'),
            })
          }
          if (!/^\d+$/.test(variable.address)) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['variables', index, 'address'],
              message: t('devices.validation.registerAddress'),
            })
          }
        })
      }

      if (type === 'opcUa') {
        if (!values.opcLocation) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['opcLocation'],
            message: t('devices.validation.endpointRequired'),
          })
        }
        values.variables.forEach((variable, index) => {
          if (!OPCUA_TYPES.includes(variable.dataType)) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['variables', index, 'dataType'],
              message: t('devices.validation.opcTypeUnsupported'),
            })
          }
          // 后端 binding:"required" 会拒绝 namespace = 0。
          if (variable.namespace < 1) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['variables', index, 'namespace'],
              message: t('devices.validation.namespacePositive'),
            })
          }
        })
      }

      if (type === 's7') {
        if (!values.s7Location) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['s7Location'],
            message: t('devices.validation.s7LocationRequired'),
          })
        }
        values.variables.forEach((variable, index) => {
          if (!S7_TYPES.includes(variable.dataType)) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['variables', index, 'dataType'],
              message: t('devices.validation.s7TypeUnsupported'),
            })
          }
        })
      }
    })
}

export type DeviceFormSchema = ReturnType<typeof createDeviceFormSchema>
export type DeviceFormValues = z.infer<DeviceFormSchema>
export type VariableFormValues = DeviceFormValues['variables'][number]

export function emptyVariable(type: DeviceType): VariableFormValues {
  return {
    name: '',
    dataType: type === 'opcUa' ? 'number' : 'int16',
    address: type === 'modbus' ? '0' : '',
    bits: 0,
    functionCode: 3,
    rate: 1,
    amount: 0,
    namespace: 2,
    defaultValue: '',
    accessMode: 'r',
  }
}

export function defaultFormValues(type: DeviceType): DeviceFormValues {
  return {
    name: '',
    deviceCode: '',
    deviceType: type,
    deviceModel: PROTOCOLS[type].defaultModel,
    collectorCycle: 1,
    variableInterval: 0,
    topic: '',
    modbusLocation: '',
    modbusPort: 502,
    modbusBaudRate: 9600,
    modbusDataBits: 8,
    modbusParity: 'noParity',
    modbusStopBits: '1',
    slave: 1,
    memoryLayout: 'ABCD',
    positionAddress: 0,
    opcLocation: 'opc.tcp://',
    opcPort: 4840,
    opcUsername: '',
    opcPassword: '',
    s7Location: '',
    s7Port: 102,
    s7Rack: 0,
    s7Slot: 1,
    variables: [emptyVariable(type)],
  }
}

function stringValue(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

export function toFormValues(device: DeviceDetail): DeviceFormValues {
  const base = defaultFormValues(device.deviceType)
  const common: Partial<DeviceFormValues> = {
    name: device.name,
    deviceCode: device.deviceCode,
    deviceType: device.deviceType,
    deviceModel: device.deviceModel,
    collectorCycle: device.collectorCycle || 1,
    variableInterval: device.variableInterval || 0,
    topic: device.topic ?? '',
  }

  if (device.deviceType === 'modbus') {
    const modbus = device as ModbusDevice
    return {
      ...base,
      ...common,
      modbusLocation: modbus.address?.location ?? '',
      modbusPort: modbus.address?.option?.port ?? 502,
      modbusBaudRate: modbus.address?.option?.baudRate ?? 9600,
      modbusDataBits: modbus.address?.option?.dataBits ?? 8,
      modbusParity: modbus.address?.option?.parity ?? 'noParity',
      modbusStopBits: modbus.address?.option?.stopBits ?? '1',
      slave: modbus.slave || 1,
      memoryLayout: modbus.memoryLayout ?? 'ABCD',
      positionAddress: modbus.positionAddress ?? 0,
      variables: modbus.variables.map((v) => ({
        name: v.name,
        dataType: v.dataType,
        address: stringValue(v.address),
        bits: v.bits ?? 0,
        functionCode: v.functionCode ?? 3,
        rate: v.rate ?? 1,
        amount: v.amount ?? 0,
        namespace: 2,
        defaultValue: stringValue(v.defaultValue),
        accessMode: v.accessMode,
      })),
    }
  }

  if (device.deviceType === 'opcUa') {
    const opcUa = device as OpcUaDevice
    return {
      ...base,
      ...common,
      opcLocation: opcUa.address?.location ?? '',
      opcPort: opcUa.address?.option?.port ?? 4840,
      opcUsername: opcUa.address?.option?.username ?? '',
      opcPassword: opcUa.address?.option?.password ?? '',
      variables: opcUa.variables.map((v) => ({
        name: v.name,
        dataType: v.dataType,
        address: stringValue(v.address),
        bits: 0,
        functionCode: 3,
        rate: 1,
        amount: 0,
        namespace: v.namespace ?? 2,
        defaultValue: stringValue(v.defaultValue),
        accessMode: v.accessMode,
      })),
    }
  }

  const s7 = device as S7Device
  return {
    ...base,
    ...common,
    s7Location: s7.address?.location ?? '',
    s7Port: s7.address?.option?.port ?? 102,
    s7Rack: s7.address?.option?.rack ?? 0,
    s7Slot: s7.address?.option?.slot ?? 1,
    variables: s7.variables.map((v) => ({
      name: v.name,
      dataType: v.dataType,
      address: stringValue(v.address),
      bits: 0,
      functionCode: 3,
      rate: v.rate ?? 1,
      amount: 0,
      namespace: 2,
      defaultValue: stringValue(v.defaultValue),
      accessMode: v.accessMode,
    })),
  }
}

export function toPayload(values: DeviceFormValues): DevicePayload {
  const base = {
    name: values.name.trim(),
    deviceCode: values.deviceCode.trim(),
    deviceType: values.deviceType,
    deviceModel: values.deviceModel,
    collectorCycle: values.collectorCycle,
    ...(values.variableInterval > 0 ? { variableInterval: values.variableInterval } : {}),
    ...(values.topic.trim() ? { topic: values.topic.trim() } : {}),
  }

  if (values.deviceType === 'modbus') {
    const serial = values.deviceModel === 'modbusRtu'
    return {
      ...base,
      deviceType: 'modbus',
      address: {
        location: values.modbusLocation.trim(),
        option: serial
          ? {
              baudRate: values.modbusBaudRate,
              dataBits: values.modbusDataBits,
              parity: values.modbusParity,
              stopBits: values.modbusStopBits,
            }
          : { port: values.modbusPort },
      },
      slave: values.slave,
      memoryLayout: values.memoryLayout as MemoryLayout,
      positionAddress: values.positionAddress,
      variables: values.variables.map((v) => ({
        name: v.name.trim(),
        dataType: v.dataType,
        address: Number.parseInt(v.address, 10),
        bits: v.dataType === 'bool' ? v.bits : 0,
        functionCode: v.functionCode,
        rate: v.rate,
        amount: v.amount,
        accessMode: v.accessMode as AccessMode,
      })),
    }
  }

  if (values.deviceType === 'opcUa') {
    return {
      ...base,
      deviceType: 'opcUa',
      address: {
        location: values.opcLocation.trim(),
        option: {
          port: values.opcPort,
          ...(values.opcUsername ? { username: values.opcUsername } : {}),
          ...(values.opcPassword ? { password: values.opcPassword } : {}),
        },
      },
      variables: values.variables.map((v) => ({
        name: v.name.trim(),
        dataType: v.dataType,
        address: v.address.trim(),
        namespace: v.namespace,
        accessMode: v.accessMode as AccessMode,
      })),
    }
  }

  return {
    ...base,
    deviceType: 's7',
    address: {
      location: values.s7Location.trim(),
      option: { port: values.s7Port, rack: values.s7Rack, slot: values.s7Slot },
    },
    variables: values.variables.map((v) => ({
      name: v.name.trim(),
      dataType: v.dataType,
      address: v.address.trim(),
      rate: v.rate,
      accessMode: v.accessMode as AccessMode,
    })),
  }
}
