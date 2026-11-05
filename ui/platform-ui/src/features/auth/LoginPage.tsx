import { useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { AlertCircle, LockKeyhole, LogIn, UserRound } from 'lucide-react'
import { z } from 'zod'
import { useLogin } from '@/api/hooks'
import { errorMessage } from '@/lib/http'
import { useAuth } from '@/providers/AuthProvider'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'

const loginSchema = z.object({
  username: z.string().trim().min(1, '请输入用户名').max(64, '用户名不能超过 64 个字符'),
  password: z.string().min(1, '请输入密码').max(512, '密码不能超过 512 个字符'),
})

type LoginForm = z.infer<typeof loginSchema>

const isMock = import.meta.env.VITE_USE_MOCK === 'true'

export function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const { token, signIn } = useAuth()
  const loginMutation = useLogin()
  const [error, setError] = useState<string | null>(null)

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
    <div className="flex min-h-dvh items-center justify-center bg-bg px-4 py-10">
      <div className="grid w-full max-w-4xl overflow-hidden rounded-3xl border border-border bg-surface shadow-xl lg:grid-cols-[1.05fr_1fr]">
        <div className="relative hidden flex-col justify-between bg-gradient-to-br from-emerald-600 via-emerald-700 to-slate-900 p-9 text-white lg:flex">
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
              <p className="text-sm font-semibold tracking-tight">AI-harns-platform</p>
              <p className="text-[11px] text-white/60">工业物联网平台</p>
            </div>
          </div>

          <div>
            <h1 className="text-2xl font-semibold leading-snug tracking-tight text-balance">
              物模型驱动的设备接入
              <br />
              从采集到平台的一体化管理
            </h1>
            <p className="mt-3 max-w-sm text-sm leading-relaxed text-white/70">
              管理物模型、设备凭证与在线状态，支持 MQTT / HTTP 上报与图片上传。
            </p>
          </div>

          <dl className="grid grid-cols-3 gap-3 text-center">
            {[
              ['物模型', '属性 / 特征'],
              ['设备', '凭证 / Topic'],
              ['认证', '会话 / 权限'],
            ].map(([title, desc]) => (
              <div key={title} className="rounded-xl bg-white/10 px-2 py-3 backdrop-blur">
                <dt className="text-xs font-medium">{title}</dt>
                <dd className="mt-0.5 text-[10px] text-white/60">{desc}</dd>
              </div>
            ))}
          </dl>
        </div>

        <div className="flex flex-col justify-center px-6 py-10 sm:px-10">
          <div className="mb-7">
            <h2 className="text-xl font-semibold tracking-tight">登录平台</h2>
            <p className="mt-1 text-sm text-muted">使用平台账号登录管理后台</p>
          </div>

          <form className="space-y-4" onSubmit={onSubmit} noValidate>
            <Field label="用户名" htmlFor="username" error={errors.username?.message} required>
              <div className="relative">
                <UserRound className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
                <Input
                  id="username"
                  autoComplete="username"
                  placeholder="请输入用户名"
                  className="pl-9"
                  {...register('username')}
                />
              </div>
            </Field>

            <Field label="密码" htmlFor="password" error={errors.password?.message} required>
              <div className="relative">
                <LockKeyhole className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
                <Input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  placeholder="请输入密码"
                  className="pl-9"
                  {...register('password')}
                />
              </div>
            </Field>

            {error ? (
              <div
                role="alert"
                className="flex items-start gap-2 rounded-lg border border-danger/25 bg-danger/5 px-3 py-2.5 text-xs text-danger"
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
              登录
            </Button>
          </form>

          {isMock ? (
            <p className="mt-5 rounded-lg border border-amber-600/25 bg-amber-500/10 px-3 py-2 text-[11px] leading-relaxed text-amber-700 dark:text-amber-300">
              Mock 模式：使用 <code className="font-mono">admin</code> /{' '}
              <code className="font-mono">harns@iot</code> 登录。
            </p>
          ) : (
            <p className="mt-5 text-[11px] leading-relaxed text-faint">
              首次部署使用初始化管理员账号登录，登录后请及时修改密码。
            </p>
          )}
        </div>
      </div>
    </div>
  )
}
