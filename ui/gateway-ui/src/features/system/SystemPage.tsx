import { useEffect, useRef, useState } from 'react'
import { Pause, Play, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { PageHeader } from '@/components/shared/PageHeader'
import { cn } from '@/lib/cn'
import { useI18n } from '@/providers/I18nProvider'

const MAX_LINES = 2000

function logClass(line: string) {
  if (line.includes('level=ERROR') || line.includes('"level":"ERROR"')) return 'text-danger'
  if (line.includes('level=WARN') || line.includes('"level":"WARN"')) return 'text-warning'
  return 'text-fg'
}

export function SystemPage() {
  const { t } = useI18n()
  const [lines, setLines] = useState<string[]>([])
  const [paused, setPaused] = useState(false)
  const [autoScroll, setAutoScroll] = useState(true)
  const [error, setError] = useState('')
  const [connected, setConnected] = useState(false)
  const pausedRef = useRef(false)
  const pendingRef = useRef<string[]>([])
  const viewportRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    pausedRef.current = paused
    if (!paused && pendingRef.current.length > 0) {
      const pending = pendingRef.current
      pendingRef.current = []
      setLines((current) => [...current, ...pending].slice(-MAX_LINES))
    }
  }, [paused])

  useEffect(() => {
    if (autoScroll && !paused) {
      viewportRef.current?.scrollTo({ top: viewportRef.current.scrollHeight })
    }
  }, [lines, autoScroll, paused])

  useEffect(() => {
    const controller = new AbortController()
    let active = true

    const appendLine = (line: string) => {
      if (!active) return
      if (pausedRef.current) {
        pendingRef.current.push(line)
        if (pendingRef.current.length > MAX_LINES) {
          pendingRef.current = pendingRef.current.slice(-MAX_LINES)
        }
        return
      }
      setLines((current) => [...current, line].slice(-MAX_LINES))
    }

    const consumeEvent = (block: string) => {
      let event = 'message'
      const data: string[] = []
      for (const tuple of block.split('\n')) {
        if (tuple.startsWith('event:')) event = tuple.slice(6).trim()
        if (tuple.startsWith('data:')) data.push(tuple.slice(5).trimStart())
      }
      if (event !== 'log' || data.length === 0) return
      try {
        const payload = JSON.parse(data.join('\n')) as { line?: string }
        if (payload.line) appendLine(payload.line)
      } catch {
        /* ignore malformed event */
      }
    }

    const run = async () => {
      try {
        const response = await fetch('/api/v1/system/logs/stream', {
          headers: { Accept: 'text/event-stream' },
          signal: controller.signal,
        })
        if (!response.ok) {
          const payload = (await response.json().catch(() => null)) as { message?: string } | null
          if (active) setError(payload?.message || t('system.connectionError'))
          return
        }
        if (!response.body) {
          if (active) setError(t('system.connectionError'))
          return
        }
        if (active) setConnected(true)
        const reader = response.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''
        while (active) {
          const { done, value } = await reader.read()
          if (done) break
          buffer += decoder.decode(value, { stream: true })
          let index = buffer.indexOf('\n\n')
          while (index >= 0) {
            consumeEvent(buffer.slice(0, index))
            buffer = buffer.slice(index + 2)
            index = buffer.indexOf('\n\n')
          }
        }
      } catch (err) {
        if (active && !(err instanceof DOMException && err.name === 'AbortError')) {
          setError(t('system.connectionError'))
        }
      } finally {
        if (active) setConnected(false)
      }
    }

    void run()
    return () => {
      active = false
      controller.abort()
    }
  }, [t])

  return (
    <>
      <PageHeader title={t('system.title')} description={t('system.description')} />
      <Card className="overflow-hidden">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border px-4 py-3">
          <div>
            <div className="flex items-center gap-2 text-sm font-medium">
              <span
                className={cn('size-1.5 rounded-full', connected ? 'bg-primary' : 'bg-faint')}
              />
              {t('system.logs')}
            </div>
            <p className="mt-1 text-xs text-muted">{t('system.logsHint')}</p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <label className="flex items-center gap-1.5 text-xs text-muted">
              <input
                type="checkbox"
                checked={autoScroll}
                onChange={(event) => setAutoScroll(event.target.checked)}
                className="size-4 accent-[var(--primary)]"
              />
              {t('system.autoScroll')}
            </label>
            <Button size="sm" onClick={() => setPaused((current) => !current)}>
              {paused ? <Play className="size-4" /> : <Pause className="size-4" />}
              {paused ? t('system.resume') : t('system.pause')}
            </Button>
            <Button size="sm" onClick={() => { setLines([]); pendingRef.current = [] }}>
              <Trash2 className="size-4" />
              {t('system.clear')}
            </Button>
          </div>
        </div>
        <div
          ref={viewportRef}
          className="h-[calc(100dvh-16rem)] min-h-80 overflow-auto bg-surface-2 px-4 py-3 font-mono text-xs leading-6"
        >
          {error ? (
            <div className="rounded-lg border border-danger/30 bg-danger/5 px-3 py-2 text-danger">{error}</div>
          ) : lines.length === 0 ? (
            <div className="text-muted">{t('system.waiting')}</div>
          ) : (
            lines.map((line, index) => (
              <div key={`${index}-${line.slice(0, 16)}`} className={cn('whitespace-pre-wrap break-all', logClass(line))}>
                {line}
              </div>
            ))
          )}
        </div>
      </Card>
    </>
  )
}
