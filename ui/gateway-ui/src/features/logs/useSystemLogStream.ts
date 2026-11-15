import { useEffect, useRef, useState } from 'react'
import { gatewayBaseUrl } from '@/api/gateway'

export type StreamLogLevel = 'info' | 'warn' | 'error'

export interface StreamLogLine {
  id: number
  level: StreamLogLevel
  text: string
}

interface LogStreamState {
  lines: StreamLogLine[]
  error: boolean
}

const MAX_LINES = 2000
const RETRY_DELAY_MS = 3000

function parseLevel(line: string): StreamLogLevel | null {
  const textMatch = line.match(/\blevel=(DEBUG|INFO|WARN|ERROR)\b/i)
  const jsonMatch = line.match(/"level"\s*:\s*"(DEBUG|INFO|WARN|ERROR)"/i)
  const value = (textMatch?.[1] ?? jsonMatch?.[1] ?? '').toLowerCase()
  if (value === 'error') return 'error'
  if (value === 'warn') return 'warn'
  if (value === 'info') return 'info'
  return null
}

function parseEvent(event: string): string | null {
  const data = event
    .split('\n')
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trimStart())
    .join('\n')
  if (!data) return null
  try {
    const payload = JSON.parse(data) as { line?: unknown }
    return typeof payload.line === 'string' ? payload.line : null
  } catch {
    return null
  }
}

async function consume(body: ReadableStream<Uint8Array>, onLine: (line: string) => void): Promise<void> {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) return
      buffer += decoder.decode(value, { stream: true })
      let boundary = buffer.indexOf('\n\n')
      while (boundary >= 0) {
        const event = buffer.slice(0, boundary)
        buffer = buffer.slice(boundary + 2)
        const line = parseEvent(event)
        if (line) onLine(line)
        boundary = buffer.indexOf('\n\n')
      }
    }
  } finally {
    reader.releaseLock()
  }
}

export function useSystemLogStream(): LogStreamState {
  const [lines, setLines] = useState<StreamLogLine[]>([])
  const [error, setError] = useState(false)
  const nextId = useRef(0)

  useEffect(() => {
    let stopped = false
    let retryTimer: number | undefined
    let controller: AbortController | null = null

    const connect = async () => {
      if (stopped) return
      controller = new AbortController()
      try {
        const response = await fetch(`${gatewayBaseUrl()}/system/logs/stream`, {
          headers: { Accept: 'text/event-stream' },
          cache: 'no-store',
          signal: controller.signal,
        })
        if (!response.ok || !response.body) {
          throw new Error(`log stream http ${response.status}`)
        }
        setError(false)
        await consume(response.body, (text) => {
          const level = parseLevel(text)
          if (!level) return
          const entry = { id: nextId.current++, level, text }
          setLines((current) => [...current, entry].slice(-MAX_LINES))
        })
      } catch (streamError) {
        if (stopped || (streamError instanceof DOMException && streamError.name === 'AbortError')) {
          return
        }
        setError(true)
      }
      if (!stopped) {
        retryTimer = window.setTimeout(connect, RETRY_DELAY_MS)
      }
    }

    void connect()
    return () => {
      stopped = true
      if (retryTimer !== undefined) window.clearTimeout(retryTimer)
      controller?.abort()
    }
  }, [])

  return { lines, error }
}
