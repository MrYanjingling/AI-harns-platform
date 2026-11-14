import { Info } from 'lucide-react'
import { Card, EmptyState, PageHeader, useI18n } from '@harns/core'

export function AboutPage() {
  const { t } = useI18n()
  return (
    <>
      <PageHeader
        title={t('gateway.title.about')}
        description={t('gateway.about.description')}
      />
      <Card>
        <EmptyState icon={<Info className="size-5" />} title={t('gateway.about.emptyTitle')} description={t('gateway.about.emptyDescription')} />
      </Card>
    </>
  )
}
