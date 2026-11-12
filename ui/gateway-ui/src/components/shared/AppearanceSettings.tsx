import { Monitor, Moon, Sun } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useI18n } from '@/providers/I18nProvider'
import { useTheme, type Theme } from '@/providers/ThemeProvider'
import type { Language } from '@/i18n/messages'

const themes: Array<{
  value: Theme
  label: 'theme.light' | 'theme.dark' | 'theme.system'
  icon: typeof Sun
}> = [
  { value: 'light', label: 'theme.light', icon: Sun },
  { value: 'dark', label: 'theme.dark', icon: Moon },
  { value: 'system', label: 'theme.system', icon: Monitor },
]

export function AppearanceSettings() {
  const { theme, setTheme } = useTheme()
  const { language, setLanguage, t } = useI18n()

  return (
    <div className="space-y-5">
      <div>
        <label className="text-sm font-medium" htmlFor="gateway-language">
          {t('settings.language')}
        </label>
        <p className="mt-1 text-xs text-muted">{t('settings.languageHint')}</p>
        <select
          id="gateway-language"
          value={language}
          onChange={(event) => setLanguage(event.target.value as Language)}
          className="mt-3 h-9 w-full rounded-lg border border-border bg-surface px-3 text-sm text-fg outline-none focus:ring-2 focus:ring-primary/30"
        >
          <option value="zh-CN">{t('language.zh')}</option>
          <option value="en-US">{t('language.en')}</option>
        </select>
      </div>

      <div>
        <div className="text-sm font-medium">{t('settings.theme')}</div>
        <p className="mt-1 text-xs text-muted">{t('settings.themeHint')}</p>
        <div className="mt-3 grid grid-cols-3 overflow-hidden rounded-lg border border-border bg-surface-2">
          {themes.map((item) => {
            const Icon = item.icon
            const active = theme === item.value
            return (
              <button
                key={item.value}
                type="button"
                title={t(item.label)}
                aria-label={t(item.label)}
                aria-pressed={active}
                onClick={() => setTheme(item.value)}
                className={cn(
                  'flex h-9 items-center justify-center border-r border-border transition-colors last:border-r-0',
                  active
                    ? 'bg-primary text-primary-fg'
                    : 'text-muted hover:bg-surface-3 hover:text-fg',
                )}
              >
                <Icon className="size-4" />
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}
