import { request } from '@harns/core'
import { platformData } from './platform'
import type {
  ApiEnvelope,
  CaptchaResponse,
  CurrentUser,
  LoginRequest,
  LoginResponse,
} from './types'

/** 后端 RSA 公钥为 Base64(X.509 DER);获取失败时回退明文,保持与后端兼容。 */
async function getPublicKey(): Promise<string | null> {
  try {
    return await platformData<string>('/auth/publicKey', { skipAuth: true })
  } catch {
    return null
  }
}

function toBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return window.btoa(binary)
}

/** 使用 RSA-OAEP(SHA-256) 加密密码;浏览器不支持时回退明文。 */
export async function encryptPassword(password: string): Promise<string> {
  const publicKey = await getPublicKey()
  if (!publicKey || !window.crypto?.subtle) return password
  try {
    const der = Uint8Array.from(window.atob(publicKey), (char) => char.charCodeAt(0))
    const key = await window.crypto.subtle.importKey(
      'spki',
      der,
      { name: 'RSA-OAEP', hash: 'SHA-256' },
      false,
      ['encrypt'],
    )
    const cipher = await window.crypto.subtle.encrypt(
      { name: 'RSA-OAEP' },
      key,
      new TextEncoder().encode(password),
    )
    return toBase64(cipher)
  } catch {
    return password
  }
}

export async function login(req: LoginRequest): Promise<LoginResponse> {
  const password = await encryptPassword(req.password)
  const { data } = await request<ApiEnvelope<LoginResponse>>('platform', '/auth/login', {
    method: 'POST',
    body: { ...req, password },
    skipAuth: true,
  })
  return data.data
}

export async function logout(): Promise<void> {
  await platformData<null>('/auth/logout', { method: 'POST' })
}

export async function getCurrentUser(refresh = false): Promise<CurrentUser> {
  return platformData<CurrentUser>(`/auth/index${refresh ? '?refresh=true' : ''}`)
}

export async function getCaptcha(): Promise<CaptchaResponse> {
  return platformData<CaptchaResponse>('/auth/verificationCode', {
    method: 'POST',
    skipAuth: true,
  })
}
