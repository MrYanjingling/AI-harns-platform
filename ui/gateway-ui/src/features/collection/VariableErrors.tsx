import { useI18n } from '@harns/core'

export function VariableErrors({ errors }: { errors: Record<string, string> }) {
  const { t } = useI18n()
  const entries = Object.entries(errors).filter(
    ([key]) => key === 'variables' || key.startsWith('variables.'),
  )
  if (entries.length === 0) return null

  return (
    <div className="space-y-1 rounded border border-danger/20 bg-danger/5 px-3 py-2 text-[11px] leading-relaxed text-danger">
      {entries.map(([key, message]) => {
        const match = /^variables\.(\d+)/.exec(key)
        return (
          <p key={key}>
            {match
              ? t('gateway.variable.rowError', {
                  row: Number(match[1]) + 1,
                  message,
                })
              : message}
          </p>
        )
      })}
    </div>
  )
}
