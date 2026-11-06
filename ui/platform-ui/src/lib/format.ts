import { translate } from '@/i18n/translate'

const pad = (n: number) => String(n).padStart(2, '0')

/** 把任意后端时间解析为 Date;兼容 "2026-07-01 10:00:00"。 */
export function parseDate(value: string | number | Date | null | undefined): Date | null {
  if (value === null || value === undefined || value === '') return null
  if (value instanceof Date) return Number.isNaN(value.getTime()) ? null : value
  if (typeof value === 'number') {
    const d = new Date(value)
    return Number.isNaN(d.getTime()) ? null : d
  }
  // Safari 对 "yyyy-MM-dd HH:mm:ss" 解析不稳定,统一补 T。
  const normalized = value.includes(' ') && !value.includes('T') ? value.replace(' ', 'T') : value
  const d = new Date(normalized)
  return Number.isNaN(d.getTime()) ? null : d
}

/** 2026-07-01 10:00:00 */
export function formatDateTime(value: string | number | Date | null | undefined): string {
  const d = parseDate(value)
  if (!d) return '—'
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(
    d.getHours(),
  )}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

/** 2026-07-01 10:00 */
export function formatDateTimeShort(value: string | number | Date | null | undefined): string {
  const d = parseDate(value)
  if (!d) return '—'
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(
    d.getHours(),
  )}:${pad(d.getMinutes())}`
}

export function formatRelative(value: string | number | Date | null | undefined): string {
  const d = parseDate(value)
  if (!d) return '—'
  const diff = Date.now() - d.getTime()
  if (diff < 0) return translate('common.time.justNow')
  const sec = Math.floor(diff / 1000)
  if (sec < 60) return translate('common.time.secondsAgo', { count: sec })
  const min = Math.floor(sec / 60)
  if (min < 60) return translate('common.time.minutesAgo', { count: min })
  const hour = Math.floor(min / 60)
  if (hour < 24) return translate('common.time.hoursAgo', { count: hour })
  const day = Math.floor(hour / 24)
  if (day < 30) return translate('common.time.daysAgo', { count: day })
  return formatDateTimeShort(d)
}

/** 后端要求的 "yyyy-MM-dd HH:mm:ss" */
export function formatApiTime(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(
    date.getHours(),
  )}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

/** 后端要求的毫秒格式 "yyyy-MM-dd HH:mm:ss.SSS" */
export function formatApiTimeMs(date: Date): string {
  return `${formatApiTime(date)}.${String(date.getMilliseconds()).padStart(3, '0')}`
}

/** Date -> <input type="datetime-local"> 的值 */
export function toDateTimeLocal(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(
    date.getHours(),
  )}:${pad(date.getMinutes())}`
}

/** <input type="datetime-local"> -> Date */
export function fromDateTimeLocal(value: string): Date | null {
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? null : d
}

/** "50%" -> 50 */
export function percentValue(value: string | number | null | undefined): number {
  if (value === null || value === undefined) return 0
  const n = typeof value === 'number' ? value : Number.parseFloat(value.replace('%', ''))
  if (!Number.isFinite(n)) return 0
  return Math.min(100, Math.max(0, n))
}

export function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(units.length - 1, Math.floor(Math.log(value) / Math.log(1024)))
  return `${(value / 1024 ** i).toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

export function maskSecret(value: string | null | undefined): string {
  if (!value) return translate('common.state.unset')
  if (value.length <= 8) return '••••••••'
  return `${value.slice(0, 4)}••••••••${value.slice(-4)}`
}
