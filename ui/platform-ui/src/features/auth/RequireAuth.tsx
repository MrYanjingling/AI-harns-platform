import { Navigate, useLocation } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useI18n } from '@harns/core'
import { useAuth } from '@/providers/AuthProvider'
import { ErrorState } from '@harns/core'
import { PageLoading } from '@harns/core'

/** 未登录跳转登录页;登录后回到原始地址。 */
export function RequireAuth({ children }: { children: ReactNode }) {
  const { t } = useI18n()
  const { token, userLoading, userError } = useAuth()
  const location = useLocation()

  if (!token) {
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  }

  if (userLoading) return <PageLoading label={t('common.loading.session')} />

  if (userError) {
    return (
      <div className="mx-auto max-w-xl px-4 py-16">
        <ErrorState error={userError} onRetry={() => window.location.reload()} />
      </div>
    )
  }

  return <>{children}</>
}
