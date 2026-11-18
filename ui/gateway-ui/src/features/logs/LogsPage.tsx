import { useEffect, useMemo, useRef, useState } from 'react'
import { Download, Trash2 } from 'lucide-react'
import { Button, cn, showErrorToast, useI18n } from '@harns/core'
import { gatewayAuthHeaders, gatewayBaseUrl, handleGatewayAuthFailure } from '@/api/gateway'
import { LogLevelSelect } from './LogLevelSelect'
import { useSystemLogStream, type StreamLogLevel } from './useSystemLogStream'

export function LogsPage() {
  const { t } = useI18n()
  const { lines, error, clear } = useSystemLogStream()
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
        { headers: gatewayAuthHeaders('text/plain'), cache: 'no-store' },
      )
      if (!response.ok) {
        await handleGatewayAuthFailure(response)
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
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="flex shrink-0 items-center gap-2">
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
        <div className="ml-auto flex items-center gap-2">
          <Button
            variant="ghost"
            icon={<Trash2 className="size-3.5" />}
            onClick={clear}
          >
            {t('gateway.logs.clear')}
          </Button>
          <Button
            variant="outline"
            icon={<Download className="size-3.5" />}
            loading={exporting}
            onClick={() => void exportLogs()}
          >
            {t('gateway.logs.export')}
          </Button>
        </div>
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
        className="min-h-0 w-full flex-1 overflow-auto rounded border-[1.5px] border-border-strong/75 bg-surface p-3 font-mono text-xs leading-5 shadow-[0_1px_3px_rgba(28,47,76,0.06)]"
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
