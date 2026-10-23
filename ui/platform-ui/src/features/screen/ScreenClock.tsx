import { useEffect, useState } from 'react'

const TIME_FORMAT = new Intl.DateTimeFormat('zh-CN', {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
})

const DATE_FORMAT = new Intl.DateTimeFormat('zh-CN', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  weekday: 'short',
})

/** 独立组件:每秒只重渲染时钟,避免整个大屏跟着刷新。 */
export function ScreenClock() {
  const [now, setNow] = useState(() => new Date())

  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 1000)
    return () => window.clearInterval(timer)
  }, [])

  return (
    <div className="text-right leading-tight">
      <p className="text-xs font-semibold tabular sm:text-sm">{TIME_FORMAT.format(now)}</p>
      <p className="mt-0.5 hidden text-[10px] whitespace-nowrap text-muted sm:block">{DATE_FORMAT.format(now)}</p>
    </div>
  )
}
