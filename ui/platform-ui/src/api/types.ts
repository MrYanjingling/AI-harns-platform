/* ============================================================
 * 后端契约类型 — 严格对应 Go 结构体的 JSON 输出。
 * 来源:pkg/runtime、pkg/v1、pkg/gateway、pkg/data、pkg/protocol/*。
 * 修改前请先阅读对应 server.go 与 type.go。
 * ============================================================ */

export type DeviceType = 'modbus' | 'opcUa' | 's7'

export type DeviceStatus =
  | 'collecting'
  | 'collectingError'
  | 'unconnected'
  | 'emptyVariable'
  | 'stopped'
  | 'error'

export type AccessMode = 'r' | 'rw'

export type DataType =
  | 'bool'
  | 'int16'
  | 'float32'
  | 'float64'
  | 'int32'
  | 'int64'
  | 'uint16'
  | 'number'
  | 'string'

export type MemoryLayout = 'ABCD' | 'BADC' | 'CDAB' | 'DCBA'

export type DeviceStatusAction = 'start' | 'restart' | 'stop'

/* ---------- 设备(折叠列表形态) ---------- */

/** GET /devices 默认返回的 8 个字段。eTag 用于 If-Match。 */
export interface DeviceSummary {
  name: string
  id: string
  eTag: string
  modTime: string
  deviceCode: string
  deviceType: DeviceType
  deviceModel: string
  collectStatus: DeviceStatus
  topic?: string
}

export interface DeviceListResponse {
  devices?: DeviceSummary[]
}

/* ---------- 协议地址 ---------- */

export interface ModbusAddressOption {
  port?: number
  baudRate?: number
  dataBits?: number
  parity?: string
  stopBits?: string
}

export interface ModbusAddress {
  location: string
  option?: ModbusAddressOption
}

export interface OpcUaAddressOption {
  port?: number
  username?: string
  password?: string
}

export interface OpcUaAddress {
  location: string
  option?: OpcUaAddressOption
}

export interface S7AddressOption {
  port?: number
  rack?: number
  slot?: number
}

export interface S7Address {
  location: string
  option?: S7AddressOption
}

/* ---------- 变量(运行态,exploded=true) ---------- */

export interface ModbusVariable {
  name: string
  dataType: DataType
  address: number
  bits: number
  functionCode: number
  rate: number
  amount: number
  defaultValue?: unknown
  value?: unknown
  accessMode: AccessMode
}

export interface OpcUaVariable {
  name: string
  dataType: DataType
  address: unknown
  namespace: number
  defaultValue?: unknown
  value?: unknown
  accessMode: AccessMode
}

export interface S7Variable {
  name: string
  dataType: DataType
  address: string
  rate: number
  defaultValue?: unknown
  value?: unknown
  accessMode: AccessMode
}

/* ---------- 设备(exploded=true) ---------- */

export interface ModbusDevice extends DeviceSummary {
  deviceType: 'modbus'
  collectorCycle: number
  variableInterval: number
  address: ModbusAddress
  slave: number
  memoryLayout: MemoryLayout
  positionAddress: number
  variables: ModbusVariable[]
}

export interface OpcUaDevice extends DeviceSummary {
  deviceType: 'opcUa'
  collectorCycle: number
  variableInterval: number
  address: OpcUaAddress
  variables: OpcUaVariable[]
}

export interface S7Device extends DeviceSummary {
  deviceType: 's7'
  collectorCycle: number
  variableInterval: number
  address: S7Address
  variables: S7Variable[]
}

export type DeviceDetail = ModbusDevice | OpcUaDevice | S7Device

/* ---------- 创建 / 更新载荷(v1 创建形状) ---------- */

export interface DeviceBasePayload {
  name: string
  deviceCode: string
  deviceType: DeviceType
  deviceModel: string
  collectorCycle: number
  variableInterval?: number
  topic?: string
}

export interface ModbusDevicePayload extends DeviceBasePayload {
  deviceType: 'modbus'
  address: ModbusAddress
  slave: number
  memoryLayout: MemoryLayout
  positionAddress?: number
  variables: Array<{
    name: string
    dataType: DataType
    address: number
    bits?: number
    functionCode: number
    rate?: number
    amount?: number
    defaultValue?: unknown
    accessMode: AccessMode
  }>
}

export interface OpcUaDevicePayload extends DeviceBasePayload {
  deviceType: 'opcUa'
  address: OpcUaAddress
  variables: Array<{
    name: string
    dataType: DataType
    address: unknown
    namespace: number
    defaultValue?: unknown
    accessMode: AccessMode
  }>
}

export interface S7DevicePayload extends DeviceBasePayload {
  deviceType: 's7'
  address: S7Address
  variables: Array<{
    name: string
    dataType: DataType
    address: string
    rate?: number
    defaultValue?: unknown
    accessMode: AccessMode
  }>
}

export type DevicePayload = ModbusDevicePayload | OpcUaDevicePayload | S7DevicePayload

/** PUT /devices/:id/action 的请求体:单键对象数组。 */
export type DeviceAction = Record<string, unknown>

/* ---------- 网关资源 ---------- */

export interface GatewayMeta {
  name: string
  id: string
  eTag: string
  secret: string
  modTime: string
}

/** 注意:后端字段名是首字母大写的 Total/Used/UsedPercent。 */
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

/* ---------- 数据服务统一响应 ---------- */

export interface DataResponse<T> {
  data?: T
  total?: number
}

export interface MetricsInfo {
  id: string
  tenantId?: string | null
  revision?: number | null
  createdBy?: string | null
  createdTime?: string | null
  updatedBy?: string | null
  updatedTime?: string | null
  dataPoint?: string | null
  metricsCode?: string | null
  metricsName?: string | null
  measureType?: string | null
  enable?: boolean | null
  metricsType?: string | null
  calculateType?: string | null
  note?: string | null
}

export interface KpiInfo {
  id: string
  tenantId?: string | null
  revision?: number | null
  createdBy?: string | null
  createdTime?: string | null
  updatedBy?: string | null
  updatedTime?: string | null
  name: string
  formula?: string | null
  formulaDataName?: string | null
  formulaOri: string
  timeGranularity: string
  targetData?: string | null
  singleQuota: boolean
  defaultTag: boolean
  canDelete: boolean
  unit?: string | null
  targetDataType: string
  status: string
}

export interface TagInfo {
  id: string
  tenantId?: string | null
  revision?: number | null
  createdBy?: string | null
  createdTime?: string | null
  updatedBy?: string | null
  updatedTime?: string | null
  tagId?: string | null
  tagName?: string | null
  tagDesc?: string | null
  lastSynTime?: string | null
  tagType?: string | null
  tagAttribute?: string | null
  enable?: number | null
  scadaId: string
  operator?: string | null
  operateValue?: string | null
}

export interface MetricsQueryRequest {
  metricsCodes: string[]
  startTime: string
  endTime: string
  granularity: 'h'
}

export interface MetricsResult {
  metricsCode: string
  startTime: string
  endTime: string
  value: number
}

export interface KpiQueryRequest {
  ids: string[]
  startTime: string
  endTime: string
  granularity: 'h'
}

export interface KpiResult {
  id: string
  startTime: string
  endTime: string
  value: number
}

export type TimeSeriesMode = 'raw' | 'mean' | 'usage'

export interface TagTimeSeriesRequest {
  select: string
  start?: string
  end?: string
  latest?: boolean
  limit?: number
  mode?: TimeSeriesMode
}

export interface TimeSeriesPoint {
  time: string
  value: string
}

export interface DataPointTimeSeries {
  tagId: string
  tagName: string
  tagDesc: string
  data: TimeSeriesPoint[]
}

export type TimeSeriesSource = 'fmcs' | 'escada'

/* ============================================================
 * 平台服务(platform)契约 — 对应 pkg/platform/{auth,identity,thingmodel,things,files}
 * 响应统一 {code,message,data};分页 data 为 {total,page,pageSize,list}。
 * ============================================================ */

export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

export interface PlatformPage<T> {
  total: number
  page: number
  pageSize: number
  list: T[]
}

export interface PlatformPageQuery {
  page: number
  pageSize: number
  keyword?: string
}

export type PlatformStatus = 'enabled' | 'disabled'
export type Gender = 'male' | 'female'
export type AuthMode = 'local' | 'ldap' | 'radius'
export type OperationType = 'LOGIN' | 'LOGOUT' | 'CREATE' | 'UPDATE' | 'DELETE' | 'QUERY'
export type AuditApp = 'IOT' | 'EMS' | 'EAM'

/* ---------- 认证 ---------- */

export interface LoginRequest {
  username: string
  password: string
  code?: string
  key?: string
}

export interface LoginResponse {
  token: string
}

export interface CaptchaResponse {
  uuid: string
  img: string
}

/** /auth/index 返回的当前登录用户快照。 */
export interface CurrentUser {
  id: string
  version: number
  userName: string
  userNo: string
  nickName: string
  status: PlatformStatus
  roleIds: string[]
  orgIds: string[]
}

/* ---------- 用户 ---------- */

export interface UserInfo {
  id: string
  version: number
  userName: string
  userNo: string
  nickName: string
  birthDate: string
  gender: Gender | ''
  mail: string
  phone: string
  note: string
  authMode: AuthMode
  lastLoginTime: string | null
  loginFailureTimes: number
  expireTimes: number
  status: PlatformStatus
  createdUser: string
  updatedUser: string
  createdTime: string
  updatedTime: string
  roleIds: string[]
  orgIds: string[]
  roles: RoleInfo[]
  orgs: OrgInfo[]
}

export interface CreateUserRequest {
  userName: string
  password?: string
  userNo: string
  nickName: string
  birthDate?: string
  gender?: Gender
  mail?: string
  phone?: string
  note?: string
  authKey?: string
  authMode?: AuthMode
  expireTimes?: number
  status?: PlatformStatus
  roleIds?: string[]
  orgIds?: string[]
}

export interface UpdateUserRequest {
  userName?: string
  password?: string
  userNo?: string
  nickName?: string
  birthDate?: string
  gender?: Gender
  mail?: string
  phone?: string
  note?: string
  authKey?: string
  authMode?: AuthMode
  expireTimes?: number
  status?: PlatformStatus
}

/* ---------- 角色 ---------- */

export interface RoleInfo {
  id: string
  version: number
  name: string
  code: string
  description: string
  status: PlatformStatus
  createdUser: string
  updatedUser: string
  createdTime: string
  updatedTime: string
}

export interface CreateRoleRequest {
  name: string
  code: string
  description?: string
  status?: PlatformStatus
}

export type UpdateRoleRequest = Partial<CreateRoleRequest>

/* ---------- 组织 ---------- */

export interface OrgInfo {
  id: string
  version: number
  code: string
  name: string
  parentId: string
  sort: number
  description: string
  remark: string
  status: PlatformStatus
  createdUser: string
  updatedUser: string
  createdTime: string
  updatedTime: string
  children?: OrgInfo[]
}

export interface CreateOrgRequest {
  code: string
  name: string
  parentId?: string
  sort?: number
  description?: string
  remark?: string
  status?: PlatformStatus
}

export type UpdateOrgRequest = Partial<CreateOrgRequest>

/* ---------- 审计日志 ---------- */

export interface AuditLogInfo {
  id: string
  app: AuditApp | string
  traceId: string
  userId: string
  userName: string
  operationType: OperationType | string
  operationDesc: string
  method: string
  path: string
  query: string
  ip: string
  userAgent: string
  requestBody?: string
  responseBody?: string
  status: number
  resultCode: number
  errorMsg: string
  durationMs: number
  createdTime: string
}

export interface AuditQuery extends PlatformPageQuery {
  app?: AuditApp | ''
  userId?: string
  path?: string
  method?: string
  resultCode?: number
  /** yyyy-MM-dd HH:mm:ss */
  start?: string
  /** yyyy-MM-dd HH:mm:ss */
  end?: string
}

/* ---------- 物模型 ---------- */

export type ThingDataType =
  | 'int'
  | 'long'
  | 'float'
  | 'double'
  | 'bool'
  | 'enum'
  | 'text'
  | 'time'
  | 'array'
  | 'struct'

export interface EnumValue {
  value: number
  desc: string
}

export interface EnumSpec {
  values: EnumValue[]
}

export interface ThingDataTypeDetail {
  type: ThingDataType
  unit?: string
  unitName?: string
  min?: string
  max?: string
  trueValue?: string
  falseValue?: string
  length?: string
  enumSpec?: EnumSpec
  arraySpec?: {
    size?: string
    element: ThingDataTypeDetail
  }
  structSpec?: {
    fields: Array<{
      identifier: string
      name: string
      dataType: ThingDataTypeDetail
    }>
  }
}

export type RwFlag = 'r' | 'w' | 'rw'

/** 新增/导入属性时的入参形状;id 等系统字段由后端维护。 */
export interface ThingModelPropertyInput {
  identifier: string
  name: string
  dataType: ThingDataTypeDetail
  rwFlag?: RwFlag
  required?: boolean
  description?: string
  custom?: Record<string, unknown>
  sort?: number
}

export interface ThingModelProperty extends ThingModelPropertyInput {
  id: string
  version: number
  createdUser: string
  updatedUser: string
  createdTime: string
  updatedTime: string
}

export interface ThingModelFeature {
  identifier: string
  name: string
  dataType: ThingDataTypeDetail
  description?: string
  custom?: Record<string, unknown>
  sort?: number
}

export interface ThingModelInfo {
  id: string
  version: number
  thingModelKey: string
  name: string
  description: string
  propertyCount: number
  createdUser: string
  updatedUser: string
  createdTime: string
  updatedTime: string
}

export interface ThingModelDetail extends ThingModelInfo {
  properties: ThingModelProperty[]
  features: ThingModelFeature[]
}

export interface CreateThingModelRequest {
  thingModelKey?: string
  name: string
  description?: string
  properties?: ThingModelPropertyInput[]
  features?: ThingModelFeature[]
}

export interface UpdateThingModelRequest {
  name?: string
  description?: string
}

/** GET/PUT /iot/thingModels/:id/tsl 的导入导出文档。 */
export interface ThingModelDocument {
  thingModelKey?: string
  name?: string
  description?: string
  properties: ThingModelPropertyInput[]
  features?: ThingModelFeature[]
}

/* ---------- 设备(Thing) ---------- */

export interface ThingInfo {
  id: string
  version: number
  thingModelId: string
  deviceName: string
  deviceKey: string
  upTopic: string
  downTopic: string
  deviceDesc: string
  mqttClientId: string
  mqttUsername: string
  enable: 0 | 1
  onlineStatus: 0 | 1
  lastActiveTime: number | null
  createdUser: string
  updatedUser: string
  createdTime: string
  updatedTime: string
}

export interface ThingDetail extends ThingInfo {
  deviceSecret: string
  token: string
  mqttPassword: string
  image: string
  features?: Record<string, unknown>
}

export interface CreateThingItem {
  deviceName: string
  deviceKey?: string
  deviceDesc?: string
  features?: Record<string, unknown>
  image?: string
}

export interface CreateThingsRequest {
  thingModelId: string
  things: CreateThingItem[]
}

export interface UpdateThingRequest {
  deviceName?: string
  deviceDesc?: string
  mqttClientId?: string
  mqttUsername?: string
  mqttPassword?: string
  enable?: 0 | 1
  features?: Record<string, unknown>
  image?: string
}

export interface DeleteThingsResult {
  deleted: number
}

/* ---------- 文件 ---------- */

export interface UploadedFile {
  key: string
}
