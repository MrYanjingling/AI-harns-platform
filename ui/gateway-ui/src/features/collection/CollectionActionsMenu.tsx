import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { MoreVertical, Play, RotateCw, Square } from 'lucide-react'
import { cn, useI18n } from '@harns/core'
import type { Collection } from '@/api/types'

export type CollectionOperation = 'start' | 'stop' | 'restart'

interface CollectionActionsMenuProps {
  collection: Collection
  onAction: (kind: CollectionOperation) => void
}

const menuWidth = 128

export function CollectionActionsMenu({
  collection,
  onAction,
}: CollectionActionsMenuProps) {
  const { t } = useI18n()
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0 })

  const place = () => {
    const rect = buttonRef.current?.getBoundingClientRect()
    if (!rect) return
    const height = 132
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
    if (open) {
      setOpen(false)
      return
    }
    place()
    setOpen(true)
  }

  const run = (action: CollectionOperation) => {
    setOpen(false)
    onAction(action)
  }

  const collecting = collection.collectStatus === 'collecting'

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={toggle}
        aria-label={t('gateway.collection.action.more')}
        title={t('gateway.collection.action.more')}
        className={cn(
          'inline-flex size-7 items-center justify-center rounded text-primary transition-colors hover:bg-surface-2 hover:text-primary-hover',
          open && 'bg-surface-2 text-primary',
        )}
      >
        <MoreVertical className="size-4" />
      </button>

      {open
        ? createPortal(
            <div
              ref={menuRef}
              role="menu"
              style={{ top: position.top, left: position.left }}
              className="animate-pop fixed z-[200] w-32 origin-top-right rounded border border-border bg-surface py-1 text-fg shadow-xl"
            >
              <button
                type="button"
                role="menuitem"
                disabled={collecting}
                onClick={() => run('start')}
                className={cn(
                  'flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg',
                  collecting && 'pointer-events-none opacity-40',
                )}
              >
                <Play className="size-3.5" />
                {t('common.action.start')}
              </button>
              <button
                type="button"
                role="menuitem"
                disabled={!collecting}
                onClick={() => run('stop')}
                className={cn(
                  'flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg',
                  !collecting && 'pointer-events-none opacity-40',
                )}
              >
                <Square className="size-3.5" />
                {t('common.action.stop')}
              </button>
              <button
                type="button"
                role="menuitem"
                onClick={() => run('restart')}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
              >
                <RotateCw className="size-3.5" />
                {t('common.action.restart')}
              </button>
            </div>,
            document.body,
          )
        : null}
    </>
  )
}
