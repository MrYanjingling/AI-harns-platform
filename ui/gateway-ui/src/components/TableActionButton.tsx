import {
  forwardRef,
  useEffect,
  useRef,
  useState,
  type ButtonHTMLAttributes,
  type ReactNode,
} from 'react'
import { createPortal } from 'react-dom'
import { cn } from '@harns/core'

export interface TableActionButtonProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  label: string
  icon: ReactNode
}

export const TableActionButton = forwardRef<HTMLButtonElement, TableActionButtonProps>(
  function TableActionButton(
    {
      label,
      icon,
      className,
      disabled,
      onMouseEnter,
      onMouseLeave,
      onFocus,
      onBlur,
      ...props
    },
    forwardedRef,
  ) {
    const buttonRef = useRef<HTMLButtonElement>(null)
    const [open, setOpen] = useState(false)
    const [position, setPosition] = useState({ top: 0, left: 0, above: true })

    const show = () => {
      if (disabled) return
      const rect = buttonRef.current?.getBoundingClientRect()
      if (!rect) return
      const estimatedWidth = Math.min(200, label.length * 12 + 16)
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

    const hide = () => setOpen(false)

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

    return (
      <>
        <button
          ref={(node) => {
            buttonRef.current = node
            if (typeof forwardedRef === 'function') forwardedRef(node)
            else if (forwardedRef) forwardedRef.current = node
          }}
          type="button"
          disabled={disabled}
          aria-label={label}
          title={label}
          onMouseEnter={(event) => {
            show()
            onMouseEnter?.(event)
          }}
          onMouseLeave={(event) => {
            hide()
            onMouseLeave?.(event)
          }}
          onFocus={(event) => {
            show()
            onFocus?.(event)
          }}
          onBlur={(event) => {
            hide()
            onBlur?.(event)
          }}
          className={cn(
            'inline-flex size-7 shrink-0 items-center justify-center rounded text-muted transition-colors',
            'hover:bg-surface-2 hover:text-fg disabled:pointer-events-none disabled:opacity-40',
            '[&_svg]:size-4',
            className,
          )}
          {...props}
        >
          {icon}
        </button>
        {open
          ? createPortal(
              <div
                role="tooltip"
                style={{
                  position: 'fixed',
                  top: position.top,
                  left: position.left,
                  width: 'max-content',
                  maxWidth: 'min(200px, calc(100vw - 1rem))',
                  transform: position.above
                    ? 'translate(-50%, -100%)'
                    : 'translate(-50%, 0)',
                }}
                className="pointer-events-none z-[400] rounded bg-header px-2 py-1 text-[11px] leading-4 text-header-fg shadow-lg"
              >
                {label}
                <span
                  className={cn(
                    'absolute left-1/2 size-2 -translate-x-1/2 rotate-45 bg-header',
                    position.above ? 'top-full -translate-y-1/2' : 'bottom-full translate-y-1/2',
                  )}
                />
              </div>,
              document.body,
            )
          : null}
      </>
    )
  },
)
