/* ============================================================
 * 网关后端契约类型 — 对应 pkg/gateway/runtime、pkg/gateway/v1
 * 以及 pkg/gateway/{device,gateway}/server.go。
 * 注意:网关与平台一样返回 {code,message,data} 信封。
 * ============================================================ */

export type DeviceType = 'modbus' | 'opcUa' | 's7'
export type DeviceModel = 'modbusTcp' | 'modbusRtu' | 'modbusRtuOverTcp' | 'opcUa' | 's71500'
export type AccessMode = 'r' | 'rw'

/** 采集状态,pkg/gateway/runtime/meta.go。 */
export type CollectStatus =
  | 'collecting'
  | 'collectingError'
  | 'unconnected'
  | 'emptyVariable'
  | 'stopped'
  | 'error'

export type DeviceOperator = 'start' | 'stop' | 'restart'

export interface GatewayDevice {
  name: string
  id: string
  /** 后端 ObjectMeta.Version 的 JSON 字段名。 */
  eTag: string
  modTime: string
  topic?: string
  deviceCode?: string
  deviceType: DeviceType | string
  deviceModel: string
  collectStatus: CollectStatus | string
}

export interface GatewayAddressOption {
  port?: number
  baudRate?: number
  dataBits?: number
  parity?: string
  stopBits?: string
  username?: string
  password?: string
  rack?: number
  slot?: number
}

export interface GatewayAddress {
  location: string
  option?: GatewayAddressOption
}

export interface GatewayVariable {
  name: string
  dataType: string
  address: number | string
  bits?: number
  functionCode?: number
  rate?: number
  amount?: number
  namespace?: number
  defaultValue?: unknown
  accessMode: AccessMode
}

export interface GatewayDeviceDetail extends GatewayDevice {
  collectorCycle: number
  variableInterval?: number
  address: GatewayAddress
  variables: GatewayVariable[]
  /** Modbus 专有 */
  slave?: number
  memoryLayout?: string
  positionAddress?: number
}

export interface DeviceListResponse {
  devices: GatewayDeviceDetail[]
}

/** 写入后端的设备结构;protocol 专有字段按 deviceType 选择。 */
export interface DevicePayload {
  name: string
  deviceCode: string
  deviceType: DeviceType
  deviceModel: string
  collectorCycle: number
  variableInterval?: number
  topic?: string
  address: GatewayAddress
  variables: GatewayVariable[]
  slave?: number
  memoryLayout?: string
  positionAddress?: number
}

/* ---------------- 网关资源 ---------------- */

export interface GatewayMeta {
  name: string
  id: string
  eTag: string
  secret: string
  modTime: string
}

/** 后端字段名首字母大写:Total / Used / UsedPercent。 */
export interface UsageInfo {
  Total: string
  Used: string
  UsedPercent: string
}

export interface GatewayCpuResponse {
  cpus?: {
    '1min': string
    '5min': string
    '15min': string
  }
}

export interface GatewayMemResponse {
  mem?: UsageInfo
}

export interface GatewayDiskResponse {
  disk?: Record<string, UsageInfo>
}

/* ---------------- 订阅 / 插件 / 日志(前端派生) ---------------- */

export type SubscriptionSource = 'custom' | 'auto'

export interface SubscriptionView {
  deviceId: string
  deviceName: string
  deviceCode: string
  deviceType: string
  deviceModel: string
  topic: string
  source: SubscriptionSource
  status: CollectStatus | string
}

export type PluginKey = DeviceType

export interface PluginCatalogItem {
  key: PluginKey
  name: string
  models: string[]
  transports: string[]
  dataTypes: string[]
  capabilities: string[]
  descriptionKey: string
}

export type OperationAction =
  | 'create'
  | 'update'
  | 'delete'
  | 'start'
  | 'stop'
  | 'restart'
  | 'control'

export type OperationResult = 'success' | 'failure'

export interface OperationLogEntry {
  id: string
  time: string
  action: OperationAction
  result: OperationResult
  target: string
  message: string
}

export type GatewayLogLevel = 'info' | 'warn' | 'error'

export interface GatewayLogEntry {
  id: string
  time: string
  level: GatewayLogLevel
  source: 'event' | 'operation'
  title: string
  message: string
  deviceId?: string
}
