/* ============================================================
 * 网关后端契约类型 — 对应 pkg/gateway/{collectionmgr,drivermgr,pushmgr}
 * 以及 pkg/gateway/{gateway,server}.go。
 * 网关与平台一样返回 {code,message,data} 信封;分页结构同平台。
 * ============================================================ */

export interface PageResp<T> {
  total: number
  page: number
  pageSize: number
  list: T[]
}

/* ---------------- 采集器 ---------------- */

export type CollectStatus =
  | 'collecting'
  | 'collectingError'
  | 'unconnected'
  | 'emptyVariable'
  | 'stopped'
  | 'error'

export type AccessMode = 'r' | 'w' | 'rw'

export interface Collection {
  id: string
  name: string
  version: string
  driverType: string
  driverModel: string
  collectStatus: CollectStatus | string
  enable: boolean
  collectorCycleMs: number
  modTime: string
}

export interface CollectionDetail extends Collection {
  protocolConfig: Record<string, unknown>
}

/** 变量定义;创建/新增变量时使用。 */
export interface CollectionVariableInput {
  name: string
  dataType: string
  address: string
  bits?: number
  accessMode: AccessMode
  rate?: number
  offset?: number
  precision?: number
  isCalculated?: boolean
  expression?: string
}

/** 变量列表响应,额外携带运行时值。 */
export interface CollectionVariable extends CollectionVariableInput {
  type?: string
  value?: unknown
}

export interface CollectionPayload {
  driverType: string
  driverModel: string
  name: string
  collectorCycleMs: number
  variables?: CollectionVariableInput[]
  protocolConfig: Record<string, unknown>
}

export interface AddVariablesResult {
  added: number
  variables: CollectionVariableInput[]
}

export interface CollectionAction {
  name: string
  value: string
}

/* ---------------- 驱动 ---------------- */

export interface DriverFieldOption {
  value: string
  labelZh: string
  labelEn: string
}

export interface DriverField {
  name: string
  labelZh: string
  labelEn: string
  regex: string
  sort: number
  type: 'input' | 'select'
  required?: boolean
  options?: DriverFieldOption[]
}

export interface VariableAddressSchema {
  labelZh: string
  labelEn: string
  regex?: string
  placeholderZh?: string
  placeholderEn?: string
  descriptionZh?: string
  descriptionEn?: string
}

export interface VariableDataType {
  value: string
  labelZh: string
  labelEn: string
  addressRegex: string
}

export interface DriverSchema {
  fields: DriverField[]
  variableAddress?: VariableAddressSchema
  variableDataTypes?: VariableDataType[]
}

export interface Driver {
  id: string
  driverType: string
  model: string
  builtinDriver: boolean
  schema: DriverSchema
  pluginName: string
  createdTime: string
}

export interface DriverGroup {
  driverType: string
  models: Driver[]
}

export interface DriverSchemaResponse {
  driverType: string
  driverModel: string
  schema: DriverSchema
}

export interface DriverPayload {
  driverType: string
  model: string
  schema: DriverSchema
  pluginName: string
}

/* ---------------- 推送 ---------------- */

export interface MQTTConfig {
  brokerUrl: string
  username?: string
  password?: string
  dataTopic: string
  commandTopic?: string
  clientId?: string
  qos?: number
  retain?: boolean
  keepAlive?: number
}

export type ConnectorStatus = 'running' | 'stopped'

export interface PushConnector {
  id: string
  type: string
  name: string
  pushConfig: MQTTConfig
  status: ConnectorStatus | string
  version: number
  createdTime: string
  updatedTime: string
}

export interface PushConnectorPayload {
  type: string
  name: string
  pushConfig: MQTTConfig
}

export interface PushGroup {
  id: string
  name: string
  connectorId: string
  pushIntervalMs: number
  allCollections: boolean
  collectionIds: string[]
  maxCollectionsPerPushMessage: number
  version: number
  createdTime: string
  updatedTime: string
}

export interface PushGroupPayload {
  name: string
  connectorId: string
  pushIntervalMs: number
  allCollections: boolean
  collectionIds: string[]
  maxCollectionsPerPushMessage?: number
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

/* ---------------- 帧报文 ---------------- */

export interface FrameEvent {
  traceId: number
  time: string
  operation: string
  hex: string
  length: number
  durationMs?: number
  error?: string
}

/* ---------------- 本地日志 ---------------- */

export type OperationAction =
  | 'create'
  | 'update'
  | 'delete'
  | 'start'
  | 'stop'
  | 'restart'
  | 'control'
  | 'enable'
  | 'push'

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
  collectionId?: string
}
