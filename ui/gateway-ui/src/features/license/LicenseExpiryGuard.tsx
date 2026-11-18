import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { CircleAlert } from 'lucide-react'
import { Button, Dialog, formatDateTime, parseDate, useI18n } from '@harns/core'
import { useLicense } from '@/api/hooks'

const DAY_MS = 24 * 60 * 60 * 1000
const CHECK_INTERVAL_MS = 30_000

export function LicenseExpiryGuard({ children }: { children: ReactNode }) {
  const { t } = useI18n()
  const [polling, setPolling] = useState(false)
  const license = useLicense(polling ? CHECK_INTERVAL_MS : false)
  const [open, setOpen] = useState(false)
  const timerRef = useRef<number | undefined>(undefined)

  const deadline = useMemo(
    () => parseDate(license.data?.license?.softwareRunDeadline),
    [license.data?.license?.softwareRunDeadline],
  )
  const deadlineMs = deadline?.getTime() ?? null

  const clearTimer = useCallback(() => {
    if (timerRef.current !== undefined) {
      window.clearInterval(timerRef.current)
      timerRef.current = undefined
    }
  }, [])

  const startTimer = useCallback(() => {
    clearTimer()
    if (deadlineMs === null) return
    timerRef.current = window.setInterval(() => {
      if (Date.now() > deadlineMs) setOpen(true)
    }, CHECK_INTERVAL_MS)
  }, [clearTimer, deadlineMs])

  useEffect(() => {
    if (deadlineMs === null) {
      setPolling(false)
      clearTimer()
      setOpen(false)
      return
    }
    const remaining = deadlineMs - Date.now()
    setPolling(remaining <= DAY_MS)
    if (remaining > DAY_MS) {
      clearTimer()
      setOpen(false)
      return
    }
    setOpen(remaining <= 0)
    startTimer()
    return clearTimer
  }, [clearTimer, deadlineMs, startTimer])

  const close = () => {
    setOpen(false)
    if (deadlineMs !== null && deadlineMs - Date.now() <= DAY_MS) {
      startTimer()
    }
  }

  return (
    <>
      {children}
      <Dialog
        open={open}
        onClose={close}
        title={
          <span className="flex items-center gap-2 text-lg font-medium text-fg">
            <CircleAlert className="size-7 shrink-0 text-warning" />
            {t('gateway.license.expired.title')}
          </span>
        }
        size="sm"
        initialFocus="panel"
        className="rounded-lg border-warning/30"
        headerClassName="items-center px-7 pt-4 pb-0"
        bodyClassName="px-7 pt-2 pb-4"
        footerClassName="gap-3 px-7 pt-0 pb-5"
        footer={
          <Button
            variant="outline"
            size="sm"
            className="h-7 rounded-lg px-2.5 text-sm"
            onClick={close}
          >
            {t('common.action.close')}
          </Button>
        }
      >
        {deadline ? (
          <p className="text-[13px] leading-relaxed text-muted">
            {t('gateway.license.expired.deadline', {
              time: formatDateTime(deadline),
            })}
          </p>
        ) : null}
      </Dialog>
    </>
  )
}
