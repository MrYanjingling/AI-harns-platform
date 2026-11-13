import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useI18n } from '../../i18n'
import { PAGE_SIZES } from '../../lib/constants'
import { Button } from '../ui/Button'
import { Select } from '../ui/Field'

export function Pagination({
  page,
  pageSize,
  total,
  onPageChange,
  onPageSizeChange,
  disabled = false,
}: {
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
  disabled?: boolean
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
        <Select
          aria-label={t('components.pagination.pageSizeAria')}
          className="h-8 w-24 text-xs"
          value={pageSize}
          onChange={(event) => onPageSizeChange(Number(event.target.value))}
          disabled={disabled}
        >
          {PAGE_SIZES.map((size) => (
            <option key={size} value={size}>
              {t('common.pagination.pageSize', { size })}
            </option>
          ))}
        </Select>
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
      </div>
    </div>
  )
}
