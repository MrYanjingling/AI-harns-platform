import { Languages } from 'lucide-react'
import { LOCALES, useI18n } from '../../i18n'

/** 顶栏语言切换:仅中英两种语言,直接互切。 */
export function LanguageSwitcher({ className }: { className?: string }) {
  const { locale, setLocale, t } = useI18n()
  const nextLocale = locale === 'zh-CN' ? 'en-US' : 'zh-CN'
  const nextLabel = LOCALES.find((item) => item.value === nextLocale)?.label ?? nextLocale

  return (
    <button
      type="button"
      onClick={() => setLocale(nextLocale)}
      aria-label={`${t('layout.languageSwitcher')}: ${nextLabel}`}
      title={`${t('layout.languageSwitcher')}: ${nextLabel}`}
      className={
        className ??
        'flex h-8 items-center gap-1 rounded px-2 text-xs text-header-muted transition-all hover:bg-header-hover hover:text-header-fg active:scale-95'
      }
    >
      <Languages className="size-4" />
      <span>{locale === 'zh-CN' ? 'EN' : '中'}</span>
    </button>
  )
}
