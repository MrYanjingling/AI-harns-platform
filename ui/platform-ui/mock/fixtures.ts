import type {
  AuditLogInfo,
  DataPointTimeSeries,
  KpiInfo,
  MetricsInfo,
  OrgInfo,
  RoleInfo,
  TagInfo,
  ThingDetail,
  ThingModelDetail,
  ThingModelProperty,
  ThingModelFeature,
  UserInfo,
} from '../src/api/types'

/** mock 内存态设备:比 DeviceDetail 宽松,便于增删改。 */
export interface StoredDevice {
  name: string
  id: string
  eTag: string
  modTime: string
  deviceCode: string
  deviceType: 'modbus' | 'opcUa' | 's7'
  deviceModel: string
  collectStatus: 'collecting' | 'collectingError' | 'unconnected' | 'emptyVariable' | 'stopped' | 'error'
  topic?: string
  collectorCycle?: number
  variableInterval?: number
  address?: Record<string, unknown>
  slave?: number
  memoryLayout?: string
  positionAddress?: number
  variables?: Array<{
    name: string
    dataType: string
    address: unknown
    bits?: number
    functionCode?: number
    rate?: number
    amount?: number
    namespace?: number
    defaultValue?: unknown
    value?: unknown
    accessMode: 'r' | 'rw'
  }>
}

const now = () => new Date().toISOString()

export const gatewayMeta = {
  name: 'ai-harns-edge-01',
  id: 'f9f7d42d40b2415d90ed79a242292337',
  eTag: '1844674407370955161',
  secret: 'hg_5e1c9a77b2f0441db8c3e6a90d2f7c41',
  modTime: '2026-08-12T03:21:58.000Z',
}

export const seedDevices: StoredDevice[] = [
  {
    name: '1# 空压机',
    id: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
    eTag: '1001',
    modTime: '2026-10-19T08:12:31.000Z',
    deviceCode: 'air-compressor-01',
    deviceType: 'modbus',
    deviceModel: 'modbusTcp',
    collectStatus: 'collecting',
    collectorCycle: 1,
    variableInterval: 10,
    slave: 1,
    memoryLayout: 'ABCD',
    positionAddress: 0,
    address: { location: '192.168.10.21', option: { port: 502 } },
    variables: [
      { name: '运行状态', dataType: 'bool', address: 0, bits: 0, functionCode: 3, accessMode: 'r', value: true },
      { name: '排气压力', dataType: 'float32', address: 2, functionCode: 3, rate: 0.1, accessMode: 'r', value: 0.72 },
      { name: '排气温度', dataType: 'int16', address: 4, functionCode: 3, accessMode: 'r', value: 78 },
      { name: '频率给定', dataType: 'uint16', address: 6, functionCode: 3, accessMode: 'rw', value: 42 },
      { name: '启停控制', dataType: 'bool', address: 8, bits: 0, functionCode: 3, accessMode: 'rw', value: true },
    ],
  },
  {
    name: '2# 冷却泵',
    id: 'b2c3d4e5f60718293a4b5c6d7e8f9012',
    eTag: '1002',
    modTime: '2026-10-19T07:48:05.000Z',
    deviceCode: 'cooling-pump-02',
    deviceType: 'modbus',
    deviceModel: 'modbusRtu',
    collectStatus: 'collectingError',
    collectorCycle: 2,
    slave: 2,
    memoryLayout: 'CDAB',
    positionAddress: 0,
    address: {
      location: '/dev/ttyS0',
      option: { baudRate: 9600, dataBits: 8, parity: 'noParity', stopBits: '1' },
    },
    variables: [
      { name: '出口压力', dataType: 'float32', address: 0, functionCode: 3, accessMode: 'r', value: 0.41 },
      { name: '电机电流', dataType: 'int16', address: 2, functionCode: 3, rate: 0.01, accessMode: 'r' },
      { name: '运行频率', dataType: 'uint16', address: 4, functionCode: 3, accessMode: 'rw', value: 35 },
    ],
  },
  {
    name: '3# 注塑机',
    id: 'c3d4e5f60718293a4b5c6d7e8f901234',
    eTag: '1003',
    modTime: '2026-10-19T09:03:44.000Z',
    deviceCode: 'injection-molding-03',
    deviceType: 'opcUa',
    deviceModel: 'opcUa',
    collectStatus: 'collecting',
    collectorCycle: 1,
    variableInterval: 5,
    address: { location: 'opc.tcp://192.168.10.35:4840', option: { port: 4840, username: 'operator', password: '••••••' } },
    variables: [
      { name: '料筒温度', dataType: 'number', address: 'ns=2;s=Barrel.Temperature', namespace: 2, accessMode: 'r', value: 216.4 },
      { name: '注射压力', dataType: 'number', address: 'ns=2;s=Injection.Pressure', namespace: 2, accessMode: 'r', value: 88.2 },
      { name: '设定温度', dataType: 'number', address: 'ns=2;s=Barrel.TargetTemperature', namespace: 2, accessMode: 'rw', value: 220 },
      { name: '模号', dataType: 'string', address: 'ns=2;s=Mold.Code', namespace: 2, accessMode: 'r', value: 'M-2207A' },
    ],
  },
  {
    name: '4# 蒸汽锅炉',
    id: 'd4e5f60718293a4b5c6d7e8f90123456',
    eTag: '1004',
    modTime: '2026-10-18T22:15:10.000Z',
    deviceCode: 'steam-boiler-04',
    deviceType: 's7',
    deviceModel: 's71500',
    collectStatus: 'stopped',
    collectorCycle: 3,
    address: { location: '192.168.10.40', option: { port: 102, rack: 0, slot: 1 } },
    variables: [
      { name: '炉膛温度', dataType: 'float32', address: 'DB1.DBD0', rate: 1, accessMode: 'r', value: 642.5 },
      { name: '蒸汽压力', dataType: 'float32', address: 'DB1.DBD4', rate: 0.01, accessMode: 'r', value: 1.26 },
      { name: '给水阀开度', dataType: 'int16', address: 'DB1.DBW8', rate: 0.1, accessMode: 'rw', value: 55 },
    ],
  },
  {
    name: '5# 传送带',
    id: 'e5f60718293a4b5c6d7e8f9012345678',
    eTag: '1005',
    modTime: '2026-10-19T06:30:00.000Z',
    deviceCode: 'conveyor-05',
    deviceType: 'modbus',
    deviceModel: 'modbusRtuOverTcp',
    collectStatus: 'unconnected',
    collectorCycle: 2,
    slave: 5,
    memoryLayout: 'ABCD',
    address: { location: '192.168.10.51', option: { port: 8899 } },
    variables: [
      { name: '线速度', dataType: 'float32', address: 0, functionCode: 3, accessMode: 'r' },
      { name: '启停', dataType: 'bool', address: 2, bits: 0, functionCode: 3, accessMode: 'rw', value: false },
    ],
  },
  {
    name: '6# 包装线',
    id: 'f60718293a4b5c6d7e8f901234567890',
    eTag: '1006',
    modTime: '2026-10-17T14:02:19.000Z',
    deviceCode: 'packaging-line-06',
    deviceType: 'opcUa',
    deviceModel: 'opcUa',
    collectStatus: 'error',
    collectorCycle: 1,
    address: { location: 'opc.tcp://192.168.10.60:4840', option: { port: 4840 } },
    variables: [
      { name: '包装速度', dataType: 'number', address: 'ns=3;s=Line.Speed', namespace: 3, accessMode: 'r' },
      { name: '产量计数', dataType: 'number', address: 'ns=3;s=Line.Counter', namespace: 3, accessMode: 'r' },
    ],
  },
]

export const metrics: MetricsInfo[] = [
  {
    id: 'm-001',
    metricsCode: 'energy_total',
    metricsName: '总能耗',
    measureType: 'kWh',
    enable: true,
    metricsType: 'A',
    calculateType: 'sum',
    dataPoint: 'plant.energy',
    note: '全厂总用电量',
  },
  {
    id: 'm-002',
    metricsCode: 'power_factor',
    metricsName: '功率因数',
    measureType: '',
    enable: true,
    metricsType: 'A',
    calculateType: 'avg',
    dataPoint: 'plant.power-factor',
    note: '并网点平均功率因数',
  },
  {
    id: 'm-003',
    metricsCode: 'water_total',
    metricsName: '总用水量',
    measureType: 'm³',
    enable: true,
    metricsType: 'A',
    calculateType: 'sum',
    dataPoint: 'plant.water',
    note: '',
  },
  {
    id: 'm-004',
    metricsCode: 'steam_total',
    metricsName: '总蒸汽量',
    measureType: 't',
    enable: true,
    metricsType: 'A',
    calculateType: 'sum',
    dataPoint: 'plant.steam',
    note: '',
  },
  {
    id: 'm-005',
    metricsCode: 'run_hours',
    metricsName: '设备运行时长',
    measureType: 'h',
    enable: false,
    metricsType: 'M',
    calculateType: 'sum',
    dataPoint: 'plant.runtime',
    note: '手动抄表',
  },
]

export const kpis: KpiInfo[] = [
  {
    id: 'kpi_001',
    name: '日总能耗',
    formula: 'energy_total',
    formulaOri: 'A',
    timeGranularity: 'd',
    singleQuota: true,
    defaultTag: false,
    canDelete: true,
    unit: 'kWh',
    targetDataType: 'number',
    status: 'active',
    targetData: '52000',
  },
  {
    id: 'kpi_002',
    name: '单位产品能耗',
    formula: 'energy_total/output_count',
    formulaOri: 'A/B',
    timeGranularity: 'd',
    singleQuota: false,
    defaultTag: false,
    canDelete: true,
    unit: 'kWh/件',
    targetDataType: 'number',
    status: 'active',
    targetData: '1.8',
  },
  {
    id: 'kpi_003',
    name: '设备综合效率 (OEE)',
    formula: 'availability*performance*quality',
    formulaOri: 'A*B*C',
    timeGranularity: 'd',
    singleQuota: false,
    defaultTag: true,
    canDelete: false,
    unit: '%',
    targetDataType: 'number',
    status: 'active',
    targetData: '85',
  },
  {
    id: 'kpi_004',
    name: '峰谷电费占比',
    formula: 'peak_cost/total_cost',
    formulaOri: 'A/B',
    timeGranularity: 'm',
    singleQuota: false,
    defaultTag: false,
    canDelete: true,
    unit: '%',
    targetDataType: 'number',
    status: 'draft',
  },
]

export const tags: TagInfo[] = [
  { id: 't-001', tagId: 'SCADA.compressor.pressure', tagName: '空压机排气压力', tagDesc: '1# 空压机', tagType: 'analog', tagAttribute: 'float', enable: 1, scadaId: 'scada-01', lastSynTime: '2026-10-19T09:20:00.000Z' },
  { id: 't-002', tagId: 'SCADA.compressor.temperature', tagName: '空压机排气温度', tagDesc: '1# 空压机', tagType: 'analog', tagAttribute: 'float', enable: 1, scadaId: 'scada-01', lastSynTime: '2026-10-19T09:20:00.000Z' },
  { id: 't-003', tagId: 'SCADA.chiller.flow', tagName: '冷却水流量', tagDesc: '循环水系统', tagType: 'analog', tagAttribute: 'float', enable: 1, scadaId: 'scada-01', lastSynTime: '2026-10-19T09:20:00.000Z' },
  { id: 't-004', tagId: 'SCADA.boiler.steam', tagName: '蒸汽流量', tagDesc: '4# 锅炉', tagType: 'analog', tagAttribute: 'float', enable: 1, scadaId: 'scada-02', lastSynTime: '2026-10-19T09:19:30.000Z' },
  { id: 't-005', tagId: 'SCADA.boiler.level', tagName: '汽包液位', tagDesc: '4# 锅炉', tagType: 'analog', tagAttribute: 'float', enable: 1, scadaId: 'scada-02', lastSynTime: '2026-10-19T09:19:30.000Z' },
  { id: 't-006', tagId: 'SCADA.molding.pressure', tagName: '注塑压力', tagDesc: '3# 注塑机', tagType: 'analog', tagAttribute: 'float', enable: 1, scadaId: 'scada-02', lastSynTime: '2026-10-19T09:18:00.000Z' },
  { id: 't-007', tagId: 'SCADA.molding.status', tagName: '注塑机运行状态', tagDesc: '3# 注塑机', tagType: 'digital', tagAttribute: 'bool', enable: 1, scadaId: 'scada-02', lastSynTime: '2026-10-19T09:18:00.000Z' },
  { id: 't-008', tagId: 'SCADA.legacy.temp', tagName: '旧线温度', tagDesc: '已停用产线', tagType: 'analog', tagAttribute: 'float', enable: 0, scadaId: 'scada-01', lastSynTime: '2026-09-30T11:02:00.000Z' },
]

const TAG_BASE: Record<string, { name: string; desc: string; base: number; amp: number }> = {
  'SCADA.compressor.pressure': { name: '空压机排气压力', desc: '1# 空压机', base: 0.72, amp: 0.06 },
  'SCADA.compressor.temperature': { name: '空压机排气温度', desc: '1# 空压机', base: 78, amp: 4 },
  'SCADA.chiller.flow': { name: '冷却水流量', desc: '循环水系统', base: 126, amp: 9 },
  'SCADA.boiler.steam': { name: '蒸汽流量', desc: '4# 锅炉', base: 12.4, amp: 1.1 },
  'SCADA.boiler.level': { name: '汽包液位', desc: '4# 锅炉', base: 52, amp: 3 },
  'SCADA.molding.pressure': { name: '注塑压力', desc: '3# 注塑机', base: 88, amp: 6 },
  'SCADA.molding.status': { name: '注塑机运行状态', desc: '3# 注塑机', base: 1, amp: 0 },
  'SCADA.legacy.temp': { name: '旧线温度', desc: '已停用产线', base: 30, amp: 0.2 },
}

const pad = (n: number) => String(n).padStart(2, '0')
function formatTime(d: Date) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

/** 生成最近 24 小时的确定性时序数据。 */
export function buildTimeSeries(select: string, limit = 48): DataPointTimeSeries[] {
  const ids = select
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
    .slice(0, 10)
  const nowDate = new Date()
  const count = Math.max(2, Math.min(limit || 48, 48))
  const stepMs = (24 * 60 * 60 * 1000) / (count - 1)

  return ids.map((tagId) => {
    const meta = TAG_BASE[tagId] ?? { name: tagId, desc: '', base: 50, amp: 5 }
    return {
      tagId,
      tagName: meta.name,
      tagDesc: meta.desc,
      data: Array.from({ length: count }, (_, i) => {
        const time = new Date(nowDate.getTime() - (count - 1 - i) * stepMs)
        const wave = Math.sin(i / 3.2) * meta.amp + Math.sin(i / 1.7) * meta.amp * 0.35
        const value = meta.amp === 0 ? meta.base : meta.base + wave
        return {
          time: formatTime(time),
          value: value.toFixed(meta.base >= 10 ? 2 : 3),
        }
      }),
    }
  })
}

export function createDeviceId(): string {
  const hex = '0123456789abcdef'
  return Array.from({ length: 32 }, () => hex[Math.floor(Math.random() * 16)]).join('')
}

export function nowIso() {
  return now()
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
