import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import {
  DEFAULT_LOCALE,
  LOCALES,
  getLocale,
  registerMessages,
  setCurrentLocale,
  translate,
  type InterpolationVars,
  type Locale,
  type MessageKey,
} from './translate'
import type { Messages } from './messages'

export type Translate = (key: MessageKey, vars?: InterpolationVars) => string

interface I18nValue {
  locale: Locale
  setLocale: (locale: Locale) => void
  t: Translate
  locales: ReadonlyArray<{ value: Locale; label: string }>
}

const I18nContext = createContext<I18nValue | null>(null)

export function I18nProvider({
  messages,
  children,
}: {
  /** 应用字典;入口已注册时可省略。重复注册是幂等的。 */
  messages?: Messages
  children: ReactNode
}) {
  // 同步注册,避免首屏渲染出 key。
  if (messages) registerMessages(messages)
  const [locale, setLocaleState] = useState<Locale>(() => getLocale() ?? DEFAULT_LOCALE)

  useEffect(() => {
    setCurrentLocale(locale)
  }, [locale])

  const setLocale = useCallback((next: Locale) => setLocaleState(next), [])
  const t = useCallback<Translate>((key, vars) => translate(key, vars, locale), [locale])

  const value = useMemo<I18nValue>(
    () => ({ locale, setLocale, t, locales: LOCALES }),
    [locale, setLocale, t],
  )

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n(): I18nValue {
  const value = useContext(I18nContext)
  if (!value) throw new Error('useI18n must be used within I18nProvider')
  return value
}
