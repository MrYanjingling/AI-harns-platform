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
    <div className={cn('mb-4 flex flex-wrap items-end justify-between gap-3', className)}>
      <div className="min-w-0">
        {backTo ? (
          <Link
            to={backTo}
            className="mb-1.5 inline-flex items-center gap-1 text-xs text-muted transition-colors hover:text-fg"
          >
            <ChevronLeft className="size-3.5" />
            {backLabel ?? t('components.pageHeader.back')}
          </Link>
        ) : null}
        <h1 className="text-lg font-semibold tracking-tight text-balance">{title}</h1>
        {description ? (
          <p className="mt-1 max-w-2xl text-xs leading-relaxed text-muted">{description}</p>
        ) : null}
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  )
}
