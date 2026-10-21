import { platformData, platformPage } from './platform'
import type {
  CreateOrgRequest,
  CreateRoleRequest,
  CreateUserRequest,
  OrgInfo,
  PlatformPage,
  RoleInfo,
  UpdateOrgRequest,
  UpdateRoleRequest,
  UpdateUserRequest,
  UserInfo,
} from './types'

export interface IdentityQuery {
  page: number
  pageSize: number
  keyword?: string
}

/* ---------- 用户 ---------- */

export function listUsers(query: IdentityQuery): Promise<PlatformPage<UserInfo>> {
  return platformPage<UserInfo>('/iot/users', {
    page: query.page,
    pageSize: query.pageSize,
    keyword: query.keyword,
  })
}

export function getUser(id: string): Promise<UserInfo> {
  return platformData<UserInfo>(`/iot/users/${id}`)
}

export function createUser(payload: CreateUserRequest): Promise<UserInfo> {
  return platformData<UserInfo>('/iot/users', { method: 'POST', body: payload })
}

export function updateUser(id: string, payload: UpdateUserRequest): Promise<UserInfo> {
  return platformData<UserInfo>(`/iot/users/${id}`, { method: 'PUT', body: payload })
}

export function deleteUser(id: string): Promise<null> {
  return platformData<null>(`/iot/users/${id}`, { method: 'DELETE' })
}

/** 后端把密码重置为「用户名@123」并返回更新后的用户。 */
export function resetUserPassword(id: string): Promise<UserInfo> {
  return platformData<UserInfo>(`/iot/users/${id}/password`, { method: 'PUT' })
}

/* ---------- 角色 ---------- */

export function listRoles(query: IdentityQuery): Promise<PlatformPage<RoleInfo>> {
  return platformPage<RoleInfo>('/iot/roles', {
    page: query.page,
    pageSize: query.pageSize,
    keyword: query.keyword,
  })
}

export function getRole(id: string): Promise<RoleInfo> {
  return platformData<RoleInfo>(`/iot/roles/${id}`)
}

export function createRole(payload: CreateRoleRequest): Promise<RoleInfo> {
  return platformData<RoleInfo>('/iot/roles', { method: 'POST', body: payload })
}

export function updateRole(id: string, payload: UpdateRoleRequest): Promise<RoleInfo> {
  return platformData<RoleInfo>(`/iot/roles/${id}`, { method: 'PUT', body: payload })
}

export function deleteRole(id: string): Promise<null> {
  return platformData<null>(`/iot/roles/${id}`, { method: 'DELETE' })
}

/* ---------- 组织 ---------- */

export function getOrgTree(): Promise<OrgInfo[]> {
  return platformData<OrgInfo[]>('/iot/orgs/tree')
}

export function getOrg(id: string): Promise<OrgInfo> {
  return platformData<OrgInfo>(`/iot/orgs/${id}`)
}

export function createOrg(payload: CreateOrgRequest): Promise<OrgInfo> {
  return platformData<OrgInfo>('/iot/orgs', { method: 'POST', body: payload })
}

export function updateOrg(id: string, payload: UpdateOrgRequest): Promise<OrgInfo> {
  return platformData<OrgInfo>(`/iot/orgs/${id}`, { method: 'PUT', body: payload })
}

export function deleteOrg(id: string): Promise<null> {
  return platformData<null>(`/iot/orgs/${id}`, { method: 'DELETE' })
}
