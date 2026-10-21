import type {
  AuditLogInfo,
  OrgInfo,
  RoleInfo,
  ThingDetail,
  ThingModelDetail,
  ThingModelProperty,
  ThingModelFeature,
  UserInfo,
} from '../src/api/types'

export const gatewayMeta = {
  name: 'ai-harns-edge-01',
  id: 'f9f7d42d40b2415d90ed79a242292337',
  eTag: '1844674407370955161',
  secret: 'hg_5e1c9a77b2f0441db8c3e6a90d2f7c41',
  modTime: '2026-08-12T03:21:58.000Z',
}

export function createHexId(): string {
  const hex = '0123456789abcdef'
  return Array.from({ length: 32 }, () => hex[Math.floor(Math.random() * 16)]).join('')
}

/* ============================================================
 * 平台服务种子数据(物模型 / 设备 / 用户 / 角色 / 组织 / 审计)
 * 与 Go 后端 JSON 形状一致,供 VITE_USE_MOCK=true 时演示。
 * ============================================================ */

const PLATFORM_TIME = '2026-10-28T09:30:00.000Z'

function makeProperty(
  id: string,
  identifier: string,
  name: string,
  dataType: ThingModelProperty['dataType'],
  extra: Partial<ThingModelProperty> = {},
): ThingModelProperty {
  return {
    id,
    version: 0,
    identifier,
    name,
    dataType,
    rwFlag: 'r',
    required: false,
    description: '',
    sort: 0,
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: PLATFORM_TIME,
    updatedTime: PLATFORM_TIME,
    ...extra,
  }
}

const THERMOSTAT_FEATURES: ThingModelFeature[] = [
  {
    identifier: 'ratedVoltage',
    name: '额定电压',
    dataType: { type: 'double', unit: 'V' },
    description: '设备额定电压',
    sort: 0,
  },
  {
    identifier: 'ratedPower',
    name: '额定功率',
    dataType: { type: 'double', unit: 'kW' },
    description: '设备额定功率',
    sort: 1,
  },
]

export const seedThingModels: ThingModelDetail[] = [
  {
    id: '01K9TM00000000000000000001',
    version: 2,
    thingModelKey: 'tm_thermostat',
    name: '温控器',
    description: '环境温控设备,采集温度并支持模式与配置下发。',
    propertyCount: 4,
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: PLATFORM_TIME,
    updatedTime: '2026-11-02T02:10:00.000Z',
    properties: [
      makeProperty(
        '01K9TP00000000000000000001',
        'temperature',
        '温度',
        { type: 'double', unit: '℃', unitName: '摄氏度', min: '-20', max: '120' },
        { required: true, description: '环境温度', sort: 0 },
      ),
      makeProperty(
        '01K9TP00000000000000000002',
        'mode',
        '模式',
        {
          type: 'enum',
          enumSpec: {
            values: [
              { value: 0, desc: '自动' },
              { value: 1, desc: '手动' },
              { value: 2, desc: '停止' },
            ],
          },
        },
        { rwFlag: 'rw', sort: 1 },
      ),
      makeProperty(
        '01K9TP00000000000000000003',
        'history',
        '历史值',
        { type: 'array', arraySpec: { size: '10', element: { type: 'double', unit: '℃' } } },
        { sort: 2 },
      ),
      makeProperty(
        '01K9TP00000000000000000004',
        'config',
        '配置',
        {
          type: 'struct',
          structSpec: {
            fields: [
              {
                identifier: 'high',
                name: '上限',
                dataType: { type: 'double', min: '0', max: '100' },
              },
            ],
          },
        },
        { rwFlag: 'rw', sort: 3 },
      ),
    ],
    features: THERMOSTAT_FEATURES,
  },
  {
    id: '01K9TM00000000000000000002',
    version: 1,
    thingModelKey: 'tm_meter',
    name: '智能电表',
    description: '三相智能电表,采集电压、电流、功率与累计电量。',
    propertyCount: 4,
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: '2026-10-29T01:20:00.000Z',
    updatedTime: '2026-11-01T06:45:00.000Z',
    properties: [
      makeProperty(
        '01K9TP00000000000000000011',
        'voltage',
        '电压',
        { type: 'double', unit: 'V', min: '0', max: '500' },
        { sort: 0 },
      ),
      makeProperty(
        '01K9TP00000000000000000012',
        'current',
        '电流',
        { type: 'double', unit: 'A', min: '0', max: '200' },
        { sort: 1 },
      ),
      makeProperty(
        '01K9TP00000000000000000013',
        'power',
        '有功功率',
        { type: 'double', unit: 'kW', min: '0' },
        { sort: 2 },
      ),
      makeProperty(
        '01K9TP00000000000000000014',
        'totalEnergy',
        '累计电量',
        { type: 'double', unit: 'kWh', min: '0' },
        { required: true, sort: 3 },
      ),
    ],
    features: [
      {
        identifier: 'meterModel',
        name: '电表型号',
        dataType: { type: 'text', length: '32' },
        sort: 0,
      },
      {
        identifier: 'manufacturer',
        name: '生产厂商',
        dataType: { type: 'text', length: '64' },
        sort: 1,
      },
    ],
  },
  {
    id: '01K9TM00000000000000000003',
    version: 3,
    thingModelKey: 'tm_pump',
    name: '水泵',
    description: '循环水泵,支持启停、流量与压力采集。',
    propertyCount: 4,
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: '2026-10-30T03:10:00.000Z',
    updatedTime: '2026-11-03T08:05:00.000Z',
    properties: [
      makeProperty(
        '01K9TP00000000000000000021',
        'running',
        '运行状态',
        { type: 'bool', trueValue: '运行', falseValue: '停止' },
        { rwFlag: 'rw', sort: 0 },
      ),
      makeProperty(
        '01K9TP00000000000000000022',
        'flow',
        '流量',
        { type: 'double', unit: 'm³/h', min: '0' },
        { sort: 1 },
      ),
      makeProperty(
        '01K9TP00000000000000000023',
        'pressure',
        '出口压力',
        { type: 'double', unit: 'MPa', min: '0', max: '2' },
        { sort: 2 },
      ),
      makeProperty(
        '01K9TP00000000000000000024',
        'speed',
        '转速',
        { type: 'int', unit: 'r/min', min: '0', max: '3000' },
        { rwFlag: 'rw', sort: 3 },
      ),
    ],
    features: [
      {
        identifier: 'ratedPower',
        name: '额定功率',
        dataType: { type: 'double', unit: 'kW' },
        sort: 0,
      },
    ],
  },
]

function makeThing(partial: Partial<ThingDetail> & Pick<ThingDetail, 'id' | 'thingModelId' | 'deviceName' | 'deviceKey'>): ThingDetail {
  const deviceSecret = partial.deviceSecret ?? '4f2a8c9e1b7d6a3f0e5c2b9d8a7f6e1c4b3a2d9f8e7c6b5a4d3f2e1c0b9a8d7f'
  return {
    version: 1,
    upTopic: `iot/${partial.deviceKey}/up`,
    downTopic: `iot/${partial.deviceKey}/down`,
    deviceDesc: '',
    mqttClientId: partial.id,
    mqttUsername: partial.deviceKey,
    enable: 1,
    onlineStatus: 0,
    lastActiveTime: null,
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: PLATFORM_TIME,
    updatedTime: PLATFORM_TIME,
    deviceSecret,
    token: '9c1e7b3a5d2f8e4c6a0b9d7e5f3a1c2b8d6e4f0a9c7b5d3e1f2a0c8b6d4e2f1a',
    mqttPassword: deviceSecret,
    image: '',
    features: {},
    ...partial,
  }
}

const ACTIVE_NOW = Date.now()

export const seedThings: ThingDetail[] = [
  makeThing({
    id: '01K9TH00000000000000000001',
    thingModelId: '01K9TM00000000000000000001',
    deviceName: '1# 温控器',
    deviceKey: 'thermostat_01',
    deviceDesc: '一号车间温控器',
    version: 3,
    enable: 1,
    onlineStatus: 1,
    lastActiveTime: ACTIVE_NOW - 30_000,
    features: { ratedVoltage: 380, ratedPower: 2.2 },
    image: 'images/2026/11/mock-thermostat-01.png',
  }),
  makeThing({
    id: '01K9TH00000000000000000002',
    thingModelId: '01K9TM00000000000000000001',
    deviceName: '2# 温控器',
    deviceKey: 'thermostat_02',
    deviceDesc: '二号车间温控器',
    version: 1,
    enable: 1,
    onlineStatus: 0,
    lastActiveTime: ACTIVE_NOW - 10 * 60_000,
    features: { ratedVoltage: 220, ratedPower: 1.5 },
  }),
  makeThing({
    id: '01K9TH00000000000000000003',
    thingModelId: '01K9TM00000000000000000002',
    deviceName: '总进线电表',
    deviceKey: 'meter_main',
    deviceDesc: '厂房总进线计量',
    version: 2,
    enable: 1,
    onlineStatus: 1,
    lastActiveTime: ACTIVE_NOW - 5_000,
    features: { meterModel: 'DTS-100', manufacturer: 'Harns' },
  }),
  makeThing({
    id: '01K9TH00000000000000000004',
    thingModelId: '01K9TM00000000000000000002',
    deviceName: '车间电表',
    deviceKey: 'meter_shop',
    deviceDesc: '装配车间分表',
    version: 1,
    enable: 0,
    onlineStatus: 0,
    lastActiveTime: null,
    features: { meterModel: 'DTS-60', manufacturer: 'Harns' },
  }),
  makeThing({
    id: '01K9TH00000000000000000005',
    thingModelId: '01K9TM00000000000000000003',
    deviceName: '1# 循环泵',
    deviceKey: 'pump_circ_01',
    deviceDesc: '冷却水循环泵',
    version: 4,
    enable: 1,
    onlineStatus: 1,
    lastActiveTime: ACTIVE_NOW - 2_000,
    features: { ratedPower: 5.5 },
  }),
  makeThing({
    id: '01K9TH00000000000000000006',
    thingModelId: '01K9TM00000000000000000003',
    deviceName: '2# 循环泵',
    deviceKey: 'pump_circ_02',
    deviceDesc: '备用循环泵',
    version: 1,
    enable: 1,
    onlineStatus: 0,
    lastActiveTime: ACTIVE_NOW - 60 * 60_000,
    features: { ratedPower: 5.5 },
  }),
]

export const seedRoles: RoleInfo[] = [
  {
    id: '01M6KGSCACPFET7BPJG8TQRSP3',
    version: 0,
    name: '超级管理员',
    code: 'admin',
    description: '内置超级管理员,拥有全部权限',
    status: 'enabled',
    createdUser: 'system',
    updatedUser: 'system',
    createdTime: PLATFORM_TIME,
    updatedTime: PLATFORM_TIME,
  },
  {
    id: '01K9RO00000000000000000002',
    version: 1,
    name: '运维人员',
    code: 'operator',
    description: '维护物模型与设备',
    status: 'enabled',
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: PLATFORM_TIME,
    updatedTime: PLATFORM_TIME,
  },
  {
    id: '01K9RO00000000000000000003',
    version: 1,
    name: '只读用户',
    code: 'viewer',
    description: '只读查看平台数据',
    status: 'enabled',
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: PLATFORM_TIME,
    updatedTime: PLATFORM_TIME,
  },
]

export const seedOrgs: OrgInfo[] = [
  {
    id: '01K9OR00000000000000000001',
    version: 0,
    code: 'HQ',
    name: '总部',
    parentId: '',
    sort: 0,
    description: '集团总部',
    remark: '',
    status: 'enabled',
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: PLATFORM_TIME,
    updatedTime: PLATFORM_TIME,
  },
  {
    id: '01K9OR00000000000000000002',
    version: 1,
    code: 'PROD',
    name: '生产部',
    parentId: '01K9OR00000000000000000001',
    sort: 0,
    description: '生产管理',
    remark: '',
    status: 'enabled',
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: PLATFORM_TIME,
    updatedTime: PLATFORM_TIME,
  },
  {
    id: '01K9OR00000000000000000003',
    version: 1,
    code: 'EQUIP',
    name: '设备部',
    parentId: '01K9OR00000000000000000001',
    sort: 1,
    description: '设备维护',
    remark: '',
    status: 'enabled',
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: PLATFORM_TIME,
    updatedTime: PLATFORM_TIME,
  },
  {
    id: '01K9OR00000000000000000004',
    version: 0,
    code: 'ASSEMBLY',
    name: '装配车间',
    parentId: '01K9OR00000000000000000002',
    sort: 0,
    description: '装配车间',
    remark: '',
    status: 'enabled',
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: PLATFORM_TIME,
    updatedTime: PLATFORM_TIME,
  },
  {
    id: '01K9OR00000000000000000005',
    version: 0,
    code: 'EAST',
    name: '华东工厂',
    parentId: '',
    sort: 1,
    description: '华东生产基地',
    remark: '',
    status: 'enabled',
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: PLATFORM_TIME,
    updatedTime: PLATFORM_TIME,
  },
]

function roleById(id: string): RoleInfo | undefined {
  return seedRoles.find((role) => role.id === id)
}

function orgById(id: string): OrgInfo | undefined {
  return seedOrgs.find((org) => org.id === id)
}

function makeUser(partial: Partial<UserInfo> & Pick<UserInfo, 'id' | 'userName' | 'userNo' | 'nickName'>): UserInfo {
  const roleIds = partial.roleIds ?? []
  const orgIds = partial.orgIds ?? []
  return {
    version: 0,
    birthDate: '',
    gender: '',
    mail: '',
    phone: '',
    note: '',
    authMode: 'local',
    lastLoginTime: null,
    loginFailureTimes: 0,
    expireTimes: 0,
    status: 'enabled',
    createdUser: 'system',
    updatedUser: 'system',
    createdTime: PLATFORM_TIME,
    updatedTime: PLATFORM_TIME,
    roleIds,
    orgIds,
    roles: roleIds.map(roleById).filter((role): role is RoleInfo => Boolean(role)),
    orgs: orgIds.map(orgById).filter((org): org is OrgInfo => Boolean(org)),
    ...partial,
  }
}

export const seedUsers: UserInfo[] = [
  makeUser({
    id: '01M6KGSCACPFET7BPJGBQ15WMF',
    userName: 'admin',
    userNo: 'admin',
    nickName: '超级管理员',
    roleIds: ['01M6KGSCACPFET7BPJG8TQRSP3'],
    orgIds: ['01K9OR00000000000000000001'],
    lastLoginTime: '2026-11-04T09:12:00.000Z',
    mail: 'admin@harns.local',
  }),
  makeUser({
    id: '01K9US00000000000000000002',
    userName: 'operator',
    userNo: 'E1001',
    nickName: '值班员',
    roleIds: ['01K9RO00000000000000000002'],
    orgIds: ['01K9OR00000000000000000002'],
    phone: '13800000001',
    lastLoginTime: '2026-11-03T22:40:00.000Z',
  }),
  makeUser({
    id: '01K9US00000000000000000003',
    userName: 'viewer',
    userNo: 'E1002',
    nickName: '访客',
    roleIds: ['01K9RO00000000000000000003'],
    orgIds: ['01K9OR00000000000000000003'],
    status: 'disabled',
  }),
]

function makeAudit(
  partial: Partial<AuditLogInfo> & Pick<AuditLogInfo, 'id' | 'operationType' | 'method' | 'path'>,
): AuditLogInfo {
  return {
    app: 'IOT',
    traceId: `trace-${partial.id}`,
    userId: '01M6KGSCACPFET7BPJGBQ15WMF',
    userName: 'admin',
    operationDesc: '',
    query: '',
    ip: '127.0.0.1',
    userAgent: 'Mozilla/5.0 (Macintosh)',
    status: 200,
    resultCode: 200,
    errorMsg: '',
    durationMs: 12,
    createdTime: '2026-11-04T09:12:00.000Z',
    ...partial,
  }
}

export const seedAuditLogs: AuditLogInfo[] = [
  makeAudit({
    id: '01K9AU00000000000000000001',
    operationType: 'LOGIN',
    operationDesc: '用户登录',
    method: 'POST',
    path: '/api/v1/auth/login',
    requestBody: '{"username":"admin","password":"***"}',
    responseBody: '{"code":200,"message":"success"}',
    createdTime: '2026-11-04T09:12:00.000Z',
    durationMs: 48,
  }),
  makeAudit({
    id: '01K9AU00000000000000000002',
    operationType: 'CREATE',
    operationDesc: '创建物模型',
    method: 'POST',
    path: '/api/v1/iot/thingModels',
    requestBody: '{"name":"水泵","thingModelKey":"tm_pump"}',
    responseBody: '{"code":200,"message":"success"}',
    createdTime: '2026-11-03T08:05:00.000Z',
    durationMs: 21,
  }),
  makeAudit({
    id: '01K9AU00000000000000000003',
    operationType: 'UPDATE',
    operationDesc: '更新设备',
    method: 'PUT',
    path: '/api/v1/iot/things/01K9TH00000000000000000001',
    requestBody: '{"enable":1}',
    responseBody: '{"code":200,"message":"success"}',
    createdTime: '2026-11-02T14:22:00.000Z',
    durationMs: 17,
  }),
  makeAudit({
    id: '01K9AU00000000000000000004',
    operationType: 'DELETE',
    operationDesc: '批量删除设备',
    method: 'DELETE',
    path: '/api/v1/iot/things',
    query: 'cascade=false',
    requestBody: '["01K9TH00000000000000000009"]',
    responseBody: '{"code":200,"message":"success","data":{"deleted":1}}',
    createdTime: '2026-11-01T11:03:00.000Z',
    durationMs: 26,
  }),
  makeAudit({
    id: '01K9AU00000000000000000005',
    operationType: 'UPDATE',
    operationDesc: '更新用户',
    method: 'PUT',
    path: '/api/v1/iot/users/01K9US00000000000000000002',
    status: 400,
    resultCode: 10019,
    errorMsg: '用户名称已存在。',
    requestBody: '{"nickName":"值班员"}',
    responseBody: '{"code":10019,"message":"用户名称已存在。"}',
    createdTime: '2026-10-31T16:10:00.000Z',
    durationMs: 9,
  }),
  makeAudit({
    id: '01K9AU00000000000000000006',
    operationType: 'LOGOUT',
    operationDesc: '用户退出',
    method: 'POST',
    path: '/api/v1/auth/logout',
    createdTime: '2026-10-30T18:00:00.000Z',
    durationMs: 3,
  }),
]
