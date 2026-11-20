import type { IncomingMessage, ServerResponse } from 'node:http'
import type {
  AISettings,
  CollectionDetail,
  CollectionVariable,
  CollectionVariableInput,
  Driver,
  FrameEvent,
  GatewayUser,
  MQTTConfig,
  PushConnector,
  PushConnectorPayload,
  PushGroup,
  PushGroupPayload,
  SimulatorDataType,
  SimulatorMode,
  SimulatorPoint,
  SimulatorState,
} from '../src/api/types'
import {
  collectionVariables,
  collections,
  connectors,
  drivers,
  gatewayMeta,
  gatewayUsers,
  license,
  systemInfo,
  pushGroups,
  ulid,
} from './fixtures'
import { readJson, send } from './utils'

function ok<T>(data: T): { code: number; message: string; data: T } {
  return { code: 200, message: 'success', data }
}

let aiSettings: AISettings = {
  assistantEnabled: false,
  mcpEnabled: false,
  name: '',
  remark: '',
  model: '',
  endpointUrl: '',
  apiKey: '',
  hasApiKey: false,
  mcpToken: '',
  updatedTime: nowIso(),
}

const simulatorState: SimulatorState = {
  running: false,
  port: 2502,
  unitId: 1,
  memoryLayout: 'CDAB',
  points: [],
}

function simulatorPayload(): SimulatorState {
  return {
    ...simulatorState,
    points: simulatorState.points.map((point) => ({ ...point })),
  }
}

function fail(res: ServerResponse, status: number, code: number, message: string) {
  send(res, status, { code, message, errors: [{ code, message }] })
}

function notFound(res: ServerResponse, message = 'resource not found') {
  fail(res, 404, 10004, message)
}

function badRequest(res: ServerResponse, message: string) {
  fail(res, 400, 10001, message)
}

function pageParams(query: URLSearchParams): { page: number; pageSize: number } {
  const page = Number(query.get('page') ?? 1)
  const pageSize = Number(query.get('pageSize') ?? 20)
  return {
    page: Number.isFinite(page) && page > 0 ? page : 1,
    pageSize: Number.isFinite(pageSize) && pageSize > 0 ? Math.min(pageSize, 100) : 20,
  }
}

function paginate<T>(items: T[], page: number, pageSize: number) {
  const start = (page - 1) * pageSize
  return {
    total: items.length,
    page,
    pageSize,
    list: items.slice(start, start + pageSize),
  }
}

function keywordMatch(values: Array<string | undefined>, keyword: string | null): boolean {
  if (!keyword) return true
  const query = keyword.trim().toLowerCase()
  if (!query) return true
  return values.join(' ').toLowerCase().includes(query)
}

function nextVersion(version: string): string {
  const value = Number(version)
  return Number.isFinite(value) ? String(value + 1) : '1'
}

function nowIso(): string {
  return new Date().toISOString()
}

function isObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

async function readObject(req: IncomingMessage): Promise<Record<string, unknown> | undefined> {
  const body = await readJson(req)
  return isObject(body) ? body : undefined
}

function findCollection(id: string): CollectionDetail | undefined {
  return collections.find((item) => item.id === id)
}

function findDriver(id: string): Driver | undefined {
  return drivers.find((item) => item.id === id)
}

function findConnector(id: string): PushConnector | undefined {
  return connectors.find((item) => item.id === id)
}

function findGroup(id: string): PushGroup | undefined {
  return pushGroups.find((item) => item.id === id)
}

interface MockFrameStream {
  traceId: number
  requests: Set<ServerResponse>
  responses: Set<ServerResponse>
  timer?: ReturnType<typeof setInterval>
}

const frameStreams = new Map<string, MockFrameStream>()

function frameStreamOf(id: string): MockFrameStream {
  const existing = frameStreams.get(id)
  if (existing) return existing
  const stream: MockFrameStream = {
    traceId: 0,
    requests: new Set(),
    responses: new Set(),
  }
  frameStreams.set(id, stream)
  return stream
}

function hexBytes(bytes: number[]): string {
  return bytes.map((byte) => byte.toString(16).padStart(2, '0').toUpperCase()).join(' ')
}

function writeFrame(res: ServerResponse, event: 'request' | 'response', frame: FrameEvent): boolean {
  try {
    res.write(`event: ${event}\ndata: ${JSON.stringify(frame)}\n\n`)
    return true
  } catch {
    return false
  }
}

function startMockFrameTimer(id: string): void {
  const stream = frameStreamOf(id)
  if (stream.timer) return
  stream.timer = setInterval(() => {
    stream.traceId += 1
    const traceId = stream.traceId
    const operation = traceId % 4 === 0 ? 'write' : 'read'
    const time = nowIso()
    const failed = traceId % 9 === 0
    const requestBytes = [0x01, 0x03, 0x00, traceId % 256, 0x00, 0x02, 0xc4, 0x0b]
    const responseBytes = [0x01, 0x03, 0x04, 0x41, 0x48 + (traceId % 16), 0x00, 0x00, 0x8f, 0x3c]
    const request: FrameEvent = {
      traceId,
      time,
      operation,
      hex: hexBytes(requestBytes),
      length: requestBytes.length,
    }
    const response: FrameEvent = {
      traceId,
      time,
      operation,
      hex: failed ? '' : hexBytes(responseBytes),
      length: failed ? 0 : responseBytes.length,
      durationMs: 6 + (traceId % 12),
      error: failed ? 'mock response timeout' : undefined,
    }

    for (const res of [...stream.requests]) {
      if (!writeFrame(res, 'request', request)) stream.requests.delete(res)
    }
    for (const res of [...stream.responses]) {
      if (!writeFrame(res, 'response', response)) stream.responses.delete(res)
    }

    if (stream.requests.size === 0 && stream.responses.size === 0) {
      if (stream.timer) clearInterval(stream.timer)
      frameStreams.delete(id)
    }
  }, 1200)
}

function variablesOf(id: string): CollectionVariable[] {
  return collectionVariables[id] ?? (collectionVariables[id] = [])
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function validateVariable(input: unknown): CollectionVariableInput | undefined {
  if (!isObject(input)) return undefined
  const name = typeof input.name === 'string' ? input.name.trim() : ''
  const dataType = typeof input.dataType === 'string' ? input.dataType.trim() : ''
  const address = typeof input.address === 'string' ? input.address.trim() : ''
  const expression = typeof input.expression === 'string' ? input.expression.trim() : ''
  const isCalculated = Boolean(input.isCalculated)
  if (!name || !dataType) return undefined
  if (isCalculated ? !expression : !address) return undefined
  return {
    ...(input as unknown as CollectionVariableInput),
    name,
    dataType,
    address: isCalculated ? '' : address,
    accessMode: isCalculated ? 'r' : (input.accessMode as CollectionVariableInput['accessMode']),
    isCalculated,
    expression: isCalculated ? expression : undefined,
  }
}

function refreshStatus(collection: CollectionDetail): void {
  if (!collection.enable) {
    collection.collectStatus = 'stopped'
    return
  }
  collection.collectStatus = variablesOf(collection.id).length === 0 ? 'emptyVariable' : 'collecting'
}

function normalizeConnectorPayload(
  body: Record<string, unknown>,
): PushConnectorPayload | undefined {
  const name = typeof body.name === 'string' ? body.name.trim() : ''
  if (!name || !isObject(body.pushConfig)) return undefined
  const pushConfig = body.pushConfig as unknown as MQTTConfig
  if (!pushConfig.brokerUrl) return undefined
  return {
    type: typeof body.type === 'string' && body.type ? body.type : 'mqtt',
    name,
    pushConfig,
  }
}

function normalizeGroupPayload(body: Record<string, unknown>): PushGroupPayload | undefined {
  const name = typeof body.name === 'string' ? body.name.trim() : ''
  const connectorId = typeof body.connectorId === 'string' ? body.connectorId : ''
  const dataTopic = typeof body.dataTopic === 'string' ? body.dataTopic.trim() : ''
  const commandTopic = typeof body.commandTopic === 'string' ? body.commandTopic.trim() : ''
  const pushIntervalMs = Number(body.pushIntervalMs)
  if (!name || !connectorId || !dataTopic || !Number.isFinite(pushIntervalMs) || pushIntervalMs <= 0) {
    return undefined
  }
  const allCollections = Boolean(body.allCollections)
  const collectionIds = Array.isArray(body.collectionIds)
    ? body.collectionIds.filter((item): item is string => typeof item === 'string')
    : []
  return {
    name,
    connectorId,
    dataTopic,
    commandTopic,
    pushIntervalMs,
    allCollections,
    collectionIds: allCollections ? [] : collectionIds,
    maxCollectionsPerPushMessage: Number(body.maxCollectionsPerPushMessage) || 50,
  }
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

  if (method === 'GET' && pathname === '/systemInfo') {
    send(res, 200, ok(systemInfo))
    return true
  }

  if (method === 'GET' && pathname === '/license') {
    send(res, 200, ok(license))
    return true
  }

  if (method === 'POST' && pathname === '/license/import') {
    const body = await readObject(req)
    if (!body || typeof body.data !== 'string' || !body.data.trim()) {
      badRequest(res, 'license data is required')
      return true
    }
    send(res, 200, ok(license))
    return true
  }

  /* ---------------- AI 配置 ---------------- */

  if (pathname === '/ai/settings' && method === 'GET') {
    send(res, 200, ok(aiSettings))
    return true
  }

  if (pathname === '/ai/settings' && method === 'PUT') {
    const body = await readObject(req)
    if (!body) {
      badRequest(res, 'invalid body')
      return true
    }
    const mcpEnabled = Boolean(body.mcpEnabled)
    let mcpToken = aiSettings.mcpToken
    if (mcpEnabled && !mcpToken) {
      mcpToken = `mcp_${Math.random().toString(36).slice(2)}${Math.random().toString(36).slice(2)}`
    } else if (!mcpEnabled) {
      mcpToken = ''
    }
    aiSettings = {
      ...aiSettings,
      assistantEnabled: Boolean(body.assistantEnabled),
      mcpEnabled,
      mcpToken,
      updatedTime: nowIso(),
    }
    send(res, 200, ok(aiSettings))
    return true
  }

  if (pathname === '/ai/model' && method === 'PUT') {
    const body = await readObject(req)
    const name = typeof body?.name === 'string' ? body.name.trim() : ''
    const remark = typeof body?.remark === 'string' ? body.remark.trim() : ''
    const model = typeof body?.model === 'string' ? body.model.trim() : ''
    const endpointUrl = typeof body?.endpointUrl === 'string' ? body.endpointUrl.trim() : ''
    const apiKey = typeof body?.apiKey === 'string' ? body.apiKey.trim() : ''
    if (!name || !model || !endpointUrl || (!aiSettings.hasApiKey && !apiKey)) {
      badRequest(res, 'name, model, endpointUrl and apiKey are required')
      return true
    }
    aiSettings = {
      ...aiSettings,
      name,
      remark,
      model,
      endpointUrl,
      hasApiKey: aiSettings.hasApiKey || Boolean(apiKey),
      updatedTime: nowIso(),
    }
    send(res, 200, ok(aiSettings))
    return true
  }

  if (pathname === '/ai/model/test' && method === 'POST') {
    send(res, 200, ok({ success: true }))
    return true
  }

  if (pathname === '/system/logs/export' && method === 'GET') {
    const level = (query.get('level') ?? '').toLowerCase()
    if (!['error', 'warn', 'info'].includes(level)) {
      badRequest(res, 'invalid log level')
      return true
    }
    const timestamp = nowIso()
    const lines = [
      `time=${timestamp} level=INFO msg=mock info log`,
      `time=${timestamp} level=WARN msg=mock warn log`,
      `time=${timestamp} level=ERROR msg=mock error log`,
    ]
    const filtered = lines.filter((line) => line.includes(`level=${level.toUpperCase()}`))
    res.statusCode = 200
    res.setHeader('Content-Type', 'text/plain; charset=utf-8')
    res.setHeader('Cache-Control', 'no-store')
    res.end(filtered.join('\n') + (filtered.length > 0 ? '\n' : ''))
    return true
  }

  if (pathname === '/users' && method === 'GET') {
    const { page, pageSize } = pageParams(query)
    const keyword = query.get('keyword')
    const list = gatewayUsers
      .filter((item) => keywordMatch([item.username, item.remark], keyword))
      .slice()
      .sort((left, right) => right.updatedTime.localeCompare(left.updatedTime))
    send(res, 200, ok(paginate(list, page, pageSize)))
    return true
  }

  if (pathname === '/users' && method === 'POST') {
    const body = await readObject(req)
    const username = typeof body?.username === 'string' ? body.username.trim() : ''
    if (!username) {
      badRequest(res, 'username is required')
      return true
    }
    if (gatewayUsers.some((item) => item.username === username)) {
      badRequest(res, 'username already exists')
      return true
    }
    const now = nowIso()
    const user: GatewayUser = {
      id: ulid(),
      version: 1,
      username,
      remark: typeof body?.remark === 'string' ? body.remark : '',
      isAdmin: false,
      createdTime: now,
      updatedTime: now,
    }
    gatewayUsers.push(user)
    send(res, 200, ok(user))
    return true
  }

  const userMatch = pathname.match(/^\/users\/([^/]+)(\/password\/reset)?$/)
  if (userMatch) {
    const user = gatewayUsers.find((item) => item.id === userMatch[1])
    if (!user) {
      notFound(res, 'user not found')
      return true
    }
    if (userMatch[2]) {
      if (method !== 'POST') {
        res.setHeader('Allow', 'POST')
        fail(res, 405, 10005, 'method not allowed')
        return true
      }
      user.version += 1
      user.updatedTime = nowIso()
      send(res, 200, ok(user))
      return true
    }
    if (method === 'PUT') {
      const body = await readObject(req)
      if (typeof body?.username === 'string' && body.username.trim()) {
        if (user.isAdmin && body.username.trim() !== user.username) {
          badRequest(res, 'superuser username cannot be changed')
          return true
        }
        user.username = body.username.trim()
      }
      if (typeof body?.remark === 'string') user.remark = body.remark
      user.version += 1
      user.updatedTime = nowIso()
      send(res, 200, ok(user))
      return true
    }
    if (method === 'DELETE') {
      if (user.isAdmin) {
        badRequest(res, 'superuser cannot be deleted')
        return true
      }
      const index = gatewayUsers.findIndex((item) => item.id === user.id)
      gatewayUsers.splice(index, 1)
      send(res, 200, ok({ deleted: true }))
      return true
    }
    res.setHeader('Allow', 'GET, PUT, DELETE')
    fail(res, 405, 10005, 'method not allowed')
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
          Total: `${(total / 1024).toFixed(1)}GB`,
          Used: `${(((total * percent) / 100) / 1024).toFixed(1)}GB`,
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
          '/': { Total: '28.4GB', Used: '11.0GB', UsedPercent: '39%' },
          '/data': { Total: '116.4GB', Used: '73.1GB', UsedPercent: '63%' },
        },
      }),
    )
    return true
  }

  /* ---------------- 采集器 ---------------- */

  if (pathname === '/collections') {
    if (method === 'GET') {
      const { page, pageSize } = pageParams(query)
      const keyword = query.get('keyword')
      const driverType = query.get('driverType')
      const driverModel = query.get('driverModel')
      const list = collections
        .filter(
          (item) =>
            keywordMatch([item.name, item.id], keyword) &&
            (!driverType || item.driverType === driverType) &&
            (!driverModel || item.driverModel === driverModel),
        )
        .sort((a, b) => new Date(b.modTime).getTime() - new Date(a.modTime).getTime())
      send(res, 200, ok(paginate(list, page, pageSize)))
      return true
    }
    if (method === 'POST') {
      const body = await readObject(req)
      if (!body) return badRequest(res, 'invalid body'), true
      const name = typeof body.name === 'string' ? body.name.trim() : ''
      const driverType = typeof body.driverType === 'string' ? body.driverType : ''
      const driverModel = typeof body.driverModel === 'string' ? body.driverModel : ''
      const cycle = Number(body.collectorCycleMs)
      if (!name || !driverType || !driverModel || !Number.isFinite(cycle) || cycle <= 0) {
        badRequest(res, 'name, driverType, driverModel and collectorCycleMs are required')
        return true
      }
      const rawVariables = Array.isArray(body.variables) ? body.variables : []
      const parsed = rawVariables
        .map(validateVariable)
        .filter((item): item is CollectionVariableInput => Boolean(item))
      if (new Set(parsed.map((item) => item.name)).size !== parsed.length) {
        badRequest(res, 'duplicate variable name')
        return true
      }
      const detail: CollectionDetail = {
        id: ulid(),
        name,
        version: '1',
        driverType,
        driverModel,
        collectStatus: 'stopped',
        enable: true,
        collectorCycleMs: cycle,
        variableCount: parsed.length,
        collectedVariableCount: parsed.filter((item) => !item.isCalculated).length,
        modTime: nowIso(),
        protocolConfig: isObject(body.protocolConfig) ? body.protocolConfig : {},
      }
      collections.unshift(detail)
      collectionVariables[detail.id] = parsed.map((item) => ({
        ...item,
        value: undefined,
      }))
      send(res, 200, ok(detail))
      return true
    }
    res.setHeader('Allow', 'GET, POST')
    fail(res, 405, 10005, 'method not allowed')
    return true
  }

  const frameStreamMatch = pathname.match(
    /^\/collections\/([^/]+)\/frames\/(requests|responses)\/stream$/,
  )
  if (frameStreamMatch && method === 'GET') {
    const [, id, kind] = frameStreamMatch
    const collection = findCollection(id)
    if (!collection) {
      notFound(res, 'collection not found')
      return true
    }
    if (!['collecting', 'collectingError'].includes(collection.collectStatus)) {
      badRequest(res, 'collection is not collecting')
      return true
    }

    res.statusCode = 200
    res.setHeader('Content-Type', 'text/event-stream')
    res.setHeader('Cache-Control', 'no-cache')
    res.setHeader('Connection', 'keep-alive')
    res.setHeader('X-Accel-Buffering', 'no')
    res.flushHeaders?.()
    res.write('event: ready\ndata: {}\n\n')

    const stream = frameStreamOf(id)
    const target = kind === 'requests' ? stream.requests : stream.responses
    target.add(res)
    startMockFrameTimer(id)

    res.on('close', () => {
      target.delete(res)
      if (stream.requests.size === 0 && stream.responses.size === 0 && stream.timer) {
        clearInterval(stream.timer)
        stream.timer = undefined
        frameStreams.delete(id)
      }
    })
    return true
  }

  const collectionAction = pathname.match(
    /^\/collections\/([^/]+)\/(start|stop|restart|enable|variables|action)$/,
  )
  if (collectionAction) {
    const [, id, action] = collectionAction
    const collection = findCollection(id)
    if (!collection) {
      notFound(res, 'collection not found')
      return true
    }
    if (action === 'variables') {
      if (method === 'GET') {
        const { page, pageSize } = pageParams(query)
        const keyword = query.get('keyword')
        const list = variablesOf(id).filter((item) =>
          keywordMatch([item.name, item.address, item.dataType], keyword),
        )
        send(res, 200, ok(paginate(list, page, pageSize)))
        return true
      }
      if (method === 'PUT') {
        const body = await readJson(req)
        const inputs = Array.isArray(body) ? body : []
        const parsed = inputs
          .map(validateVariable)
          .filter((item): item is CollectionVariableInput => Boolean(item))
        if (parsed.length === 0 || parsed.length !== inputs.length) {
          badRequest(res, 'invalid variables payload')
          return true
        }
        const existing = variablesOf(id)
        for (const input of parsed) {
          const index = existing.findIndex((item) => item.name === input.name)
          if (index === -1) {
            notFound(res, 'variable not found')
            return true
          }
          existing[index] = { ...input, value: existing[index].value }
        }
        collection.version = nextVersion(collection.version)
        collection.modTime = nowIso()
        send(res, 200, ok(parsed))
        return true
      }
      if (method === 'POST') {
        const body = await readJson(req)
        const inputs = Array.isArray(body) ? body : []
        const parsed = inputs
          .map(validateVariable)
          .filter((item): item is CollectionVariableInput => Boolean(item))
        if (parsed.length === 0 || parsed.length !== inputs.length) {
          badRequest(res, 'invalid variables payload')
          return true
        }
        const existing = variablesOf(id)
        const names = new Set(existing.map((item) => item.name))
        if (parsed.some((item) => names.has(item.name))) {
          badRequest(res, 'duplicate variable name')
          return true
        }
        if (new Set(parsed.map((item) => item.name)).size !== parsed.length) {
          badRequest(res, 'duplicate variable name')
          return true
        }
        for (const input of parsed) existing.push({ ...input, value: undefined })
        collection.version = nextVersion(collection.version)
        collection.modTime = nowIso()
        send(res, 200, ok({ added: parsed.length, variables: parsed }))
        return true
      }
      if (method === 'DELETE') {
        const body = await readJson(req)
        const names = Array.isArray(body)
          ? body.filter((item): item is string => typeof item === 'string')
          : []
        if (names.length === 0) {
          badRequest(res, 'variable names are required')
          return true
        }
        const before = variablesOf(id).length
        collectionVariables[id] = variablesOf(id).filter((item) => !names.includes(item.name))
        const deleted = before - collectionVariables[id].length
        collection.version = nextVersion(collection.version)
        collection.modTime = nowIso()
        send(res, 200, ok({ deleted }))
        return true
      }
      res.setHeader('Allow', 'GET, POST, PUT, DELETE')
      fail(res, 405, 10005, 'method not allowed')
      return true
    }

    if (method !== 'POST') {
      res.setHeader('Allow', 'POST')
      fail(res, 405, 10005, 'method not allowed')
      return true
    }
    if (action !== 'stop' && !collection.enable) {
      badRequest(res, 'collection is disabled')
      return true
    }
    if (action === 'stop') {
      collection.collectStatus = 'stopped'
    } else {
      refreshStatus(collection)
    }
    collection.version = nextVersion(collection.version)
    collection.modTime = nowIso()
    send(
      res,
      200,
      ok(action === 'restart' ? { restarted: true } : { status: collection.collectStatus }),
    )
    return true
  }

  const collectionMatch = pathname.match(/^\/collections\/([^/]+)$/)
  if (collectionMatch) {
    const collection = findCollection(collectionMatch[1])
    if (!collection) {
      notFound(res, 'collection not found')
      return true
    }
    if (method === 'GET') {
      send(res, 200, ok(collection))
      return true
    }
    if (method === 'PUT') {
      const body = await readObject(req)
      if (!body) {
        badRequest(res, 'invalid body')
        return true
      }
      const name = typeof body.name === 'string' ? body.name.trim() : collection.name
      const cycle = Number(body.collectorCycleMs)
      if (!name) {
        badRequest(res, 'name is required')
        return true
      }
      if (!Number.isFinite(cycle) || cycle <= 0) {
        badRequest(res, 'collectorCycleMs must be positive')
        return true
      }
      collection.name = name
      collection.collectorCycleMs = cycle
      if (isObject(body.protocolConfig)) collection.protocolConfig = body.protocolConfig
      collection.version = nextVersion(collection.version)
      collection.modTime = nowIso()
      send(res, 200, ok(collection))
      return true
    }
    if (method === 'DELETE') {
      const index = collections.findIndex((item) => item.id === collection.id)
      collections.splice(index, 1)
      delete collectionVariables[collection.id]
      send(res, 200, ok({ deleted: 1 }))
      return true
    }
    res.setHeader('Allow', 'GET, PUT, DELETE')
    fail(res, 405, 10005, 'method not allowed')
    return true
  }

  /* ---------------- 驱动 ---------------- */

  if (pathname === '/drivers/schema') {
    const driverModel = query.get('driverModel')
    const driver = drivers.find((item) => item.model === driverModel)
    if (!driver) {
      notFound(res, `driver ${driverModel} not found`)
      return true
    }
    send(
      res,
      200,
      ok({ driverType: driver.driverType, driverModel: driver.model, schema: driver.schema }),
    )
    return true
  }

  if (pathname === '/drivers') {
    if (method === 'GET') {
      const driverType = query.get('driverType')
      const list = drivers.filter((item) => !driverType || item.driverType === driverType)
      if (query.get('flat') === 'true') {
        send(res, 200, ok({ drivers: list }))
        return true
      }
      const grouped = new Map<string, Driver[]>()
      for (const driver of list) {
        const models = grouped.get(driver.driverType) ?? []
        models.push(driver)
        grouped.set(driver.driverType, models)
      }
      send(
        res,
        200,
        ok({
          groups: [...grouped.entries()].map(([type, models]) => ({
            driverType: type,
            models,
          })),
        }),
      )
      return true
    }
    if (method === 'POST') {
      const body = await readObject(req)
      if (!body) {
        badRequest(res, 'invalid body')
        return true
      }
      const driverType = typeof body.driverType === 'string' ? body.driverType.trim() : ''
      const model = typeof body.model === 'string' ? body.model.trim() : ''
      if (!driverType || !model || !isObject(body.schema) || !Array.isArray(body.schema.fields)) {
        badRequest(res, 'driverType, model and schema.fields are required')
        return true
      }
      if (drivers.some((item) => item.model === model)) {
        badRequest(res, `driver model ${model} already exists`)
        return true
      }
      const driver: Driver = {
        id: ulid(),
        driverType,
        model,
        builtinDriver: false,
        schema: body.schema as unknown as Driver['schema'],
        pluginName: typeof body.pluginName === 'string' ? body.pluginName : '',
        createdTime: nowIso(),
      }
      drivers.push(driver)
      send(res, 200, ok(driver))
      return true
    }
    res.setHeader('Allow', 'GET, POST')
    fail(res, 405, 10005, 'method not allowed')
    return true
  }

  const driverPluginMatch = pathname.match(/^\/drivers\/([^/]+)\/plugin$/)
  if (driverPluginMatch) {
    if (method !== 'POST') {
      res.setHeader('Allow', 'POST')
      fail(res, 405, 10005, 'method not allowed')
      return true
    }
    const driver = findDriver(driverPluginMatch[1])
    if (!driver) {
      notFound(res, 'driver not found')
      return true
    }
    if (driver.builtinDriver) {
      badRequest(res, 'built-in driver cannot upload plugins')
      return true
    }
    send(res, 200, ok(driver))
    return true
  }

  const driverMatch = pathname.match(/^\/drivers\/([^/]+)$/)
  if (driverMatch) {
    const driver = findDriver(driverMatch[1])
    if (!driver) {
      notFound(res, 'driver not found')
      return true
    }
    if (method === 'GET') {
      send(res, 200, ok(driver))
      return true
    }
    if (method === 'PUT') {
      const body = await readObject(req)
      if (!body || !isObject(body.schema) || !Array.isArray(body.schema.fields)) {
        badRequest(res, 'schema.fields is required')
        return true
      }
      driver.schema = body.schema as unknown as Driver['schema']
      if (typeof body.pluginName === 'string') driver.pluginName = body.pluginName
      send(res, 200, ok(driver))
      return true
    }
    if (method === 'DELETE') {
      if (driver.builtinDriver) {
        badRequest(res, 'built-in driver cannot be deleted')
        return true
      }
      drivers.splice(drivers.indexOf(driver), 1)
      send(res, 200, ok({ deleted: true }))
      return true
    }
    res.setHeader('Allow', 'GET, PUT, DELETE')
    fail(res, 405, 10005, 'method not allowed')
    return true
  }

  /* ---------------- 推送:连接器 ---------------- */

  if (pathname === '/push/connectors') {
    if (method === 'GET') {
      const { page, pageSize } = pageParams(query)
      const keyword = query.get('keyword')
      const type = query.get('type')
      const list = connectors
        .filter(
          (item) =>
            keywordMatch([item.name, item.id, item.pushConfig?.brokerUrl], keyword) &&
            (!type || item.type === type),
        )
        .map((item) => ({
          ...item,
          groupCount: pushGroups.filter((group) => group.connectorId === item.id).length,
        }))
      send(res, 200, ok(paginate(list, page, pageSize)))
      return true
    }
    if (method === 'POST') {
      const body = await readObject(req)
      const payload = body ? normalizeConnectorPayload(body) : undefined
      if (!payload) {
        badRequest(res, 'name and pushConfig.brokerUrl are required')
        return true
      }
      const connector: PushConnector = {
        id: ulid(),
        type: payload.type,
        name: payload.name,
        pushConfig: payload.pushConfig,
        runState: 'running',
        status: 'running',
        version: 1,
        createdTime: nowIso(),
        updatedTime: nowIso(),
      }
      connectors.unshift(connector)
      send(res, 200, ok(connector))
      return true
    }
    res.setHeader('Allow', 'GET, POST')
    fail(res, 405, 10005, 'method not allowed')
    return true
  }

  const connectorAction = pathname.match(/^\/push\/connectors\/([^/]+)\/(start|stop|restart)$/)
  if (connectorAction) {
    if (method !== 'POST') {
      res.setHeader('Allow', 'POST')
      fail(res, 405, 10005, 'method not allowed')
      return true
    }
    const connector = findConnector(connectorAction[1])
    if (!connector) {
      notFound(res, 'connector not found')
      return true
    }
    const connectorActionName = connectorAction[2]
    if (connectorActionName === 'start') {
      connector.runState = 'running'
      connector.status = 'running'
    } else if (connectorActionName === 'stop') {
      connector.runState = 'stopped'
      connector.status = 'stopped'
    } else if (connector.runState !== 'running') {
      badRequest(res, 'connection is stopped')
      return true
    } else {
      connector.status = 'running'
    }
    connector.updatedTime = nowIso()
    send(res, 200, ok({ status: connector.status }))
    return true
  }

  const connectorMatch = pathname.match(/^\/push\/connectors\/([^/]+)$/)
  if (connectorMatch) {
    const connector = findConnector(connectorMatch[1])
    if (!connector) {
      notFound(res, 'connector not found')
      return true
    }
    if (method === 'PUT') {
      const body = await readObject(req)
      const payload = body ? normalizeConnectorPayload(body) : undefined
      if (!payload) {
        badRequest(res, 'name and pushConfig.brokerUrl are required')
        return true
      }
      connector.name = payload.name
      connector.type = payload.type
      connector.pushConfig = payload.pushConfig
      connector.status = connector.runState === 'running' ? 'running' : 'stopped'
      connector.version += 1
      connector.updatedTime = nowIso()
      send(res, 200, ok(connector))
      return true
    }
    if (method === 'DELETE') {
      if (connector.status === 'running') {
        badRequest(res, 'stop the connector before deleting it')
        return true
      }
      const cascade = query.get('cascade') === 'true'
      if (cascade) {
        for (let index = pushGroups.length - 1; index >= 0; index -= 1) {
          if (pushGroups[index].connectorId === connector.id) pushGroups.splice(index, 1)
        }
      } else if (pushGroups.some((group) => group.connectorId === connector.id)) {
        badRequest(res, 'connector is referenced by push groups; delete them first or use cascade')
        return true
      }
      connectors.splice(connectors.indexOf(connector), 1)
      send(res, 200, ok({ deleted: true }))
      return true
    }
    res.setHeader('Allow', 'PUT, DELETE')
    fail(res, 405, 10005, 'method not allowed')
    return true
  }

  /* ---------------- 推送:推送组 ---------------- */

  if (pathname === '/push/groups') {
    if (method === 'GET') {
      const { page, pageSize } = pageParams(query)
      const keyword = query.get('keyword')
      const connectorId = query.get('connectorId')
      const list = pushGroups.filter(
        (item) =>
          keywordMatch([item.name, item.id], keyword) &&
          (!connectorId || item.connectorId === connectorId),
      )
      send(res, 200, ok(paginate(list, page, pageSize)))
      return true
    }
    if (method === 'POST') {
      const body = await readObject(req)
      const payload = body ? normalizeGroupPayload(body) : undefined
      if (!payload) {
        badRequest(res, 'name, connectorId, dataTopic and pushIntervalMs are required')
        return true
      }
      if (!findConnector(payload.connectorId)) {
        badRequest(res, 'connector not found')
        return true
      }
      if (!payload.allCollections && payload.collectionIds.length === 0) {
        badRequest(res, 'at least one collection is required when allCollections is false')
        return true
      }
      const group: PushGroup = {
        id: ulid(),
        ...payload,
        maxCollectionsPerPushMessage: payload.maxCollectionsPerPushMessage ?? 50,
        runState: 'running',
        status: findConnector(payload.connectorId)?.status === 'running' ? 'running' : 'stopped',
        version: 1,
        createdTime: nowIso(),
        updatedTime: nowIso(),
      }
      pushGroups.unshift(group)
      send(res, 200, ok(group))
      return true
    }
    res.setHeader('Allow', 'GET, POST')
    fail(res, 405, 10005, 'method not allowed')
    return true
  }

  const groupAction = pathname.match(/^\/push\/groups\/([^/]+)\/(start|stop|restart|payloads\/stream|commands\/stream)$/)
  if (groupAction) {
    const [, id, action] = groupAction
    const group = findGroup(id)
    if (!group) {
      notFound(res, 'push group not found')
      return true
    }
    if (action === 'payloads/stream') {
      if (method !== 'GET') {
        res.setHeader('Allow', 'GET')
        fail(res, 405, 10005, 'method not allowed')
        return true
      }
      res.writeHead(200, {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        Connection: 'keep-alive',
      })
      res.write('event: ready\ndata: {}\n\n')
      const selected = group.allCollections
        ? collections
        : collections.filter((item) => group.collectionIds.includes(item.id))
      const timer = setInterval(() => {
        if (group.runState !== 'running' || group.status !== 'running') return
        const source = selected[0]
        if (!source) return
        const values = variablesOf(source.id).reduce<Record<string, string>>((acc, item) => {
          acc[item.name] = formatValue(item.value)
          return acc
        }, {})
        const event = {
          groupId: group.id,
          connectorId: group.connectorId,
          connectorType: findConnector(group.connectorId)?.type ?? 'mqtt',
          topic: group.dataTopic,
          key: group.id,
          time: nowIso(),
          durationMs: 3,
          payload: {
            type: 'property',
            payload: [{ collectionId: source.id, time: Date.now(), property: values }],
          },
        }
        group.lastPushTime = event.time
        res.write(`event: payload\ndata: ${JSON.stringify(event)}\n\n`)
      }, Math.max(500, Math.min(5000, group.pushIntervalMs)))
      req.on('close', () => clearInterval(timer))
      return true
    }
    if (action === 'commands/stream') {
      if (method !== 'GET') {
        res.setHeader('Allow', 'GET')
        fail(res, 405, 10005, 'method not allowed')
        return true
      }
      res.writeHead(200, {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        Connection: 'keep-alive',
      })
      res.write('event: ready\ndata: {}\n\n')
      const target = group.allCollections
        ? collections[0]
        : collections.find((item) => group.collectionIds.includes(item.id))
      const timer = setInterval(() => {
        if (group.runState !== 'running' || group.status !== 'running' || !group.commandTopic) return
        const variable = target ? variablesOf(target.id)[0] : undefined
        const event = {
          groupId: group.id,
          connectorId: group.connectorId,
          connectorType: findConnector(group.connectorId)?.type ?? 'mqtt',
          topic: group.commandTopic,
          time: nowIso(),
          payload: {
            type: 'command',
            payload: [
              {
                collectionId: target?.id ?? 'collection',
                messageId: ulid(),
                property: variable ? { [variable.name]: '1' } : {},
              },
            ],
          },
        }
        res.write(`event: command\ndata: ${JSON.stringify(event)}\n\n`)
      }, Math.max(2500, Math.min(8000, group.pushIntervalMs + 1500)))
      req.on('close', () => clearInterval(timer))
      return true
    }
    if (method !== 'POST') {
      res.setHeader('Allow', 'POST')
      fail(res, 405, 10005, 'method not allowed')
      return true
    }
    const connector = findConnector(group.connectorId)
    if (action === 'start') {
      if (!connector || connector.status !== 'running') {
        badRequest(res, 'connection is not running')
        return true
      }
      group.runState = 'running'
      group.status = 'running'
      group.lastError = undefined
      group.lastPushTime = nowIso()
    } else if (action === 'stop') {
      group.runState = 'stopped'
      group.status = 'stopped'
    } else {
      if (group.runState !== 'running') {
        badRequest(res, 'push group is stopped')
        return true
      }
      if (!connector || connector.status !== 'running') {
        badRequest(res, 'connection is not running')
        return true
      }
      group.status = 'running'
      group.lastError = undefined
    }
    group.version += 1
    group.updatedTime = nowIso()
    send(res, 200, ok({ status: group.status }))
    return true
  }

  const groupMatch = pathname.match(/^\/push\/groups\/([^/]+)$/)
  if (groupMatch) {
    const group = findGroup(groupMatch[1])
    if (!group) {
      notFound(res, 'push group not found')
      return true
    }
    if (method === 'PUT') {
      const body = await readObject(req)
      const payload = body ? normalizeGroupPayload(body) : undefined
      if (!payload) {
        badRequest(res, 'name, connectorId, dataTopic and pushIntervalMs are required')
        return true
      }
      if (!findConnector(payload.connectorId)) {
        badRequest(res, 'connector not found')
        return true
      }
      Object.assign(group, payload)
      group.status = group.runState === 'running' &&
        findConnector(group.connectorId)?.status === 'running'
        ? 'running'
        : 'stopped'
      group.version += 1
      group.updatedTime = nowIso()
      send(res, 200, ok(group))
      return true
    }
    if (method === 'DELETE') {
      pushGroups.splice(pushGroups.indexOf(group), 1)
      send(res, 200, ok({ deleted: true }))
      return true
    }
    res.setHeader('Allow', 'PUT, DELETE')
    fail(res, 405, 10005, 'method not allowed')
    return true
  }

  if (pathname === '/simulator' && method === 'GET') {
    send(res, 200, ok(simulatorPayload()))
    return true
  }

  if (pathname === '/simulator/open' && method === 'POST') {
    const body = await readObject(req)
    const port = Number(body?.port ?? 2502)
    if (!Number.isInteger(port) || port < 1 || port > 65535) {
      badRequest(res, 'invalid simulator port')
      return true
    }
    simulatorState.port = port
    simulatorState.running = true
    send(res, 200, ok(simulatorPayload()))
    return true
  }

  if (pathname === '/simulator/close' && method === 'POST') {
    simulatorState.running = false
    send(res, 200, ok(simulatorPayload()))
    return true
  }

  if (pathname === '/simulator/points' && method === 'POST') {
    const body = await readObject(req)
    const address = Number(body?.address)
    const mode = body?.mode
    const dataType =
      body?.dataType === 'int32' || body?.dataType === 'float' ? body.dataType : 'int16'
    const words = dataType === 'int16' ? 1 : 2
    if (!Number.isInteger(address) || address < 1 || address + words - 1 > 65535) {
      badRequest(res, 'invalid simulator address')
      return true
    }
    if (mode !== 'random' && mode !== 'increment' && mode !== 'decrement') {
      badRequest(res, 'invalid simulator mode')
      return true
    }
    if (
      simulatorState.points.some((point) => {
        const pointWords = point.dataType === 'int16' ? 1 : 2
        return address < point.address + pointWords && address + words > point.address
      })
    ) {
      badRequest(res, 'simulator point already exists')
      return true
    }
    const point: SimulatorPoint = {
      address,
      name: typeof body?.name === 'string' ? body.name : '',
      description: typeof body?.description === 'string' ? body.description : '',
      registerType: 'holdingRegister',
      dataType: dataType as SimulatorDataType,
      mode: mode as SimulatorMode,
      value: 0,
    }
    simulatorState.points.push(point)
    simulatorState.points.sort((left, right) => left.address - right.address)
    send(res, 200, ok(point))
    return true
  }

  const simulatorPointMatch = pathname.match(/^\/simulator\/points\/(\d+)$/)
  if (simulatorPointMatch && method === 'DELETE') {
    const address = Number(simulatorPointMatch[1])
    const index = simulatorState.points.findIndex((point) => point.address === address)
    if (index < 0) {
      notFound(res, 'simulator point not found')
      return true
    }
    simulatorState.points.splice(index, 1)
    send(res, 200, ok({ deleted: true }))
    return true
  }

  return false
}
