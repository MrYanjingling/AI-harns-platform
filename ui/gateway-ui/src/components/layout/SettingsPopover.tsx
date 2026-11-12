import { useEffect, useRef, useState } from 'react'
import { Settings } from 'lucide-react'
import { AppearanceSettings } from '@/components/shared/AppearanceSettings'
import { useI18n } from '@/providers/I18nProvider'

export function SettingsPopover() {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onPointerDown = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false)
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        aria-label={t('nav.settings')}
        aria-haspopup="dialog"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
        className="inline-flex size-8 items-center justify-center rounded-lg text-muted transition-colors hover:bg-surface-2 hover:text-fg"
      >
        <Settings className="size-4.5" />
      </button>
      {open ? (
        <div
          role="dialog"
          aria-label={t('nav.settings')}
          className="absolute right-0 top-10 z-50 w-72 rounded-xl border border-border bg-surface p-4 shadow-xl"
        >
          <div className="mb-4 text-sm font-semibold">{t('nav.settings')}</div>
          <AppearanceSettings />
        </div>
      ) : null}
    </div>
  )
}
