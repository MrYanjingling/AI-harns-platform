import type { IncomingMessage, ServerResponse } from 'node:http'
import type {
  AuditLogInfo,
  OrgInfo,
  RoleInfo,
  ThingDetail,
  ThingInfo,
  ThingModelDetail,
  ThingModelFeature,
  ThingModelProperty,
  ThingModelPropertyInput,
  UserInfo,
} from '../src/api/types'
import {
  createDeviceId,
  seedAuditLogs,
  seedOrgs,
  seedRoles,
  seedThingModels,
  seedThings,
  seedUsers,
} from './fixtures'
import { readJson, readRaw, send } from './utils'

/** 平台服务的有状态 mock:会话、用户、角色、组织、物模型、设备、审计。 */

let users: UserInfo[] = structuredClone(seedUsers)
let roles: RoleInfo[] = structuredClone(seedRoles)
let orgs: OrgInfo[] = structuredClone(seedOrgs)
let models: ThingModelDetail[] = structuredClone(seedThingModels)
let things: ThingDetail[] = structuredClone(seedThings)
const auditLogs: AuditLogInfo[] = structuredClone(seedAuditLogs)

const sessions = new Map<string, string>()
const passwords = new Map<string, string>(
  seedUsers.map((user) => [
    user.id,
    user.userName === 'admin' ? 'harns@iot' : `${user.userName}@qwer`,
  ]),
)

const nowIso = () => new Date().toISOString()
const newId = () => createDeviceId()
const IDENTIFIER_PATTERN = /^[A-Za-z][A-Za-z0-9_]*$/
const DEVICE_KEY_PATTERN = /^[a-zA-Z][a-zA-Z0-9_-]{3,63}$/

class MockError extends Error {
  constructor(
    override message: string,
    readonly code: number,
    readonly status = 400,
    readonly issues?: Array<{ field: string; message: string }>,
  ) {
    super(message)
  }
}

function ok(res: ServerResponse, data: unknown, headers?: Record<string, string>) {
  send(res, 200, { code: 200, message: 'success', data }, headers)
}

function fail(res: ServerResponse, error: MockError) {
  send(res, error.status, {
    code: error.code,
    message: error.message,
    errors:
      error.issues && error.issues.length > 0
        ? error.issues.map((issue) => ({
            code: error.code,
            field: issue.field,
            message: issue.message,
          }))
        : [{ code: error.code, message: error.message }],
  })
}

function unauthorized(res: ServerResponse) {
  fail(res, new MockError('登录已过期,请重新登录。', 10032, 401))
}

function bearer(req: IncomingMessage): string {
  const header = req.headers.authorization ?? ''
  return header.replace(/^Bearer\s+/i, '').trim()
}

function currentUser(req: IncomingMessage): UserInfo | null {
  const userId = sessions.get(bearer(req))
  if (!userId) return null
  return users.find((user) => user.id === userId) ?? null
}

function requireUser(req: IncomingMessage, res: ServerResponse): UserInfo | null {
  const user = currentUser(req)
  if (!user) unauthorized(res)
  return user
}

function expandUser(user: UserInfo): UserInfo {
  return {
    ...user,
    roles: user.roleIds
      .map((id) => roles.find((role) => role.id === id))
      .filter((role): role is RoleInfo => Boolean(role)),
    orgs: user.orgIds
      .map((id) => orgs.find((org) => org.id === id))
      .filter((org): org is OrgInfo => Boolean(org)),
  }
}

function pagination(query: URLSearchParams) {
  const page = Math.max(1, Number(query.get('page')) || 1)
  const pageSize = Math.min(100, Math.max(1, Number(query.get('pageSize')) || 10))
  return { page, pageSize, offset: (page - 1) * pageSize }
}

function pageResult<T>(list: T[], page: number, pageSize: number) {
  return { total: list.length, page, pageSize, list: list.slice((page - 1) * pageSize, page * pageSize) }
}

function modelSummary(model: ThingModelDetail) {
  return {
    id: model.id,
    version: model.version,
    thingModelKey: model.thingModelKey,
    name: model.name,
    description: model.description,
    propertyCount: model.properties.length,
    createdUser: model.createdUser,
    updatedUser: model.updatedUser,
    createdTime: model.createdTime,
    updatedTime: model.updatedTime,
  }
}

function thingSummary(thing: ThingDetail): ThingInfo {
  const { deviceSecret, token, mqttPassword, image, features, ...summary } = thing
  void deviceSecret
  void token
  void mqttPassword
  void image
  void features
  return summary
}

const ONLINE_TTL_MS = 60_000

/** 与后端一致:enable=1 且最近 60 秒内有上报才算在线,不依赖存储值。 */
function effectiveThing(thing: ThingDetail): ThingDetail {
  const online =
    thing.enable === 1 &&
    thing.lastActiveTime !== null &&
    Date.now() - thing.lastActiveTime <= ONLINE_TTL_MS
  return { ...thing, onlineStatus: online ? 1 : 0 }
}

function toPropertyInput(property: ThingModelProperty): ThingModelPropertyInput {
  const { id, version, createdUser, updatedUser, createdTime, updatedTime, ...input } = property
  void id
  void version
  void createdUser
  void updatedUser
  void createdTime
  void updatedTime
  return input
}

function orgTree(): OrgInfo[] {
  const build = (parentId: string): OrgInfo[] =>
    orgs
      .filter((org) => (org.parentId || '') === parentId)
      .sort((a, b) => (a.sort ?? 0) - (b.sort ?? 0))
      .map((org) => ({ ...org, children: build(org.id) }))
  return build('')
}

function hasChildOrg(id: string): boolean {
  return orgs.some((org) => org.parentId === id)
}

function makeDeviceKey(): string {
  const alphabet = 'abcdefghijklmnopqrstuvwxyz0123456789'
  let key = ''
  for (let index = 0; index < 16; index += 1) {
    key += alphabet[Math.floor(Math.random() * alphabet.length)]
  }
  if (!/[a-z]/.test(key)) key = `a${key.slice(1)}`
  if (!/\d/.test(key)) key = `${key.slice(0, -1)}1`
  return `dk_${key}`
}

function randomHex(length: number): string {
  const hex = '0123456789abcdef'
  return Array.from({ length }, () => hex[Math.floor(Math.random() * 16)]).join('')
}

function ensureUnique(condition: boolean, message: string, code: number) {
  if (condition) throw new MockError(message, code)
}

function validatePropertyInput(
  property: ThingModelPropertyInput,
  fieldPrefix: string,
  seenIdentifiers: Set<string>,
  seenNames: Set<string>,
) {
  const identifier = (property.identifier ?? '').trim()
  const name = (property.name ?? '').trim()
  if (!IDENTIFIER_PATTERN.test(identifier) || identifier.length > 64) {
    throw new MockError('物模型校验失败。', 10045, 400, [
      {
        field: `${fieldPrefix}identifier`,
        message: '属性标识符只能由字母、数字和下划线组成,且必须以字母开头。',
      },
    ])
  }
  if (identifier === 'time' || identifier === 'device_id') {
    throw new MockError('物模型校验失败。', 10045, 400, [
      { field: `${fieldPrefix}identifier`, message: `属性标识符不能使用保留字 ${identifier}。` },
    ])
  }
  if (!name || !/^[A-Za-z]/.test(name) || name.length > 64) {
    throw new MockError('物模型校验失败。', 10045, 400, [
      { field: `${fieldPrefix}name`, message: '属性名称必须以字母开头且不超过 64 个字符。' },
    ])
  }
  if (seenIdentifiers.has(identifier)) {
    throw new MockError('物模型校验失败。', 10045, 400, [
      { field: `${fieldPrefix}identifier`, message: '属性标识符不能重复。' },
    ])
  }
  if (seenNames.has(name)) {
    throw new MockError('物模型校验失败。', 10045, 400, [
      { field: `${fieldPrefix}name`, message: '属性名称不能重复。' },
    ])
  }
  if (!property.dataType || !property.dataType.type) {
    throw new MockError('物模型校验失败。', 10045, 400, [
      { field: `${fieldPrefix}dataType.type`, message: '数据类型不能为空。' },
    ])
  }
  seenIdentifiers.add(identifier)
  seenNames.add(name)
}

function validateFeatureKeys(model: ThingModelDetail, features: Record<string, unknown> | undefined) {
  if (!features) return
  const defined = new Set(model.features.map((feature) => feature.identifier))
  for (const key of Object.keys(features)) {
    if (!defined.has(key)) {
      throw new MockError('设备校验失败。', 10054, 400, [
        { field: `features.${key}`, message: `物模型未定义特征 ${key}。` },
      ])
    }
  }
}

function createThingItem(
  model: ThingModelDetail,
  item: {
    deviceName: string
    deviceKey?: string
    deviceDesc?: string
    features?: Record<string, unknown>
    image?: string
  },
  usedNames: Set<string>,
  usedKeys: Set<string>,
): ThingDetail {
  const deviceName = (item.deviceName ?? '').trim()
  if (!deviceName || deviceName.length > 64) {
    throw new MockError('设备校验失败。', 10054, 400, [
      { field: 'deviceName', message: '设备名称不能为空且不超过 64 个字符。' },
    ])
  }
  if (usedNames.has(deviceName)) {
    throw new MockError('设备名称已存在。', 10049)
  }
  usedNames.add(deviceName)

  let deviceKey = (item.deviceKey ?? '').trim()
  if (deviceKey) {
    if (!DEVICE_KEY_PATTERN.test(deviceKey)) {
      throw new MockError('设备校验失败。', 10054, 400, [
        { field: 'deviceKey', message: '设备 Key 格式不正确。' },
      ])
    }
    ensureUnique(usedKeys.has(deviceKey), '设备 Key 已存在。', 10050)
  } else {
    do {
      deviceKey = makeDeviceKey()
    } while (usedKeys.has(deviceKey))
  }
  usedKeys.add(deviceKey)

  if (item.image && !item.image.startsWith('images/')) {
    throw new MockError('设备校验失败。', 10054, 400, [
      { field: 'image', message: '图片必须是 /iot/files 返回的 key。' },
    ])
  }
  validateFeatureKeys(model, item.features)

  const id = newId()
  const secret = randomHex(64)
  const timestamp = nowIso()
  return {
    id,
    version: 0,
    thingModelId: model.id,
    deviceName,
    deviceKey,
    upTopic: `iot/${deviceKey}/up`,
    downTopic: `iot/${deviceKey}/down`,
    deviceDesc: item.deviceDesc ?? '',
    mqttClientId: id,
    mqttUsername: deviceKey,
    enable: 1,
    onlineStatus: 0,
    lastActiveTime: null,
    createdUser: 'admin',
    updatedUser: 'admin',
    createdTime: timestamp,
    updatedTime: timestamp,
    deviceSecret: secret,
    token: randomHex(64),
    mqttPassword: secret,
    image: item.image ?? '',
    features: item.features ?? {},
  }
}

export async function handlePlatform(
  req: IncomingMessage,
  res: ServerResponse,
  method: string,
  pathname: string,
  query: URLSearchParams,
): Promise<boolean> {
  try {
    return await route(req, res, method, pathname, query)
  } catch (error) {
    if (error instanceof MockError) {
      fail(res, error)
      return true
    }
    fail(res, new MockError(error instanceof Error ? error.message : 'mock error', 10001))
    return true
  }
}

async function route(
  req: IncomingMessage,
  res: ServerResponse,
  method: string,
  pathname: string,
  query: URLSearchParams,
): Promise<boolean> {
  /* ---------------- 认证 ---------------- */

  if (method === 'POST' && pathname === '/auth/login') {
    const body = (await readJson(req)) as { username?: string; password?: string } | undefined
    const userName = (body?.username ?? '').trim()
    const password = body?.password ?? ''
    const user = users.find((item) => item.userName === userName)
    if (!user) throw new MockError('用户名或密码错误。', 10015)
    if (user.status !== 'enabled') throw new MockError('账号已禁用。', 10016)
    if (passwords.get(user.id) !== password) throw new MockError('用户名或密码错误。', 10015)
    const token = `mock-${user.id}-${Date.now()}`
    sessions.set(token, user.id)
    user.lastLoginTime = nowIso()
    ok(res, { token })
    return true
  }

  if (method === 'POST' && pathname === '/auth/logout') {
    sessions.delete(bearer(req))
    ok(res, null)
    return true
  }

  if (method === 'GET' && pathname === '/auth/index') {
    const user = requireUser(req, res)
    if (!user) return true
    ok(res, expandUser(user))
    return true
  }

  if (method === 'GET' && pathname === '/auth/publicKey') {
    // 返回空串时前端回退明文密码,避免 mock 需要实现 RSA。
    ok(res, '')
    return true
  }

  if (method === 'POST' && pathname === '/auth/verificationCode') {
    ok(res, { uuid: newId(), img: '' })
    return true
  }

  if (!pathname.startsWith('/iot')) return false

  /* ---------------- 机器接口(不走用户会话) ---------------- */

  if (pathname === '/iot/thingsAuth') {
    send(res, 200, {
      result: 'allow',
      is_superuser: false,
      client_attrs: { device_key: 'mock-device' },
      acl: [],
    })
    return true
  }
  if (pathname === '/iot/thingsPubMessage') {
    send(res, 204)
    return true
  }
  if (pathname === '/iot/thingsData') {
    ok(res, null)
    return true
  }

  const user = requireUser(req, res)
  if (!user) return true

  /* ---------------- 文件上传 ---------------- */

  if (method === 'POST' && pathname === '/iot/files') {
    const contentType = String(req.headers['content-type'] ?? '')
    if (!contentType.startsWith('multipart/form-data')) {
      throw new MockError('请求体不是 multipart/form-data。', 10001)
    }
    const raw = await readRaw(req)
    if (raw.length === 0) throw new MockError('缺少上传文件。', 10001)
    const now = new Date()
    const month = String(now.getMonth() + 1).padStart(2, '0')
    const ext = raw[0] === 0x89 && raw[1] === 0x50 ? 'png' : raw[0] === 0xff ? 'jpg' : 'png'
    ok(res, { key: `images/${now.getFullYear()}/${month}/mock-${newId()}.${ext}` })
    return true
  }

  /* ---------------- 用户 ---------------- */

  if (pathname === '/iot/users' && method === 'GET') {
    const keyword = (query.get('keyword') ?? '').toLowerCase()
    const filtered = users
      .filter((item) =>
        keyword
          ? [item.userName, item.userNo, item.nickName].some((value) =>
              value.toLowerCase().includes(keyword),
            )
          : true,
      )
      .sort((a, b) => a.userName.localeCompare(b.userName))
    const { page, pageSize } = pagination(query)
    ok(res, pageResult(filtered.map(expandUser), page, pageSize))
    return true
  }

  if (pathname === '/iot/users' && method === 'POST') {
    const body = (await readJson(req)) as Partial<UserInfo> & {
      password?: string
      roleIds?: string[]
      orgIds?: string[]
    }
    const userName = (body.userName ?? '').trim()
    const userNo = (body.userNo ?? '').trim()
    const nickName = (body.nickName ?? '').trim()
    if (!userName || !userNo || !nickName) {
      throw new MockError('用户名、工号与昵称不能为空。', 10001)
    }
    ensureUnique(
      users.some((item) => item.userName === userName),
      `用户名称 [${userName}] 已存在。`,
      10019,
    )
    const roleIds = body.roleIds ?? []
    const orgIds = body.orgIds ?? []
    for (const roleId of roleIds) {
      if (!roles.some((role) => role.id === roleId)) {
        throw new MockError('角色不存在。', 10021)
      }
    }
    for (const orgId of orgIds) {
      if (!orgs.some((org) => org.id === orgId)) {
        throw new MockError('组织不存在。', 10025)
      }
    }
    const id = newId()
    const created: UserInfo = {
      id,
      version: 0,
      userName,
      userNo,
      nickName,
      birthDate: body.birthDate ?? '',
      gender: body.gender ?? '',
      mail: body.mail ?? '',
      phone: body.phone ?? '',
      note: body.note ?? '',
      authMode: body.authMode ?? 'local',
      lastLoginTime: null,
      loginFailureTimes: 0,
      expireTimes: body.expireTimes ?? 0,
      status: body.status ?? 'enabled',
      createdUser: user.userName,
      updatedUser: user.userName,
      createdTime: nowIso(),
      updatedTime: nowIso(),
      roleIds,
      orgIds,
      roles: [],
      orgs: [],
    }
    passwords.set(id, body.password || `${userName}@qwer`)
    users = [...users, created]
    ok(res, expandUser(created), { ETag: '0' })
    return true
  }

  const userPasswordMatch = pathname.match(/^\/iot\/users\/([^/]+)\/password$/)
  if (userPasswordMatch && method === 'PUT') {
    const target = users.find((item) => item.id === userPasswordMatch[1])
    if (!target) throw new MockError('用户不存在。', 10018)
    passwords.set(target.id, `${target.userName}@qwer`)
    target.version += 1
    target.updatedTime = nowIso()
    target.updatedUser = user.userName
    for (const [token, userId] of sessions) {
      if (userId === target.id) sessions.delete(token)
    }
    ok(res, expandUser(target), { ETag: String(target.version) })
    return true
  }

  const userMatch = pathname.match(/^\/iot\/users\/([^/]+)$/)
  if (userMatch) {
    const target = users.find((item) => item.id === userMatch[1])
    if (!target) throw new MockError('用户不存在。', 10018)
    if (method === 'GET') {
      ok(res, expandUser(target), { ETag: String(target.version) })
      return true
    }
    if (method === 'DELETE') {
      users = users.filter((item) => item.id !== target.id)
      passwords.delete(target.id)
      for (const [token, userId] of sessions) {
        if (userId === target.id) sessions.delete(token)
      }
      ok(res, null)
      return true
    }
    if (method === 'PUT') {
      const body = (await readJson(req)) as Partial<UserInfo> & { password?: string }
      if (body.userName) {
        ensureUnique(
          users.some((item) => item.id !== target.id && item.userName === body.userName),
          `用户名称 [${body.userName}] 已存在。`,
          10019,
        )
        target.userName = body.userName
      }
      const mutable: Array<keyof UserInfo> = [
        'userNo',
        'nickName',
        'birthDate',
        'gender',
        'mail',
        'phone',
        'note',
        'authMode',
        'expireTimes',
        'status',
      ]
      for (const key of mutable) {
        const value = body[key]
        if (value !== undefined) {
          // 仅复制请求显式提供的字段,与后端指针语义一致。
          Object.assign(target, { [key]: value })
        }
      }
      if (body.password) passwords.set(target.id, body.password)
      target.version += 1
      target.updatedTime = nowIso()
      target.updatedUser = user.userName
      ok(res, expandUser(target), { ETag: String(target.version) })
      return true
    }
  }

  /* ---------------- 角色 ---------------- */

  if (pathname === '/iot/roles' && method === 'GET') {
    const keyword = (query.get('keyword') ?? '').toLowerCase()
    const filtered = roles
      .filter((item) =>
        keyword
          ? [item.name, item.code].some((value) => value.toLowerCase().includes(keyword))
          : true,
      )
      .sort((a, b) => a.name.localeCompare(b.name))
    const { page, pageSize } = pagination(query)
    ok(res, pageResult(filtered, page, pageSize))
    return true
  }

  if (pathname === '/iot/roles' && method === 'POST') {
    const body = (await readJson(req)) as Partial<RoleInfo>
    const name = (body.name ?? '').trim()
    const code = (body.code ?? '').trim()
    if (!name || !code) throw new MockError('角色名称与编码不能为空。', 10001)
    ensureUnique(
      roles.some((item) => item.name === name),
      `角色名称 [${name}] 已存在。`,
      10023,
    )
    ensureUnique(
      roles.some((item) => item.code === code),
      `角色编码 [${code}] 已存在。`,
      10024,
    )
    const created: RoleInfo = {
      id: newId(),
      version: 0,
      name,
      code,
      description: body.description ?? '',
      status: body.status ?? 'enabled',
      createdUser: user.userName,
      updatedUser: user.userName,
      createdTime: nowIso(),
      updatedTime: nowIso(),
    }
    roles = [...roles, created]
    ok(res, created, { ETag: '0' })
    return true
  }

  const roleMatch = pathname.match(/^\/iot\/roles\/([^/]+)$/)
  if (roleMatch) {
    const target = roles.find((item) => item.id === roleMatch[1])
    if (!target) throw new MockError('角色不存在。', 10022)
    if (method === 'GET') {
      ok(res, target, { ETag: String(target.version) })
      return true
    }
    if (method === 'DELETE') {
      roles = roles.filter((item) => item.id !== target.id)
      users = users.map((item) => ({
        ...item,
        roleIds: item.roleIds.filter((id) => id !== target.id),
      }))
      ok(res, null)
      return true
    }
    if (method === 'PUT') {
      const body = (await readJson(req)) as Partial<RoleInfo>
      if (body.name) target.name = body.name
      if (body.code) target.code = body.code
      if (body.description !== undefined) target.description = body.description
      if (body.status) target.status = body.status
      target.version += 1
      target.updatedTime = nowIso()
      target.updatedUser = user.userName
      ok(res, target, { ETag: String(target.version) })
      return true
    }
  }

  /* ---------------- 组织 ---------------- */

  if (pathname === '/iot/orgs/tree' && method === 'GET') {
    ok(res, orgTree())
    return true
  }

  if (pathname === '/iot/orgs' && method === 'POST') {
    const body = (await readJson(req)) as Partial<OrgInfo>
    const code = (body.code ?? '').trim()
    const name = (body.name ?? '').trim()
    if (!code || !name) throw new MockError('组织编码与名称不能为空。', 10001)
    const parentId = !body.parentId || body.parentId === '0' ? '' : body.parentId
    if (parentId && !orgs.some((org) => org.id === parentId)) {
      throw new MockError('上级组织不存在。', 10025)
    }
    const created: OrgInfo = {
      id: newId(),
      version: 0,
      code,
      name,
      parentId,
      sort: body.sort ?? 0,
      description: body.description ?? '',
      remark: body.remark ?? '',
      status: body.status ?? 'enabled',
      createdUser: user.userName,
      updatedUser: user.userName,
      createdTime: nowIso(),
      updatedTime: nowIso(),
    }
    orgs = [...orgs, created]
    ok(res, created, { ETag: '0' })
    return true
  }

  const orgMatch = pathname.match(/^\/iot\/orgs\/([^/]+)$/)
  if (orgMatch) {
    const target = orgs.find((item) => item.id === orgMatch[1])
    if (!target) throw new MockError('组织不存在。', 10026)
    if (method === 'GET') {
      ok(res, { ...target, children: orgTree().find((item) => item.id === target.id)?.children ?? [] })
      return true
    }
    if (method === 'DELETE') {
      if (hasChildOrg(target.id)) {
        throw new MockError('组织存在子组织,无法删除。', 10027)
      }
      orgs = orgs.filter((item) => item.id !== target.id)
      users = users.map((item) => ({
        ...item,
        orgIds: item.orgIds.filter((id) => id !== target.id),
      }))
      ok(res, null)
      return true
    }
    if (method === 'PUT') {
      const body = (await readJson(req)) as Partial<OrgInfo>
      if (body.code) target.code = body.code
      if (body.name) target.name = body.name
      if (body.parentId !== undefined) {
        target.parentId = body.parentId === '0' ? '' : body.parentId
      }
      if (body.sort !== undefined) target.sort = body.sort
      if (body.description !== undefined) target.description = body.description
      if (body.remark !== undefined) target.remark = body.remark
      if (body.status) target.status = body.status
      target.version += 1
      target.updatedTime = nowIso()
      target.updatedUser = user.userName
      ok(res, target, { ETag: String(target.version) })
      return true
    }
  }

  /* ---------------- 审计日志 ---------------- */

  if (pathname === '/iot/auditLogs' && method === 'GET') {
    const app = query.get('app') ?? ''
    const userId = query.get('userId') ?? ''
    const path = (query.get('path') ?? '').toLowerCase()
    const methodFilter = query.get('method') ?? ''
    const resultCode = query.get('resultCode')
    const start = query.get('start')
    const end = query.get('end')
    const filtered = auditLogs
      .filter((log) => (app ? log.app === app : true))
      .filter((log) => (userId ? log.userId === userId : true))
      .filter((log) => (path ? log.path.toLowerCase().includes(path) : true))
      .filter((log) => (methodFilter ? log.method === methodFilter : true))
      .filter((log) => (resultCode ? String(log.resultCode) === resultCode : true))
      .filter((log) => {
        if (!start) return true
        const time = new Date(log.createdTime).getTime()
        return time >= new Date(start.replace(' ', 'T')).getTime()
      })
      .filter((log) => {
        if (!end) return true
        const time = new Date(log.createdTime).getTime()
        return time <= new Date(end.replace(' ', 'T')).getTime()
      })
      .sort((a, b) => new Date(b.createdTime).getTime() - new Date(a.createdTime).getTime())
    const { page, pageSize } = pagination(query)
    ok(res, pageResult(filtered, page, pageSize))
    return true
  }

  /* ---------------- 物模型 ---------------- */

  if (pathname === '/iot/thingModels' && method === 'GET') {
    const keyword = (query.get('keyword') ?? '').toLowerCase()
    const filtered = models
      .filter((model) =>
        keyword
          ? [model.name, model.thingModelKey].some((value) =>
              value.toLowerCase().includes(keyword),
            )
          : true,
      )
      .sort((a, b) => new Date(b.createdTime).getTime() - new Date(a.createdTime).getTime())
    const { page, pageSize } = pagination(query)
    ok(res, pageResult(filtered.map(modelSummary), page, pageSize))
    return true
  }

  if (pathname === '/iot/thingModels' && method === 'POST') {
    const body = (await readJson(req)) as {
      thingModelKey?: string
      name?: string
      description?: string
      properties?: ThingModelPropertyInput[]
      features?: ThingModelFeature[]
    }
    const name = (body.name ?? '').trim()
    if (!name) throw new MockError('物模型名称不能为空。', 10045)
    ensureUnique(
      models.some((model) => model.name === name),
      `物模型名称 [${name}] 已存在。`,
      10043,
    )
    const key = (body.thingModelKey ?? '').trim() || `tm_${randomHex(13)}`
    ensureUnique(
      models.some((model) => model.thingModelKey === key),
      `物模型 Key [${key}] 已存在。`,
      10047,
    )
    const seenIdentifiers = new Set<string>()
    const seenNames = new Set<string>()
    const properties = (body.properties ?? []).map((property, index) => {
      validatePropertyInput(property, `properties[${index}].`, seenIdentifiers, seenNames)
      const timestamp = nowIso()
      return {
        ...property,
        id: newId(),
        version: 0,
        createdUser: user.userName,
        updatedUser: user.userName,
        createdTime: timestamp,
        updatedTime: timestamp,
      }
    })
    const created: ThingModelDetail = {
      id: newId(),
      version: 0,
      thingModelKey: key,
      name,
      description: body.description ?? '',
      propertyCount: properties.length,
      createdUser: user.userName,
      updatedUser: user.userName,
      createdTime: nowIso(),
      updatedTime: nowIso(),
      properties,
      features: body.features ?? [],
    }
    models = [...models, created]
    ok(res, created)
    return true
  }

  const modelTslMatch = pathname.match(/^\/iot\/thingModels\/([^/]+)\/tsl$/)
  if (modelTslMatch) {
    const model = models.find((item) => item.id === modelTslMatch[1])
    if (!model) throw new MockError('物模型不存在。', 10042)
    if (method === 'GET') {
      ok(res, {
        thingModelKey: model.thingModelKey,
        name: model.name,
        description: model.description,
        properties: model.properties.map(toPropertyInput),
        features: model.features,
      })
      return true
    }
    if (method === 'PUT') {
      const body = (await readJson(req)) as {
        thingModelKey?: string
        name?: string
        description?: string
        properties?: ThingModelPropertyInput[]
        features?: ThingModelFeature[]
      }
      if (body.thingModelKey && body.thingModelKey !== model.thingModelKey) {
        throw new MockError('物模型 Key 不可修改。', 10045, 400, [
          { field: 'thingModelKey', message: '物模型 Key 不可修改。' },
        ])
      }
      if (!Array.isArray(body.properties)) {
        throw new MockError('物模型校验失败。', 10045, 400, [
          { field: 'properties', message: 'properties 不能为空。' },
        ])
      }
      if (body.name) {
        ensureUnique(
          models.some((item) => item.id !== model.id && item.name === body.name),
          `物模型名称 [${body.name}] 已存在。`,
          10043,
        )
        model.name = body.name
      }
      if (body.description !== undefined) model.description = body.description
      const seenIdentifiers = new Set<string>()
      const seenNames = new Set<string>()
      const timestamp = nowIso()
      model.properties = body.properties.map((property, index) => {
        validatePropertyInput(property, `properties[${index}].`, seenIdentifiers, seenNames)
        return {
          ...property,
          id: newId(),
          version: 0,
          createdUser: user.userName,
          updatedUser: user.userName,
          createdTime: timestamp,
          updatedTime: timestamp,
        }
      })
      if (Array.isArray(body.features)) model.features = body.features
      model.propertyCount = model.properties.length
      model.version += 1
      model.updatedTime = nowIso()
      model.updatedUser = user.userName
      ok(res, model)
      return true
    }
  }

  const modelPropertiesMatch = pathname.match(/^\/iot\/thingModels\/([^/]+)\/properties$/)
  if (modelPropertiesMatch) {
    const model = models.find((item) => item.id === modelPropertiesMatch[1])
    if (!model) throw new MockError('物模型不存在。', 10042)
    if (method === 'POST') {
      const body = (await readJson(req)) as ThingModelPropertyInput[]
      if (!Array.isArray(body) || body.length === 0) {
        throw new MockError('属性不能为空。', 10045)
      }
      const seenIdentifiers = new Set(model.properties.map((property) => property.identifier))
      const seenNames = new Set(model.properties.map((property) => property.name))
      const timestamp = nowIso()
      const created = body.map((property, index) => {
        validatePropertyInput(property, `[${index}].`, seenIdentifiers, seenNames)
        return {
          ...property,
          id: newId(),
          version: 0,
          createdUser: user.userName,
          updatedUser: user.userName,
          createdTime: timestamp,
          updatedTime: timestamp,
        }
      })
      model.properties = [...model.properties, ...created]
      model.propertyCount = model.properties.length
      model.version += 1
      model.updatedTime = timestamp
      ok(res, created)
      return true
    }
    if (method === 'DELETE') {
      const body = (await readJson(req)) as string[]
      if (!Array.isArray(body)) throw new MockError('请求体必须是标识符数组。', 10001)
      const before = model.properties.length
      model.properties = model.properties.filter(
        (property) => !body.includes(property.identifier),
      )
      model.propertyCount = model.properties.length
      model.version += 1
      model.updatedTime = nowIso()
      ok(res, { deleted: before - model.properties.length })
      return true
    }
  }

  const modelPropertyMatch = pathname.match(
    /^\/iot\/thingModels\/([^/]+)\/properties\/([^/]+)$/,
  )
  if (modelPropertyMatch && method === 'PUT') {
    const model = models.find((item) => item.id === modelPropertyMatch[1])
    if (!model) throw new MockError('物模型不存在。', 10042)
    const identifier = decodeURIComponent(modelPropertyMatch[2])
    const property = model.properties.find((item) => item.identifier === identifier)
    if (!property) throw new MockError('属性不存在。', 10044)
    const body = (await readJson(req)) as ThingModelPropertyInput
    if (body.identifier && body.identifier !== identifier) {
      throw new MockError('物模型校验失败。', 10045, 400, [
        { field: 'identifier', message: '路径中的标识符与请求体不一致。' },
      ])
    }
    const seenIdentifiers = new Set(
      model.properties.filter((item) => item.identifier !== identifier).map((item) => item.identifier),
    )
    const seenNames = new Set(
      model.properties.filter((item) => item.identifier !== identifier).map((item) => item.name),
    )
    validatePropertyInput(body, '', seenIdentifiers, seenNames)
    Object.assign(property, body, {
      identifier,
      version: property.version + 1,
      updatedUser: user.userName,
      updatedTime: nowIso(),
    })
    model.version += 1
    model.updatedTime = nowIso()
    ok(res, property)
    return true
  }

  const modelMatch = pathname.match(/^\/iot\/thingModels\/([^/]+)$/)
  if (modelMatch) {
    const model = models.find((item) => item.id === modelMatch[1])
    if (!model) throw new MockError('物模型不存在。', 10042)
    if (method === 'GET') {
      ok(res, model)
      return true
    }
    if (method === 'DELETE') {
      models = models.filter((item) => item.id !== model.id)
      ok(res, null)
      return true
    }
    if (method === 'PUT') {
      const body = (await readJson(req)) as { name?: string; description?: string }
      if (body.name) {
        ensureUnique(
          models.some((item) => item.id !== model.id && item.name === body.name),
          `物模型名称 [${body.name}] 已存在。`,
          10043,
        )
        model.name = body.name
      }
      if (body.description !== undefined) model.description = body.description
      model.version += 1
      model.updatedTime = nowIso()
      model.updatedUser = user.userName
      ok(res, modelSummary(model))
      return true
    }
  }

  /* ---------------- 设备(Thing) ---------------- */

  if (pathname === '/iot/things' && method === 'GET') {
    const keyword = (query.get('keyword') ?? '').toLowerCase()
    const filtered = things
      .filter((thing) =>
        keyword
          ? [thing.deviceName, thing.deviceKey].some((value) =>
              value.toLowerCase().includes(keyword),
            )
          : true,
      )
      .sort((a, b) => new Date(b.createdTime).getTime() - new Date(a.createdTime).getTime())
      .map((thing) => thingSummary(effectiveThing(thing)))
    const { page, pageSize } = pagination(query)
    ok(res, pageResult(filtered, page, pageSize))
    return true
  }

  if (pathname === '/iot/things' && method === 'POST') {
    const body = (await readJson(req)) as {
      thingModelId?: string
      things?: Array<{
        deviceName: string
        deviceKey?: string
        deviceDesc?: string
        features?: Record<string, unknown>
        image?: string
      }>
    }
    const model = models.find((item) => item.id === body.thingModelId)
    if (!model) throw new MockError('物模型不存在。', 10042)
    if (!Array.isArray(body.things) || body.things.length < 1 || body.things.length > 100) {
      throw new MockError('单批创建 1~100 台设备。', 10054)
    }
    const usedNames = new Set(
      things.filter((thing) => thing.thingModelId === model.id).map((thing) => thing.deviceName),
    )
    const usedKeys = new Set(things.map((thing) => thing.deviceKey))
    const created = body.things.map((item) =>
      createThingItem(model, item, usedNames, usedKeys),
    )
    things = [...things, ...created]
    ok(res, created)
    return true
  }

  if (pathname === '/iot/things' && method === 'DELETE') {
    const body = (await readJson(req)) as string[]
    if (!Array.isArray(body)) throw new MockError('请求体必须是 id 数组。', 10001)
    const before = things.length
    things = things.filter((thing) => !body.includes(thing.id))
    ok(res, { deleted: before - things.length })
    return true
  }

  const thingMatch = pathname.match(/^\/iot\/things\/([^/]+)$/)
  if (thingMatch) {
    const thing = things.find((item) => item.id === thingMatch[1])
    if (!thing) throw new MockError('设备不存在。', 10048)
    if (method === 'GET') {
      ok(res, effectiveThing(thing))
      return true
    }
    if (method === 'PUT') {
      const body = (await readJson(req)) as Partial<ThingDetail>
      if (body.deviceName) {
        const name = body.deviceName.trim()
        if (!name || name.length > 64) throw new MockError('设备名称不能为空。', 10054)
        ensureUnique(
          things.some(
            (item) =>
              item.id !== thing.id &&
              item.thingModelId === thing.thingModelId &&
              item.deviceName === name,
          ),
          '设备名称已存在。',
          10049,
        )
        thing.deviceName = name
      }
      if (body.deviceDesc !== undefined) thing.deviceDesc = body.deviceDesc
      if (body.mqttClientId) {
        ensureUnique(
          things.some((item) => item.id !== thing.id && item.mqttClientId === body.mqttClientId),
          'MQTT clientId 已存在。',
          10052,
        )
        thing.mqttClientId = body.mqttClientId
      }
      if (body.mqttUsername) {
        ensureUnique(
          things.some((item) => item.id !== thing.id && item.mqttUsername === body.mqttUsername),
          'MQTT username 已存在。',
          10053,
        )
        thing.mqttUsername = body.mqttUsername
      }
      if (body.mqttPassword) thing.mqttPassword = body.mqttPassword
      if (body.enable !== undefined) thing.enable = body.enable
      if (body.features !== undefined) {
        const model = models.find((item) => item.id === thing.thingModelId)
        if (model) validateFeatureKeys(model, body.features)
        thing.features = body.features
      }
      if (body.image !== undefined) {
        if (body.image && !body.image.startsWith('images/')) {
          throw new MockError('图片必须是 /iot/files 返回的 key。', 10054)
        }
        thing.image = body.image
      }
      thing.version += 1
      thing.updatedTime = nowIso()
      thing.updatedUser = user.userName
      ok(res, effectiveThing(thing))
      return true
    }
  }

  return false
}
