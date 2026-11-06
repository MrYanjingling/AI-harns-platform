import { messages, type MessageKey } from './messages'

export type Locale = 'zh-CN' | 'en-US'
export type InterpolationVars = Record<string, string | number>

export const LOCALES: ReadonlyArray<{ value: Locale; label: string }> = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English' },
]

export const DEFAULT_LOCALE: Locale = 'zh-CN'
export const LOCALE_STORAGE_KEY = 'harns-locale'

export function isLocale(value: unknown): value is Locale {
  return value === 'zh-CN' || value === 'en-US'
}

function readStoredLocale(): Locale {
  try {
    const saved = localStorage.getItem(LOCALE_STORAGE_KEY)
    return isLocale(saved) ? saved : DEFAULT_LOCALE
  } catch {
    return DEFAULT_LOCALE
  }
}

// 非 React 模块（format、http、schema）通过模块级当前语言取文案;
// Provider 切换语言时调用 setCurrentLocale 保持同步。
let currentLocale: Locale = readStoredLocale()

export function getLocale(): Locale {
  return currentLocale
}

export function setCurrentLocale(locale: Locale): void {
  currentLocale = locale
  try {
    localStorage.setItem(LOCALE_STORAGE_KEY, locale)
  } catch {
    // localStorage 不可用时仅保留内存态
  }
  if (typeof document !== 'undefined') document.documentElement.lang = locale
}

/** 带数量的文案优先取 `key.one` / `key.other`。 */
function resolveKey(key: MessageKey, vars?: InterpolationVars): string {
  if (typeof vars?.count !== 'number') return key
  const pluralKey = `${key}.${vars.count === 1 ? 'one' : 'other'}`
  return pluralKey in messages['zh-CN'] ? pluralKey : key
}

export function translate(
  key: MessageKey,
  vars?: InterpolationVars,
  locale: Locale = currentLocale,
): string {
  const dictionary = messages[locale] as Record<string, string>
  const fallback = messages[DEFAULT_LOCALE] as Record<string, string>
  const template = dictionary[resolveKey(key, vars)] ?? fallback[key] ?? key
  if (!vars) return template
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    vars[name] === undefined ? match : String(vars[name]),
  )
}
