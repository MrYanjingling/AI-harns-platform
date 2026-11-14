import { useEffect, useRef, useState, type FormEvent } from 'react'
import { KeyRound, LogOut, UserRound } from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  clearSessionToken,
  Dialog,
  Field,
  Input,
  useI18n,
} from '@harns/core'

interface PasswordForm {
  oldPassword: string
  newPassword: string
  confirmPassword: string
}

interface PasswordErrors {
  oldPassword?: string
  newPassword?: string
  confirmPassword?: string
}

const emptyForm: PasswordForm = {
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
}

export function GatewayAccountMenu() {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const [passwordOpen, setPasswordOpen] = useState(false)
  const [form, setForm] = useState<PasswordForm>(emptyForm)
  const [errors, setErrors] = useState<PasswordErrors>({})
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const handlePointerDown = (event: PointerEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false)
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('pointerdown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [open])

  useEffect(() => {
    if (!passwordOpen) {
      setForm(emptyForm)
      setErrors({})
    }
  }, [passwordOpen])

  const submitPassword = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const nextErrors: PasswordErrors = {}
    if (!form.oldPassword) nextErrors.oldPassword = t('gateway.account.passwordRequired')
    if (!form.newPassword) {
      nextErrors.newPassword = t('gateway.account.passwordRequired')
    } else if (form.newPassword.length < 6 || form.newPassword.length > 64) {
      nextErrors.newPassword = t('gateway.account.passwordInvalid')
    }
    if (!form.confirmPassword) {
      nextErrors.confirmPassword = t('gateway.account.passwordRequired')
    } else if (form.confirmPassword !== form.newPassword) {
      nextErrors.confirmPassword = t('gateway.account.passwordMismatch')
    }
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return

    setPasswordOpen(false)
    toast.info(t('gateway.account.notAvailable'))
  }

  const logout = () => {
    clearSessionToken()
    setOpen(false)
    toast.success(t('gateway.account.loggedOut'))
  }

  return (
    <>
      <div ref={containerRef} className="relative">
        <button
          type="button"
          onClick={() => setOpen((current) => !current)}
          aria-haspopup="menu"
          aria-expanded={open}
          aria-label={t('gateway.account.title')}
          title={t('common.user.notSignedIn')}
          className="flex size-7 items-center justify-center rounded-full bg-primary text-primary-fg transition-opacity hover:opacity-90"
        >
          <UserRound className="size-3.5" />
        </button>

        {open ? (
          <div
            role="menu"
            className="animate-pop absolute top-full right-0 z-[60] mt-2 w-[100px] origin-top-right rounded border border-border bg-surface py-1 text-fg shadow-xl"
          >
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setOpen(false)
                setPasswordOpen(true)
              }}
              className="flex w-full items-center gap-1.5 px-2 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
            >
              <KeyRound className="size-3.5" />
              {t('gateway.account.changePassword')}
            </button>
            <button
              type="button"
              role="menuitem"
              onClick={logout}
              className="flex w-full items-center gap-1.5 border-t border-border px-2 py-2 text-left text-sm text-danger transition-colors hover:bg-danger/5"
            >
              <LogOut className="size-3.5" />
              {t('gateway.account.logout')}
            </button>
          </div>
        ) : null}
      </div>

      <Dialog
        open={passwordOpen}
        onClose={() => setPasswordOpen(false)}
        title={t('gateway.account.changePassword')}
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={() => setPasswordOpen(false)}>
              {t('common.action.cancel')}
            </Button>
            <Button type="submit" form="gateway-change-password" variant="primary">
              {t('common.action.confirm')}
            </Button>
          </>
        }
      >
        <form id="gateway-change-password" className="space-y-3" onSubmit={submitPassword}>
          <Field label={t('gateway.account.oldPassword')} htmlFor="gateway-old-password" error={errors.oldPassword} required>
            <Input
              id="gateway-old-password"
              type="password"
              autoComplete="current-password"
              value={form.oldPassword}
              onChange={(event) => setForm({ ...form, oldPassword: event.target.value })}
            />
          </Field>
          <Field label={t('gateway.account.newPassword')} htmlFor="gateway-new-password" error={errors.newPassword} required>
            <Input
              id="gateway-new-password"
              type="password"
              autoComplete="new-password"
              value={form.newPassword}
              onChange={(event) => setForm({ ...form, newPassword: event.target.value })}
            />
          </Field>
          <Field label={t('gateway.account.confirmPassword')} htmlFor="gateway-confirm-password" error={errors.confirmPassword} required>
            <Input
              id="gateway-confirm-password"
              type="password"
              autoComplete="new-password"
              value={form.confirmPassword}
              onChange={(event) => setForm({ ...form, confirmPassword: event.target.value })}
            />
          </Field>
        </form>
      </Dialog>
    </>
  )
}
