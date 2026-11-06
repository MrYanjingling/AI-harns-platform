import { useCallback, useState } from 'react'
import { useI18n, type MessageKey } from '@/i18n'
import { Field, Input } from '@/components/ui/Field'
import { Button } from '@/components/ui/Button'
import { formatApiTime, fromDateTimeLocal, toDateTimeLocal } from '@/lib/format'

export interface ApiTimeRange {
  startTime: string
  endTime: string
}

/** 数据服务查询的公共时间范围状态。 */
export function useTimeRange(hours = 24) {
  const [start, setStart] = useState(() =>
    toDateTimeLocal(new Date(Date.now() - hours * 60 * 60 * 1000)),
  )
  const [end, setEnd] = useState(() => toDateTimeLocal(new Date()))

  const applyPreset = useCallback((presetHours: number) => {
    const now = new Date()
    setEnd(toDateTimeLocal(now))
    setStart(toDateTimeLocal(new Date(now.getTime() - presetHours * 60 * 60 * 1000)))
  }, [])

  const toApi = useCallback((): ApiTimeRange | null => {
    const startDate = fromDateTimeLocal(start)
    const endDate = fromDateTimeLocal(end)
    if (!startDate || !endDate || startDate.getTime() >= endDate.getTime()) return null
    return { startTime: formatApiTime(startDate), endTime: formatApiTime(endDate) }
  }, [start, end])

  return { start, end, setStart, setEnd, applyPreset, toApi, valid: Boolean(toApi()) }
}

const PRESETS: Array<{ hours: number; labelKey: MessageKey }> = [
  { hours: 1, labelKey: 'components.timeRange.hours1' },
  { hours: 24, labelKey: 'components.timeRange.hours24' },
  { hours: 24 * 7, labelKey: 'components.timeRange.days7' },
]

export function TimeRangeFields({
  range,
}: {
  range: ReturnType<typeof useTimeRange>
}) {
  const { t } = useI18n()
  return (
    <div className="flex flex-wrap items-end gap-3">
      <Field label={t('components.timeRange.start')} className="w-52">
        <Input
          type="datetime-local"
          value={range.start}
          onChange={(event) => range.setStart(event.target.value)}
        />
      </Field>
      <Field label={t('components.timeRange.end')} className="w-52">
        <Input
          type="datetime-local"
          value={range.end}
          onChange={(event) => range.setEnd(event.target.value)}
        />
      </Field>
      <div className="flex items-center gap-1 pb-0.5">
        {PRESETS.map((preset) => (
          <Button
            key={preset.hours}
            size="sm"
            variant="ghost"
            onClick={() => range.applyPreset(preset.hours)}
          >
            {t(preset.labelKey)}
          </Button>
        ))}
      </div>
      {!range.valid ? (
        <p className="pb-2 text-xs text-danger">{t('components.timeRange.invalid')}</p>
      ) : null}
    </div>
  )
}
