import { Gauge } from 'lucide-react'
import { Card, EmptyState, PageHeader, useI18n } from '@harns/core'

export function MonitoringPointsPage() {
  const { t } = useI18n()
  return (
    <>
      <PageHeader
        title={t('gateway.title.monitorPoints')}
        description={t('gateway.monitoring.description')}
      />
      <Card>
        <EmptyState icon={<Gauge className="size-5" />} title={t('gateway.monitoring.emptyTitle')} description={t('gateway.monitoring.emptyDescription')} />
      </Card>
    </>
  )
}
