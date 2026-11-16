import { useEffect, useMemo, useRef, useState } from 'react'
import { Download } from 'lucide-react'
import { Button, cn, showErrorToast, useI18n } from '@harns/core'
import { gatewayBaseUrl } from '@/api/gateway'
import { LogLevelSelect } from './LogLevelSelect'
import { useSystemLogStream, type StreamLogLevel } from './useSystemLogStream'

export function LogsPage() {
  const { t } = useI18n()
  const { lines, error } = useSystemLogStream()
  const [level, setLevel] = useState<StreamLogLevel>('info')
  const [exporting, setExporting] = useState(false)
  const boxRef = useRef<HTMLDivElement>(null)
  const stickToBottom = useRef(true)

  const filtered = useMemo(() => lines.filter((line) => line.level === level), [lines, level])

  const exportLogs = async () => {
    setExporting(true)
    try {
      const response = await fetch(
        `${gatewayBaseUrl()}/system/logs/export?level=${encodeURIComponent(level)}`,
        { headers: { Accept: 'text/plain' }, cache: 'no-store' },
      )
      if (!response.ok) {
        let message = t('gateway.logs.exportFailed')
        try {
          const payload = (await response.json()) as { message?: unknown }
          if (typeof payload.message === 'string') message = payload.message
        } catch {
        }
        throw new Error(message)
      }
      const blob = await response.blob()
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = `harns-gateway-${level}-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.log`
      anchor.click()
      URL.revokeObjectURL(url)
    } catch (error) {
      showErrorToast(error)
    } finally {
      setExporting(false)
    }
  }

  useEffect(() => {
    if (!stickToBottom.current) return
    const box = boxRef.current
    if (box) box.scrollTop = box.scrollHeight
  }, [filtered])

  return (
    <div className="flex min-h-full flex-col gap-3">
      <div className="flex items-center gap-2">
        <label htmlFor="gateway-log-level" className="text-sm text-muted">
          {t('gateway.logs.filter.level')}
        </label>
        <LogLevelSelect
          value={level}
          onChange={(next) => {
            stickToBottom.current = true
            setLevel(next)
          }}
        />
        <Button
          variant="outline"
          className="ml-auto"
          icon={<Download className="size-3.5" />}
          loading={exporting}
          onClick={() => void exportLogs()}
        >
          {t('gateway.logs.export')}
        </Button>
      </div>

      <div
        ref={boxRef}
        role="log"
        aria-live="polite"
        onScroll={() => {
          const box = boxRef.current
          if (!box) return
          stickToBottom.current = box.scrollHeight - box.scrollTop - box.clientHeight < 24
        }}
        className="h-[400px] w-full overflow-auto rounded border-[1.5px] border-border-strong/75 bg-surface p-3 font-mono text-xs leading-5 shadow-[0_1px_3px_rgba(28,47,76,0.06)]"
      >
        {error ? <p className="text-danger">{t('gateway.logs.streamError')}</p> : null}
        {filtered.length === 0 && !error ? (
          <p className="text-faint">{t('gateway.logs.waiting')}</p>
        ) : null}
        {filtered.map((line) => (
          <p
            key={line.id}
            className={cn(
              'whitespace-pre-wrap break-all',
              line.level === 'error' && 'text-danger',
              line.level === 'warn' && 'text-warning',
              line.level === 'info' && 'text-fg',
            )}
          >
            {line.text}
          </p>
        ))}
      </div>
    </div>
  )
}
