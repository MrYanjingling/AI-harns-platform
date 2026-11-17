import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { MoreVertical, Play, Radio, RotateCw, Square } from 'lucide-react'
import { cn, useI18n } from '@harns/core'
import type { PushGroup } from '@/api/types'

interface PushGroupActionsMenuProps {
  group: PushGroup
  connectorStatus?: string
  disabled?: boolean
  onStart: () => void
  onStop: () => void
  onRestart: () => void
  onViewPayloads: () => void
}

const menuWidth = 140

export function PushGroupActionsMenu({
  group,
  connectorStatus,
  disabled = false,
  onStart,
  onStop,
  onRestart,
  onViewPayloads,
}: PushGroupActionsMenuProps) {
  const { t } = useI18n()
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0 })

  const place = () => {
    const rect = buttonRef.current?.getBoundingClientRect()
    if (!rect) return
    const height = 156
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

  const run = (callback: () => void) => {
    setOpen(false)
    callback()
  }

  const canStart = group.runState === 'stopped'
  const canStop = group.status !== 'stopped'
  const canRestart = group.runState === 'running'
  const canViewPayloads = group.status === 'running' && connectorStatus === 'running'

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => {
          if (disabled) return
          if (open) {
            setOpen(false)
            return
          }
          place()
          setOpen(true)
        }}
        aria-label={t('gateway.push.action.more')}
        title={t('gateway.push.action.more')}
        className={cn(
          'inline-flex size-7 items-center justify-center rounded text-primary transition-colors hover:bg-surface-2 hover:text-primary-hover disabled:cursor-not-allowed disabled:opacity-40',
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
              className="animate-pop fixed z-[200] w-[140px] origin-top-right rounded border border-border bg-surface py-1 text-fg shadow-xl"
            >
              <button
                type="button"
                role="menuitem"
                disabled={!canViewPayloads}
                onClick={() => run(onViewPayloads)}
                className={cn(
                  'flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg',
                  !canViewPayloads && 'pointer-events-none opacity-40',
                )}
              >
                <Radio className="size-3.5" />
                {t('gateway.push.payload.view')}
              </button>
              <button
                type="button"
                role="menuitem"
                disabled={!canStart}
                onClick={() => run(onStart)}
                className={cn(
                  'flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg',
                  !canStart && 'pointer-events-none opacity-40',
                )}
              >
                <Play className="size-3.5" />
                {t('common.action.start')}
              </button>
              <button
                type="button"
                role="menuitem"
                disabled={!canStop}
                onClick={() => run(onStop)}
                className={cn(
                  'flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg',
                  !canStop && 'pointer-events-none opacity-40',
                )}
              >
                <Square className="size-3.5" />
                {t('common.action.stop')}
              </button>
              <button
                type="button"
                role="menuitem"
                disabled={!canRestart}
                onClick={() => run(onRestart)}
                className={cn(
                  'flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg',
                  !canRestart && 'pointer-events-none opacity-40',
                )}
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
