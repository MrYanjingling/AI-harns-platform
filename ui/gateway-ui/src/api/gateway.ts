import { buildQuery, request, type RequestOptions } from '@harns/core'
import type {
  DeviceListResponse,
  DeviceOperator,
  DevicePayload,
  GatewayCpuResponse,
  GatewayDeviceDetail,
  GatewayDiskResponse,
  GatewayMemResponse,
  GatewayMeta,
} from './types'

/** 网关与平台共用统一响应信封。 */
interface GatewayEnvelope<T> {
  code: number
  message?: string
  data: T
}

async function gatewayData<T>(
  path: string,
  options?: RequestOptions,
): Promise<{ data: T; etag?: string }> {
  const { data, etag } = await request<GatewayEnvelope<T>>('gateway', path, options)
  return { data: data.data, etag }
}

/* ---------------- 设备(采集通道) ---------------- */

export async function listDevices(exploded = true): Promise<GatewayDeviceDetail[]> {
  const { data } = await gatewayData<DeviceListResponse>(
    `/devices${buildQuery({ exploded })}`,
  )
  return data.devices ?? []
}

export async function getDevice(id: string): Promise<GatewayDeviceDetail> {
  const { data } = await gatewayData<GatewayDeviceDetail>(`/devices/${id}${buildQuery({ exploded: true })}`)
  return data
}

export async function createDevice(payload: DevicePayload): Promise<GatewayDeviceDetail> {
  const { data } = await gatewayData<GatewayDeviceDetail>('/devices', {
    method: 'POST',
    body: payload,
  })
  return data
}

/** 全量更新;If-Match 使用详情里的 eTag(后端版本号)。 */
export async function updateDevice(
  id: string,
  eTag: string,
  payload: DevicePayload,
): Promise<GatewayDeviceDetail> {
  const { data } = await gatewayData<GatewayDeviceDetail>(`/devices/${id}`, {
    method: 'PUT',
    body: payload,
    ifMatch: eTag,
  })
  return data
}

export async function deleteDevice(id: string, eTag: string): Promise<void> {
  await gatewayData<unknown>(`/devices/${id}`, { method: 'DELETE', ifMatch: eTag })
}

export async function switchDeviceStatus(id: string, status: DeviceOperator): Promise<void> {
  await gatewayData<unknown>(`/devices/${id}/${status}`, { method: 'PUT' })
}

/** actions 是 [{ 变量名: 值 }] 的数组,后端逐个校验读写属性。 */
export async function controlDevice(
  id: string,
  actions: Array<Record<string, unknown>>,
): Promise<void> {
  await gatewayData<unknown>(`/devices/${id}/action`, { method: 'PUT', body: actions })
}

/* ---------------- 网关资源 ---------------- */

export async function getGatewayMeta(): Promise<GatewayMeta> {
  const { data, etag } = await gatewayData<GatewayMeta>('/gatewayMeta')
  return { ...data, eTag: data.eTag || etag || '' }
}

export async function getGatewayCpu(): Promise<GatewayCpuResponse> {
  return (await gatewayData<GatewayCpuResponse>('/gatewayCpu')).data
}

export async function getGatewayMem(): Promise<GatewayMemResponse> {
  return (await gatewayData<GatewayMemResponse>('/gatewayMem')).data
}

export async function getGatewayDisk(): Promise<GatewayDiskResponse> {
  return (await gatewayData<GatewayDiskResponse>('/gatewayDisk')).data
}
