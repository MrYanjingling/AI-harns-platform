export class ApiError extends Error {
  readonly status: number
  readonly code?: number

  constructor(message: string, status: number, code?: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

interface ApiPayload {
  code?: number
  message?: string
  data?: unknown
  errors?: Array<{ code?: number; message?: string }>
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(`/api/v1${path}`, { ...init, headers })
  let payload: ApiPayload | undefined
  try {
    payload = (await response.json()) as ApiPayload
  } catch {
    /* non-json response */
  }

  if (!response.ok || (payload?.code !== undefined && payload.code !== 200)) {
    const message =
      payload?.errors?.map((item) => item.message).filter(Boolean).join('; ') ||
      payload?.message ||
      `请求失败 (HTTP ${response.status})`
    throw new ApiError(message, response.status, payload?.code)
  }
  return payload?.data as T
}
