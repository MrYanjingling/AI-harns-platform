import type { Locale } from '@harns/core'

/** 驱动 Schema 的标签由后端同时提供中英文,按当前语言选择。 */
export function localized(zh: string | undefined, en: string | undefined, locale: Locale): string {
  return (locale === 'en-US' ? en : zh) || zh || en || ''
}
