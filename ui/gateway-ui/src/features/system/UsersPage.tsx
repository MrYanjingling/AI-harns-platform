import { Users } from 'lucide-react'
import { Card, EmptyState, PageHeader, useI18n } from '@harns/core'

export function UsersPage() {
  const { t } = useI18n()
  return (
    <>
      <PageHeader
        title={t('gateway.title.users')}
        description={t('gateway.users.description')}
      />
      <Card>
        <EmptyState icon={<Users className="size-5" />} title={t('gateway.users.emptyTitle')} description={t('gateway.users.emptyDescription')} />
      </Card>
    </>
  )
}
