import type { MessageKey, StatusTone } from '@harns/core'
import type { AccessMode, CollectStatus, ConnectorStatus } from '@/api/types'

export type { StatusTone }

export interface StatusMeta {
  labelKey: MessageKey
  descriptionKey: MessageKey
  tone: StatusTone
}

export const COLLECT_STATUS: Record<CollectStatus, StatusMeta> = {
  collecting: {
    labelKey: 'gateway.status.collecting.label',
    descriptionKey: 'gateway.status.collecting.description',
    tone: 'success',
  },
  collectingError: {
    labelKey: 'gateway.status.collectingError.label',
    descriptionKey: 'gateway.status.collectingError.description',
    tone: 'danger',
  },
  unconnected: {
    labelKey: 'gateway.status.unconnected.label',
    descriptionKey: 'gateway.status.unconnected.description',
    tone: 'danger',
  },
  emptyVariable: {
    labelKey: 'gateway.status.emptyVariable.label',
    descriptionKey: 'gateway.status.emptyVariable.description',
    tone: 'warning',
  },
  stopped: {
    labelKey: 'gateway.status.stopped.label',
    descriptionKey: 'gateway.status.stopped.description',
    tone: 'warning',
  },
  error: {
    labelKey: 'gateway.status.error.label',
    descriptionKey: 'gateway.status.error.description',
    tone: 'danger',
  },
}

export const UPDATE_STATUS: Record<string, StatusMeta> = {
  received: {
    labelKey: 'gateway.update.status.received',
    descriptionKey: 'gateway.update.status.receivedHint',
    tone: 'info',
  },
  downloading: {
    labelKey: 'gateway.update.status.downloading',
    descriptionKey: 'gateway.update.status.downloadingHint',
    tone: 'warning',
  },
  ready: {
    labelKey: 'gateway.update.status.ready',
    descriptionKey: 'gateway.update.status.readyHint',
    tone: 'info',
  },
  applying: {
    labelKey: 'gateway.update.status.applying',
    descriptionKey: 'gateway.update.status.applyingHint',
    tone: 'warning',
  },
  restarting: {
    labelKey: 'gateway.update.status.restarting',
    descriptionKey: 'gateway.update.status.restartingHint',
    tone: 'warning',
  },
  succeeded: {
    labelKey: 'gateway.update.status.succeeded',
    descriptionKey: 'gateway.update.status.succeededHint',
    tone: 'success',
  },
  failed: {
    labelKey: 'gateway.update.status.failed',
    descriptionKey: 'gateway.update.status.failedHint',
    tone: 'danger',
  },
  rejected: {
    labelKey: 'gateway.update.status.rejected',
    descriptionKey: 'gateway.update.status.rejectedHint',
    tone: 'neutral',
  },
  ignored: {
    labelKey: 'gateway.update.status.ignored',
    descriptionKey: 'gateway.update.status.ignoredHint',
    tone: 'neutral',
  },
}

export function otaStatusMeta(status: string | undefined): StatusMeta {
  return (status && UPDATE_STATUS[status]) || UNKNOWN_COLLECT_STATUS
}

export const UNKNOWN_COLLECT_STATUS: StatusMeta = {
  labelKey: 'common.state.unknown',
  descriptionKey: 'common.state.unknown',
  tone: 'neutral',
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
  },
  stopped: {
    labelKey: 'gateway.push.status.stopped',
    descriptionKey: 'gateway.push.status.stoppedHint',
    tone: 'warning',
  },
  error: {
    labelKey: 'gateway.push.status.error',
    descriptionKey: 'gateway.push.status.errorHint',
    tone: 'danger',
  },
}

export function connectorStatusMeta(status: string | undefined): StatusMeta {
  if (status === 'running' || status === 'stopped' || status === 'error') {
    return CONNECTOR_STATUS[status]
  }
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
  siemens: 'Siemens',
  opcUa: 'OPC UA',
  omron: 'Omron',
  Panasonic: 'Panasonic',
  ethernetIp: 'EtherNet/IP',
  ethercat: 'EtherCAT',
  profinet: 'PROFINET',
  dnp3: 'DNP3',
  dlt645: 'DLT 645',
  cjt188: 'CJT 188',
  dlt698: 'DLT 698',
  KNX: 'KNX',
  CNC: 'CNC',
  Kuka: 'Kuka',
  Codesys: 'Codesys',
  'Allen-Bradley': 'Allen-Bradley',
  HJ212: 'HJ212',
  BACnet: 'BACnet',
  TwinCAT: 'TwinCAT',
  'GE/Emerson': 'GE/Emerson',
  SNMP: 'SNMP',
  IEC: 'IEC',
  ABB: 'ABB',
  Mitsubishi: 'Mitsubishi',
}

const DRIVER_MODEL_LABELS: Record<string, string> = {
  modbusTcp: 'Modbus TCP',
  modbusRtu: 'Modbus RTU',
  modbusRtuOverTcp: 'Modbus RTU over TCP',
  opcUa: 'OPC UA',
  's7-1500': 'S7-1500',
  s71500: 'S7-1500',
  's7-200': 'S7-200',
  mpi: 'MPI',
  ppi: 'PPI',
  'KNXnet/IP': 'KNXnet/IP',
  KND: 'KND',
  heidenhain: 'Heidenhain',
  'fanuc focas': 'Fanuc FOCAS',
  MTConnect: 'MTConnect',
  Kuka: 'Kuka',
  'codesys-v3': 'CODESYS V3',
  ab5000: 'AB5000',
  DF1: 'DF1',
  HJ212: 'HJ212',
  BACnet: 'BACnet',
  'Beckhoff ADS': 'Beckhoff ADS',
  SRTP: 'SRTP',
  'SNMP-V2': 'SNMP-V2',
  IEC61850: 'IEC 61850',
  fins: 'Omron FINS',
  hostlinkCmode: 'Omron HostLink C-Mode',
  omronCip: 'Omron EtherNet/IP CIP',
  Mewtocol: 'Panasonic Mewtocol',
  ethernetIp: 'EtherNet/IP',
  'EtherNet/IP-ControlLogic 55xx': 'EtherNet/IP-ControlLogic 55xx',
  'EtherNet/IP-MicroLogix 1400': 'EtherNet/IP-MicroLogix 1400',
  ethercat: 'EtherCAT (TwinCAT ADS)',
  profinet: 'PROFINET',
  dnp3: 'DNP3',
  dlt645_2007: 'DLT 645-2007',
  dlt645_1997: 'DLT 645-1997',
  cjt188_2004: 'CJT 188-2004',
  cjt188_2018: 'CJT 188-2018',
  dlt698_45: 'DLT 698.45',
  COMLI: 'COMLI',
  'MC-FX': 'MC-FX',
  'MC-1E': 'MC-1E',
  'MC-3E': 'MC-3E',
  'MC-4E': 'MC-4E',
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
  'link',
  'positionAddress',
  'concurrency',
  'baudRate',
  'dataBits',
  'port',
  'rack',
  'slot',
  'pduSize',
  'connectionType',
  'localTsap',
  'remoteTsap',
  'type',
  'expires',
  'maxSize',
  'timeoutMs',
  'intervalMs',
  'interval',
  'cipSize',
  'logixArraySize',
  'masterId',
  'slaveId',
  'class0123Interval',
  'class123Interval',
  'timeSync',
  'maxReadWords',
  'baud',
  'data',
  'parity',
  'stop',
  'retry',
  'transport_mode',
  'src-port',
  'amsPort',
  'localAmsPort',
  'srcPort',
  'bbmd',
  'srcAdsPort',
  'dstAdsPort',
  'localPort',
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
  return `${ms}ms`
}
