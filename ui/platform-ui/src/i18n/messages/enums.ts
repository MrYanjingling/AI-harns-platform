import { defineMessages } from '@harns/core'

export const enums = defineMessages({
  'zh-CN': {
    'enums.platformStatus.enabled.label': '启用',
    'enums.platformStatus.enabled.description': '账号或资源处于启用状态',
    'enums.platformStatus.disabled.label': '禁用',
    'enums.platformStatus.disabled.description': '账号或资源处于禁用状态',

    'enums.thingDataType.int': '32 位整数',
    'enums.thingDataType.long': '64 位整数',
    'enums.thingDataType.float': '单精度浮点',
    'enums.thingDataType.double': '双精度浮点',
    'enums.thingDataType.bool': '布尔值',
    'enums.thingDataType.enum': '枚举值',
    'enums.thingDataType.text': '文本',
    'enums.thingDataType.time': '时间',
    'enums.thingDataType.array': '数组',
    'enums.thingDataType.struct': '结构体',

    'enums.rw.r.label': '只读 r',
    'enums.rw.r.description': '仅设备上报',
    'enums.rw.w.label': '只写 w',
    'enums.rw.w.description': '仅平台下发',
    'enums.rw.rw.label': '读写 rw',
    'enums.rw.rw.description': '平台与设备双向',

    'enums.operation.login': '登录',
    'enums.operation.logout': '退出',
    'enums.operation.create': '创建',
    'enums.operation.update': '更新',
    'enums.operation.delete': '删除',
    'enums.operation.query': '查询',

    'enums.gender.male': '男',
    'enums.gender.female': '女',

    'enums.authMode.local': '本地认证',
    'enums.authMode.ldap': 'LDAP',
    'enums.authMode.radius': 'RADIUS',
  },
  'en-US': {
    'enums.platformStatus.enabled.label': 'Enabled',
    'enums.platformStatus.enabled.description': 'Account or resource is enabled',
    'enums.platformStatus.disabled.label': 'Disabled',
    'enums.platformStatus.disabled.description': 'Account or resource is disabled',

    'enums.thingDataType.int': '32-bit integer',
    'enums.thingDataType.long': '64-bit integer',
    'enums.thingDataType.float': 'Single-precision float',
    'enums.thingDataType.double': 'Double-precision float',
    'enums.thingDataType.bool': 'Boolean',
    'enums.thingDataType.enum': 'Enumeration',
    'enums.thingDataType.text': 'Text',
    'enums.thingDataType.time': 'Time',
    'enums.thingDataType.array': 'Array',
    'enums.thingDataType.struct': 'Structure',

    'enums.rw.r.label': 'Read-only r',
    'enums.rw.r.description': 'Reported by the device only',
    'enums.rw.w.label': 'Write-only w',
    'enums.rw.w.description': 'Sent by the platform only',
    'enums.rw.rw.label': 'Read-write rw',
    'enums.rw.rw.description': 'Bidirectional between platform and device',

    'enums.operation.login': 'Login',
    'enums.operation.logout': 'Logout',
    'enums.operation.create': 'Create',
    'enums.operation.update': 'Update',
    'enums.operation.delete': 'Delete',
    'enums.operation.query': 'Query',

    'enums.gender.male': 'Male',
    'enums.gender.female': 'Female',

    'enums.authMode.local': 'Local',
    'enums.authMode.ldap': 'LDAP',
    'enums.authMode.radius': 'RADIUS',
  },
})
