import { useCallback, useEffect, useRef, useState } from 'react'

export function useAutoRefresh(
  onRefresh: () => void | Promise<void>,
  defaultSeconds = 10,
) {
  const [active, setActive] = useState(false)
  const [seconds, setSeconds] = useState(defaultSeconds)
  const [refreshing, setRefreshing] = useState(false)
  const callbackRef = useRef(onRefresh)

  useEffect(() => {
    callbackRef.current = onRefresh
  }, [onRefresh])

  const refresh = useCallback(async () => {
    setRefreshing(true)
    try {
      await callbackRef.current()
    } finally {
      setRefreshing(false)
    }
  }, [])

  useEffect(() => {
    if (!active) return
    const timer = window.setInterval(() => {
      void callbackRef.current()
    }, seconds * 1000)
    return () => window.clearInterval(timer)
  }, [active, seconds])

  return {
    active,
    seconds,
    refreshing,
    setActive,
    setSeconds,
    refresh,
  }
}
