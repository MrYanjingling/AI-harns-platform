import { useEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'
import { PAGE_SIZES } from '../../lib/constants'
import { Button } from '../ui/Button'

function PageSizeSelect({
  value,
  disabled,
  onChange,
}: {
  value: number
  disabled: boolean
  onChange: (value: number) => void
}) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0 })
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const menuWidth = 112
  const menuHeight = PAGE_SIZES.length * 32 + 8

  const place = () => {
    const rect = buttonRef.current?.getBoundingClientRect()
    if (!rect) return
    const left = Math.min(
      Math.max(8, rect.left),
      Math.max(8, window.innerWidth - menuWidth - 8),
    )
    const top =
      rect.bottom + 4 + menuHeight > window.innerHeight
        ? Math.max(8, rect.top - menuHeight - 4)
        : rect.bottom + 4
    setPosition({ top, left })
  }

  useEffect(() => {
    if (!open) return
    const handleClick = (event: MouseEvent) => {
      const target = event.target as Node
      if (!menuRef.current?.contains(target) && !buttonRef.current?.contains(target)) {
        setOpen(false)
      }
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false)
    }
    const handleViewportChange = () => window.requestAnimationFrame(place)
    document.addEventListener('click', handleClick)
    document.addEventListener('keydown', handleKeyDown)
    window.addEventListener('scroll', handleViewportChange, true)
    window.addEventListener('resize', handleViewportChange)
    return () => {
      document.removeEventListener('click', handleClick)
      document.removeEventListener('keydown', handleKeyDown)
      window.removeEventListener('scroll', handleViewportChange, true)
      window.removeEventListener('resize', handleViewportChange)
    }
  }, [open])

  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        disabled={disabled}
        aria-label={t('components.pagination.pageSizeAria')}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => {
          if (open) {
            setOpen(false)
            return
          }
          place()
          setOpen(true)
        }}
        className="flex h-8 w-28 items-center justify-between gap-1 rounded border border-border-strong bg-surface px-2.5 text-xs text-fg transition-colors hover:border-faint focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/15 disabled:cursor-not-allowed disabled:opacity-60"
      >
        <span className="truncate">{t('common.pagination.pageSize', { size: value })}</span>
        <ChevronDown
          className={cn('size-3.5 shrink-0 text-faint transition-transform', open && 'rotate-180')}
        />
      </button>
      {open
        ? createPortal(
            <div
              ref={menuRef}
              role="listbox"
              style={{ top: position.top, left: position.left }}
              className="fixed z-[200] w-28 rounded border border-border bg-surface p-1 shadow-[0_3px_6px_-4px_rgba(0,0,0,0.12),0_6px_16px_0_rgba(0,0,0,0.08),0_9px_28px_8px_rgba(0,0,0,0.05)]"
            >
              {PAGE_SIZES.map((size) => {
                const selected = size === value
                return (
                  <button
                    key={size}
                    type="button"
                    role="option"
                    aria-selected={selected}
                    onClick={() => {
                      onChange(size)
                      setOpen(false)
                    }}
                    className={cn(
                      'flex min-h-8 w-full items-center rounded px-3 py-1.5 text-left text-sm transition-colors',
                      selected
                        ? 'bg-primary-soft font-medium text-primary'
                        : 'text-fg hover:bg-surface-2',
                    )}
                  >
                    {t('common.pagination.pageSize', { size })}
                  </button>
                )
              })}
            </div>,
            document.body,
          )
        : null}
    </>
  )
}

export function Pagination({
  page,
  pageSize,
  total,
  onPageChange,
  onPageSizeChange,
  disabled = false,
  actions,
}: {
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
  disabled?: boolean
  actions?: ReactNode
}) {
  const { t } = useI18n()
  const pageCount = Math.max(1, Math.ceil(total / pageSize))
  const safePage = Math.min(Math.max(1, page), pageCount)

  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border bg-surface px-3 py-2">
      <p className="text-xs text-muted">
        {t('common.pagination.pageInfo', { total, page: safePage, pages: pageCount })}
      </p>
      <div className="flex items-center gap-2">
        <PageSizeSelect
          value={pageSize}
          disabled={disabled}
          onChange={onPageSizeChange}
        />
        <Button
          size="sm"
          variant="outline"
          icon={<ChevronLeft className="size-3.5" />}
          onClick={() => onPageChange(safePage - 1)}
          disabled={disabled || safePage <= 1}
        >
          {t('common.pagination.prev')}
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => onPageChange(safePage + 1)}
          disabled={disabled || safePage >= pageCount}
        >
          {t('common.pagination.next')}
          <ChevronRight className="size-3.5" />
        </Button>
        {actions}
      </div>
    </div>
  )
}
