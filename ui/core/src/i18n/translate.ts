import { coreMessages, type Messages } from './messages'

export type Locale = 'zh-CN' | 'en-US'
export type InterpolationVars = Record<string, string | number>

/** 菜单、标题等跨应用引用的 key 由各应用自行约束,核心层按字符串处理。 */
export type MessageKey = string

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

// 非 React 模块（format、http）没有 Provider 上下文,统一读取这里注册的字典。
const registry: Record<Locale, Record<string, string>> = {
  'zh-CN': { ...coreMessages['zh-CN'] },
  'en-US': { ...coreMessages['en-US'] },
}

/** 合并多个应用的字典;应用入口渲染前调用,重复注册是幂等的。 */
export function registerMessages(messages: Messages): void {
  for (const locale of LOCALES) {
    Object.assign(registry[locale.value], messages[locale.value])
  }
}

export function mergeMessages(...parts: Messages[]): Messages {
  const merged: Messages = { 'zh-CN': {}, 'en-US': {} }
  for (const part of parts) {
    Object.assign(merged['zh-CN'], part['zh-CN'])
    Object.assign(merged['en-US'], part['en-US'])
  }
  return merged
}

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
function resolveKey(key: string, vars?: InterpolationVars): string {
  if (typeof vars?.count !== 'number') return key
  const pluralKey = `${key}.${vars.count === 1 ? 'one' : 'other'}`
  return pluralKey in registry['zh-CN'] ? pluralKey : key
}

export function translate(
  key: string,
  vars?: InterpolationVars,
  locale: Locale = currentLocale,
): string {
  const resolved = resolveKey(key, vars)
  const dictionary = registry[locale]
  const fallback = registry[DEFAULT_LOCALE]
  const template = dictionary[resolved] ?? fallback[resolved] ?? dictionary[key] ?? key
  if (!vars) return template
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    vars[name] === undefined ? match : String(vars[name]),
  )
}
