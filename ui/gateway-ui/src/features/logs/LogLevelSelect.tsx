import { SearchSelect, useI18n, type SearchSelectOption } from '@harns/core'
import type { StreamLogLevel } from './useSystemLogStream'

interface LogLevelSelectProps {
  value: StreamLogLevel
  onChange: (value: StreamLogLevel) => void
}

const options: SearchSelectOption[] = [
  { value: 'error', label: 'Error' },
  { value: 'warn', label: 'Warn' },
  { value: 'info', label: 'Info' },
]

export function LogLevelSelect({ value, onChange }: LogLevelSelectProps) {
  const { t } = useI18n()
  return (
    <SearchSelect
      value={value}
      options={options}
      onChange={(next) => onChange(next as StreamLogLevel)}
      ariaLabel={t('gateway.logs.filter.level')}
      emptyText={t('gateway.logs.filter.empty')}
      className="w-[140px]"
    />
  )
}
