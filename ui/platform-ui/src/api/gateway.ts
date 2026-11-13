import { request, type RequestOptions } from '@harns/core'
import type {
  GatewayCpuResponse,
  GatewayDiskResponse,
  GatewayMemResponse,
  GatewayMeta,
} from './types'

/** 网关与平台共用统一响应信封 {code,message,data}。 */
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

export async function getGatewayMeta(): Promise<GatewayMeta> {
  const { data, etag } = await gatewayData<GatewayMeta>('/gatewayMeta')
  // 后端同时写入 ETag 响应头与 body.eTag,优先使用 body。
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
