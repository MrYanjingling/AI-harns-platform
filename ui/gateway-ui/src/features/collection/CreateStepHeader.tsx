import { Check } from 'lucide-react'
import { cn, useI18n } from '@harns/core'

export function CreateStepHeader({ current }: { current: 1 | 2 }) {
  const { t } = useI18n()
  const steps = [t('gateway.form.step.collection'), t('gateway.form.step.variables')]

  return (
    <div className="flex flex-wrap items-center gap-2 rounded border border-border bg-surface-2 px-3 py-2 text-xs text-muted">
      {steps.map((label, index) => {
        const step = (index + 1) as 1 | 2
        const done = step < current
        const active = step === current
        return (
          <div key={label} className="flex items-center gap-2">
            {index > 0 ? <span className="h-px w-8 bg-border" /> : null}
            <span
              className={cn(
                'flex size-5 items-center justify-center rounded-full border text-[10px] font-medium',
                active
                  ? 'border-primary bg-primary text-primary-fg'
                  : done
                    ? 'border-primary/25 bg-primary-soft text-primary'
                    : 'border-border-strong bg-surface text-muted',
              )}
            >
              {done ? <Check className="size-3" /> : step}
            </span>
            <span className="text-[13px] font-semibold tracking-tight text-fg">{label}</span>
          </div>
        )
      })}
    </div>
  )
}
