import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { createPortal } from 'react-dom'
import { ChevronDown, Search } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'

export interface SearchSelectOption {
  value: string
  label: string
}

interface SearchSelectProps {
  id?: string
  value: string
  options: SearchSelectOption[]
  onChange: (value: string) => void
  ariaLabel: string
  emptyText?: string
  placeholderValue?: string
  searchable?: boolean
  disabled?: boolean
  className?: string
  panelClassName?: string
}

export function SearchSelect({
  id,
  value,
  options,
  onChange,
  ariaLabel,
  emptyText,
  placeholderValue,
  searchable = true,
  disabled = false,
  className,
  panelClassName,
}: SearchSelectProps) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const containerRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLDivElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const inputId = useId()
  const [panelPosition, setPanelPosition] = useState({ top: 0, left: 0, width: 0 })

  const selected = options.find((option) => option.value === value)
  const filtered = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    return keyword
      ? options.filter((option) => option.label.toLowerCase().includes(keyword))
      : options
  }, [options, query])

  const close = () => {
    setOpen(false)
    setQuery('')
  }

  const placePanel = () => {
    const rect = triggerRef.current?.getBoundingClientRect()
    if (!rect) return
    const width = Math.max(rect.width, 1)
    const padding = 8
    const left = Math.min(
      Math.max(padding, rect.left),
      Math.max(padding, window.innerWidth - width - padding),
    )
    const estimatedHeight = Math.min(224, Math.max(40, options.length * 32 + 8))
    const top =
      rect.bottom + 4 + estimatedHeight > window.innerHeight
        ? Math.max(padding, rect.top - estimatedHeight - 4)
        : rect.bottom + 4
    setPanelPosition({ top, left, width })
  }

  const openSelect = () => {
    if (disabled) return
    placePanel()
    setOpen(true)
    if (searchable) {
      window.requestAnimationFrame(() => {
        inputRef.current?.focus()
        inputRef.current?.select()
      })
    }
  }

  const select = (next: string) => {
    onChange(next)
    close()
  }

  useEffect(() => {
    if (!open) return
    setActiveIndex(options.findIndex((option) => option.value === value))
    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target as Node
      if (!containerRef.current?.contains(target) && !panelRef.current?.contains(target)) {
        close()
      }
    }
    const handleViewportChange = () => close()
    document.addEventListener('pointerdown', handlePointerDown)
    window.addEventListener('scroll', handleViewportChange, true)
    window.addEventListener('resize', handleViewportChange)
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown)
      window.removeEventListener('scroll', handleViewportChange, true)
      window.removeEventListener('resize', handleViewportChange)
    }
  }, [open, options, value])

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (disabled) return
    if (event.key === 'Escape') {
      event.preventDefault()
      close()
      return
    }
    if (!open) {
      if (event.key === 'ArrowDown' || event.key === 'Enter' || event.key === ' ') {
        event.preventDefault()
        openSelect()
      }
      return
    }
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      setActiveIndex((index) => (filtered.length ? (index + 1) % filtered.length : 0))
    } else if (event.key === 'ArrowUp') {
      event.preventDefault()
      setActiveIndex((index) =>
        filtered.length
          ? index < 0
            ? filtered.length - 1
            : (index - 1 + filtered.length) % filtered.length
          : -1,
      )
    } else if (event.key === 'Enter') {
      event.preventDefault()
      const next = filtered[activeIndex]
      if (next) select(next.value)
    }
  }

  return (
    <div
      ref={containerRef}
      className="relative"
      aria-disabled={disabled}
      onKeyDown={handleKeyDown}
    >
      <div
        ref={triggerRef}
        className={cn(
          'flex h-[var(--select-height)] items-center rounded border bg-surface text-sm transition-colors',
          open
            ? 'border-primary ring-2 ring-primary/15'
            : 'border-border-strong hover:border-faint',
          disabled && 'cursor-not-allowed opacity-60 hover:border-border-strong',
          className,
        )}
      >
        {searchable ? (
          <>
            <input
              id={id ?? inputId}
              ref={inputRef}
              value={open ? query : (selected?.label ?? '')}
              onChange={(event) => {
                if (!open) setOpen(true)
                setQuery(event.target.value)
                setActiveIndex(0)
              }}
              onClick={() => {
                if (!open) openSelect()
              }}
              onFocus={() => {
                if (!open) openSelect()
              }}
              placeholder={selected?.label ?? ariaLabel}
              aria-label={ariaLabel}
              disabled={disabled}
              aria-haspopup="listbox"
              aria-expanded={open}
              className={cn(
                'h-full min-w-0 flex-1 bg-transparent pl-2 text-sm outline-none placeholder:text-faint',
                !open && selected?.value === placeholderValue ? 'text-faint' : 'text-fg',
              )}
            />
            <button
              type="button"
              aria-label={t('common.action.search')}
              disabled={disabled}
              onClick={openSelect}
              className="flex h-full items-center px-2 text-faint transition-colors hover:text-fg"
            >
              <Search className="size-3.5" />
            </button>
          </>
        ) : (
          <button
            type="button"
            id={id ?? inputId}
            aria-label={ariaLabel}
            aria-haspopup="listbox"
            aria-expanded={open}
            disabled={disabled}
            onClick={() => (open ? close() : openSelect())}
            className="flex h-full min-w-0 flex-1 items-center gap-1 bg-transparent pr-2 pl-2 text-left text-sm outline-none"
          >
            <span
              className={cn(
                'min-w-0 flex-1 truncate',
                selected?.value === placeholderValue ? 'text-faint' : 'text-fg',
              )}
            >
              {selected?.label ?? ''}
            </span>
            <ChevronDown className="size-3.5 shrink-0 text-faint" />
          </button>
        )}
      </div>

      {open
        ? createPortal(
            <div
              ref={panelRef}
              style={{
                position: 'fixed',
                top: panelPosition.top,
                left: panelPosition.left,
                width: panelPosition.width,
              }}
              className={cn(
                'z-[120] rounded border border-border bg-surface p-1 text-fg shadow-[0_3px_6px_-4px_rgba(0,0,0,0.12),0_6px_16px_0_rgba(0,0,0,0.08),0_9px_28px_8px_rgba(0,0,0,0.05)]',
                panelClassName,
              )}
            >
              <div role="listbox" className="max-h-[var(--select-panel-max-height)] overflow-auto py-1">
                {filtered.length === 0 ? (
                  <p className="px-3 py-2 text-xs text-faint">
                    {emptyText ?? t('common.state.empty')}
                  </p>
                ) : (
                  filtered.map((option, index) => {
                    const isSelected = option.value === value
                    return (
                      <button
                        key={option.value}
                        type="button"
                        role="option"
                        aria-selected={isSelected}
                        onMouseEnter={() => setActiveIndex(index)}
                        onClick={() => select(option.value)}
                        className={cn(
                          'flex min-h-8 w-full items-center rounded px-3 py-1.5 text-left text-sm transition-colors',
                          isSelected
                            ? 'bg-primary-soft font-medium text-primary'
                            : index === activeIndex
                              ? 'bg-surface-2 text-fg'
                              : 'text-fg',
                        )}
                      >
                        {option.label}
                      </button>
                    )
                  })
                )}
              </div>
            </div>,
            document.body,
          )
        : null}
    </div>
  )
}
