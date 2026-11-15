import { useEffect, useId, useRef, type KeyboardEvent, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'
import { Button } from './Button'

export interface DialogProps {
  open: boolean
  onClose: () => void
  title?: ReactNode
  description?: ReactNode
  children?: ReactNode
  footer?: ReactNode
  size?: 'sm' | 'md' | 'lg' | 'xl'
  closeOnBackdrop?: boolean
  className?: string
  headerClassName?: string
  bodyClassName?: string
  footerClassName?: string
}

const sizes = {
  sm: 'max-w-sm',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
  xl: 'max-w-4xl',
}

const focusableSelector =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'

let bodyLockCount = 0
let bodyOverflow = ''

function lockBodyScroll() {
  if (bodyLockCount === 0) {
    bodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
  }
  bodyLockCount += 1
}

function unlockBodyScroll() {
  bodyLockCount = Math.max(0, bodyLockCount - 1)
  if (bodyLockCount === 0) {
    document.body.style.overflow = bodyOverflow
  }
}

function focusableElements(panel: HTMLDivElement | null): HTMLElement[] {
  if (!panel) return []
  return Array.from(panel.querySelectorAll<HTMLElement>(focusableSelector))
}

export function Dialog({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  size = 'md',
  closeOnBackdrop = true,
  className,
  headerClassName,
  bodyClassName,
  footerClassName,
}: DialogProps) {
  const { t } = useI18n()
  const panelRef = useRef<HTMLDivElement>(null)
  const titleID = useId()

  useEffect(() => {
    if (!open) return
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    lockBodyScroll()
    const timer = window.setTimeout(() => {
      const panel = panelRef.current
      if (!panel || panel.contains(document.activeElement)) return
      const [first] = focusableElements(panel)
      ;(first ?? panel).focus()
    }, 0)
    return () => {
      window.clearTimeout(timer)
      unlockBodyScroll()
      if (previousFocus?.isConnected) previousFocus.focus()
    }
  }, [open])

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Escape') {
      event.preventDefault()
      onClose()
      return
    }
    if (event.key !== 'Tab') return
    const panel = panelRef.current
    if (!panel) return
    const items = focusableElements(panel)
    if (items.length === 0) {
      event.preventDefault()
      panel.focus()
      return
    }
    const first = items[0]
    const last = items[items.length - 1]
    const active = document.activeElement
    if (event.shiftKey && (active === first || active === panel)) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && active === last) {
      event.preventDefault()
      first.focus()
    }
  }

  if (!open || typeof document === 'undefined') return null

  return createPortal(
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4">
      <div
        aria-hidden
        className="absolute inset-0 bg-black/45 backdrop-blur-[2px]"
        onClick={() => {
          if (closeOnBackdrop) onClose()
        }}
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? titleID : undefined}
        tabIndex={-1}
        onKeyDown={handleKeyDown}
        className={cn(
          'relative z-10 max-h-[calc(100dvh-2rem)] w-[calc(100vw-2rem)] overflow-y-auto rounded border border-border bg-surface p-0 text-fg shadow-2xl outline-none animate-fade-up',
          sizes[size],
          className,
        )}
      >
        <div className={cn('flex items-start justify-between gap-3 px-4 pt-4', headerClassName)}>
          <div className="min-w-0">
            {title ? (
              <h2 id={titleID} className="text-sm font-semibold tracking-tight">
                {title}
              </h2>
            ) : null}
            {description ? (
              <p className="mt-1 text-[11px] leading-relaxed text-muted">{description}</p>
            ) : null}
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label={t('components.dialog.close')}>
            <X className="size-4" />
          </Button>
        </div>
        {children ? <div className={cn('px-4 py-3', bodyClassName)}>{children}</div> : null}
        {footer ? (
          <div className={cn('flex items-center justify-end gap-2 border-t border-border px-4 py-3', footerClassName)}>
            {footer}
          </div>
        ) : null}
      </div>
    </div>,
    document.body,
  )
}
