import { EmptyState } from './EmptyState'

export interface DataColumn {
  key: string
  title: string
  className?: string
}

export function DataTable({
  columns,
  emptyText,
  selectable = true,
}: {
  columns: DataColumn[]
  emptyText: string
  selectable?: boolean
}) {
  return (
    <div className="overflow-hidden rounded-lg border border-border bg-surface">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="bg-surface-2 text-xs text-muted">
            <tr>
              {selectable ? (
                <th className="w-10 px-4 py-3">
                  <input type="checkbox" aria-label="select all" className="size-4 accent-[var(--primary)]" />
                </th>
              ) : null}
              {columns.map((column) => (
                <th key={column.key} className={`px-4 py-3 font-medium ${column.className ?? ''}`}>
                  {column.title}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            <tr>
              <td colSpan={columns.length + (selectable ? 1 : 0)}>
                <EmptyState text={emptyText} />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  )
}
