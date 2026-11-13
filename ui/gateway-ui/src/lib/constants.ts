import type { MessageKey } from '@harns/core'
import type { AccessMode, CollectStatus, ConnectorStatus } from '@/api/types'

export type StatusTone = 'neutral' | 'success' | 'warning' | 'danger' | 'info'

export interface StatusMeta {
  labelKey: MessageKey
  descriptionKey: MessageKey
  tone: StatusTone
  dotClassName: string
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

export const CONNECTOR_STATUS: Record<ConnectorStatus, StatusMeta> = {
  running: {
    labelKey: 'gateway.push.status.running',
    descriptionKey: 'gateway.push.status.runningHint',
    tone: 'success',
    dotClassName: 'bg-emerald-500',
  },
  stopped: {
    labelKey: 'gateway.push.status.stopped',
    descriptionKey: 'gateway.push.status.stoppedHint',
    tone: 'neutral',
    dotClassName: 'bg-zinc-400',
  },
}

export function connectorStatusMeta(status: string | undefined): StatusMeta {
  if (status === 'running' || status === 'stopped') return CONNECTOR_STATUS[status]
  return UNKNOWN_COLLECT_STATUS
}

export const ACCESS_MODES: Array<{ value: AccessMode; labelKey: MessageKey }> = [
  { value: 'r', labelKey: 'gateway.variable.accessMode.r' },
  { value: 'w', labelKey: 'gateway.variable.accessMode.w' },
  { value: 'rw', labelKey: 'gateway.variable.accessMode.rw' },
]

/** 内置驱动类型的技术名称;未知类型原样展示。 */
const DRIVER_TYPE_LABELS: Record<string, string> = {
  modbus: 'Modbus',
  s7: 'S7',
  opcUa: 'OPC UA',
}

const DRIVER_MODEL_LABELS: Record<string, string> = {
  modbusTcp: 'Modbus TCP',
  modbusRtu: 'Modbus RTU',
  modbusRtuOverTcp: 'Modbus RTU over TCP',
  opcUa: 'OPC UA',
  s71500: 'S7-1500',
  's7-200': 'S7-200',
}

export function driverTypeLabel(type: string): string {
  return DRIVER_TYPE_LABELS[type] ?? type
}

export function driverModelLabel(model: string): string {
  return DRIVER_MODEL_LABELS[model] ?? model
}

/**
 * 驱动 Schema 的协议字段均以字符串编辑,但后端结构体要求数字。
 * 这些字段在提交前转换为数字;localTsap/remoteTsap 支持 0x 前缀。
 */
export const NUMERIC_PROTOCOL_FIELDS = new Set([
  'positionAddress',
  'tcpCount',
  'baudRate',
  'dataBits',
  'port',
  'rack',
  'slot',
  'pduSize',
  'connectionType',
  'localTsap',
  'remoteTsap',
])

export const HEX_PROTOCOL_FIELDS = new Set(['localTsap', 'remoteTsap'])

/** OPC UA 详情会脱敏密码;编辑时保留原值而不回写。 */
export const REDACTED_VALUE = '***'

export function toProtocolValue(name: string, raw: string): string | number | undefined {
  const value = raw.trim()
  if (value === '') return undefined
  if (HEX_PROTOCOL_FIELDS.has(name)) {
    const parsed = Number.parseInt(value, value.toLowerCase().startsWith('0x') ? 16 : 10)
    return Number.isFinite(parsed) ? parsed : undefined
  }
  if (NUMERIC_PROTOCOL_FIELDS.has(name)) {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : value
  }
  return value
}

/** 采集周期以毫秒存储,界面按秒展示(支持小数)。 */
export function formatCycle(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return '—'
  const seconds = ms / 1000
  return Number.isInteger(seconds) ? `${seconds}s` : `${seconds.toFixed(1)}s`
}
