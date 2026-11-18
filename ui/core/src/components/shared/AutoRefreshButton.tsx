import { useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { cn } from '../../lib/cn'
import { useI18n } from '../../i18n'
import { useAutoRefresh } from '../../hooks/useAutoRefresh'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { Input } from '../ui/Field'

export function AutoRefreshButton({
  onRefresh,
  defaultSeconds = 10,
  className,
}: {
  onRefresh: () => void | Promise<void>
  defaultSeconds?: number
  className?: string
}) {
  const { t } = useI18n()
  const autoRefresh = useAutoRefresh(onRefresh, defaultSeconds)
  const [open, setOpen] = useState(false)
  const [input, setInput] = useState(String(autoRefresh.seconds))
  const [error, setError] = useState('')

  const openDialog = () => {
    setInput(String(autoRefresh.seconds))
    setError('')
    setOpen(true)
  }

  const closeDialog = () => {
    setOpen(false)
    setError('')
  }

  const stop = () => {
    autoRefresh.setActive(false)
    closeDialog()
  }

  const confirm = () => {
    const seconds = Number(input.trim())
    if (!Number.isInteger(seconds) || seconds < 1 || seconds > 3600) {
      setError(t('common.autoRefresh.invalid'))
      return
    }
    autoRefresh.setSeconds(seconds)
    autoRefresh.setActive(true)
    closeDialog()
    void autoRefresh.refresh()
  }

  return (
    <>
      <Button
        variant={autoRefresh.active ? 'primary' : 'outline'}
        className={cn('app-auto-refresh-button', className)}
        loading={autoRefresh.refreshing}
        onClick={openDialog}
        icon={<RefreshCw />}
      >
        {autoRefresh.active
          ? t('common.autoRefresh.active', { seconds: autoRefresh.seconds })
          : t('common.autoRefresh.label')}
      </Button>

      <Dialog
        open={open}
        onClose={closeDialog}
        title={t('common.autoRefresh.title')}
        className="w-[320px] max-w-[calc(100vw-2rem)] rounded-lg"
        footer={
          <>
            {autoRefresh.active ? (
              <Button variant="outline" onClick={stop}>
                {t('common.autoRefresh.stop')}
              </Button>
            ) : null}
            <Button variant="outline" onClick={closeDialog}>
              {t('common.action.cancel')}
            </Button>
            <Button variant="primary" onClick={confirm}>
              {t('common.action.confirm')}
            </Button>
          </>
        }
      >
        <div className="space-y-2">
          <label htmlFor="app-auto-refresh-seconds" className="block text-sm text-fg">
            {t('common.autoRefresh.seconds')}
          </label>
          <Input
            id="app-auto-refresh-seconds"
            type="number"
            min={1}
            max={3600}
            step={1}
            autoFocus
            value={input}
            aria-invalid={Boolean(error)}
            className={
              error
                ? 'border-danger hover:border-danger focus:border-danger focus:ring-danger/15'
                : undefined
            }
            onChange={(event) => {
              setInput(event.target.value)
              setError('')
            }}
          />
          <p className="app-dialog-error">{error}</p>
        </div>
      </Dialog>
    </>
  )
}
