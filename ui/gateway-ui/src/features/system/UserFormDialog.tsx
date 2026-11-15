import { useEffect, useState, type FormEvent } from 'react'
import { CircleHelp } from 'lucide-react'
import { Button, Dialog, Field, Input, Label, Textarea, showErrorToast, useI18n } from '@harns/core'
import { toast } from 'sonner'
import { useCreateUser, useUpdateUser } from '@/api/hooks'
import type { GatewayUser } from '@/api/types'

interface UserFormDialogProps {
  open: boolean
  user?: GatewayUser
  onClose: () => void
}

export function UserFormDialog({ open, user, onClose }: UserFormDialogProps) {
  const { t } = useI18n()
  const createUser = useCreateUser()
  const updateUser = useUpdateUser()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [remark, setRemark] = useState('')
  const [errors, setErrors] = useState<{ username?: string; password?: string; remark?: string }>({})

  useEffect(() => {
    if (!open) return
    setUsername(user?.username ?? '')
    setPassword('')
    setRemark(user?.remark ?? '')
    setErrors({})
  }, [open, user])

  const submitting = createUser.isPending || updateUser.isPending

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const nextErrors: typeof errors = {}
    const nextUsername = username.trim()
    const nextRemark = remark.trim()
    if (!nextUsername || nextUsername.length > 64) {
      nextErrors.username = t('gateway.users.form.usernameInvalid')
    }
    if (!user && password && (password.length < 6 || password.length > 64)) {
      nextErrors.password = t('gateway.users.form.passwordInvalid')
    }
    if (nextRemark.length > 255) {
      nextErrors.remark = t('gateway.users.form.remarkInvalid')
    }
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return

    const onSuccess = (saved: GatewayUser) => {
      toast.success(
        t(user ? 'gateway.users.updatedToast' : 'gateway.users.createdToast', {
          name: saved.username,
        }),
      )
      onClose()
    }
    if (user) {
      updateUser.mutate(
        { id: user.id, payload: { username: nextUsername, remark: nextRemark } },
        { onSuccess, onError: showErrorToast },
      )
      return
    }
    createUser.mutate(
      { username: nextUsername, password: password || undefined, remark: nextRemark },
      { onSuccess, onError: showErrorToast },
    )
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t(user ? 'gateway.users.editTitle' : 'gateway.users.createTitle')}
      size="sm"
      className="rounded-lg"
      footer={
        <>
          <Button variant="outline" onClick={onClose}>
            {t('common.action.cancel')}
          </Button>
          <Button type="submit" form="gateway-user-form" variant="primary" loading={submitting}>
            {t('common.action.save')}
          </Button>
        </>
      }
    >
      <form id="gateway-user-form" className="space-y-3" onSubmit={submit}>
        <Field label={t('gateway.users.form.username')} htmlFor="gateway-user-name" error={errors.username} required>
          <Input
            id="gateway-user-name"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            disabled={Boolean(user?.isAdmin)}
            maxLength={64}
          />
        </Field>
        {!user ? (
          <div className="space-y-1.5">
            <div className="flex items-center gap-1.5">
              <Label htmlFor="gateway-user-password">{t('gateway.users.form.password')}</Label>
              <span className="group relative inline-flex" title={t('gateway.users.form.passwordHint')}>
                <CircleHelp
                  className="size-3.5 text-faint"
                  tabIndex={0}
                  aria-label={t('gateway.users.form.passwordHint')}
                />
                <span
                  role="tooltip"
                  className="pointer-events-none invisible absolute top-full left-0 z-20 mt-1 w-56 rounded border border-border bg-surface px-2 py-1.5 text-xs leading-relaxed text-fg shadow-lg group-hover:visible group-focus-within:visible"
                >
                  {t('gateway.users.form.passwordHint')}
                </span>
              </span>
            </div>
            <Input
              id="gateway-user-password"
              type="password"
              autoComplete="new-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              maxLength={64}
            />
            {errors.password ? <p className="text-xs text-danger">{errors.password}</p> : null}
          </div>
        ) : null}
        <Field label={t('gateway.users.form.remark')} htmlFor="gateway-user-remark" error={errors.remark}>
          <Textarea
            id="gateway-user-remark"
            value={remark}
            onChange={(event) => setRemark(event.target.value)}
            maxLength={255}
            rows={3}
            className="min-h-20 resize-none"
          />
        </Field>
      </form>
    </Dialog>
  )
}
