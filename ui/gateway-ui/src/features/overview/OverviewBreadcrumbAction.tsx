import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { Button, Dialog, Input, useI18n } from '@harns/core'

export function OverviewBreadcrumbAction() {
  const { t } = useI18n()
  const queryClient = useQueryClient()
  const [refreshing, setRefreshing] = useState(false)
  const [autoRefreshOpen, setAutoRefreshOpen] = useState(false)
  const [autoRefreshInput, setAutoRefreshInput] = useState('10')
  const [autoRefreshSeconds, setAutoRefreshSeconds] = useState(10)
  const [autoRefreshActive, setAutoRefreshActive] = useState(false)
  const [autoRefreshError, setAutoRefreshError] = useState('')

  const refresh = async () => {
    setRefreshing(true)
    try {
      await queryClient.invalidateQueries({ queryKey: ['gateway'] })
    } finally {
      setRefreshing(false)
    }
  }

  useEffect(() => {
    if (!autoRefreshActive) return
    const timer = window.setInterval(() => {
      void queryClient.invalidateQueries({ queryKey: ['gateway'] })
    }, autoRefreshSeconds * 1000)
    return () => window.clearInterval(timer)
  }, [autoRefreshActive, autoRefreshSeconds, queryClient])

  const openAutoRefresh = () => {
    setAutoRefreshInput(String(autoRefreshSeconds))
    setAutoRefreshError('')
    setAutoRefreshOpen(true)
  }

  const closeAutoRefresh = () => {
    setAutoRefreshOpen(false)
    setAutoRefreshError('')
  }

  const stopAutoRefresh = () => {
    setAutoRefreshActive(false)
    closeAutoRefresh()
  }

  const confirmAutoRefresh = () => {
    const seconds = Number(autoRefreshInput.trim())
    if (!Number.isInteger(seconds) || seconds < 1 || seconds > 3600) {
      setAutoRefreshError(t('gateway.overview.autoRefreshInvalid'))
      return
    }
    setAutoRefreshSeconds(seconds)
    setAutoRefreshActive(true)
    setAutoRefreshOpen(false)
    setAutoRefreshError('')
    void refresh()
  }

  return (
    <>
      <Button
        variant={autoRefreshActive ? 'primary' : 'outline'}
        className="h-9"
        loading={refreshing}
        onClick={openAutoRefresh}
        icon={<RefreshCw className="size-3.5" />}
      >
        {autoRefreshActive
          ? t('gateway.overview.autoRefreshActive', { seconds: autoRefreshSeconds })
          : t('gateway.overview.autoRefresh')}
      </Button>

      <Dialog
        open={autoRefreshOpen}
        onClose={closeAutoRefresh}
        title={t('gateway.overview.autoRefreshTitle')}
        description={t('gateway.overview.autoRefreshDescription')}
        className="w-[320px] max-w-[calc(100vw-2rem)] rounded-lg"
        footer={
          <>
            {autoRefreshActive ? (
              <Button variant="outline" onClick={stopAutoRefresh}>
                {t('gateway.overview.autoRefreshStop')}
              </Button>
            ) : null}
            <Button variant="outline" onClick={closeAutoRefresh}>
              {t('common.action.cancel')}
            </Button>
            <Button variant="primary" onClick={confirmAutoRefresh}>
              {t('common.action.confirm')}
            </Button>
          </>
        }
      >
        <div className="space-y-2">
          <label
            htmlFor="overview-auto-refresh-seconds"
            className="block text-sm text-fg"
          >
            {t('gateway.overview.autoRefreshSeconds')}
          </label>
          <Input
            id="overview-auto-refresh-seconds"
            type="number"
            min={1}
            max={3600}
            step={1}
            autoFocus
            value={autoRefreshInput}
            aria-invalid={Boolean(autoRefreshError)}
            className={
              autoRefreshError
                ? 'border-danger hover:border-danger focus:border-danger focus:ring-danger/15'
                : undefined
            }
            onChange={(event) => {
              setAutoRefreshInput(event.target.value)
              setAutoRefreshError('')
            }}
          />
          {autoRefreshError ? (
            <p className="text-xs text-danger">{autoRefreshError}</p>
          ) : null}
        </div>
      </Dialog>
    </>
  )
}
