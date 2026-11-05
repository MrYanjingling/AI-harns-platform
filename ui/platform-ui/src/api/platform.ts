import { buildQuery, request, type RequestOptions } from '@/lib/http'
import type { ApiEnvelope, PlatformPage } from './types'

type QueryValue = string | number | boolean | undefined | null

/** 平台接口统一解包 {code,message,data},失败时由 request 抛 ApiError。 */
export async function platformData<T>(path: string, options?: RequestOptions): Promise<T> {
  const { data } = await request<ApiEnvelope<T>>('platform', path, options)
  return data.data
}

export function platformPage<T>(
  path: string,
  params: Record<string, QueryValue>,
): Promise<PlatformPage<T>> {
  return platformData<PlatformPage<T>>(`${path}${buildQuery(params)}`)
}

export function platformQuery(path: string, params: Record<string, QueryValue>): string {
  return `${path}${buildQuery(params)}`
}
