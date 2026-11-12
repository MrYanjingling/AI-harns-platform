import { PlaceholderPage } from '@/components/shared/PlaceholderPage'
import { useI18n } from '@/providers/I18nProvider'

export function SystemPage() {
  const { t } = useI18n()
  return <PlaceholderPage title={t('system.title')} description={t('system.description')} />
}
