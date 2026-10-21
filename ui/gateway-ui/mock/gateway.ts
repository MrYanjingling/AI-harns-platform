import type { IncomingMessage, ServerResponse } from 'node:http'
import type { DevicePayload, GatewayDeviceDetail } from '../src/api/types'
import { devices, gatewayMeta } from './fixtures'
import { readJson, send } from './utils'

function ok<T>(data: T): { code: number; message: string; data: T } {
  return { code: 200, message: 'success', data }
}

function fail(res: ServerResponse, status: number, code: number, message: string) {
  send(res, status, { code, message, errors: [{ code, message }] })
}

function autoTopic(deviceId: string): string {
  return `data/${gatewayMeta.id}/v1/${deviceId}`
}

function nextVersion(current: string): string {
  const value = Number(current)
  if (Number.isFinite(value)) return String(value + 1)
  return String(Date.now())
}

function newId(): string {
  return Array.from({ length: 32 }, () => Math.floor(Math.random() * 16).toString(16)).join('')
}

function matchesFilter(device: GatewayDeviceDetail, raw: string | null): boolean {
  if (!raw) return true
  try {
    const filter = JSON.parse(raw) as Record<string, unknown>
    const nameFilter = filter.name
    if (typeof nameFilter === 'string' && !device.name.includes(nameFilter)) return false
    if (nameFilter && typeof nameFilter === 'object') {
      const condition = nameFilter as Record<string, string>
      if (condition.eq && device.name !== condition.eq) return false
      if (condition.contains && !device.name.includes(condition.contains)) return false
      if (condition.startsWith && !device.name.startsWith(condition.startsWith)) return false
      if (condition.endsWith && !device.name.endsWith(condition.endsWith)) return false
    }
    const idFilter = filter.id
    if (typeof idFilter === 'string' && device.id !== idFilter) return false
    return true
  } catch {
    return true
  }
}

function applyPayload(device: GatewayDeviceDetail, payload: DevicePayload) {
  device.name = payload.name
  device.deviceCode = payload.deviceCode
  device.deviceType = payload.deviceType
  device.deviceModel = payload.deviceModel
  device.collectorCycle = payload.collectorCycle
  device.variableInterval = payload.variableInterval
  device.topic = payload.topic
  device.address = payload.address ?? { location: '' }
  device.variables = payload.variables ?? []
  device.slave = payload.slave
  device.memoryLayout = payload.memoryLayout
  device.positionAddress = payload.positionAddress
}

export async function handleGateway(
  req: IncomingMessage,
  res: ServerResponse,
  method: string,
  pathname: string,
  query: URLSearchParams,
): Promise<boolean> {
  /* ---------------- 网关资源 ---------------- */

  if (method === 'GET' && pathname === '/gatewayMeta') {
    send(res, 200, ok(gatewayMeta), { ETag: gatewayMeta.eTag })
    return true
  }

  if (method === 'GET' && pathname === '/gatewayCpu') {
    const second = new Date().getSeconds()
    const base = 22 + Math.sin(second / 7) * 8
    send(
      res,
      200,
      ok({
        cpus: {
          '1min': `${base.toFixed(0)}%`,
          '5min': `${(base - 2).toFixed(0)}%`,
          '15min': `${(base - 4).toFixed(0)}%`,
        },
      }),
    )
    return true
  }

  if (method === 'GET' && pathname === '/gatewayMem') {
    const percent = 58 + Math.sin(new Date().getSeconds() / 12) * 4
    const total = 3906
    send(
      res,
      200,
      ok({
        mem: {
          Total: `${total}MB`,
          Used: `${Math.round((total * percent) / 100)}MB`,
          UsedPercent: `${percent.toFixed(0)}%`,
        },
      }),
    )
    return true
  }

  if (method === 'GET' && pathname === '/gatewayDisk') {
    send(
      res,
      200,
      ok({
        disk: {
          '/': { Total: '29072MB', Used: '11248MB', UsedPercent: '39%' },
          '/data': { Total: '119234MB', Used: '74821MB', UsedPercent: '63%' },
        },
      }),
    )
    return true
  }

  /* ---------------- 设备 ---------------- */

  if (method === 'GET' && pathname === '/devices') {
    const filter = query.get('filter')
    send(res, 200, ok({ devices: devices.filter((device) => matchesFilter(device, filter)) }))
    return true
  }

  if (method === 'POST' && pathname === '/devices') {
    const payload = (await readJson(req)) as DevicePayload | undefined
    if (!payload?.name || !payload.deviceCode || !payload.deviceModel || !payload.deviceType) {
      fail(res, 400, 10001, '设备名称、编码、类型和型号不能为空')
      return true
    }
    const id = newId()
    const now = new Date().toISOString()
    const topic = payload.topic?.trim() || autoTopic(id)
    const variables = payload.variables ?? []
    const device: GatewayDeviceDetail = {
      id,
      eTag: String(Date.now()),
      name: payload.name,
      deviceCode: payload.deviceCode,
      deviceType: payload.deviceType,
      deviceModel: payload.deviceModel,
      collectStatus: variables.length ? 'collecting' : 'emptyVariable',
      modTime: now,
      topic,
      collectorCycle: payload.collectorCycle,
      variableInterval: payload.variableInterval,
      address: payload.address ?? { location: '' },
      variables,
      slave: payload.slave,
      memoryLayout: payload.memoryLayout,
      positionAddress: payload.positionAddress,
    }
    devices.push(device)
    send(res, 200, ok(device), { ETag: device.eTag })
    return true
  }

  const detailMatch = pathname.match(/^\/devices\/([0-9a-fA-F]{32})$/)
  if (detailMatch) {
    const id = detailMatch[1]
    const device = devices.find((item) => item.id === id)
    if (method === 'GET') {
      if (!device) {
        send(res, 404)
        return true
      }
      send(res, 200, ok(device), { ETag: device.eTag })
      return true
    }
    if (method === 'DELETE') {
      const ifMatch = req.headers['if-match']
      if (!ifMatch) {
        send(res, 428)
        return true
      }
      if (!device) {
        send(res, 404)
        return true
      }
      if (String(ifMatch) !== device.eTag) {
        send(res, 412)
        return true
      }
      const index = devices.indexOf(device)
      devices.splice(index, 1)
      send(res, 200, ok(device))
      return true
    }
    if (method === 'PUT') {
      const ifMatch = req.headers['if-match']
      if (!ifMatch) {
        send(res, 428)
        return true
      }
      if (!device) {
        send(res, 404)
        return true
      }
      if (String(ifMatch) !== device.eTag) {
        send(res, 412)
        return true
      }
      const payload = (await readJson(req)) as DevicePayload | undefined
      if (!payload) {
        fail(res, 400, 10001, '请求体不能为空')
        return true
      }
      applyPayload(device, payload)
      device.eTag = nextVersion(device.eTag)
      device.modTime = new Date().toISOString()
      if (!device.topic && device.collectStatus === 'collecting') {
        device.topic = autoTopic(device.id)
      }
      send(res, 200, ok(device), { ETag: device.eTag })
      return true
    }
    return false
  }

  const actionMatch = pathname.match(/^\/devices\/([0-9a-fA-F]{32})\/action$/)
  if (method === 'PUT' && actionMatch) {
    const device = devices.find((item) => item.id === actionMatch[1])
    if (!device) {
      send(res, 404)
      return true
    }
    const actions = (await readJson(req)) as Array<Record<string, unknown>> | undefined
    if (!Array.isArray(actions) || actions.length === 0) {
      fail(res, 400, 10005, '没有可执行的反控变量')
      return true
    }
    const writable = new Set(
      device.variables.filter((item) => item.accessMode === 'rw').map((item) => item.name),
    )
    for (const action of actions) {
      for (const key of Object.keys(action)) {
        if (!writable.has(key)) {
          fail(res, 400, 10004, `变量 [${key}] 不存在或不可写`)
          return true
        }
      }
    }
    if (device.collectStatus === 'unconnected') {
      fail(res, 400, 10014, `设备 [${device.name}] 未连接`)
      return true
    }
    send(res, 202)
    return true
  }

  const statusMatch = pathname.match(/^\/devices\/([0-9a-fA-F]{32})\/(start|stop|restart)$/)
  if (method === 'PUT' && statusMatch) {
    const device = devices.find((item) => item.id === statusMatch[1])
    if (!device) {
      send(res, 404)
      return true
    }
    const status = statusMatch[2]
    if (status === 'stop') {
      device.collectStatus = 'stopped'
    } else {
      device.collectStatus = device.variables.length > 0 ? 'collecting' : 'emptyVariable'
      if (device.collectStatus === 'collecting' && !device.topic) {
        device.topic = autoTopic(device.id)
      }
    }
    device.modTime = new Date().toISOString()
    send(res, 202)
    return true
  }

  return false
}
