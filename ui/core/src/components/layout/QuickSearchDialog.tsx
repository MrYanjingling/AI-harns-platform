import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CornerDownLeft, Search } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'
import { Dialog } from '../ui/Dialog'
import { Input } from '../ui/Field'
import type { NavEntry } from './nav'

export function QuickSearchDialog({
  open,
  onClose,
  entries,
}: {
  open: boolean
  onClose: () => void
  entries: NavEntry[]
}) {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    setQuery('')
    // 原生 dialog 打开后会把焦点交给关闭按钮,这里显式回到搜索框。
    const timer = window.setTimeout(() => {
      containerRef.current?.querySelector('input')?.focus()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [open])

  const results = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (!keyword) return entries
    return entries.filter((entry) =>
      [
        t(entry.labelKey),
        entry.hintKey ? t(entry.hintKey) : '',
        t(entry.sectionLabelKey),
        entry.groupLabelKey ? t(entry.groupLabelKey) : '',
      ]
        .join(' ')
        .toLowerCase()
        .includes(keyword),
    )
  }, [entries, query, t])

  const go = (to: string) => {
    onClose()
    setQuery('')
    navigate(to)
  }

  return (
    <Dialog open={open} onClose={onClose} size="md" title={t('layout.shell.quickSearch.title')}>
      <div ref={containerRef} className="-mt-2">
        <div className="relative">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-faint" />
          <Input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && results[0]) go(results[0].to)
            }}
            placeholder={t('layout.shell.quickSearch.placeholder')}
            aria-label={t('layout.shell.searchMenu')}
            className="pl-8"
          />
        </div>

        <div className="mt-2 max-h-80 overflow-y-auto">
          {results.length === 0 ? (
            <p className="px-2 py-8 text-center text-xs text-faint">
              {t('layout.shell.quickSearch.empty')}
            </p>
          ) : (
            results.map((entry) => (
              <button
                key={`${entry.sectionKey}-${entry.to}`}
                type="button"
                onClick={() => go(entry.to)}
                className={cn(
                  'flex w-full items-center justify-between gap-3 rounded px-2 py-2 text-left transition-colors',
                  'hover:bg-surface-2 focus-visible:bg-surface-2 focus-visible:outline-none',
                )}
              >
                <span className="min-w-0">
                  <span className="block text-sm text-fg">{t(entry.labelKey)}</span>
                  <span className="block truncate text-xs text-faint">
                    {t(entry.sectionLabelKey)}
                    {entry.hintKey ? ` · ${t(entry.hintKey)}` : ''}
                  </span>
                </span>
                <CornerDownLeft className="size-3.5 shrink-0 text-faint" />
              </button>
            ))
          )}
        </div>
      </div>
    </Dialog>
  )
}
