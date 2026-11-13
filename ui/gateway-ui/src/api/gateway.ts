import { buildQuery, request, type RequestOptions } from '@harns/core'
import type {
  AddVariablesResult,
  Collection,
  CollectionAction,
  CollectionDetail,
  CollectionPayload,
  CollectionVariable,
  CollectionVariableInput,
  Driver,
  DriverGroup,
  DriverPayload,
  DriverSchemaResponse,
  GatewayCpuResponse,
  GatewayDiskResponse,
  GatewayMemResponse,
  GatewayMeta,
  PageResp,
  PushConnector,
  PushConnectorPayload,
  PushGroup,
  PushGroupPayload,
} from './types'

/** 网关与平台共用统一响应信封。 */
interface GatewayEnvelope<T> {
  code: number
  message?: string
  data: T
}

async function gatewayData<T>(path: string, options?: RequestOptions): Promise<T> {
  const { data } = await request<GatewayEnvelope<T>>('gateway', path, options)
  return data.data
}

export function gatewayBaseUrl(): string {
  return import.meta.env.VITE_GATEWAY_BASE || '/api/v1'
}

interface ListQuery {
  page?: number
  pageSize?: number
  keyword?: string
  driverType?: string
  driverModel?: string
}

/* ---------------- 采集器 ---------------- */

export function listCollections(query: ListQuery = {}): Promise<PageResp<Collection>> {
  return gatewayData<PageResp<Collection>>(
    `/collections${buildQuery({
      page: query.page,
      pageSize: query.pageSize,
      keyword: query.keyword,
      driverType: query.driverType,
      driverModel: query.driverModel,
    })}`,
  )
}

export function getCollection(id: string): Promise<CollectionDetail> {
  return gatewayData<CollectionDetail>(`/collections/${id}`)
}

export function createCollection(payload: CollectionPayload): Promise<CollectionDetail> {
  return gatewayData<CollectionDetail>('/collections', { method: 'POST', body: payload })
}

export function updateCollection(
  id: string,
  payload: CollectionPayload,
): Promise<CollectionDetail> {
  return gatewayData<CollectionDetail>(`/collections/${id}`, {
    method: 'PUT',
    body: payload,
  })
}

export function deleteCollection(id: string): Promise<{ deleted: number }> {
  return gatewayData<{ deleted: number }>(`/collections/${id}`, { method: 'DELETE' })
}

export function startCollection(id: string): Promise<{ status: string }> {
  return gatewayData<{ status: string }>(`/collections/${id}/start`, { method: 'POST' })
}

export function stopCollection(id: string): Promise<{ status: string }> {
  return gatewayData<{ status: string }>(`/collections/${id}/stop`, { method: 'POST' })
}

export function restartCollection(id: string): Promise<{ restarted: boolean }> {
  return gatewayData<{ restarted: boolean }>(`/collections/${id}/restart`, { method: 'POST' })
}

export function setCollectionEnable(id: string, enable: boolean): Promise<{ enable: boolean }> {
  return gatewayData<{ enable: boolean }>(`/collections/${id}/enable`, {
    method: 'PUT',
    body: { enable },
  })
}

export function listCollectionVariables(
  id: string,
  query: ListQuery = {},
): Promise<PageResp<CollectionVariable>> {
  return gatewayData<PageResp<CollectionVariable>>(
    `/collections/${id}/variables${buildQuery({
      page: query.page,
      pageSize: query.pageSize,
      keyword: query.keyword,
    })}`,
  )
}

export function addCollectionVariables(
  id: string,
  variables: CollectionVariableInput[],
): Promise<AddVariablesResult> {
  return gatewayData<AddVariablesResult>(`/collections/${id}/variables`, {
    method: 'POST',
    body: variables,
  })
}

export function deleteCollectionVariables(
  id: string,
  names: string[],
): Promise<{ deleted: number }> {
  return gatewayData<{ deleted: number }>(`/collections/${id}/variables`, {
    method: 'DELETE',
    body: names,
  })
}

export function deliverCollectionAction(
  id: string,
  actions: CollectionAction[],
): Promise<{ accepted: number }> {
  return gatewayData<{ accepted: number }>(`/collections/${id}/action`, {
    method: 'PUT',
    body: { actions },
  })
}

/* ---------------- 驱动 ---------------- */

export async function listDriverGroups(driverType?: string): Promise<DriverGroup[]> {
  const data = await gatewayData<{ groups: DriverGroup[] }>(
    `/drivers${buildQuery({ driverType })}`,
  )
  return data.groups ?? []
}

export async function listDriversFlat(driverType?: string): Promise<Driver[]> {
  const data = await gatewayData<{ drivers: Driver[] }>(
    `/drivers${buildQuery({ driverType, flat: true })}`,
  )
  return data.drivers ?? []
}

export function getDriver(id: string): Promise<Driver> {
  return gatewayData<Driver>(`/drivers/${id}`)
}

export function getDriverSchema(driverModel: string): Promise<DriverSchemaResponse> {
  return gatewayData<DriverSchemaResponse>(
    `/drivers/schema${buildQuery({ driverModel })}`,
  )
}

export function createDriver(payload: DriverPayload): Promise<Driver> {
  return gatewayData<Driver>('/drivers', { method: 'POST', body: payload })
}

export function updateDriver(id: string, payload: DriverPayload): Promise<Driver> {
  return gatewayData<Driver>(`/drivers/${id}`, { method: 'PUT', body: payload })
}

export function deleteDriver(id: string): Promise<null> {
  return gatewayData<null>(`/drivers/${id}`, { method: 'DELETE' })
}

/* ---------------- 推送 ---------------- */

export function listPushConnectors(
  query: Pick<ListQuery, 'page' | 'pageSize' | 'keyword'> = {},
): Promise<PageResp<PushConnector>> {
  return gatewayData<PageResp<PushConnector>>(
    `/push/connectors${buildQuery({
      page: query.page,
      pageSize: query.pageSize,
      keyword: query.keyword,
    })}`,
  )
}

export function createPushConnector(payload: PushConnectorPayload): Promise<PushConnector> {
  return gatewayData<PushConnector>('/push/connectors', { method: 'POST', body: payload })
}

export function updatePushConnector(
  id: string,
  payload: PushConnectorPayload,
): Promise<PushConnector> {
  return gatewayData<PushConnector>(`/push/connectors/${id}`, {
    method: 'PUT',
    body: payload,
  })
}

export function deletePushConnector(
  id: string,
  cascade = false,
): Promise<{ deleted: boolean }> {
  return gatewayData<{ deleted: boolean }>(
    `/push/connectors/${id}${buildQuery({ cascade })}`,
    { method: 'DELETE' },
  )
}

export function startPushConnector(id: string): Promise<{ status: string }> {
  return gatewayData<{ status: string }>(`/push/connectors/${id}/start`, {
    method: 'POST',
  })
}

export function stopPushConnector(id: string): Promise<{ status: string }> {
  return gatewayData<{ status: string }>(`/push/connectors/${id}/stop`, {
    method: 'POST',
  })
}

export function listPushGroups(
  query: Pick<ListQuery, 'page' | 'pageSize' | 'keyword'> = {},
): Promise<PageResp<PushGroup>> {
  return gatewayData<PageResp<PushGroup>>(
    `/push/groups${buildQuery({
      page: query.page,
      pageSize: query.pageSize,
      keyword: query.keyword,
    })}`,
  )
}

export function createPushGroup(payload: PushGroupPayload): Promise<PushGroup> {
  return gatewayData<PushGroup>('/push/groups', { method: 'POST', body: payload })
}

export function updatePushGroup(
  id: string,
  payload: PushGroupPayload,
): Promise<PushGroup> {
  return gatewayData<PushGroup>(`/push/groups/${id}`, { method: 'PUT', body: payload })
}

export function deletePushGroup(id: string): Promise<{ deleted: boolean }> {
  return gatewayData<{ deleted: boolean }>(`/push/groups/${id}`, { method: 'DELETE' })
}

/* ---------------- 网关资源 ---------------- */

export async function getGatewayMeta(): Promise<GatewayMeta> {
  const { data, etag } = await request<GatewayEnvelope<GatewayMeta>>('gateway', '/gatewayMeta')
  return { ...data.data, eTag: data.data.eTag || etag || '' }
}

export function getGatewayCpu(): Promise<GatewayCpuResponse> {
  return gatewayData<GatewayCpuResponse>('/gatewayCpu')
}

export function getGatewayMem(): Promise<GatewayMemResponse> {
  return gatewayData<GatewayMemResponse>('/gatewayMem')
}

export function getGatewayDisk(): Promise<GatewayDiskResponse> {
  return gatewayData<GatewayDiskResponse>('/gatewayDisk')
}
