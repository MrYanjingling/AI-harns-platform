import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/providers/AuthProvider'

export function AuthGate({ children }: { children: ReactNode }) {
  const { signedIn } = useAuth()
  const location = useLocation()

  if (!signedIn) {
    return (
      <Navigate
        to="/login"
        replace
        state={{ from: `${location.pathname}${location.search}` }}
      />
    )
  }
  return <>{children}</>
}
