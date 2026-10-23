import type { DataType, DeviceDetail } from '@/api/types'
import { formatNumber } from '@/lib/format'

export interface DisplayVariable {
  name: string
  dataType: DataType
  value: unknown
  accessMode: 'r' | 'rw'
}

/** 三种协议共享 name/dataType/value/accessMode,大屏只读展示无需分支。 */
export function readVariables(device: DeviceDetail): DisplayVariable[] {
  return device.variables.map((variable) => ({
    name: variable.name,
    dataType: variable.dataType,
    value: variable.value,
    accessMode: variable.accessMode,
  }))
}

/** 展示连接地址;端口存在时拼成 host:port。 */
export function deviceAddress(device: DeviceDetail): string {
  const port = device.address.option?.port
  return port ? `${device.address.location}:${port}` : device.address.location
}

export function formatVariableValue(value: unknown, dataType: DataType): string {
  if (value === null || value === undefined || value === '') return '—'
  if (dataType === 'bool') return value ? 'true' : 'false'
  if (typeof value === 'number') return formatNumber(value, 3)
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value)
    } catch {
      return '—'
    }
  }
  return String(value)
}
