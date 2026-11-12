import { Construction } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { PageHeader } from './PageHeader'
import { useI18n } from '@/providers/I18nProvider'

export function PlaceholderPage({ title, description }: { title: string; description: string }) {
  const { t } = useI18n()
  return (
    <>
      <PageHeader title={title} description={description} />
      <Card className="flex min-h-64 flex-col items-center justify-center gap-3 p-8 text-center">
        <span className="flex size-12 items-center justify-center rounded-full bg-primary-soft text-primary">
          <Construction className="size-6" />
        </span>
        <div>
          <p className="font-medium">{t('placeholder.ready')}</p>
          <p className="mt-1 text-sm text-muted">{t('placeholder.waiting')}</p>
        </div>
      </Card>
    </>
  )
}
