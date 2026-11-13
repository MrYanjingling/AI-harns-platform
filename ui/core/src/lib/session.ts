const TOKEN_KEY = 'harns-session-token'

/** 平台会话失效事件:由 http 层派发,AuthProvider 负责清理并跳转登录页。 */
export const SESSION_EXPIRED_EVENT = 'harns:session-expired'

export function getSessionToken(): string | null {
  try {
    return window.localStorage.getItem(TOKEN_KEY)
  } catch {
    return null
  }
}

export function setSessionToken(token: string): void {
  try {
    window.localStorage.setItem(TOKEN_KEY, token)
  } catch {
    // 隐私模式下 localStorage 可能不可写,此时仅当前页面可用。
  }
}

export function clearSessionToken(): void {
  try {
    window.localStorage.removeItem(TOKEN_KEY)
  } catch {
    // ignore
  }
}

/** 后端返回 401 时调用:清理本地令牌并通知上层。 */
export function notifySessionExpired(): void {
  clearSessionToken()
  window.dispatchEvent(new Event(SESSION_EXPIRED_EVENT))
}
