import { useEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { MoreVertical } from 'lucide-react'
import { cn } from '../../lib/cn'
import { TableActionButton } from '../table/TableActionButton'

export interface ActionMenuItem {
  key: string
  label: ReactNode
  icon?: ReactNode
  disabled?: boolean
  tone?: 'default' | 'danger'
  onSelect: () => void
}

export function ActionMenu({
  label,
  items,
  disabled = false,
  menuWidth = 140,
  className,
}: {
  label: string
  items: ActionMenuItem[]
  disabled?: boolean
  menuWidth?: number
  className?: string
}) {
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0 })

  const place = () => {
    const rect = buttonRef.current?.getBoundingClientRect()
    if (!rect) return
    const height = items.length * 40 + 8
    const left = Math.min(
      Math.max(8, rect.right - menuWidth),
      Math.max(8, window.innerWidth - menuWidth - 8),
    )
    const top =
      rect.bottom + 4 + height > window.innerHeight
        ? Math.max(8, rect.top - height - 4)
        : rect.bottom + 4
    setPosition({ top, left })
  }

  useEffect(() => {
    if (!open) return
    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target as Node
      if (!menuRef.current?.contains(target) && !buttonRef.current?.contains(target)) {
        setOpen(false)
      }
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false)
    }
    const handleViewportChange = () => setOpen(false)
    document.addEventListener('pointerdown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)
    window.addEventListener('scroll', handleViewportChange, true)
    window.addEventListener('resize', handleViewportChange)
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
      window.removeEventListener('scroll', handleViewportChange, true)
      window.removeEventListener('resize', handleViewportChange)
    }
  }, [open])

  const toggle = () => {
    if (disabled) return
    if (open) {
      setOpen(false)
      return
    }
    place()
    setOpen(true)
  }

  const run = (item: ActionMenuItem) => {
    if (item.disabled) return
    setOpen(false)
    item.onSelect()
  }

  return (
    <>
      <TableActionButton
        ref={buttonRef}
        label={label}
        icon={<MoreVertical />}
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={toggle}
        className={cn(open && 'bg-surface-2', className)}
      />
      {open
        ? createPortal(
            <div
              ref={menuRef}
              role="menu"
              style={{ top: position.top, left: position.left, width: menuWidth }}
              className="animate-pop fixed z-[200] origin-top-right rounded border border-border bg-surface py-1 text-fg shadow-xl"
            >
              {items.map((item, index) => (
                <button
                  key={item.key}
                  type="button"
                  role="menuitem"
                  disabled={item.disabled}
                  onClick={() => run(item)}
                  className={cn(
                    'flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors',
                    index > 0 && item.tone === 'danger' && 'border-t border-border',
                    item.tone === 'danger'
                      ? 'text-danger hover:bg-danger/5'
                      : 'text-muted hover:bg-surface-2 hover:text-fg',
                    item.disabled && 'pointer-events-none opacity-40',
                  )}
                >
                  {item.icon ? (
                    <span className="flex shrink-0 [&_svg]:size-3.5">{item.icon}</span>
                  ) : null}
                  {item.label}
                </button>
              ))}
            </div>,
            document.body,
          )
        : null}
    </>
  )
}
