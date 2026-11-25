import { useEffect, useRef, useState, type KeyboardEvent, type ReactNode } from 'react'
import { X } from 'lucide-react'
import { cn } from '../../lib/cn'
import { useI18n } from '../../i18n'
import { Button } from './Button'

export interface RightDrawerProps {
  open: boolean
  onClose: () => void
  title: string
  resizeLabel: string
  icon?: ReactNode
  headerActions?: ReactNode
  children: ReactNode
  defaultWidth?: number
  minWidth?: number
  maxWidth?: number
  bodyClassName?: string
}

/** A resizable, full-height drawer that participates in the page layout. */
export function RightDrawer({
  open,
  onClose,
  title,
  resizeLabel,
  icon,
  headerActions,
  children,
  defaultWidth = 480,
  minWidth = 360,
  maxWidth = 960,
  bodyClassName,
}: RightDrawerProps) {
  const { t } = useI18n()
  const drawerRef = useRef<HTMLElement>(null)
  const [width, setWidth] = useState(() => {
    if (typeof window === 'undefined') return defaultWidth
    return Math.min(defaultWidth, Math.max(280, window.innerWidth - 48))
  })
  const [resizing, setResizing] = useState(false)

  const clampWidth = (value: number) => {
    if (typeof window === 'undefined') return value
    const max = Math.min(maxWidth, Math.max(280, window.innerWidth - 48))
    const min = Math.min(minWidth, max)
    return Math.min(max, Math.max(min, value))
  }

  useEffect(() => {
    const handleResize = () => setWidth((current) => clampWidth(current))
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  useEffect(() => {
    if (!resizing) return
    const handlePointerMove = (event: PointerEvent) => {
      setWidth(clampWidth(window.innerWidth - event.clientX))
    }
    const handlePointerUp = () => setResizing(false)
    window.addEventListener('pointermove', handlePointerMove)
    window.addEventListener('pointerup', handlePointerUp)
    return () => {
      window.removeEventListener('pointermove', handlePointerMove)
      window.removeEventListener('pointerup', handlePointerUp)
    }
  }, [resizing])

  useEffect(() => {
    if (!open) return
    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key !== 'Escape') return
      event.preventDefault()
      onClose()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onClose, open])

  useEffect(() => {
    const node = drawerRef.current
    if (!node) return
    if (open) {
      node.removeAttribute('inert')
    } else {
      node.setAttribute('inert', '')
    }
  }, [open])

  const handleResizeKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return
    event.preventDefault()
    const delta = event.key === 'ArrowLeft' ? 16 : -16
    setWidth((current) => clampWidth(current + delta))
  }

  return (
    <aside
      ref={drawerRef}
      aria-label={title}
      aria-hidden={!open}
      className={cn(
        'relative flex h-full shrink-0 flex-col overflow-hidden bg-surface transition-[width] duration-200 ease-out',
        open ? 'border-l border-border' : 'border-l-0 pointer-events-none',
        resizing && 'transition-none select-none',
      )}
      style={{
        width: open ? `${width}px` : '0px',
        height: 'calc(100dvh - var(--layout-footer-height))',
      }}
    >
      <div
        role="separator"
        aria-label={resizeLabel}
        aria-orientation="vertical"
        aria-valuemin={minWidth}
        aria-valuemax={maxWidth}
        aria-valuenow={Math.round(width)}
        tabIndex={0}
        onPointerDown={(event) => {
          event.preventDefault()
          setResizing(true)
        }}
        onKeyDown={handleResizeKeyDown}
        className="group absolute inset-y-0 left-0 z-10 hidden w-2 -translate-x-1/2 cursor-col-resize touch-none md:block"
      >
        <span className="absolute inset-y-0 left-1/2 w-0.5 -translate-x-1/2 bg-transparent transition-colors group-hover:bg-primary/60 group-focus-visible:bg-primary group-active:bg-primary" />
      </div>

      <div className="flex h-12 shrink-0 items-center justify-between border-b border-border px-4">
        <div className="flex min-w-0 items-center gap-2 text-sm font-semibold text-fg">
          {icon ? <span className="shrink-0">{icon}</span> : null}
          <span className="truncate">{title}</span>
        </div>
        <div className="flex min-w-0 shrink-0 items-center gap-1.5">
          {headerActions}
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={t('components.dialog.close')}
            title={t('components.dialog.close')}
            onClick={onClose}
          >
            <X className="size-4" />
          </Button>
        </div>
      </div>

      <div className={cn('flex min-h-0 flex-1 flex-col overflow-hidden', bodyClassName)}>{children}</div>
    </aside>
  )
}
