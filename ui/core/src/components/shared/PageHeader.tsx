import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { ChevronLeft } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'

export function PageHeader({
  title,
  description,
  actions,
  backTo,
  backLabel,
  className,
}: {
  title: ReactNode
  description?: ReactNode
  actions?: ReactNode
  backTo?: string
  backLabel?: string
  className?: string
}) {
  const { t } = useI18n()
  return (
    <div className={cn('mb-3 flex flex-wrap items-end justify-between gap-2.5', className)}>
      <div className="min-w-0">
        {backTo ? (
          <Link
            to={backTo}
            className="mb-1 inline-flex items-center gap-1 text-[11px] text-muted transition-all hover:-translate-x-0.5 hover:text-fg"
          >
            <ChevronLeft className="size-3" />
            {backLabel ?? t('components.pageHeader.back')}
          </Link>
        ) : null}
        <div className="flex items-center gap-2">
          <span
            className="gradient-brand h-4 w-1 shrink-0 rounded-full"
            aria-hidden="true"
          />
          <h1 className="text-base font-semibold tracking-tight text-balance">{title}</h1>
        </div>
        {description ? (
          <p className="mt-0.5 max-w-2xl text-[11px] leading-relaxed text-muted">{description}</p>
        ) : null}
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-1.5">{actions}</div> : null}
    </div>
  )
}
