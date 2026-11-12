import { Card } from '@/components/ui/Card'
import { PageHeader } from '@/components/shared/PageHeader'
import { AppearanceSettings } from '@/components/shared/AppearanceSettings'
import { useI18n } from '@/providers/I18nProvider'

export function SettingsPage() {
  const { t } = useI18n()

  return (
    <>
      <PageHeader title={t('settings.title')} description={t('settings.description')} />
      <Card className="p-5">
        <AppearanceSettings />
      </Card>
    </>
  )
}
