import type { MessageKey } from '@harns/core'
import type {
  AccessMode,
  CollectStatus,
  DeviceModel,
  DeviceType,
  PluginCatalogItem,
} from '@/api/types'

export type StatusTone = 'neutral' | 'success' | 'warning' | 'danger' | 'info'

export interface StatusMeta {
  labelKey: MessageKey
  descriptionKey: MessageKey
  tone: StatusTone
  dotClassName: string
}

export interface DeviceModelMeta {
  value: DeviceModel
  /** 技术名称不翻译 */
  label: string
  descriptionKey: MessageKey
}

export interface DeviceTypeMeta {
  value: DeviceType
  label: string
  descriptionKey: MessageKey
  models: DeviceModelMeta[]
}

export const DEVICE_TYPES: DeviceTypeMeta[] = [
  {
    value: 'modbus',
    label: 'Modbus',
    descriptionKey: 'gateway.protocol.modbus.description',
    models: [
      {
        value: 'modbusTcp',
        label: 'Modbus TCP',
        descriptionKey: 'gateway.model.modbusTcp.description',
      },
      {
        value: 'modbusRtu',
        label: 'Modbus RTU',
        descriptionKey: 'gateway.model.modbusRtu.description',
      },
      {
        value: 'modbusRtuOverTcp',
        label: 'Modbus RTU over TCP',
        descriptionKey: 'gateway.model.modbusRtuOverTcp.description',
      },
    ],
  },
  {
    value: 'opcUa',
    label: 'OPC UA',
    descriptionKey: 'gateway.protocol.opcUa.description',
    models: [
      {
        value: 'opcUa',
        label: 'OPC UA',
        descriptionKey: 'gateway.model.opcUa.description',
      },
    ],
  },
  {
    value: 's7',
    label: 'S7',
    descriptionKey: 'gateway.protocol.s7.description',
    models: [
      {
        value: 's71500',
        label: 'S7-1500',
        descriptionKey: 'gateway.model.s71500.description',
      },
    ],
  },
]

export function deviceTypeMeta(type: string): DeviceTypeMeta | undefined {
  return DEVICE_TYPES.find((item) => item.value === type)
}

export function deviceModelMeta(type: string, model: string): DeviceModelMeta | undefined {
  return deviceTypeMeta(type)?.models.find((item) => item.value === model)
}

export const COLLECT_STATUS: Record<CollectStatus, StatusMeta> = {
  collecting: {
    labelKey: 'gateway.status.collecting.label',
    descriptionKey: 'gateway.status.collecting.description',
    tone: 'success',
    dotClassName: 'bg-emerald-500',
  },
  collectingError: {
    labelKey: 'gateway.status.collectingError.label',
    descriptionKey: 'gateway.status.collectingError.description',
    tone: 'danger',
    dotClassName: 'bg-rose-500',
  },
  unconnected: {
    labelKey: 'gateway.status.unconnected.label',
    descriptionKey: 'gateway.status.unconnected.description',
    tone: 'warning',
    dotClassName: 'bg-amber-500',
  },
  emptyVariable: {
    labelKey: 'gateway.status.emptyVariable.label',
    descriptionKey: 'gateway.status.emptyVariable.description',
    tone: 'warning',
    dotClassName: 'bg-amber-500',
  },
  stopped: {
    labelKey: 'gateway.status.stopped.label',
    descriptionKey: 'gateway.status.stopped.description',
    tone: 'neutral',
    dotClassName: 'bg-zinc-400',
  },
  error: {
    labelKey: 'gateway.status.error.label',
    descriptionKey: 'gateway.status.error.description',
    tone: 'danger',
    dotClassName: 'bg-rose-500',
  },
}

export const UNKNOWN_COLLECT_STATUS: StatusMeta = {
  labelKey: 'common.state.unknown',
  descriptionKey: 'common.state.unknown',
  tone: 'neutral',
  dotClassName: 'bg-zinc-400',
}

export function collectStatusMeta(status: string | undefined): StatusMeta {
  if (status && status in COLLECT_STATUS) {
    return COLLECT_STATUS[status as CollectStatus]
  }
  return UNKNOWN_COLLECT_STATUS
}

/** 非采集中状态视为异常,用于概览统计与日志事件。 */
export function isAbnormalStatus(status: string | undefined): boolean {
  return (
    status === 'collectingError' ||
    status === 'unconnected' ||
    status === 'emptyVariable' ||
    status === 'error'
  )
}

export const ACCESS_MODES: Array<{ value: AccessMode; labelKey: MessageKey }> = [
  { value: 'r', labelKey: 'gateway.form.accessMode.r' },
  { value: 'rw', labelKey: 'gateway.form.accessMode.rw' },
]

export const MODBUS_DATA_TYPES = [
  'bool',
  'int16',
  'float32',
  'float64',
  'int32',
  'int64',
  'uint16',
] as const

export const OPCUA_DATA_TYPES = ['string', 'number'] as const

export const S7_DATA_TYPES = [
  'bool',
  'int16',
  'float32',
  'float64',
  'int32',
  'int64',
  'uint16',
] as const

export function dataTypesFor(deviceType: DeviceType | string): readonly string[] {
  if (deviceType === 'opcUa') return OPCUA_DATA_TYPES
  if (deviceType === 's7') return S7_DATA_TYPES
  return MODBUS_DATA_TYPES
}

export const BAUD_RATES = [1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200] as const
export const DATA_BITS = [5, 6, 7, 8] as const
export const PARITIES = ['noParity', 'oddParity', 'evenParity', 'markParity', 'spaceParity'] as const
export const STOP_BITS = ['1', '1.5', '2'] as const
export const MEMORY_LAYOUTS = ['ABCD', 'BADC', 'CDAB', 'DCBA'] as const
export const FUNCTION_CODES = [1, 2, 3, 4] as const

export const SERIAL_MODELS: readonly string[] = ['modbusRtu']

export function usesSerial(deviceModel: string): boolean {
  return SERIAL_MODELS.includes(deviceModel)
}

/** 内置协议插件目录;后端通过 registry.DeviceTypeMap 注册,前端只读展示。 */
export const PLUGIN_CATALOG: PluginCatalogItem[] = [
  {
    key: 'modbus',
    name: 'Modbus',
    models: ['modbusTcp', 'modbusRtu', 'modbusRtuOverTcp'],
    transports: ['TCP', 'RTU', 'RTU over TCP'],
    dataTypes: [...MODBUS_DATA_TYPES],
    capabilities: [
      'gateway.plugins.capability.read',
      'gateway.plugins.capability.write',
      'gateway.plugins.capability.cycle',
    ],
    descriptionKey: 'gateway.protocol.modbus.description',
  },
  {
    key: 'opcUa',
    name: 'OPC UA',
    models: ['opcUa'],
    transports: ['TCP'],
    dataTypes: [...OPCUA_DATA_TYPES],
    capabilities: ['gateway.plugins.capability.read', 'gateway.plugins.capability.cycle'],
    descriptionKey: 'gateway.protocol.opcUa.description',
  },
  {
    key: 's7',
    name: 'S7',
    models: ['s71500'],
    transports: ['TCP'],
    dataTypes: [...S7_DATA_TYPES],
    capabilities: [
      'gateway.plugins.capability.read',
      'gateway.plugins.capability.write',
      'gateway.plugins.capability.cycle',
    ],
    descriptionKey: 'gateway.protocol.s7.description',
  },
]

/** 网关 MQTT 发布 topic 规则,与 pkg/gateway/device/manager.go 一致。 */
export function defaultTopic(gatewayId: string, deviceId: string): string {
  return `data/${gatewayId}/v1/${deviceId}`
}
