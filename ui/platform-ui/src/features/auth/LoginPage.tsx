import { useMemo, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { AlertCircle, LockKeyhole, LogIn, UserRound } from 'lucide-react'
import { z } from 'zod'
import { useLogin } from '@/api/hooks'
import { useI18n, type MessageKey } from '@harns/core'
import { errorMessage } from '@harns/core'
import { useAuth } from '@/providers/AuthProvider'
import { TechBackdrop } from '@harns/core'
import { Button } from '@harns/core'
import { Field, Input } from '@harns/core'

const isMock = import.meta.env.VITE_USE_MOCK === 'true'

const LOGIN_FEATURES: Array<[MessageKey, MessageKey]> = [
  ['auth.login.feature.thingModel.title', 'auth.login.feature.thingModel.description'],
  ['auth.login.feature.device.title', 'auth.login.feature.device.description'],
  ['auth.login.feature.auth.title', 'auth.login.feature.auth.description'],
]

export function LoginPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const { token, signIn } = useAuth()
  const loginMutation = useLogin()
  const [error, setError] = useState<string | null>(null)

  const loginSchema = useMemo(
    () =>
      z.object({
        username: z
          .string()
          .trim()
          .min(1, t('auth.validation.usernameRequired'))
          .max(64, t('auth.validation.usernameMax')),
        password: z
          .string()
          .min(1, t('auth.validation.passwordRequired'))
          .max(512, t('auth.validation.passwordMax')),
      }),
    [t],
  )

  type LoginForm = z.infer<typeof loginSchema>

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
    defaultValues: { username: '', password: '' },
  })

  if (token) return <Navigate to="/" replace />

  const onSubmit = handleSubmit(async (values) => {
    setError(null)
    try {
      const result = await loginMutation.mutateAsync(values)
      signIn(result.token)
      const from = (location.state as { from?: string } | null)?.from
      navigate(from && from.startsWith('/') ? from : '/', { replace: true })
    } catch (err) {
      setError(errorMessage(err))
    }
  })

  return (
    <div className="relative flex min-h-dvh items-center justify-center overflow-hidden bg-bg px-4 py-10">
      <TechBackdrop />
      <div className="relative z-10 grid w-full max-w-4xl overflow-hidden rounded-2xl border border-border bg-surface shadow-xl lg:grid-cols-[1.05fr_1fr]">
        <div className="relative hidden flex-col justify-between bg-gradient-to-br from-primary via-primary-hover to-header p-9 text-white lg:flex">
          <div className="flex items-center gap-2.5">
            <div className="flex size-9 items-center justify-center rounded-xl bg-white/15 backdrop-blur">
              <svg viewBox="0 0 24 24" className="size-5" fill="none" aria-hidden="true">
                <path
                  d="M7 15V8m0 3.5h5M12 15V8m0 3.5h5V15"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </div>
            <div>
              <p className="text-sm font-semibold tracking-wide">HARNS</p>
              <p className="text-[11px] text-white/60">{t('auth.login.brandSubtitle')}</p>
            </div>
          </div>

          <div>
            <h1 className="text-2xl font-semibold leading-snug tracking-tight text-balance">
              {t('auth.login.headlineLine1')}
              <br />
              {t('auth.login.headlineLine2')}
            </h1>
            <p className="mt-3 max-w-sm text-sm leading-relaxed text-white/70">
              {t('auth.login.description')}
            </p>
          </div>

          <dl className="grid grid-cols-3 gap-3 text-center">
            {LOGIN_FEATURES.map(([title, desc]) => (
              <div key={title} className="rounded-xl bg-white/10 px-2 py-3 backdrop-blur">
                <dt className="text-xs font-medium">{t(title)}</dt>
                <dd className="mt-0.5 text-[10px] text-white/60">{t(desc)}</dd>
              </div>
            ))}
          </dl>
        </div>

        <div className="flex flex-col justify-center px-6 py-10 sm:px-10">
          <div className="mb-7">
            <h2 className="text-xl font-semibold tracking-tight">{t('auth.login.title')}</h2>
            <p className="mt-1 text-sm text-muted">{t('auth.login.subtitle')}</p>
          </div>

          <form className="space-y-4" onSubmit={onSubmit} noValidate>
            <Field
              label={t('auth.login.username')}
              htmlFor="username"
              error={errors.username?.message}
              required
            >
              <div className="relative">
                <UserRound className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
                <Input
                  id="username"
                  autoComplete="username"
                  placeholder={t('auth.login.usernamePlaceholder')}
                  className="pl-9"
                  {...register('username')}
                />
              </div>
            </Field>

            <Field
              label={t('auth.login.password')}
              htmlFor="password"
              error={errors.password?.message}
              required
            >
              <div className="relative">
                <LockKeyhole className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
                <Input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  placeholder={t('auth.login.passwordPlaceholder')}
                  className="pl-9"
                  {...register('password')}
                />
              </div>
            </Field>

            {error ? (
              <div
                role="alert"
                className="flex items-start gap-2 rounded-xl border border-danger/25 bg-danger/5 px-3 py-2.5 text-xs text-danger"
              >
                <AlertCircle className="mt-0.5 size-4 shrink-0" />
                <span className="leading-relaxed">{error}</span>
              </div>
            ) : null}

            <Button
              type="submit"
              variant="primary"
              size="lg"
              className="w-full"
              loading={loginMutation.isPending}
              icon={<LogIn className="size-4" />}
            >
              {loginMutation.isPending ? t('auth.login.signingIn') : t('auth.login.submit')}
            </Button>
          </form>

          {isMock ? (
            <p className="mt-5 rounded-xl border border-amber-600/25 bg-amber-500/10 px-3 py-2 text-[11px] leading-relaxed text-amber-700 dark:text-amber-300">
              {t('auth.login.mockHint')}
            </p>
          ) : (
            <p className="mt-5 text-[11px] leading-relaxed text-faint">
              {t('auth.login.firstDeployHint')}
            </p>
          )}
        </div>
      </div>
    </div>
  )
}
