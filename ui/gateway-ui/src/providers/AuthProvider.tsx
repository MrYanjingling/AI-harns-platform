import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { useQueryClient } from '@tanstack/react-query'
import {
  SESSION_EXPIRED_EVENT,
  clearSessionToken,
  getSessionToken,
  setSessionToken,
  translate,
} from '@harns/core'
import { logout as logoutRequest } from '@/api/gateway'
import { clearAssistantSession } from '@/lib/assistant-session'
import type { GatewayUser, LoginResult } from '@/api/types'

const USER_KEY = 'harns-gateway-auth-user'

interface AuthContextValue {
  user: GatewayUser | null
  signedIn: boolean
  signIn: (result: LoginResult) => void
  signOut: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

function readUser(): GatewayUser | null {
  try {
    const raw = window.localStorage.getItem(USER_KEY)
    return raw ? (JSON.parse(raw) as GatewayUser) : null
  } catch {
    return null
  }
}

function writeUser(user: GatewayUser | null): void {
  try {
    if (user) window.localStorage.setItem(USER_KEY, JSON.stringify(user))
    else window.localStorage.removeItem(USER_KEY)
  } catch {
    // ignore storage failures
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const [token, setToken] = useState(() => getSessionToken())
  const [user, setUser] = useState<GatewayUser | null>(() => readUser())

  useEffect(() => {
    const handleExpired = () => {
      clearSessionToken()
      clearAssistantSession()
      writeUser(null)
      setToken(null)
      setUser(null)
      queryClient.clear()
    }
    window.addEventListener(SESSION_EXPIRED_EVENT, handleExpired)
    return () => window.removeEventListener(SESSION_EXPIRED_EVENT, handleExpired)
  }, [queryClient])

  const signIn = useCallback(
    (result: LoginResult) => {
      clearAssistantSession()
      queryClient.clear()
      setSessionToken(result.token)
      writeUser(result.user)
      setToken(result.token)
      setUser(result.user)
    },
    [queryClient],
  )

  const signOut = useCallback(async () => {
    try {
      if (token) await logoutRequest()
    } catch {
      // 会话可能已过期,本地状态仍需要清理。
    } finally {
      clearSessionToken()
      clearAssistantSession()
      writeUser(null)
      setToken(null)
      setUser(null)
      queryClient.clear()
    }
  }, [queryClient, token])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      signedIn: Boolean(token),
      signIn,
      signOut,
    }),
    [signIn, signOut, token, user],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) throw new Error(translate('gateway.auth.contextMissing'))
  return context
}
