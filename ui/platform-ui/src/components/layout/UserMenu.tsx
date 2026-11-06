import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Languages, LogOut, Moon, Settings, Sun } from 'lucide-react'
import { LOCALES, useI18n } from '@/i18n'
import { cn } from '@/lib/cn'
import { useAuth } from '@/providers/AuthProvider'
import { useTheme } from '@/providers/ThemeProvider'

export function UserMenu() {
  const { t, locale, setLocale } = useI18n()
  const { user, signOut } = useAuth()
  const { theme, toggle } = useTheme()
  const [open, setOpen] = useState(false)
  const [signingOut, setSigningOut] = useState(false)
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

  const name = user?.nickName || user?.userName || t('common.user.notSignedIn')
  const initial = name.slice(0, 1).toUpperCase()
  const nextLocale = locale === 'zh-CN' ? 'en-US' : 'zh-CN'
  const nextLocaleLabel = LOCALES.find((item) => item.value === nextLocale)?.label ?? nextLocale

  const handleSignOut = async () => {
    setSigningOut(true)
    try {
      await signOut()
    } finally {
      setSigningOut(false)
      setOpen(false)
    }
  }

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={t('common.user.menu')}
        title={name}
        className="flex size-7 items-center justify-center rounded-full bg-primary text-xs font-medium text-primary-fg transition-opacity hover:opacity-90"
      >
        {initial}
      </button>

      {open ? (
        <div
          role="menu"
          className="absolute top-full right-0 z-50 mt-2 w-56 rounded border border-border bg-surface py-1 text-fg shadow-xl"
        >
          <div className="border-b border-border px-3 py-2">
            <p className="truncate text-sm font-medium">{name}</p>
            <p className="truncate text-xs text-faint">{user?.userNo || user?.userName || ''}</p>
          </div>
          <Link
            to="/settings"
            role="menuitem"
            onClick={() => setOpen(false)}
            className="flex items-center gap-2 px-3 py-2 text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
          >
            <Settings className="size-3.5" />
            {t('common.user.settings')}
          </Link>
          <button
            type="button"
            role="menuitem"
            onClick={toggle}
            className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
          >
            {theme === 'dark' ? <Sun className="size-3.5" /> : <Moon className="size-3.5" />}
            {theme === 'dark' ? t('common.theme.toggleToLight') : t('common.theme.toggleToDark')}
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={() => setLocale(nextLocale)}
            className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
          >
            <Languages className="size-3.5" />
            {nextLocaleLabel}
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={handleSignOut}
            disabled={signingOut}
            className={cn(
              'flex w-full items-center gap-2 border-t border-border px-3 py-2 text-left text-sm text-danger transition-colors',
              'hover:bg-danger/5 disabled:pointer-events-none disabled:opacity-60',
            )}
          >
            <LogOut className="size-3.5" />
            {signingOut ? t('common.user.signingOut') : t('common.user.signOut')}
          </button>
        </div>
      ) : null}
    </div>
  )
}
