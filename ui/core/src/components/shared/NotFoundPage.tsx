import { Link } from 'react-router-dom'
import { Compass } from 'lucide-react'
import { useI18n } from '../../i18n'
import { Card } from '../ui/Card'

export function NotFoundPage({ homeTo = '/' }: { homeTo?: string }) {
  const { t } = useI18n()
  return (
    <Card className="mx-auto mt-10 flex max-w-lg flex-col items-center px-6 py-16 text-center">
      <div className="gradient-brand-soft mb-5 flex size-12 items-center justify-center rounded-2xl border border-border text-primary">
        <Compass className="size-5" />
      </div>
      <p className="text-gradient text-4xl font-semibold tracking-tight tabular">404</p>
      <h1 className="mt-2 text-sm font-semibold">{t('common.notFound.title')}</h1>
      <p className="mt-1 text-xs text-muted">{t('common.notFound.description')}</p>
      <Link
        to={homeTo}
        className="gradient-brand mt-5 inline-flex h-9.5 items-center rounded-lg px-4 text-sm font-medium text-primary-fg shadow-sm transition-[filter] hover:brightness-110 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
      >
        {t('common.notFound.backHome')}
      </Link>
    </Card>
  )
}
