import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { Search } from 'lucide-react'
import { cn, useI18n } from '@harns/core'
import type { StreamLogLevel } from './useSystemLogStream'

interface LogLevelSelectProps {
  value: StreamLogLevel
  onChange: (value: StreamLogLevel) => void
}

const options: Array<{ value: StreamLogLevel; label: string }> = [
  { value: 'error', label: 'Error' },
  { value: 'warn', label: 'Warn' },
  { value: 'info', label: 'Info' },
]

export function LogLevelSelect({ value, onChange }: LogLevelSelectProps) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const selected = options.find((option) => option.value === value) ?? options[2]
  const filtered = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (!keyword) return options
    return options.filter((option) => option.label.toLowerCase().includes(keyword))
  }, [query])

  const close = () => {
    setOpen(false)
    setQuery('')
  }

  const openSelect = () => {
    setOpen(true)
    window.requestAnimationFrame(() => {
      inputRef.current?.focus()
      inputRef.current?.select()
    })
  }

  const select = (next: StreamLogLevel) => {
    onChange(next)
    close()
  }

  useEffect(() => {
    if (!open) return
    setActiveIndex(Math.max(0, options.findIndex((option) => option.value === value)))
    const handlePointerDown = (event: PointerEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) close()
    }
    document.addEventListener('pointerdown', handlePointerDown)
    return () => document.removeEventListener('pointerdown', handlePointerDown)
  }, [open, value])

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
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
        filtered.length ? (index - 1 + filtered.length) % filtered.length : 0,
      )
    } else if (event.key === 'Enter') {
      event.preventDefault()
      const next = filtered[activeIndex]
      if (next) select(next.value)
    }
  }

  return (
    <div ref={containerRef} className="relative" onKeyDown={handleKeyDown}>
      <div
        className={cn(
          'flex h-9 w-[110px] items-center rounded border bg-surface text-sm transition-colors',
          open
            ? 'border-primary ring-2 ring-primary/15'
            : 'border-border-strong hover:border-faint',
        )}
      >
        <input
          id="gateway-log-level"
          ref={inputRef}
          value={open ? query : selected.label}
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
          placeholder={selected.label}
          aria-label={t('gateway.logs.filter.level')}
          aria-haspopup="listbox"
          aria-expanded={open}
          className="h-full min-w-0 flex-1 bg-transparent pl-2 text-sm text-fg outline-none placeholder:text-faint"
        />
        <button
          type="button"
          aria-label={t('common.action.search')}
          onClick={openSelect}
          className="flex h-full items-center px-2 text-faint transition-colors hover:text-fg"
        >
          <Search className="size-3.5" />
        </button>
      </div>

      {open ? (
        <div className="absolute top-full left-0 z-50 mt-1 w-44 rounded border border-border bg-surface p-1 text-fg shadow-[0_3px_6px_-4px_rgba(0,0,0,0.12),0_6px_16px_0_rgba(0,0,0,0.08),0_9px_28px_8px_rgba(0,0,0,0.05)]">
          <div role="listbox" className="max-h-56 overflow-auto py-1">
            {filtered.length === 0 ? (
              <p className="px-3 py-2 text-xs text-faint">{t('gateway.logs.filter.empty')}</p>
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
        </div>
      ) : null}
    </div>
  )
}
