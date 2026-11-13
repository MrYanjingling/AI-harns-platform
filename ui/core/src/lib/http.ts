import { getLocale, translate } from '../i18n/translate'
import { getSessionToken, notifySessionExpired } from './session'

export type Service = 'gateway' | 'platform'

const BASES: Record<Service, string> = {
  gateway: import.meta.env.VITE_GATEWAY_BASE || '/api/v1',
  platform: import.meta.env.VITE_PLATFORM_BASE || '/api/v1',
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  /** multipart 上传;与 body 互斥,设置后不再写 Content-Type。 */
  formData?: FormData
  /** 乐观锁:网关设备写入时必须携带最近一次读取到的 eTag。 */
  ifMatch?: string
  /** 覆盖默认的 application/json,用于 merge-patch / json-patch。 */
  contentType?: string
  headers?: Record<string, string>
  /** 登录、验证码等公开接口不携带 Authorization。 */
  skipAuth?: boolean
  signal?: AbortSignal
}

export interface ApiResponse<T> {
  data: T
  etag?: string
}

export interface ApiFieldError {
  code?: number
  message?: string
  field?: string
}

export class ApiError extends Error {
  readonly status: number
  readonly code?: number
  readonly errors: ApiFieldError[]

  constructor(message: string, status: number, code?: number, errors: ApiFieldError[] = []) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.errors = errors
  }
}

interface ErrorPayload {
  code?: number
  message?: string
  errors?: ApiFieldError[]
  data?: unknown
}

function extractMessage(
  payload: unknown,
  status: number,
): { message: string; code?: number; errors: ApiFieldError[] } {
  if (payload && typeof payload === 'object') {
    const p = payload as ErrorPayload
    if (Array.isArray(p.errors) && p.errors.length > 0) {
      const messages = p.errors.map((e) => e?.message).filter(Boolean)
      if (messages.length > 0) {
        return { message: messages.join(';'), code: p.errors[0]?.code, errors: p.errors }
      }
    }
    if (typeof p.message === 'string' && p.message) {
      return { message: p.message, code: p.code, errors: p.errors ?? [] }
    }
    if (typeof p.data === 'string' && p.data) return { message: p.data, errors: [] }
    if (p.data && typeof p.data === 'object' && 'message' in p.data) {
      const m = (p.data as { message?: unknown }).message
      if (typeof m === 'string') return { message: m, errors: [] }
    }
  }
  return { message: statusMessage(status), errors: [] }
}

function statusMessage(status: number): string {
  switch (status) {
    case 400:
      return translate('common.error.badRequest')
    case 404:
      return translate('common.error.notFound')
    case 405:
      return translate('common.error.methodNotAllowed')
    case 409:
      return translate('common.error.conflict')
    case 412:
      return translate('common.error.stale')
    case 428:
      return translate('common.error.missingIfMatch')
    case 500:
      return translate('common.error.server')
    case 502:
    case 503:
    case 504:
      return translate('common.error.unreachable')
    default:
      return translate('common.error.http', { status })
  }
}

/** 统一请求入口:JSON 编解码、错误归一化、ETag 读取、空响应兼容。 */
export async function request<T>(
  service: Service,
  path: string,
  options: RequestOptions = {},
): Promise<ApiResponse<T>> {
  const {
    method = 'GET',
    body,
    formData,
    ifMatch,
    contentType,
    headers = {},
    skipAuth,
    signal,
  } = options
  const url = `${BASES[service]}${path.startsWith('/') ? path : `/${path}`}`

  const finalHeaders: Record<string, string> = {
    Accept: 'application/json',
    'Accept-Language': getLocale(),
    ...headers,
  }
  if (service === 'platform' && !skipAuth) {
    const token = getSessionToken()
    if (token) finalHeaders.Authorization = `Bearer ${token}`
  }
  if (formData) {
    delete finalHeaders['Content-Type']
  } else if (body !== undefined) {
    finalHeaders['Content-Type'] = contentType ?? 'application/json'
  }
  if (ifMatch) finalHeaders['If-Match'] = ifMatch

  let response: Response
  try {
    response = await fetch(url, {
      method,
      headers: finalHeaders,
      body: formData ?? (body === undefined ? undefined : JSON.stringify(body)),
      signal,
    })
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error
    throw new ApiError(translate('common.error.network'), 0)
  }

  const text = await response.text()
  let payload: unknown = undefined
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      payload = text
    }
  }

  if (!response.ok) {
    const { message, code, errors } = extractMessage(payload, response.status)
    // 平台服务统一使用 Bearer 会话;401 表示会话过期,清理后由上层跳转登录。
    if (service === 'platform' && response.status === 401 && !skipAuth) {
      notifySessionExpired()
    }
    throw new ApiError(message, response.status, code, errors)
  }

  return {
    data: payload as T,
    etag: response.headers.get('ETag') ?? undefined,
  }
}

/** 拼接查询串,自动忽略空值。 */
export function buildQuery(params: Record<string, string | number | boolean | undefined | null>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    search.set(key, String(value))
  }
  const query = search.toString()
  return query ? `?${query}` : ''
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError
}

export function errorMessage(error: unknown): string {
  if (isApiError(error)) return error.message
  if (error instanceof Error) return error.message
  return translate('common.error.unknown')
}
