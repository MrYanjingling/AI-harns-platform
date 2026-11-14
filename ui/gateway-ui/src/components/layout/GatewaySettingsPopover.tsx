import { useEffect, useRef, useState } from 'react'
import { Moon, Settings, Sun } from 'lucide-react'
import { LOCALES, useI18n, useTheme } from '@harns/core'

export function GatewaySettingsPopover() {
  const { t, locale, setLocale } = useI18n()
  const { theme, setTheme } = useTheme()
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

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-label={t('gateway.settings.title')}
        title={t('gateway.settings.title')}
        className="flex size-8 items-center justify-center rounded text-header-muted transition-colors hover:bg-header-hover hover:text-header-fg"
      >
        <Settings className="size-4" />
      </button>

      {open ? (
        <div
          role="dialog"
          aria-label={t('gateway.settings.title')}
          className="animate-pop absolute top-full right-0 z-[60] mt-2 w-56 origin-top-right rounded border border-border bg-surface p-3 text-fg shadow-xl"
        >
          <p className="mb-2 text-xs font-medium text-faint">{t('gateway.settings.theme')}</p>
          <div className="grid grid-cols-2 gap-1">
            <label className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm text-fg hover:bg-surface-2">
              <input
                type="radio"
                name="gateway-theme"
                checked={theme === 'light'}
                onChange={() => setTheme('light')}
                className="size-4 accent-primary"
              />
              <Sun className="size-3.5 text-muted" />
              {t('common.theme.light')}
            </label>
            <label className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm text-fg hover:bg-surface-2">
              <input
                type="radio"
                name="gateway-theme"
                checked={theme === 'dark'}
                onChange={() => setTheme('dark')}
                className="size-4 accent-primary"
              />
              <Moon className="size-3.5 text-muted" />
              {t('common.theme.dark')}
            </label>
          </div>

          <div className="my-3 border-t border-border" />

          <p className="mb-2 text-xs font-medium text-faint">{t('gateway.settings.language')}</p>
          <div className="grid grid-cols-2 gap-1">
            {LOCALES.map((item) => (
              <label
                key={item.value}
                className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm text-fg hover:bg-surface-2"
              >
                <input
                  type="radio"
                  name="gateway-language"
                  checked={locale === item.value}
                  onChange={() => setLocale(item.value)}
                  className="size-4 accent-primary"
                />
                {item.label}
              </label>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  )
}
