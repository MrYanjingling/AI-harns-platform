import type { Locale } from '@harns/core'

const uptimePattern = /^(\d+)d(\d+)h(\d+)m(\d+)s$/

export function formatUptime(value: string | undefined, locale: Locale): string {
  if (!value) return '—'
  const match = value.match(uptimePattern)
  if (!match) return value
  const [, days, hours, minutes, seconds] = match
  if (locale === 'en-US') return `${days}d${hours}h${minutes}m${seconds}s`
  return `${days}天${hours}时${minutes}分${seconds}秒`
}
