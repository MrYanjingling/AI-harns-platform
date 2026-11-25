import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { toast } from 'sonner'
import { Tooltip, cn, useI18n } from '@harns/core'

/** Wrap content with a tooltip only when it is wider than the available cell. */
export function OverflowTooltip({
  text,
  children,
  className,
}: {
  text: string
  children?: ReactNode
  className?: string
}) {
  const { t } = useI18n()
  const contentRef = useRef<HTMLSpanElement>(null)
  const [overflowing, setOverflowing] = useState(false)

  useLayoutEffect(() => {
    const node = contentRef.current
    if (!node) return
    const measure = () => {
      setOverflowing(node.scrollWidth > node.clientWidth + 1)
    }
    measure()
    const frame = window.requestAnimationFrame(measure)
    const timer = window.setTimeout(measure, 80)
    void document.fonts?.ready.then(measure)
    if (typeof ResizeObserver === 'undefined') {
      return () => {
        window.cancelAnimationFrame(frame)
        window.clearTimeout(timer)
      }
    }
    const observer = new ResizeObserver(measure)
    observer.observe(node)
    return () => {
      window.cancelAnimationFrame(frame)
      window.clearTimeout(timer)
      observer.disconnect()
    }
  }, [children, text])

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text)
      toast.success(t('components.copy.success'), { duration: 1500 })
    } catch {
      toast.error(t('components.copy.failed'))
    }
  }

  const copyOnClick = !children
  const content = (
    <span
      ref={contentRef}
      className={cn('block min-w-0 truncate', className)}
      tabIndex={overflowing && copyOnClick ? 0 : undefined}
      onClick={copyOnClick ? () => void copy() : undefined}
      onKeyDown={(event) => {
        if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'c') {
          if (window.getSelection()?.toString()) return
          event.preventDefault()
          void copy()
        }
      }}
    >
      {children ?? text}
    </span>
  )

  return (
    <Tooltip content={text} disabled={!overflowing} className="block w-full">
      {content}
    </Tooltip>
  )
}

/** Truncate one table cell and show the full value only when it overflows. */
export function TruncatedCell({
  text,
  children,
  className,
}: {
  text?: string | null
  children?: ReactNode
  className?: string
}) {
  const value = (text ?? '').trim()
  if (!value) {
    return <span className={cn('block truncate', className)}>—</span>
  }
  return (
    <OverflowTooltip text={value} className={className}>
      {children}
    </OverflowTooltip>
  )
}
