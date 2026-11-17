import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { MoreVertical, Play, Plus, RotateCw, Square } from 'lucide-react'
import { cn, useI18n } from '@harns/core'
import { TableActionButton } from '@/components/TableActionButton'
import type { PushConnector } from '@/api/types'

export type ConnectorOperation = 'start' | 'stop' | 'restart'

interface ConnectorActionsMenuProps {
  connector: PushConnector
  disabled?: boolean
  onAction: (kind: ConnectorOperation) => void
  onCreateGroup: () => void
}

const menuWidth = 140

export function ConnectorActionsMenu({
  connector,
  disabled = false,
  onAction,
  onCreateGroup,
}: ConnectorActionsMenuProps) {
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

  const toggle = () => {
    if (disabled) return
    if (open) {
      setOpen(false)
      return
    }
    place()
    setOpen(true)
  }

  const run = (action: ConnectorOperation) => {
    setOpen(false)
    onAction(action)
  }

  const canStart = connector.runState === 'stopped'
  const canStop = connector.status !== 'stopped'
  const canRestart = connector.runState === 'running'

  return (
    <>
      <TableActionButton
        ref={buttonRef}
        label={t('gateway.push.action.more')}
        icon={<MoreVertical />}
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={toggle}
        className={cn(open && 'bg-surface-2')}
      />

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
                onClick={() => {
                  setOpen(false)
                  onCreateGroup()
                }}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-fg"
              >
                <Plus className="size-3.5" />
                {t('gateway.push.action.createGroup')}
              </button>
              <button
                type="button"
                role="menuitem"
                disabled={!canStart}
                onClick={() => run('start')}
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
                onClick={() => run('stop')}
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
                onClick={() => run('restart')}
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
