import { useEffect, useMemo, useState } from 'react'
import { useI18n } from '@/i18n'

function createTimeFormat(locale: string): Intl.DateTimeFormat {
  return new Intl.DateTimeFormat(locale, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}

function createDateFormat(locale: string): Intl.DateTimeFormat {
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    weekday: 'short',
  })
}

/** 独立组件:每秒只重渲染时钟,避免整个大屏跟着刷新。 */
export function ScreenClock() {
  const { locale } = useI18n()
  const [now, setNow] = useState(() => new Date())
  const timeFormat = useMemo(() => createTimeFormat(locale), [locale])
  const dateFormat = useMemo(() => createDateFormat(locale), [locale])

  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 1000)
    return () => window.clearInterval(timer)
  }, [])

  return (
    <div className="text-right leading-tight">
      <p className="text-xs font-semibold tabular sm:text-sm">{timeFormat.format(now)}</p>
      <p className="mt-0.5 hidden text-[10px] whitespace-nowrap text-muted sm:block">
        {dateFormat.format(now)}
      </p>
    </div>
  )
}
