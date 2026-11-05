import { ChevronLeft, ChevronRight } from 'lucide-react'
import { PAGE_SIZES } from '@/lib/constants'
import { Button } from '@/components/ui/Button'
import { Select } from '@/components/ui/Field'

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
  const pageCount = Math.max(1, Math.ceil(total / pageSize))
  const safePage = Math.min(Math.max(1, page), pageCount)

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border px-5 py-3">
      <p className="text-xs text-muted">
        共 <span className="font-medium text-fg tabular">{total}</span> 条 · 第{' '}
        <span className="tabular">{safePage}</span> / <span className="tabular">{pageCount}</span>{' '}
        页
      </p>
      <div className="flex items-center gap-2">
        <Select
          aria-label="每页条数"
          className="h-8 w-24 text-xs"
          value={pageSize}
          onChange={(event) => onPageSizeChange(Number(event.target.value))}
          disabled={disabled}
        >
          {PAGE_SIZES.map((size) => (
            <option key={size} value={size}>
              {size} 条/页
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
          上一页
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => onPageChange(safePage + 1)}
          disabled={disabled || safePage >= pageCount}
        >
          下一页
          <ChevronRight className="size-3.5" />
        </Button>
      </div>
    </div>
  )
}
