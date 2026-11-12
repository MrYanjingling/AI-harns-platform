import type { IncomingMessage, ServerResponse } from 'node:http'
import type {
  CollectionDetail,
  CollectionVariable,
  CollectionVariableInput,
  Driver,
  MQTTConfig,
  PushConnector,
  PushConnectorPayload,
  PushGroup,
  PushGroupPayload,
} from '../src/api/types'
import {
  collectionVariables,
  collections,
  connectors,
  drivers,
  gatewayMeta,
  pushGroups,
  ulid,
} from './fixtures'
import { readJson, send } from './utils'

function ok<T>(data: T): { code: number; message: string; data: T } {
  return { code: 200, message: 'success', data }
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

function variablesOf(id: string): CollectionVariable[] {
  return collectionVariables[id] ?? (collectionVariables[id] = [])
}

function validateVariable(input: unknown): CollectionVariableInput | undefined {
  if (!isObject(input)) return undefined
  const name = typeof input.name === 'string' ? input.name.trim() : ''
  const dataType = typeof input.dataType === 'string' ? input.dataType.trim() : ''
  const address = typeof input.address === 'string' ? input.address.trim() : ''
  if (!name || !dataType || !address) return undefined
  return {
    ...(input as unknown as CollectionVariableInput),
    name,
    dataType,
    address,
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
  if (!pushConfig.brokerUrl || !pushConfig.dataTopic) return undefined
  return {
    type: typeof body.type === 'string' && body.type ? body.type : 'mqtt',
    name,
    pushConfig,
  }
}

function normalizeGroupPayload(body: Record<string, unknown>): PushGroupPayload | undefined {
  const name = typeof body.name === 'string' ? body.name.trim() : ''
  const connectorId = typeof body.connectorId === 'string' ? body.connectorId : ''
  const pushIntervalMs = Number(body.pushIntervalMs)
  if (!name || !connectorId || !Number.isFinite(pushIntervalMs) || pushIntervalMs <= 0) {
    return undefined
  }
  const allCollections = Boolean(body.allCollections)
  const collectionIds = Array.isArray(body.collectionIds)
    ? body.collectionIds.filter((item): item is string => typeof item === 'string')
    : []
  return {
    name,
    connectorId,
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
      if (parsed.length === 0) {
        badRequest(res, 'at least one variable is required')
        return true
      }
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
      res.setHeader('Allow', 'GET, POST, DELETE')
      fail(res, 405, 10005, 'method not allowed')
      return true
    }

    if (action === 'enable') {
      if (method !== 'PUT') {
        res.setHeader('Allow', 'PUT')
        fail(res, 405, 10005, 'method not allowed')
        return true
      }
      const body = await readObject(req)
      const enable = Boolean(body?.enable)
      if (!enable && collection.collectStatus === 'collecting') {
        badRequest(res, 'stop the collection before disabling it')
        return true
      }
      collection.enable = enable
      if (!enable) collection.collectStatus = 'stopped'
      collection.version = nextVersion(collection.version)
      collection.modTime = nowIso()
      send(res, 200, ok({ enable }))
      return true
    }

    if (action === 'action') {
      if (method !== 'PUT') {
        res.setHeader('Allow', 'PUT')
        fail(res, 405, 10005, 'method not allowed')
        return true
      }
      if (collection.collectStatus !== 'collecting') {
        badRequest(res, 'collection is not collecting')
        return true
      }
      const body = await readObject(req)
      const actions = Array.isArray(body?.actions) ? body.actions : []
      let accepted = 0
      for (const item of actions) {
        if (!isObject(item) || typeof item.name !== 'string') continue
        const target = variablesOf(id).find((variable) => variable.name === item.name)
        if (!target || target.accessMode === 'r') continue
        target.value = item.value
        accepted += 1
      }
      if (accepted === 0) {
        badRequest(res, 'no writable variable matched')
        return true
      }
      send(res, 200, ok({ accepted }))
      return true
    }

    if (method !== 'POST') {
      res.setHeader('Allow', 'POST')
      fail(res, 405, 10005, 'method not allowed')
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
      const list = connectors.filter((item) =>
        keywordMatch([item.name, item.id, item.pushConfig?.brokerUrl], keyword),
      )
      send(res, 200, ok(paginate(list, page, pageSize)))
      return true
    }
    if (method === 'POST') {
      const body = await readObject(req)
      const payload = body ? normalizeConnectorPayload(body) : undefined
      if (!payload) {
        badRequest(res, 'name, pushConfig.brokerUrl and pushConfig.dataTopic are required')
        return true
      }
      const connector: PushConnector = {
        id: ulid(),
        type: payload.type,
        name: payload.name,
        pushConfig: payload.pushConfig,
        status: 'stopped',
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

  const connectorAction = pathname.match(/^\/push\/connectors\/([^/]+)\/(start|stop)$/)
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
    connector.status = connectorAction[2] === 'start' ? 'running' : 'stopped'
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
        badRequest(res, 'name, pushConfig.brokerUrl and pushConfig.dataTopic are required')
        return true
      }
      connector.name = payload.name
      connector.type = payload.type
      connector.pushConfig = payload.pushConfig
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
      const list = pushGroups.filter((item) => keywordMatch([item.name, item.id], keyword))
      send(res, 200, ok(paginate(list, page, pageSize)))
      return true
    }
    if (method === 'POST') {
      const body = await readObject(req)
      const payload = body ? normalizeGroupPayload(body) : undefined
      if (!payload) {
        badRequest(res, 'name, connectorId and pushIntervalMs are required')
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
        badRequest(res, 'name, connectorId and pushIntervalMs are required')
        return true
      }
      if (!findConnector(payload.connectorId)) {
        badRequest(res, 'connector not found')
        return true
      }
      Object.assign(group, payload)
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

  return false
}
