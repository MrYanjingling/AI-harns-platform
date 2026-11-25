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
  variableCount: number
  collectedVariableCount: number
  lastError?: string
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
export interface CollectionVariable
  extends Omit<CollectionVariableInput, 'bits' | 'rate' | 'offset' | 'precision'> {
  type?: string
  bits?: number | null
  rate?: number | null
  offset?: number | null
  precision?: number | null
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
  variables: CollectionVariable[]
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

export interface DriverFieldCondition {
  field: string
  equals: string
}

export interface DriverFieldVariant {
  when: DriverFieldCondition
  labelZh: string
  labelEn: string
  regex?: string
  placeholderZh?: string
  placeholderEn?: string
}

export interface DriverField {
  name: string
  labelZh: string
  labelEn: string
  regex: string
  sort: number
  type: 'input' | 'select'
  required?: boolean
  placeholderZh?: string
  placeholderEn?: string
  options?: DriverFieldOption[]
  variants?: DriverFieldVariant[]
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
  remark: string
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
  remark: string
}

/* ---------------- 推送 ---------------- */

export type KafkaSASLMechanism = 'none' | 'plain' | 'scram-sha-256' | 'scram-sha-512'

export interface ConnectorConfig {
  brokerUrl: string
  username?: string
  password?: string
  clientId?: string
  groupId?: string
  saslMechanism?: KafkaSASLMechanism
  qos?: number
  retain?: boolean
  keepAlive?: number
  dataTopic?: string
  commandTopic?: string
}

export type MQTTConfig = ConnectorConfig
export type KafkaConfig = ConnectorConfig

export type ConnectorRunState = 'running' | 'stopped'
export type ConnectorStatus = 'running' | 'stopped' | 'error'

export interface PushConnector {
  id: string
  type: string
  name: string
  pushConfig: ConnectorConfig
  runState: ConnectorRunState
  groupCount?: number
  status: ConnectorStatus | string
  lastError?: string
  version: number
  createdTime: string
  updatedTime: string
}

export interface PushConnectorPayload {
  type: string
  name: string
  pushConfig: ConnectorConfig
}

export interface PushGroup {
  id: string
  name: string
  connectorId: string
  dataTopic: string
  commandTopic: string
  runState: ConnectorRunState
  status: ConnectorStatus | string
  lastError?: string
  pushError?: string
  subscriptionError?: string
  lastPushTime?: string
  pendingMessages: number
  pendingBatches: number
  oldestPendingTime?: string
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
  dataTopic: string
  commandTopic: string
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
  initTime: string
  currentUptime: string
  totalUptime: string
}

export interface SystemInfoMeta {
  version: string
  commitId: string
  hardwareId: string
  architecture: string
  cpuModel: string
  osVersion: string
  kernelVersion: string
  backendVersion?: string
  frontendVersion?: string
  releaseVersion?: string
}

export interface GatewayUpdateInstalled {
  releaseVersion: string
  backendVersion: string
  backendSha256: string
  frontendVersion: string
  frontendSha256?: string
}

export type GatewayUpdateTaskState =
  | 'received'
  | 'downloading'
  | 'ready'
  | 'applying'
  | 'restarting'
  | 'succeeded'
  | 'failed'
  | 'rejected'
  | 'ignored'

export interface GatewayUpdateTask {
  taskId: string
  manifestUrl: string
  state: GatewayUpdateTaskState | string
  releaseVersion?: string
  backendVersion?: string
  frontendVersion?: string
  backendSha256?: string
  frontendSha256?: string
  restartRequired: boolean
  progress: number
  errorCode?: string
  message?: string
  updatedAt: string
}

export interface GatewayUpdateSettings {
  enabled: boolean
  brokerUrl: string
  username: string
  hasPassword: boolean
  updatedTime?: string
}

export interface GatewayUpdateSettingsPayload {
  enabled: boolean
  brokerUrl: string
  username: string
  password: string
}

export interface GatewayUpdateState {
  installed: GatewayUpdateInstalled
  task?: GatewayUpdateTask
  settings: GatewayUpdateSettings
}

export interface LicenseInfo {
  maxVariable: number
  contact: string
  contactEmail: string
  company: string
  licenseExpiresAt: string
  softwareRunDeadline: string
}

export interface LicenseStatus {
  status: 'authorized' | 'expired' | 'unauthorized' | string
  license?: LicenseInfo
}

export interface GatewayUser {
  id: string
  version: number
  username: string
  remark: string
  isAdmin: boolean
  createdTime: string
  updatedTime: string
}

export interface GatewayUserPayload {
  username: string
  password?: string
  remark?: string
}

export interface LoginPayload {
  username: string
  password: string
}

export interface LoginResult {
  token: string
  expiresAt: string
  user: GatewayUser
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

export interface PushPayloadEvent {
  groupId: string
  connectorId: string
  connectorType: string
  topic: string
  key?: string
  time: string
  durationMs?: number
  payload: unknown
  error?: string
}

export interface PushCommandEvent {
  groupId: string
  connectorId: string
  connectorType: string
  topic: string
  time: string
  payload: unknown
}

export type SimulatorMode = 'random' | 'increment' | 'decrement'
export type SimulatorDataType = 'int16' | 'int32' | 'float'

export interface SimulatorPoint {
  address: number
  name: string
  description: string
  registerType: string
  dataType: SimulatorDataType
  mode: SimulatorMode
  value: number
}

export interface SimulatorState {
  running: boolean
  port: number
  unitId: number
  memoryLayout: string
  points: SimulatorPoint[]
}

/* ---------------- AI 配置 ---------------- */

export interface AISettings {
  assistantEnabled: boolean
  mcpEnabled: boolean
  name: string
  remark: string
  model: string
  endpointUrl: string
  apiKey: string
  hasApiKey: boolean
  mcpToken: string
  updatedTime: string
}

export interface AIModelPayload {
  name: string
  remark: string
  model: string
  apiKey?: string
  endpointUrl: string
}

/* ---------------- AI 助手 ---------------- */

export interface AIChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export interface AIChatAttachment {
  name: string
  content: string
}

export interface AIChatResponse {
  reply: string
  tools?: string[]
}
