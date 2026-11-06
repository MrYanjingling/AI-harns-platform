import type { MessageKey } from '@/i18n'
import type {
  DeviceStatus,
  DeviceStatusAction,
  DeviceType,
  DataType,
  OperationType,
  PlatformStatus,
  RwFlag,
  ThingDataType,
} from '@/api/types'

export interface StatusMeta {
  labelKey: MessageKey
  /** tailwind 类:文本 + 背景 + 边框 */
  className: string
  dotClassName: string
  descriptionKey: MessageKey
}

export const DEVICE_STATUS: Record<DeviceStatus, StatusMeta> = {
  collecting: {
    labelKey: 'enums.deviceStatus.collecting.label',
    className:
      'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:border-emerald-600/30',
    dotClassName: 'bg-emerald-500',
    descriptionKey: 'enums.deviceStatus.collecting.description',
  },
  collectingError: {
    labelKey: 'enums.deviceStatus.collectingError.label',
    className:
      'bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-500/10 dark:text-amber-300 dark:border-amber-600/30',
    dotClassName: 'bg-amber-500',
    descriptionKey: 'enums.deviceStatus.collectingError.description',
  },
  unconnected: {
    labelKey: 'enums.deviceStatus.unconnected.label',
    className:
      'bg-zinc-50 text-zinc-600 border-zinc-200 dark:bg-zinc-500/10 dark:text-zinc-300 dark:border-zinc-600/30',
    dotClassName: 'bg-zinc-400',
    descriptionKey: 'enums.deviceStatus.unconnected.description',
  },
  emptyVariable: {
    labelKey: 'enums.deviceStatus.emptyVariable.label',
    className:
      'bg-violet-50 text-violet-700 border-violet-200 dark:bg-violet-500/10 dark:text-violet-300 dark:border-violet-600/30',
    dotClassName: 'bg-violet-500',
    descriptionKey: 'enums.deviceStatus.emptyVariable.description',
  },
  stopped: {
    labelKey: 'enums.deviceStatus.stopped.label',
    className:
      'bg-slate-100 text-slate-600 border-slate-300 dark:bg-slate-500/10 dark:text-slate-300 dark:border-slate-600/30',
    dotClassName: 'bg-slate-400',
    descriptionKey: 'enums.deviceStatus.stopped.description',
  },
  error: {
    labelKey: 'enums.deviceStatus.error.label',
    className:
      'bg-rose-50 text-rose-700 border-rose-200 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-600/30',
    dotClassName: 'bg-rose-500',
    descriptionKey: 'enums.deviceStatus.error.description',
  },
}

export const UNKNOWN_STATUS: StatusMeta = {
  labelKey: 'enums.deviceStatus.unknown.label',
  className:
    'bg-zinc-50 text-zinc-600 border-zinc-200 dark:bg-zinc-500/10 dark:text-zinc-300 dark:border-zinc-600/30',
  dotClassName: 'bg-zinc-400',
  descriptionKey: 'enums.deviceStatus.unknown.description',
}

export interface ProtocolMeta {
  type: DeviceType
  label: string
  short: string
  descriptionKey: MessageKey
  defaultModel: string
  models: Array<{ value: string; label: string; hintKey: MessageKey }>
  /** 用于徽标配色 */
  accent: string
}

export const PROTOCOLS: Record<DeviceType, ProtocolMeta> = {
  modbus: {
    type: 'modbus',
    label: 'Modbus',
    short: 'MB',
    descriptionKey: 'enums.protocol.modbus.description',
    defaultModel: 'modbusTcp',
    models: [
      {
        value: 'modbusTcp',
        label: 'Modbus TCP',
        hintKey: 'enums.protocol.modbus.model.modbusTcp.hint',
      },
      {
        value: 'modbusRtu',
        label: 'Modbus RTU',
        hintKey: 'enums.protocol.modbus.model.modbusRtu.hint',
      },
      {
        value: 'modbusRtuOverTcp',
        label: 'Modbus RTU over TCP',
        hintKey: 'enums.protocol.modbus.model.modbusRtuOverTcp.hint',
      },
    ],
    accent: 'emerald',
  },
  opcUa: {
    type: 'opcUa',
    label: 'OPC UA',
    short: 'UA',
    descriptionKey: 'enums.protocol.opcUa.description',
    defaultModel: 'opcUa',
    models: [
      { value: 'opcUa', label: 'OPC UA', hintKey: 'enums.protocol.opcUa.model.opcUa.hint' },
    ],
    accent: 'sky',
  },
  s7: {
    type: 's7',
    label: 'S7',
    short: 'S7',
    descriptionKey: 'enums.protocol.s7.description',
    defaultModel: 's71500',
    models: [{ value: 's71500', label: 'S7-1500', hintKey: 'enums.protocol.s7.model.s71500.hint' }],
    accent: 'amber',
  },
}

export const DEVICE_STATUS_ACTIONS: Array<{
  action: DeviceStatusAction
  labelKey: MessageKey
}> = [
  { action: 'start', labelKey: 'enums.deviceAction.start' },
  { action: 'restart', labelKey: 'enums.deviceAction.restart' },
  { action: 'stop', labelKey: 'enums.deviceAction.stop' },
]

export const DATA_TYPES: Array<{ value: DataType; label: string }> = [
  { value: 'bool', label: 'bool' },
  { value: 'int16', label: 'int16' },
  { value: 'uint16', label: 'uint16' },
  { value: 'int32', label: 'int32' },
  { value: 'int64', label: 'int64' },
  { value: 'float32', label: 'float32' },
  { value: 'float64', label: 'float64' },
  { value: 'number', label: 'number' },
  { value: 'string', label: 'string' },
]

export const MEMORY_LAYOUTS = ['ABCD', 'BADC', 'CDAB', 'DCBA'] as const

export const BAUD_RATES = [1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200]
export const DATA_BITS = [5, 6, 7, 8]
export const PARITIES: Array<{ value: string; labelKey: MessageKey }> = [
  { value: 'noParity', labelKey: 'enums.parity.none' },
  { value: 'oddParity', labelKey: 'enums.parity.odd' },
  { value: 'evenParity', labelKey: 'enums.parity.even' },
]
export const STOP_BITS = [
  { value: '1', label: '1' },
  { value: '1.5', label: '1.5' },
  { value: '2', label: '2' },
]

/* ---------------- 平台服务 ---------------- */

export const PLATFORM_STATUS: Record<PlatformStatus, StatusMeta> = {
  enabled: {
    labelKey: 'enums.platformStatus.enabled.label',
    className:
      'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:border-emerald-600/30',
    dotClassName: 'bg-emerald-500',
    descriptionKey: 'enums.platformStatus.enabled.description',
  },
  disabled: {
    labelKey: 'enums.platformStatus.disabled.label',
    className:
      'bg-zinc-50 text-zinc-600 border-zinc-200 dark:bg-zinc-500/10 dark:text-zinc-300 dark:border-zinc-600/30',
    dotClassName: 'bg-zinc-400',
    descriptionKey: 'enums.platformStatus.disabled.description',
  },
}

/** 物模型 dataType.type 的展示与规格能力。 */
export const THING_DATA_TYPES: Array<{
  value: ThingDataType
  label: string
  descriptionKey: MessageKey
}> = [
  { value: 'int', label: 'int', descriptionKey: 'enums.thingDataType.int' },
  { value: 'long', label: 'long', descriptionKey: 'enums.thingDataType.long' },
  { value: 'float', label: 'float', descriptionKey: 'enums.thingDataType.float' },
  { value: 'double', label: 'double', descriptionKey: 'enums.thingDataType.double' },
  { value: 'bool', label: 'bool', descriptionKey: 'enums.thingDataType.bool' },
  { value: 'enum', label: 'enum', descriptionKey: 'enums.thingDataType.enum' },
  { value: 'text', label: 'text', descriptionKey: 'enums.thingDataType.text' },
  { value: 'time', label: 'time', descriptionKey: 'enums.thingDataType.time' },
  { value: 'array', label: 'array', descriptionKey: 'enums.thingDataType.array' },
  { value: 'struct', label: 'struct', descriptionKey: 'enums.thingDataType.struct' },
]

/** 需要 unit / min / max 的数值类型。 */
export const NUMERIC_DATA_TYPES: ThingDataType[] = ['int', 'long', 'float', 'double']

export const RW_FLAGS: Array<{ value: RwFlag; labelKey: MessageKey; descriptionKey: MessageKey }> =
  [
    {
      value: 'r',
      labelKey: 'enums.rw.r.label',
      descriptionKey: 'enums.rw.r.description',
    },
    {
      value: 'w',
      labelKey: 'enums.rw.w.label',
      descriptionKey: 'enums.rw.w.description',
    },
    {
      value: 'rw',
      labelKey: 'enums.rw.rw.label',
      descriptionKey: 'enums.rw.rw.description',
    },
  ]

export const OPERATION_TYPES: Array<{ value: OperationType; labelKey: MessageKey }> = [
  { value: 'LOGIN', labelKey: 'enums.operation.login' },
  { value: 'LOGOUT', labelKey: 'enums.operation.logout' },
  { value: 'CREATE', labelKey: 'enums.operation.create' },
  { value: 'UPDATE', labelKey: 'enums.operation.update' },
  { value: 'DELETE', labelKey: 'enums.operation.delete' },
  { value: 'QUERY', labelKey: 'enums.operation.query' },
]

export const AUDIT_APPS = ['IOT', 'EMS', 'EAM'] as const
export const AUDIT_METHODS = ['POST', 'PUT', 'PATCH', 'DELETE'] as const

export const GENDERS: Array<{ value: 'male' | 'female'; labelKey: MessageKey }> = [
  { value: 'male', labelKey: 'enums.gender.male' },
  { value: 'female', labelKey: 'enums.gender.female' },
]

export const AUTH_MODES: Array<{ value: 'local' | 'ldap' | 'radius'; labelKey: MessageKey }> = [
  { value: 'local', labelKey: 'enums.authMode.local' },
  { value: 'ldap', labelKey: 'enums.authMode.ldap' },
  { value: 'radius', labelKey: 'enums.authMode.radius' },
]

export const PAGE_SIZES = [10, 20, 50, 100] as const
