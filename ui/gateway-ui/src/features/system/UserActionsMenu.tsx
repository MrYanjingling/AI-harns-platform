import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { KeyRound, MoreHorizontal, Pencil, Trash2 } from 'lucide-react'
import { cn, useI18n } from '@harns/core'
import type { GatewayUser } from '@/api/types'

interface UserActionsMenuProps {
  user: GatewayUser
  onEdit: () => void
  onReset: () => void
  onDelete: () => void
}

const menuWidth = 144
const menuHeight = 118

export function UserActionsMenu({ user, onEdit, onReset, onDelete }: UserActionsMenuProps) {
  const { t } = useI18n()
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0 })

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
    if (open) {
      setOpen(false)
      return
    }
    const rect = buttonRef.current?.getBoundingClientRect()
    if (!rect) return
    const left = Math.min(
      Math.max(8, rect.right - menuWidth),
      Math.max(8, window.innerWidth - menuWidth - 8),
    )
    const top =
      rect.bottom + 4 + menuHeight > window.innerHeight
        ? Math.max(8, rect.top - menuHeight - 4)
        : rect.bottom + 4
    setPosition({ top, left })
    setOpen(true)
  }

  const run = (action: () => void) => {
    setOpen(false)
    action()
  }

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={t('common.field.actions')}
        onClick={toggle}
        className="flex size-7 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
      >
        <MoreHorizontal className="size-4" />
      </button>
      {open
        ? createPortal(
            <div
              ref={menuRef}
              role="menu"
              style={{ top: position.top, left: position.left }}
              className="animate-pop fixed z-[200] w-36 origin-top-right rounded border border-border bg-surface py-1 text-fg shadow-xl"
            >
              <button
                type="button"
                role="menuitem"
                onClick={() => run(onEdit)}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
              >
                <Pencil className="size-3.5" />
                {t('common.action.edit')}
              </button>
              <button
                type="button"
                role="menuitem"
                onClick={() => run(onReset)}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
              >
                <KeyRound className="size-3.5" />
                {t('gateway.users.resetPassword')}
              </button>
              <button
                type="button"
                role="menuitem"
                disabled={user.isAdmin}
                onClick={() => run(onDelete)}
                className={cn(
                  'flex w-full items-center gap-2 border-t border-border px-3 py-2 text-left text-sm text-danger transition-colors hover:bg-danger/5',
                  user.isAdmin && 'pointer-events-none opacity-40',
                )}
              >
                <Trash2 className="size-3.5" />
                {t('common.action.delete')}
              </button>
            </div>,
            document.body,
          )
        : null}
    </>
  )
}
