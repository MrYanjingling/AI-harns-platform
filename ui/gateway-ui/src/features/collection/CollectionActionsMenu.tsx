import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { ChevronRight, Eye, Pencil, Play, RotateCw, Square, Trash2 } from 'lucide-react'
import { cn, useI18n } from '@harns/core'
import type { Collection } from '@/api/types'

type CollectionOperation = 'start' | 'stop' | 'restart' | 'delete'

interface CollectionActionsMenuProps {
  collection: Collection
  onView: () => void
  onEdit: () => void
  onAction: (kind: CollectionOperation) => void
}

const menuWidth = 144

export function CollectionActionsMenu({
  collection,
  onView,
  onEdit,
  onAction,
}: CollectionActionsMenuProps) {
  const { t } = useI18n()
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [expanded, setExpanded] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0 })

  const place = (nextExpanded: boolean) => {
    const rect = buttonRef.current?.getBoundingClientRect()
    if (!rect) return
    const height = nextExpanded ? 286 : 124
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
    setExpanded(false)
    place(false)
    setOpen(true)
  }

  const toggleExpanded = () => {
    const next = !expanded
    setExpanded(next)
    place(next)
  }

  const run = (action: () => void) => {
    setOpen(false)
    action()
  }

  const startDisabled = collection.collectStatus === 'collecting'
  const stopDisabled = !startDisabled

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
        <span className="text-lg leading-none">...</span>
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
                onClick={() => run(onView)}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
              >
                <Eye className="size-3.5" />
                {t('gateway.collection.action.view')}
              </button>
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
                aria-expanded={expanded}
                onClick={toggleExpanded}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
              >
                <ChevronRight
                  className={cn('size-3.5 transition-transform', expanded && 'rotate-90')}
                />
                {t('gateway.collection.action.operate')}
              </button>
              {expanded ? (
                <div className="border-t border-border pt-1">
                  <button
                    type="button"
                    role="menuitem"
                    disabled={startDisabled}
                    onClick={() => run(() => onAction('start'))}
                    className={cn(
                      'flex w-full items-center gap-2 py-2 pr-3 pl-8 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg',
                      startDisabled && 'pointer-events-none opacity-40',
                    )}
                  >
                    <Play className="size-3.5" />
                    {t('common.action.start')}
                  </button>
                  <button
                    type="button"
                    role="menuitem"
                    disabled={stopDisabled}
                    onClick={() => run(() => onAction('stop'))}
                    className={cn(
                      'flex w-full items-center gap-2 py-2 pr-3 pl-8 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg',
                      stopDisabled && 'pointer-events-none opacity-40',
                    )}
                  >
                    <Square className="size-3.5" />
                    {t('common.action.stop')}
                  </button>
                  <button
                    type="button"
                    role="menuitem"
                    onClick={() => run(() => onAction('restart'))}
                    className="flex w-full items-center gap-2 py-2 pr-3 pl-8 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
                  >
                    <RotateCw className="size-3.5" />
                    {t('common.action.restart')}
                  </button>
                  <button
                    type="button"
                    role="menuitem"
                    onClick={() => run(() => onAction('delete'))}
                    className="flex w-full items-center gap-2 py-2 pr-3 pl-8 text-left text-sm text-danger transition-colors hover:bg-danger/5"
                  >
                    <Trash2 className="size-3.5" />
                    {t('common.action.delete')}
                  </button>
                </div>
              ) : null}
            </div>,
            document.body,
          )
        : null}
    </>
  )
}
