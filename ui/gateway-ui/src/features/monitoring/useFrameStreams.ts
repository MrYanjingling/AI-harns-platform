import { useCallback, useEffect, useRef, useState } from 'react'
import { gatewayAuthHeaders, gatewayBaseUrl, handleGatewayAuthFailure } from '@/api/gateway'
import type { FrameEvent } from '@/api/types'

const MAX_EVENTS = 500
const RETRY_DELAY_MS = 3000

type FrameKind = 'request' | 'response'

function parseFrameEvent(block: string): FrameEvent | null {
  const data = block
    .split('\n')
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trimStart())
    .join('\n')
  if (!data || data === '{}') return null
  try {
    const payload = JSON.parse(data) as Partial<FrameEvent>
    if (
      typeof payload.traceId !== 'number' ||
      typeof payload.time !== 'string' ||
      typeof payload.operation !== 'string' ||
      typeof payload.hex !== 'string'
    ) {
      return null
    }
    return {
      traceId: payload.traceId,
      time: payload.time,
      operation: payload.operation,
      hex: payload.hex,
      length: typeof payload.length === 'number' ? payload.length : 0,
      durationMs: payload.durationMs,
      error: payload.error,
    }
  } catch {
    return null
  }
}

async function consume(
  body: ReadableStream<Uint8Array>,
  onEvent: (event: FrameEvent) => void,
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
        const event = parseFrameEvent(block)
        if (event) onEvent(event)
        boundary = buffer.indexOf('\n\n')
      }
    }
  } finally {
    reader.releaseLock()
  }
}

export interface FrameStreamsState {
  requests: FrameEvent[]
  responses: FrameEvent[]
  connected: boolean
  error: boolean
  paused: boolean
  setPaused: (paused: boolean) => void
  clear: () => void
}

/** 订阅一个采集实例的请求/响应帧;仅在 enabled 时建立连接。 */
export function useFrameStreams(collectionId: string | undefined, enabled: boolean): FrameStreamsState {
  const [requests, setRequests] = useState<FrameEvent[]>([])
  const [responses, setResponses] = useState<FrameEvent[]>([])
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState(false)
  const [paused, setPaused] = useState(false)
  const pausedRef = useRef(false)

  useEffect(() => {
    pausedRef.current = paused
  }, [paused])

  useEffect(() => {
    setRequests([])
    setResponses([])
    setConnected(false)
    setError(false)
    setPaused(false)
  }, [collectionId])

  useEffect(() => {
    if (!collectionId || !enabled) {
      setConnected(false)
      setError(false)
      return
    }

    let stopped = false
    const live: Record<FrameKind, boolean> = { request: false, response: false }
    const controllers = new Set<AbortController>()
    const timers = new Set<number>()

    const syncConnected = () => {
      if (stopped) return
      setConnected(live.request || live.response)
    }

    const setLive = (kind: FrameKind, value: boolean) => {
      live[kind] = value
      syncConnected()
    }

    const connect = async (kind: FrameKind) => {
      if (stopped) return
      const controller = new AbortController()
      controllers.add(controller)
      try {
        const url = `${gatewayBaseUrl()}/collections/${encodeURIComponent(collectionId)}/frames/${kind}s/stream`
        const response = await fetch(url, {
          headers: gatewayAuthHeaders('text/event-stream'),
          cache: 'no-store',
          signal: controller.signal,
        })
        if (!response.ok || !response.body) {
          await handleGatewayAuthFailure(response)
          throw new Error(`frame stream http ${response.status}`)
        }
        if (stopped) return
        setError(false)
        setLive(kind, true)
        await consume(response.body, (event) => {
          if (stopped || pausedRef.current) return
          if (kind === 'request') {
            setRequests((current) => [...current, event].slice(-MAX_EVENTS))
          } else {
            setResponses((current) => [...current, event].slice(-MAX_EVENTS))
          }
        })
      } catch (streamError) {
        if (stopped || (streamError instanceof DOMException && streamError.name === 'AbortError')) {
          return
        }
        setError(true)
      } finally {
        controllers.delete(controller)
        setLive(kind, false)
      }
      if (!stopped) {
        const timer = window.setTimeout(() => void connect(kind), RETRY_DELAY_MS)
        timers.add(timer)
      }
    }

    void connect('request')
    void connect('response')

    return () => {
      stopped = true
      for (const timer of timers) window.clearTimeout(timer)
      for (const controller of controllers) controller.abort()
    }
  }, [collectionId, enabled])

  const clear = useCallback(() => {
    setRequests([])
    setResponses([])
  }, [])

  return { requests, responses, connected, error, paused, setPaused, clear }
}
