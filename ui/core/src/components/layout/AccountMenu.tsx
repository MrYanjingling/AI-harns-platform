import { useEffect, useRef, useState } from 'react'
import { Languages, LogOut, Moon, Settings2, Sun } from 'lucide-react'
import { LOCALES, useI18n } from '../../i18n'
import { cn } from '../../lib/cn'
import { useTheme } from '../../providers/ThemeProvider'

export interface AccountMenuProps {
  name?: string
  subtitle?: string
  onSignOut?: () => void | Promise<void>
  signingOut?: boolean
}

/** 顶栏账号菜单:主题、语言与可选的退出登录。 */
export function AccountMenu({ name, subtitle, onSignOut, signingOut }: AccountMenuProps) {
  const { t, locale, setLocale } = useI18n()
  const { theme, toggle } = useTheme()
  const [open, setOpen] = useState(false)
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

  const displayName = name || t('common.user.notSignedIn')
  const initial = name ? name.slice(0, 1).toUpperCase() : ''
  const nextLocale = locale === 'zh-CN' ? 'en-US' : 'zh-CN'
  const nextLocaleLabel = LOCALES.find((item) => item.value === nextLocale)?.label ?? nextLocale

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={t('layout.shell.accountMenu')}
        title={displayName}
        className="flex size-7 items-center justify-center rounded-full bg-primary text-xs font-medium text-primary-fg transition-opacity hover:opacity-90"
      >
        {initial || <Settings2 className="size-3.5" />}
      </button>

      {open ? (
        <div
          role="menu"
          className="animate-pop absolute top-full right-0 z-[60] mt-2 w-56 origin-top-right rounded border border-border bg-surface py-1 text-fg shadow-xl"
        >
          <div className="border-b border-border px-3 py-2">
            <p className="truncate text-sm font-medium">{displayName}</p>
            <p className="truncate text-xs text-faint">{subtitle || t('layout.shell.preferences')}</p>
          </div>
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
          {onSignOut ? (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                void Promise.resolve(onSignOut()).finally(() => setOpen(false))
              }}
              disabled={signingOut}
              className={cn(
                'flex w-full items-center gap-2 border-t border-border px-3 py-2 text-left text-sm text-danger transition-colors',
                'hover:bg-danger/5 disabled:pointer-events-none disabled:opacity-60',
              )}
            >
              <LogOut className="size-3.5" />
              {signingOut ? t('common.user.signingOut') : t('common.user.signOut')}
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
