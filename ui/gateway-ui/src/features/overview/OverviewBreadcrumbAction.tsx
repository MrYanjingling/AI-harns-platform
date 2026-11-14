import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { Button, useI18n } from '@harns/core'

export function OverviewBreadcrumbAction() {
  const { t } = useI18n()
  const queryClient = useQueryClient()
  const [refreshing, setRefreshing] = useState(false)

  const refresh = async () => {
    setRefreshing(true)
    try {
      await queryClient.invalidateQueries({ queryKey: ['gateway'] })
    } finally {
      setRefreshing(false)
    }
  }

  return (
    <Button
      variant="outline"
      size="sm"
      loading={refreshing}
      onClick={() => void refresh()}
      icon={<RefreshCw className="size-3.5" />}
    >
      {t('common.action.refresh')}
    </Button>
  )
}
