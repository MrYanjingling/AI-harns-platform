import type { MessageKey } from '@harns/core'
import type { OperationType, PlatformStatus, RwFlag, ThingDataType } from '@/api/types'

export interface StatusMeta {
  labelKey: MessageKey
  /** tailwind 类:文本 + 背景 + 边框 */
  className: string
  dotClassName: string
  descriptionKey: MessageKey
}

export const UNKNOWN_STATUS: StatusMeta = {
  labelKey: 'common.state.unknown',
  className:
    'bg-zinc-50 text-zinc-600 border-zinc-200 dark:bg-zinc-500/10 dark:text-zinc-300 dark:border-zinc-600/30',
  dotClassName: 'bg-zinc-400',
  descriptionKey: 'common.state.unknown',
}

/* ---------------- 平台服务 ---------------- */

export const PLATFORM_STATUS: Record<PlatformStatus, StatusMeta> = {
  enabled: {
    labelKey: 'enums.platformStatus.enabled.label',
    className:
      'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:border-emerald-600/30',
    dotClassName: 'bg-emerald-500',
    descriptionKey: 'enums.platformStatus.enabled.description',
  },
  disabled: {
    labelKey: 'enums.platformStatus.disabled.label',
    className:
      'bg-zinc-50 text-zinc-600 border-zinc-200 dark:bg-zinc-500/10 dark:text-zinc-300 dark:border-zinc-600/30',
    dotClassName: 'bg-zinc-400',
    descriptionKey: 'enums.platformStatus.disabled.description',
  },
}

/** 物模型 dataType.type 的展示与规格能力。 */
export const THING_DATA_TYPES: Array<{
  value: ThingDataType
  label: string
  descriptionKey: MessageKey
}> = [
  { value: 'int', label: 'int', descriptionKey: 'enums.thingDataType.int' },
  { value: 'long', label: 'long', descriptionKey: 'enums.thingDataType.long' },
  { value: 'float', label: 'float', descriptionKey: 'enums.thingDataType.float' },
  { value: 'double', label: 'double', descriptionKey: 'enums.thingDataType.double' },
  { value: 'bool', label: 'bool', descriptionKey: 'enums.thingDataType.bool' },
  { value: 'enum', label: 'enum', descriptionKey: 'enums.thingDataType.enum' },
  { value: 'text', label: 'text', descriptionKey: 'enums.thingDataType.text' },
  { value: 'time', label: 'time', descriptionKey: 'enums.thingDataType.time' },
  { value: 'array', label: 'array', descriptionKey: 'enums.thingDataType.array' },
  { value: 'struct', label: 'struct', descriptionKey: 'enums.thingDataType.struct' },
]

/** 需要 unit / min / max 的数值类型。 */
export const NUMERIC_DATA_TYPES: ThingDataType[] = ['int', 'long', 'float', 'double']

export const RW_FLAGS: Array<{ value: RwFlag; labelKey: MessageKey; descriptionKey: MessageKey }> =
  [
    {
      value: 'r',
      labelKey: 'enums.rw.r.label',
      descriptionKey: 'enums.rw.r.description',
    },
    {
      value: 'w',
      labelKey: 'enums.rw.w.label',
      descriptionKey: 'enums.rw.w.description',
    },
    {
      value: 'rw',
      labelKey: 'enums.rw.rw.label',
      descriptionKey: 'enums.rw.rw.description',
    },
  ]

export const OPERATION_TYPES: Array<{ value: OperationType; labelKey: MessageKey }> = [
  { value: 'LOGIN', labelKey: 'enums.operation.login' },
  { value: 'LOGOUT', labelKey: 'enums.operation.logout' },
  { value: 'CREATE', labelKey: 'enums.operation.create' },
  { value: 'UPDATE', labelKey: 'enums.operation.update' },
  { value: 'DELETE', labelKey: 'enums.operation.delete' },
  { value: 'QUERY', labelKey: 'enums.operation.query' },
]

export const AUDIT_APPS = ['IOT', 'EMS', 'EAM'] as const
export const AUDIT_METHODS = ['POST', 'PUT', 'PATCH', 'DELETE'] as const

export const GENDERS: Array<{ value: 'male' | 'female'; labelKey: MessageKey }> = [
  { value: 'male', labelKey: 'enums.gender.male' },
  { value: 'female', labelKey: 'enums.gender.female' },
]

export const AUTH_MODES: Array<{ value: 'local' | 'ldap' | 'radius'; labelKey: MessageKey }> = [
  { value: 'local', labelKey: 'enums.authMode.local' },
  { value: 'ldap', labelKey: 'enums.authMode.ldap' },
  { value: 'radius', labelKey: 'enums.authMode.radius' },
]

export const PAGE_SIZES = [10, 20, 50, 100] as const
