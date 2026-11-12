import { PlaceholderPage } from '@/components/shared/PlaceholderPage'
import { useI18n } from '@/providers/I18nProvider'

export function UsersPage() {
  const { t } = useI18n()
  return <PlaceholderPage title={t('users.title')} description={t('users.description')} />
}
