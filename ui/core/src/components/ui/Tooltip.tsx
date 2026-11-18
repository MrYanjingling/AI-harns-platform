import { useEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { cn } from '../../lib/cn'

const TOOLTIP_WIDTH = 200

export function Tooltip({
  content,
  children,
  className,
  disabled = false,
  focusable = false,
  ariaLabel,
}: {
  content: ReactNode
  children: ReactNode
  className?: string
  disabled?: boolean
  focusable?: boolean
  ariaLabel?: string
}) {
  const rootRef = useRef<HTMLSpanElement>(null)
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0, above: true })

  useEffect(() => {
    if (!open) return
    const close = () => setOpen(false)
    window.addEventListener('scroll', close, true)
    window.addEventListener('resize', close)
    return () => {
      window.removeEventListener('scroll', close, true)
      window.removeEventListener('resize', close)
    }
  }, [open])

  const show = () => {
    if (disabled || !content) return
    const rect = rootRef.current?.getBoundingClientRect()
    if (!rect) return
    const textLength = typeof content === 'string' ? content.length : TOOLTIP_WIDTH / 12
    const estimatedWidth = Math.min(TOOLTIP_WIDTH, textLength * 12 + 16)
    const half = estimatedWidth / 2
    const center = rect.left + rect.width / 2
    const left = Math.min(
      Math.max(8 + half, center),
      Math.max(8 + half, window.innerWidth - 8 - half),
    )
    const above = rect.top > 40
    setPosition({
      top: above ? rect.top - 6 : rect.bottom + 6,
      left,
      above,
    })
    setOpen(true)
  }

  return (
    <span
      ref={rootRef}
      className={cn('inline-flex min-w-0', className)}
      onMouseEnter={show}
      onMouseLeave={() => setOpen(false)}
      onFocus={(event) => {
        if ((event.target as HTMLElement).matches(':focus-visible')) show()
      }}
      onBlur={() => setOpen(false)}
      tabIndex={focusable ? 0 : undefined}
      aria-label={ariaLabel}
    >
      {children}
      {open
        ? createPortal(
            <div
              role="tooltip"
              style={{
                position: 'fixed',
                top: position.top,
                left: position.left,
                width: 'max-content',
                maxWidth: `min(${TOOLTIP_WIDTH}px, calc(100vw - 1rem))`,
                transform: position.above
                  ? 'translate(-50%, -100%)'
                  : 'translate(-50%, 0)',
              }}
              className="app-tooltip"
            >
              {content}
              <span
                className={cn(
                  'app-tooltip-arrow',
                  position.above ? 'app-tooltip-arrow--above' : 'app-tooltip-arrow--below',
                )}
              />
            </div>,
            document.body,
          )
        : null}
    </span>
  )
}
