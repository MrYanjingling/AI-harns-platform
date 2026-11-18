import { useEffect, useState, type FormEvent } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { Eye, EyeOff } from 'lucide-react'
import { Button, Copyright, Field, Input, TechBackdrop, showErrorToast, useI18n } from '@harns/core'
import { useLogin } from '@/api/hooks'
import { useAuth } from '@/providers/AuthProvider'

const REMEMBERED_USERNAME_KEY = 'harns-gateway-login-username'

interface LoginErrors {
  username?: string
  password?: string
}

export function LoginPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const { signedIn, signIn } = useAuth()
  const loginMutation = useLogin()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(false)
  const [errors, setErrors] = useState<LoginErrors>({})
  const [showPassword, setShowPassword] = useState(false)

  useEffect(() => {
    const saved = window.localStorage.getItem(REMEMBERED_USERNAME_KEY)
    if (saved) {
      setUsername(saved)
      setRemember(true)
    }
  }, [])

  const from = (location.state as { from?: string } | null)?.from
  const target = from && from.startsWith('/') && from !== '/login' ? from : '/overview'

  if (signedIn) return <Navigate to={target} replace />

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const nextErrors: LoginErrors = {}
    if (!username.trim()) nextErrors.username = t('gateway.auth.login.usernameRequired')
    if (!password) nextErrors.password = t('gateway.auth.login.passwordRequired')
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return

    try {
      const result = await loginMutation.mutateAsync({
        username: username.trim(),
        password,
      })
      if (remember) {
        window.localStorage.setItem(REMEMBERED_USERNAME_KEY, username.trim())
      } else {
        window.localStorage.removeItem(REMEMBERED_USERNAME_KEY)
      }
      signIn(result)
      navigate(target, { replace: true })
    } catch (error) {
      showErrorToast(error)
    }
  }

  return (
    <div className="dark relative flex min-h-dvh items-center justify-center overflow-hidden bg-[#030812] px-4 py-10 text-fg">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_0%,rgba(37,99,235,0.28),transparent_42%),radial-gradient(circle_at_85%_80%,rgba(6,182,212,0.14),transparent_38%)]" />
      <div className="pointer-events-none absolute inset-0 opacity-40 [background-image:linear-gradient(rgba(96,165,250,0.08)_1px,transparent_1px),linear-gradient(90deg,rgba(96,165,250,0.08)_1px,transparent_1px)] [background-size:64px_64px] [mask-image:linear-gradient(to_bottom,black,transparent_88%)]" />
      <TechBackdrop />
      <div className="pointer-events-none absolute inset-x-0 bottom-0 h-48 bg-gradient-to-t from-[#030812] to-transparent" />

      <div className="relative z-10 w-full max-w-[400px]">
        <div className="mb-6 flex flex-col items-center text-center">
          <h1 className="flex h-[30px] w-[220px] items-center justify-center text-[28px] leading-[1.2] font-semibold tracking-[0.02em] text-header-fg">Harns gateway</h1>
        </div>

        <div className="rounded-xl border border-white/10 bg-[#081120]/85 p-6 shadow-[0_24px_80px_-24px_rgba(0,0,0,0.9)] backdrop-blur-xl sm:p-7">
          <form className="space-y-4" onSubmit={submit} noValidate>
            <Field
              label={t('gateway.auth.login.username')}
              htmlFor="gateway-login-username"
              labelClassName="text-[13px] font-medium text-header-muted"
              error={errors.username}
            >
              <Input
                id="gateway-login-username"
                value={username}
                autoComplete="username"
                autoFocus
                placeholder={t('gateway.auth.login.usernamePlaceholder')}
                aria-invalid={Boolean(errors.username)}
                className="h-11 rounded-lg border-white/10 bg-white/[0.04] px-3 text-header-fg placeholder:text-header-muted/60 hover:border-primary/50 focus:border-primary focus:ring-primary/20"
                onChange={(event) => {
                  setUsername(event.target.value)
                  setErrors((current) => ({ ...current, username: undefined }))
                }}
              />
            </Field>

            <Field
              label={t('gateway.auth.login.password')}
              htmlFor="gateway-login-password"
              labelClassName="text-[13px] font-medium text-header-muted"
              error={errors.password}
            >
              <div className="relative">
                <Input
                  id="gateway-login-password"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  autoComplete="current-password"
                  placeholder={t('gateway.auth.login.passwordPlaceholder')}
                  aria-invalid={Boolean(errors.password)}
                  className="h-11 rounded-lg border-white/10 bg-white/[0.04] pr-11 pl-3 text-header-fg placeholder:text-header-muted/60 hover:border-primary/50 focus:border-primary focus:ring-primary/20"
                  onChange={(event) => {
                    setPassword(event.target.value)
                    setErrors((current) => ({ ...current, password: undefined }))
                  }}
                />
                <button
                  type="button"
                  aria-label={t(
                    showPassword
                      ? 'gateway.auth.login.hidePassword'
                      : 'gateway.auth.login.showPassword',
                  )}
                  title={t(
                    showPassword
                      ? 'gateway.auth.login.hidePassword'
                      : 'gateway.auth.login.showPassword',
                  )}
                  onClick={() => setShowPassword((current) => !current)}
                  className="absolute top-1/2 right-2.5 flex size-7 -translate-y-1/2 items-center justify-center rounded text-header-muted/40 transition-colors hover:bg-white/5 hover:text-header-muted/80"
                >
                  {showPassword ? <Eye className="size-4" /> : <EyeOff className="size-4" />}
                </button>
              </div>
            </Field>

            <label className="flex w-fit cursor-pointer items-center gap-2 text-[13px] text-header-muted">
              <input
                type="checkbox"
                checked={remember}
                className="size-4 rounded border-white/20 bg-white/5 accent-primary"
                onChange={(event) => setRemember(event.target.checked)}
              />
              {t('gateway.auth.login.remember')}
            </label>

            <Button
              type="submit"
              variant="primary"
              loading={loginMutation.isPending}
              className="mt-2 h-11 w-full rounded-lg text-sm"
            >
              {t('gateway.auth.login.submit')}
            </Button>
          </form>
        </div>

        <p className="mt-5 text-center">
          <Copyright className="text-[10px] text-faint" />
        </p>
      </div>
    </div>
  )
}
