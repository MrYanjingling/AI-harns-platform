import type { DeviceDetail, DeviceStatus, DeviceType } from '@/api/types'

/** 工厂按协议分区,顺序固定,保证同一批设备每次进入大屏位置一致。 */
export const ZONE_ORDER: DeviceType[] = ['modbus', 'opcUa', 's7']

export const ZONE_X: Record<DeviceType, number> = {
  modbus: -8.6,
  opcUa: 0,
  s7: 8.6,
}

export const ZONE_WIDTH = 7.2
export const ZONE_DEPTH = 13.2

/** 3D 场景固定色板:three.js 只接受数值色,无法复用 Tailwind 语义类。 */
export const SCREEN_COLORS = {
  background: 0x050b10,
  floor: 0x07141a,
  grid: 0x123039,
  platform: 0x0a1a21,
  body: 0x102b35,
} as const

export const STATUS_COLOR: Record<DeviceStatus, number> = {
  collecting: 0x34d399,
  collectingError: 0xfbbf24,
  unconnected: 0x94a3b8,
  emptyVariable: 0xa78bfa,
  stopped: 0x64748b,
  error: 0xf43f5e,
}

export const UNKNOWN_STATUS_COLOR = 0x94a3b8

export const ZONE_ACCENT: Record<DeviceType, number> = {
  modbus: 0x34d399,
  opcUa: 0x38bdf8,
  s7: 0xfbbf24,
}

/** 与 3D 分区同色的文本/进度条类,保证场景与面板视觉一致。 */
export const ZONE_TEXT: Record<DeviceType, string> = {
  modbus: 'text-emerald-300',
  opcUa: 'text-sky-300',
  s7: 'text-amber-300',
}

export const ZONE_BAR: Record<DeviceType, string> = {
  modbus: 'bg-emerald-400',
  opcUa: 'bg-sky-400',
  s7: 'bg-amber-400',
}

export interface FactoryDevice {
  device: DeviceDetail
  /** 场景坐标:x 为横向,z 为纵深,y 由高度决定。 */
  x: number
  z: number
  height: number
}

export function statusColor(status: DeviceStatus): number {
  return STATUS_COLOR[status] ?? UNKNOWN_STATUS_COLOR
}

function hashString(value: string): number {
  let hash = 0
  for (let i = 0; i < value.length; i += 1) {
    hash = (hash * 31 + value.charCodeAt(i)) >>> 0
  }
  return hash
}

/**
 * 同一协议内按名称排序后铺进网格。
 * 位置只由设备 ID/协议/同协议序号决定,轮询刷新不会让设备跳动。
 */
export function buildFactoryLayout(devices: DeviceDetail[]): FactoryDevice[] {
  return ZONE_ORDER.flatMap((type) => {
    const group = devices
      .filter((device) => device.deviceType === type)
      .sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))

    const columns = group.length <= 4 ? 2 : group.length <= 9 ? 3 : 4
    const rows = Math.max(1, Math.ceil(group.length / columns))
    const columnGap = 2.45
    const rowGap = 2.7

    return group.map((device, index) => {
      const column = index % columns
      const row = Math.floor(index / columns)
      return {
        device,
        x: ZONE_X[type] + (column - (columns - 1) / 2) * columnGap,
        z: (row - (rows - 1) / 2) * rowGap,
        // 设备以固定高度差呈现层次,便于在斜视角下辨认。
        height: 0.9 + (hashString(device.id) % 5) * 0.18,
      }
    })
  })
}
