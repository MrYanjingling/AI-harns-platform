import type { DeviceDetail, DeviceStatus, DeviceType } from '@/api/types'
import { translate } from '@/i18n/translate'
import { DEVICE_STATUS, PROTOCOLS, UNKNOWN_STATUS } from '@/lib/constants'

export type TreeMode = 'protocol' | 'model'

export interface DeviceFilters {
  name: string
  code: string
  protocol: DeviceType | ''
  status: DeviceStatus | ''
  /** 仅由左侧树选择,不提供独立输入 */
  model: string
}

export const EMPTY_DEVICE_FILTERS: DeviceFilters = {
  name: '',
  code: '',
  protocol: '',
  status: '',
  model: '',
}

export interface DeviceTreeNode {
  key: string
  label: string
  protocol?: DeviceType
  model?: string
  count: number
  children?: DeviceTreeNode[]
}

const PROTOCOL_ORDER: DeviceType[] = ['modbus', 'opcUa', 's7']

function modelLabel(protocol: DeviceType, model: string): string {
  return PROTOCOLS[protocol].models.find((item) => item.value === model)?.label ?? model
}

function statusLabel(status: DeviceStatus | string): string {
  const meta = DEVICE_STATUS[status as DeviceStatus] ?? UNKNOWN_STATUS
  return translate(meta.labelKey)
}

function addressOf(device: DeviceDetail): string {
  return device.address?.location || '—'
}

export function buildDeviceTree(devices: DeviceDetail[], mode: TreeMode): DeviceTreeNode[] {
  if (mode === 'model') {
    const counts = new Map<string, { protocol: DeviceType; count: number }>()
    for (const device of devices) {
      const current = counts.get(device.deviceModel)
      counts.set(device.deviceModel, {
        protocol: device.deviceType,
        count: (current?.count ?? 0) + 1,
      })
    }
    return [...counts.entries()]
      .sort((a, b) => {
        const protocolDelta =
          PROTOCOL_ORDER.indexOf(a[1].protocol) - PROTOCOL_ORDER.indexOf(b[1].protocol)
        if (protocolDelta !== 0) return protocolDelta
        return a[0].localeCompare(b[0])
      })
      .map(([model, info]) => ({
        key: `model:${model}`,
        label: modelLabel(info.protocol, model),
        protocol: info.protocol,
        model,
        count: info.count,
      }))
  }

  const grouped = new Map<DeviceType, Map<string, number>>()
  for (const device of devices) {
    const models = grouped.get(device.deviceType) ?? new Map<string, number>()
    models.set(device.deviceModel, (models.get(device.deviceModel) ?? 0) + 1)
    grouped.set(device.deviceType, models)
  }

  return PROTOCOL_ORDER.filter((protocol) => grouped.has(protocol)).map((protocol) => {
    const models = grouped.get(protocol) ?? new Map<string, number>()
    const children = [...models.entries()]
      .sort((a, b) => modelLabel(protocol, a[0]).localeCompare(modelLabel(protocol, b[0])))
      .map(([model, count]) => ({
        key: `model:${model}`,
        label: modelLabel(protocol, model),
        protocol,
        model,
        count,
      }))
    return {
      key: `protocol:${protocol}`,
      label: PROTOCOLS[protocol].label,
      protocol,
      count: children.reduce((sum, child) => sum + child.count, 0),
      children,
    }
  })
}

export function filterTree(nodes: DeviceTreeNode[], keyword: string): DeviceTreeNode[] {
  const query = keyword.trim().toLowerCase()
  if (!query) return nodes
  return nodes.flatMap((node) => {
    const children = node.children ? filterTree(node.children, keyword) : undefined
    if (node.label.toLowerCase().includes(query)) return [node]
    if (children && children.length > 0) return [{ ...node, children }]
    return []
  })
}

export function filterDevices(
  devices: DeviceDetail[],
  filters: DeviceFilters,
): DeviceDetail[] {
  const name = filters.name.trim().toLowerCase()
  const code = filters.code.trim().toLowerCase()
  return devices
    .filter((device) => {
      if (name && !device.name.toLowerCase().includes(name)) return false
      if (code && !device.deviceCode.toLowerCase().includes(code)) return false
      if (filters.protocol && device.deviceType !== filters.protocol) return false
      if (filters.status && device.collectStatus !== filters.status) return false
      if (filters.model && device.deviceModel !== filters.model) return false
      return true
    })
    .sort((a, b) => a.deviceCode.localeCompare(b.deviceCode))
}

/** 防止以 = + - @ 开头的单元格在 Excel 中被当作公式执行。 */
function csvCell(value: unknown): string {
  let text = String(value ?? '')
  if (/^[=+\-@]/.test(text)) text = `'${text}`
  return `"${text.replaceAll('"', '""')}"`
}

export function exportDevicesCsv(devices: DeviceDetail[]): void {
  const header = [
    translate('devices.csv.deviceCode'),
    translate('devices.csv.deviceName'),
    translate('devices.csv.protocol'),
    translate('devices.csv.model'),
    translate('devices.csv.address'),
    translate('devices.csv.status'),
    translate('devices.csv.variables'),
    translate('devices.csv.updatedAt'),
    translate('devices.csv.remark'),
  ]
  const rows = devices.map((device) => [
    device.deviceCode,
    device.name,
    PROTOCOLS[device.deviceType]?.label ?? device.deviceType,
    device.deviceModel,
    addressOf(device),
    statusLabel(device.collectStatus),
    device.variables.length,
    device.modTime,
    device.topic || translate('devices.defaultTopic'),
  ])
  const content = `﻿${[header, ...rows]
    .map((row) => row.map(csvCell).join(','))
    .join('\r\n')}`
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  const stamp = new Date().toISOString().slice(0, 16).replaceAll(':', '').replace('T', '-')
  link.href = url
  link.download = translate('devices.exportFileName', { stamp })
  link.click()
  URL.revokeObjectURL(url)
}
