import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'
import { Copyright } from './Copyright'
import { TechBackdrop } from '../layout/TechBackdrop'

export function NotFoundPage({
  homeTo = '/',
  fullScreen = false,
}: {
  homeTo?: string
  fullScreen?: boolean
}) {
  const { t } = useI18n()
  const heightClass = fullScreen ? 'min-h-dvh' : 'min-h-full'

  return (
    <div
      className={cn(
        'dark relative isolate overflow-hidden bg-[#030812] text-header-fg',
        heightClass,
      )}
    >
      <TechBackdrop />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_0%,rgba(37,99,235,0.22),transparent_42%),radial-gradient(circle_at_82%_78%,rgba(6,182,212,0.12),transparent_38%)]" />
      <div className="pointer-events-none absolute inset-0 opacity-40 [background-image:linear-gradient(rgba(96,165,250,0.08)_1px,transparent_1px),linear-gradient(90deg,rgba(96,165,250,0.08)_1px,transparent_1px)] [background-size:64px_64px] [mask-image:linear-gradient(to_bottom,black,transparent_90%)]" />
      <div className={cn('relative z-10 flex flex-col px-6 pt-6 sm:px-10', heightClass)}>
        <main className="flex flex-1 flex-col items-center justify-center py-16 text-center">
          <p className="font-mono text-[clamp(7rem,22vw,16rem)] leading-none font-semibold tracking-[-0.09em] text-white/[0.06] tabular">
            404
          </p>
          <h1 className="mt-3 text-2xl font-semibold tracking-tight text-header-fg">
            {t('common.notFound.title')}
          </h1>
          <Link
            to={homeTo}
            className="mt-7 inline-flex h-10 items-center gap-2 rounded-lg bg-primary px-5 text-sm font-medium text-primary-fg shadow-[0_0_36px_-12px_var(--primary)] transition-colors hover:bg-primary-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
          >
            <ArrowLeft className="size-4" />
            {t('common.notFound.backHome')}
          </Link>
        </main>

        <footer className="flex h-5 shrink-0 items-center justify-center font-mono text-[10px] tracking-[0.16em] text-header-muted/50">
          <Copyright />
        </footer>
      </div>
    </div>
  )
}
