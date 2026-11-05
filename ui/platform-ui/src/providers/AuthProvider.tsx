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
import { logout as logoutRequest } from '@/api/auth'
import { useCurrentUser } from '@/api/hooks'
import type { CurrentUser } from '@/api/types'
import { clearSessionToken, getSessionToken, SESSION_EXPIRED_EVENT, setSessionToken } from '@/lib/session'

interface AuthContextValue {
  token: string | null
  user: CurrentUser | undefined
  userLoading: boolean
  userError: unknown
  signIn: (token: string) => void
  signOut: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => getSessionToken())
  const queryClient = useQueryClient()
  const currentUser = useCurrentUser(Boolean(token))

  useEffect(() => {
    const handleExpired = () => {
      setToken(null)
      queryClient.clear()
    }
    window.addEventListener(SESSION_EXPIRED_EVENT, handleExpired)
    return () => window.removeEventListener(SESSION_EXPIRED_EVENT, handleExpired)
  }, [queryClient])

  const signIn = useCallback(
    (nextToken: string) => {
      setSessionToken(nextToken)
      setToken(nextToken)
      void queryClient.invalidateQueries({ queryKey: ['platform', 'current-user'] })
    },
    [queryClient],
  )

  const signOut = useCallback(async () => {
    try {
      await logoutRequest()
    } catch {
      // 会话可能已过期,本地清理仍然必须执行。
    } finally {
      clearSessionToken()
      setToken(null)
      queryClient.clear()
    }
  }, [queryClient])

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      user: currentUser.data,
      userLoading: currentUser.isLoading,
      userError: currentUser.error,
      signIn,
      signOut,
    }),
    [token, currentUser.data, currentUser.isLoading, currentUser.error, signIn, signOut],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth 必须在 AuthProvider 内使用')
  return ctx
}
