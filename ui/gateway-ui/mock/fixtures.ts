import type {
  CollectionDetail,
  CollectionVariable,
  Driver,
  DriverSchema,
  GatewayMeta,
  PushConnector,
  PushGroup,
} from '../src/api/types'

export const gatewayMeta: GatewayMeta = {
  name: 'harnsgateway',
  id: 'f9f7d42d40b2415d90ed79a242292337',
  eTag: '1732684500123456789',
  secret: 'gw-secret-5f8c1d7a9b2e4c6d',
  modTime: new Date(Date.now() - 1000 * 60 * 12).toISOString(),
}

export function iso(minutesAgo: number): string {
  return new Date(Date.now() - minutesAgo * 60_000).toISOString()
}

export function ulid(): string {
  return Array.from({ length: 32 }, () => Math.floor(Math.random() * 16).toString(16)).join('')
}

const MEMORY_LAYOUT = {
  name: 'memoryLayout',
  labelZh: '内存布局',
  labelEn: 'Memory layout',
  regex: '^(ABCD|CDAB|BADC|DCBA)$',
  sort: 6,
  type: 'select' as const,
  required: true,
  options: [
    { value: 'ABCD', labelZh: 'ABCD', labelEn: 'ABCD' },
    { value: 'CDAB', labelZh: 'CDAB', labelEn: 'CDAB' },
    { value: 'BADC', labelZh: 'BADC', labelEn: 'BADC' },
    { value: 'DCBA', labelZh: 'DCBA', labelEn: 'DCBA' },
  ],
}

const MODBUS_TYPES: DriverSchema['variableDataTypes'] = [
  { value: 'bool', labelZh: '布尔', labelEn: 'Boolean', addressRegex: '^\\d+(\\.\\d+)?$' },
  { value: 'uint16', labelZh: '无符号 16 位', labelEn: 'Unsigned 16-bit', addressRegex: '^\\d+$' },
  { value: 'int16', labelZh: '有符号 16 位', labelEn: 'Signed 16-bit', addressRegex: '^\\d+$' },
  { value: 'uint32', labelZh: '无符号 32 位', labelEn: 'Unsigned 32-bit', addressRegex: '^\\d+$' },
  { value: 'int32', labelZh: '有符号 32 位', labelEn: 'Signed 32-bit', addressRegex: '^\\d+$' },
  { value: 'float32', labelZh: '单精度浮点', labelEn: 'Float 32-bit', addressRegex: '^\\d+$' },
  { value: 'float64', labelZh: '双精度浮点', labelEn: 'Float 64-bit', addressRegex: '^\\d+$' },
]

const MODBUS_ADDRESS = {
  labelZh: '寄存器地址',
  labelEn: 'Register address',
  regex: '^\\d+$',
  placeholderZh: '例如 40001',
  placeholderEn: 'e.g. 40001',
  descriptionZh: '填写从站寄存器偏移;布尔量可用「地址.位」表示。',
  descriptionEn: 'Register offset on the slave; use "address.bit" for coils.',
}

const SERIAL_FIELDS = [
  {
    name: 'baudRate',
    labelZh: '波特率',
    labelEn: 'Baud rate',
    regex: '^\\d+$',
    sort: 2,
    type: 'select' as const,
    required: true,
    options: ['4800', '9600', '19200', '38400', '115200'].map((value) => ({
      value,
      labelZh: value,
      labelEn: value,
    })),
  },
  {
    name: 'dataBits',
    labelZh: '数据位',
    labelEn: 'Data bits',
    regex: '^(7|8)$',
    sort: 3,
    type: 'select' as const,
    required: true,
    options: [
      { value: '7', labelZh: '7', labelEn: '7' },
      { value: '8', labelZh: '8', labelEn: '8' },
    ],
  },
  {
    name: 'parity',
    labelZh: '校验位',
    labelEn: 'Parity',
    regex: '^(noParity|odd|even)$',
    sort: 4,
    type: 'select' as const,
    required: true,
    options: [
      { value: 'noParity', labelZh: '无校验', labelEn: 'None' },
      { value: 'odd', labelZh: '奇校验', labelEn: 'Odd' },
      { value: 'even', labelZh: '偶校验', labelEn: 'Even' },
    ],
  },
  {
    name: 'stopBits',
    labelZh: '停止位',
    labelEn: 'Stop bits',
    regex: '^(1|2)$',
    sort: 5,
    type: 'select' as const,
    required: true,
    options: [
      { value: '1', labelZh: '1', labelEn: '1' },
      { value: '2', labelZh: '2', labelEn: '2' },
    ],
  },
]

const LOCATION_FIELD = {
  name: 'location',
  labelZh: '设备地址',
  labelEn: 'Device address',
  regex: '^.+$',
  sort: 1,
  type: 'input' as const,
  required: true,
}

const PORT_FIELD = {
  name: 'port',
  labelZh: '端口',
  labelEn: 'Port',
  regex: '^\\d+$',
  sort: 2,
  type: 'input' as const,
  required: true,
}

const SLAVE_FIELD = {
  name: 'slave',
  labelZh: '从站地址',
  labelEn: 'Slave address',
  regex: '^\\d+$',
  sort: 5,
  type: 'input' as const,
  required: true,
}

const TCP_COUNT_FIELD = {
  name: 'tcpCount',
  labelZh: 'TCP 连接数',
  labelEn: 'TCP connections',
  regex: '^\\d+$',
  sort: 7,
  type: 'input' as const,
}

const POSITION_FIELD = {
  name: 'positionAddress',
  labelZh: '起始地址偏移',
  labelEn: 'Position offset',
  regex: '^\\d+$',
  sort: 8,
  type: 'input' as const,
}

const S7_TYPES: DriverSchema['variableDataTypes'] = [
  { value: 'bool', labelZh: '布尔', labelEn: 'Boolean', addressRegex: '^DB\\d+\\.DBX\\d+\\.\\d+$' },
  { value: 'int16', labelZh: '有符号 16 位', labelEn: 'Signed 16-bit', addressRegex: '^DB\\d+\\.DBW\\d+$' },
  { value: 'uint16', labelZh: '无符号 16 位', labelEn: 'Unsigned 16-bit', addressRegex: '^DB\\d+\\.DBW\\d+$' },
  { value: 'int32', labelZh: '有符号 32 位', labelEn: 'Signed 32-bit', addressRegex: '^DB\\d+\\.DBD\\d+$' },
  { value: 'uint32', labelZh: '无符号 32 位', labelEn: 'Unsigned 32-bit', addressRegex: '^DB\\d+\\.DBD\\d+$' },
  { value: 'float32', labelZh: '单精度浮点', labelEn: 'Float 32-bit', addressRegex: '^DB\\d+\\.DBD\\d+$' },
]

const S7_ADDRESS = {
  labelZh: 'DB 地址',
  labelEn: 'DB address',
  regex: '^DB\\d+\\.DB[XBWD]\\d+(\\.\\d+)?$',
  placeholderZh: '例如 DB1.DBD0 / DB1.DBX4.0',
  placeholderEn: 'e.g. DB1.DBD0 / DB1.DBX4.0',
  descriptionZh: '按 S7 数据块寻址:DBX 位、DBW 字、DBD 双字。',
  descriptionEn: 'S7 data block addressing: DBX bit, DBW word, DBD double word.',
}

const S7_FIELDS = [
  LOCATION_FIELD,
  { ...PORT_FIELD, sort: 2, required: false },
  {
    name: 'rack',
    labelZh: '机架号',
    labelEn: 'Rack',
    regex: '^\\d+$',
    sort: 3,
    type: 'input' as const,
    required: true,
  },
  {
    name: 'slot',
    labelZh: '槽位号',
    labelEn: 'Slot',
    regex: '^\\d+$',
    sort: 4,
    type: 'input' as const,
    required: true,
  },
  {
    name: 'connectionType',
    labelZh: '连接类型',
    labelEn: 'Connection type',
    regex: '^[123]$',
    sort: 5,
    type: 'select' as const,
    required: true,
    options: [
      { value: '1', labelZh: 'PG', labelEn: 'PG' },
      { value: '2', labelZh: 'OP', labelEn: 'OP' },
      { value: '3', labelZh: 'Basic', labelEn: 'Basic' },
    ],
  },
  {
    name: 'pduSize',
    labelZh: 'PDU 大小',
    labelEn: 'PDU size',
    regex: '^\\d+$',
    sort: 6,
    type: 'input' as const,
  },
  {
    name: 'localTsap',
    labelZh: '本地 TSAP',
    labelEn: 'Local TSAP',
    regex: '^(0x)?[0-9a-fA-F]+$',
    sort: 7,
    type: 'input' as const,
  },
  {
    name: 'remoteTsap',
    labelZh: '远端 TSAP',
    labelEn: 'Remote TSAP',
    regex: '^(0x)?[0-9a-fA-F]+$',
    sort: 8,
    type: 'input' as const,
  },
]

const S7_200_FIELDS = [
  LOCATION_FIELD,
  { ...PORT_FIELD, sort: 2, required: false },
  {
    name: 'localTsap',
    labelZh: '本地 TSAP',
    labelEn: 'Local TSAP',
    regex: '^(0x)?[0-9a-fA-F]+$',
    sort: 3,
    type: 'input' as const,
    required: true,
  },
  {
    name: 'remoteTsap',
    labelZh: '远端 TSAP',
    labelEn: 'Remote TSAP',
    regex: '^(0x)?[0-9a-fA-F]+$',
    sort: 4,
    type: 'input' as const,
    required: true,
  },
  {
    name: 'pduSize',
    labelZh: 'PDU 大小',
    labelEn: 'PDU size',
    regex: '^\\d+$',
    sort: 5,
    type: 'input' as const,
  },
]

const OPCUA_SCHEMA: DriverSchema = {
  fields: [
    LOCATION_FIELD,
    { ...PORT_FIELD, sort: 2, required: true },
    {
      name: 'username',
      labelZh: '用户名',
      labelEn: 'Username',
      regex: '^.*$',
      sort: 3,
      type: 'input' as const,
    },
    {
      name: 'password',
      labelZh: '密码',
      labelEn: 'Password',
      regex: '^.*$',
      sort: 4,
      type: 'input' as const,
    },
    {
      name: 'securityPolicy',
      labelZh: '安全策略',
      labelEn: 'Security policy',
      regex: '^(None|Basic256Sha256|Aes128Sha256RsaOaep)$',
      sort: 5,
      type: 'select' as const,
      options: [
        { value: 'None', labelZh: 'None', labelEn: 'None' },
        { value: 'Basic256Sha256', labelZh: 'Basic256Sha256', labelEn: 'Basic256Sha256' },
        {
          value: 'Aes128Sha256RsaOaep',
          labelZh: 'Aes128Sha256RsaOaep',
          labelEn: 'Aes128Sha256RsaOaep',
        },
      ],
    },
  ],
  variableAddress: {
    labelZh: '节点 ID',
    labelEn: 'Node ID',
    regex: '^ns=\\d+;[isgb]=.+$',
    placeholderZh: '例如 ns=2;s=Boiler.Temp',
    placeholderEn: 'e.g. ns=2;s=Boiler.Temp',
    descriptionZh: '使用 OPC UA NodeId 语法。',
    descriptionEn: 'Use OPC UA NodeId syntax.',
  },
  variableDataTypes: [
    { value: 'number', labelZh: '数值', labelEn: 'Number', addressRegex: '^ns=\\d+;[isgb]=.+$' },
    { value: 'boolean', labelZh: '布尔', labelEn: 'Boolean', addressRegex: '^ns=\\d+;[isgb]=.+$' },
    { value: 'string', labelZh: '字符串', labelEn: 'String', addressRegex: '^ns=\\d+;[isgb]=.+$' },
  ],
}

const MODBUS_TCP_SCHEMA: DriverSchema = {
  fields: [
    LOCATION_FIELD,
    { ...PORT_FIELD, sort: 2, required: true },
    { ...SLAVE_FIELD, sort: 3 },
    { ...TCP_COUNT_FIELD, sort: 4 },
    MEMORY_LAYOUT,
    { ...POSITION_FIELD, sort: 7 },
  ],
  variableAddress: MODBUS_ADDRESS,
  variableDataTypes: MODBUS_TYPES,
}

const MODBUS_RTU_SCHEMA: DriverSchema = {
  fields: [LOCATION_FIELD, ...SERIAL_FIELDS, SLAVE_FIELD, MEMORY_LAYOUT, POSITION_FIELD],
  variableAddress: MODBUS_ADDRESS,
  variableDataTypes: MODBUS_TYPES,
}

const MODBUS_RTU_TCP_SCHEMA: DriverSchema = {
  fields: [
    LOCATION_FIELD,
    { ...PORT_FIELD, sort: 2, required: true },
    ...SERIAL_FIELDS,
    SLAVE_FIELD,
    MEMORY_LAYOUT,
    { ...TCP_COUNT_FIELD, sort: 8 },
  ],
  variableAddress: MODBUS_ADDRESS,
  variableDataTypes: MODBUS_TYPES,
}

const S71500_SCHEMA: DriverSchema = {
  fields: S7_FIELDS,
  variableAddress: S7_ADDRESS,
  variableDataTypes: S7_TYPES,
}

const S7200_SCHEMA: DriverSchema = {
  fields: S7_200_FIELDS,
  variableAddress: S7_ADDRESS,
  variableDataTypes: S7_TYPES,
}

function builtin(
  driverType: string,
  model: string,
  schema: DriverSchema,
  minutesAgo: number,
): Driver {
  return {
    id: ulid(),
    driverType,
    model,
    builtinDriver: true,
    schema,
    pluginName: `builtin-${driverType}`,
    createdTime: iso(minutesAgo),
  }
}

export const drivers: Driver[] = [
  builtin('modbus', 'modbusTcp', MODBUS_TCP_SCHEMA, 60 * 24 * 30),
  builtin('modbus', 'modbusRtu', MODBUS_RTU_SCHEMA, 60 * 24 * 30),
  builtin('modbus', 'modbusRtuOverTcp', MODBUS_RTU_TCP_SCHEMA, 60 * 24 * 30),
  builtin('modbus', 'myModbusTcp', { ...MODBUS_TCP_SCHEMA }, 60 * 24 * 2),
  builtin('opcUa', 'opcUa', OPCUA_SCHEMA, 60 * 24 * 30),
  builtin('s7', 's71500', S71500_SCHEMA, 60 * 24 * 30),
  builtin('s7', 's7-200', S7200_SCHEMA, 60 * 24 * 30),
]

drivers[3].builtinDriver = false
drivers[3].pluginName = 'my-modbus-plugin'

export const collections: CollectionDetail[] = [
  {
    id: 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
    name: '一号线电表',
    version: '4',
    driverType: 'modbus',
    driverModel: 'modbusTcp',
    collectStatus: 'collecting',
    enable: true,
    collectorCycleMs: 5000,
    modTime: iso(2),
    protocolConfig: {
      location: '192.168.1.10',
      port: 502,
      slave: 1,
      tcpCount: 1,
      memoryLayout: 'ABCD',
      positionAddress: 0,
    },
  },
  {
    id: 'b2c3d4e5f60718293a4b5c6d7e8f90a1',
    name: '空压机 RTU',
    version: '2',
    driverType: 'modbus',
    driverModel: 'modbusRtu',
    collectStatus: 'collecting',
    enable: true,
    collectorCycleMs: 10000,
    modTime: iso(4),
    protocolConfig: {
      location: '/dev/ttyS0',
      baudRate: 9600,
      dataBits: 8,
      parity: 'noParity',
      stopBits: 1,
      slave: 2,
      memoryLayout: 'CDAB',
      positionAddress: 0,
    },
  },
  {
    id: 'c3d4e5f60718293a4b5c6d7e8f90a1b2',
    name: '锅炉 OPC UA',
    version: '7',
    driverType: 'opcUa',
    driverModel: 'opcUa',
    collectStatus: 'collecting',
    enable: true,
    collectorCycleMs: 3000,
    modTime: iso(1),
    protocolConfig: {
      location: '192.168.1.20',
      port: 4840,
      username: 'operator',
      password: '***',
      securityPolicy: 'None',
    },
  },
  {
    id: 'd4e5f60718293a4b5c6d7e8f90a1b2c3',
    name: '包装线 S7',
    version: '3',
    driverType: 's7',
    driverModel: 's71500',
    collectStatus: 'collectingError',
    enable: true,
    collectorCycleMs: 2000,
    modTime: iso(6),
    protocolConfig: {
      location: '192.168.1.30',
      port: 102,
      rack: 0,
      slot: 1,
      connectionType: 1,
      pduSize: 480,
      localTsap: 0x1000,
      remoteTsap: 0x1000,
    },
  },
  {
    id: 'e5f60718293a4b5c6d7e8f90a1b2c3d4',
    name: '冷却塔 OPC UA',
    version: '1',
    driverType: 'opcUa',
    driverModel: 'opcUa',
    collectStatus: 'unconnected',
    enable: true,
    collectorCycleMs: 5000,
    modTime: iso(18),
    protocolConfig: {
      location: '192.168.1.21',
      port: 4840,
      securityPolicy: 'None',
    },
  },
  {
    id: 'f60718293a4b5c6d7e8f90a1b2c3d4e5',
    name: '备用泵',
    version: '2',
    driverType: 'modbus',
    driverModel: 'modbusTcp',
    collectStatus: 'stopped',
    enable: false,
    collectorCycleMs: 15000,
    modTime: iso(45),
    protocolConfig: {
      location: '192.168.1.40',
      port: 502,
      slave: 3,
      tcpCount: 1,
      memoryLayout: 'ABCD',
      positionAddress: 0,
    },
  },
  {
    id: '0718293a4b5c6d7e8f90a1b2c3d4e5f6',
    name: '包装线 S7-200',
    version: '1',
    driverType: 's7',
    driverModel: 's7-200',
    collectStatus: 'emptyVariable',
    enable: true,
    collectorCycleMs: 1000,
    modTime: iso(12),
    protocolConfig: {
      location: '192.168.1.31',
      port: 102,
      localTsap: 0x1000,
      remoteTsap: 0x1000,
      pduSize: 480,
    },
  },
]

function variable(
  name: string,
  dataType: string,
  address: string,
  value: unknown,
  options: Partial<CollectionVariable> = {},
): CollectionVariable {
  return {
    name,
    dataType,
    address,
    accessMode: 'r',
    value,
    ...options,
  }
}

export const collectionVariables: Record<string, CollectionVariable[]> = {
  [collections[0].id]: [
    variable('voltage', 'float32', '0', 220.4, { rate: 0.1 }),
    variable('current', 'float32', '2', 12.86, { rate: 0.01 }),
    variable('energy', 'float64', '4', 18234.62),
  ],
  [collections[1].id]: [
    variable('pressure', 'int16', '0', 6.42, { rate: 0.01 }),
    variable('running', 'bool', '0.0', true),
    variable('setpoint', 'int16', '10', 7, { accessMode: 'rw' }),
  ],
  [collections[2].id]: [
    variable('temperature', 'number', 'ns=2;s=Boiler.Temp', 178.5),
    variable('pressure', 'number', 'ns=2;s=Boiler.Press', 1.24),
    variable('targetTemp', 'number', 'ns=2;s=Boiler.Target', 180, { accessMode: 'rw' }),
  ],
  [collections[3].id]: [
    variable('speed', 'float32', 'DB1.DBD0', 1480.5),
    variable('running', 'bool', 'DB1.DBX4.0', true),
    variable('output', 'int32', 'DB1.DBD6', 72, { accessMode: 'rw' }),
  ],
  [collections[4].id]: [
    variable('fanSpeed', 'number', 'ns=3;s=Fan.Speed', 0),
  ],
  [collections[5].id]: [
    variable('flow', 'uint16', '0', 0),
    variable('enabled', 'bool', '0.1', false, { accessMode: 'rw' }),
  ],
  [collections[6].id]: [],
}

export const connectors: PushConnector[] = [
  {
    id: '11111111111111111111111111111111',
    type: 'mqtt',
    name: '主 MQTT Broker',
    pushConfig: {
      brokerUrl: 'tcp://192.168.1.5:1883',
      username: 'harns',
      password: '******',
      dataTopic: 'data/{gatewayId}/v1/#',
      commandTopic: 'command/{gatewayId}/#',
      clientId: 'harns-gateway-main',
      qos: 1,
      retain: false,
      keepAlive: 60,
    },
    status: 'running',
    version: 3,
    createdTime: iso(60 * 24 * 5),
    updatedTime: iso(30),
  },
  {
    id: '22222222222222222222222222222222',
    type: 'mqtt',
    name: '云端备份 Broker',
    pushConfig: {
      brokerUrl: 'ssl://mqtt.example.com:8883',
      username: 'gateway',
      dataTopic: 'harns/{gatewayId}/data',
      qos: 2,
      retain: true,
      keepAlive: 30,
    },
    status: 'stopped',
    version: 1,
    createdTime: iso(60 * 24 * 2),
    updatedTime: iso(60 * 8),
  },
]

export const pushGroups: PushGroup[] = [
  {
    id: '33333333333333333333333333333333',
    name: '全部通道 · 5 秒',
    connectorId: connectors[0].id,
    pushIntervalMs: 5000,
    allCollections: true,
    collectionIds: [],
    maxCollectionsPerPushMessage: 50,
    version: 2,
    createdTime: iso(60 * 24 * 4),
    updatedTime: iso(15),
  },
  {
    id: '44444444444444444444444444444444',
    name: '关键设备 · 秒级',
    connectorId: connectors[0].id,
    pushIntervalMs: 1000,
    allCollections: false,
    collectionIds: [collections[0].id, collections[2].id, collections[3].id],
    maxCollectionsPerPushMessage: 10,
    version: 1,
    createdTime: iso(60 * 24),
    updatedTime: iso(60 * 5),
  },
]
