import { useEffect, useState } from 'react'
import { gatewayBaseUrl } from '@/api/gateway'
import type { PushCommandEvent } from '@/api/types'

const MAX_EVENTS = 200
const RETRY_DELAY_MS = 3000

function parseEvent(block: string): PushCommandEvent | null {
  const data = block
    .split('\n')
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trimStart())
    .join('\n')
  if (!data || data === '{}') return null
  try {
    const payload = JSON.parse(data) as Partial<PushCommandEvent>
    if (
      typeof payload.groupId !== 'string' ||
      typeof payload.connectorId !== 'string' ||
      typeof payload.connectorType !== 'string' ||
      typeof payload.topic !== 'string' ||
      typeof payload.time !== 'string'
    ) {
      return null
    }
    return {
      groupId: payload.groupId,
      connectorId: payload.connectorId,
      connectorType: payload.connectorType,
      topic: payload.topic,
      time: payload.time,
      payload: payload.payload ?? null,
    }
  } catch {
    return null
  }
}

async function consume(
  body: ReadableStream<Uint8Array>,
  onEvent: (event: PushCommandEvent) => void,
): Promise<void> {
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
        const block = buffer.slice(0, boundary)
        buffer = buffer.slice(boundary + 2)
        const event = parseEvent(block)
        if (event) onEvent(event)
        boundary = buffer.indexOf('\n\n')
      }
    }
  } finally {
    reader.releaseLock()
  }
}

export function usePushGroupCommandStream(groupId: string | undefined, enabled: boolean) {
  const [events, setEvents] = useState<PushCommandEvent[]>([])
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState(false)

  useEffect(() => {
    setEvents([])
    setConnected(false)
    setError(false)
  }, [groupId])

  useEffect(() => {
    if (!groupId || !enabled) {
      setEvents([])
      setConnected(false)
      setError(false)
      return
    }

    let stopped = false
    let controller: AbortController | null = null
    let retryTimer: number | undefined

    const connect = async () => {
      if (stopped) return
      controller = new AbortController()
      try {
        const response = await fetch(
          `${gatewayBaseUrl()}/push/groups/${encodeURIComponent(groupId)}/commands/stream`,
          {
            headers: { Accept: 'text/event-stream' },
            cache: 'no-store',
            signal: controller.signal,
          },
        )
        if (!response.ok || !response.body) {
          throw new Error(`push command stream http ${response.status}`)
        }
        if (stopped) return
        setConnected(true)
        setError(false)
        await consume(response.body, (event) => {
          if (stopped) return
          setEvents((current) => [...current, event].slice(-MAX_EVENTS))
        })
      } catch (streamError) {
        if (stopped || (streamError instanceof DOMException && streamError.name === 'AbortError')) {
          return
        }
        setError(true)
      } finally {
        setConnected(false)
      }
      if (!stopped) {
        retryTimer = window.setTimeout(() => void connect(), RETRY_DELAY_MS)
      }
    }

    void connect()
    return () => {
      stopped = true
      if (retryTimer) window.clearTimeout(retryTimer)
      controller?.abort()
    }
  }, [enabled, groupId])

  return { events, connected, error }
}
