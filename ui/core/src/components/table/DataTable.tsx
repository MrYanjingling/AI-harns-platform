import type {
  HTMLAttributes,
  TdHTMLAttributes,
  ThHTMLAttributes,
} from 'react'
import {
  Table,
  TBody,
  TD,
  TH,
  THead,
  TR,
} from '../ui/Table'
import { cn } from '../../lib/cn'

const headCellClass =
  "relative border-0 border-b border-border px-[var(--table-cell-padding-x)] py-4 text-[length:var(--table-head-font-size)] font-medium text-fg after:absolute after:top-1/2 after:right-0 after:h-[1.6em] after:w-px after:-translate-y-1/2 after:bg-border after:content-[''] last:after:hidden"

const bodyCellClass =
  'h-[var(--table-row-height)] border-0 border-b border-border px-[var(--table-cell-padding-x)] py-2 align-middle text-[length:var(--table-data-font-size)] leading-5 font-normal font-table text-table-fg'

export function DataTable({ className, ...props }: HTMLAttributes<HTMLTableElement>) {
  return <Table className={cn('text-sm', className)} {...props} />
}

export function DataTableHead({
  className,
  ...props
}: HTMLAttributes<HTMLTableSectionElement>) {
  return (
    <THead
      className={cn(
        'sticky top-0 z-[3] bg-surface-2 text-[length:var(--table-head-font-size)] font-medium text-fg',
        className,
      )}
      {...props}
    />
  )
}

export function DataTableHeadCell({
  className,
  ...props
}: ThHTMLAttributes<HTMLTableCellElement>) {
  return <TH className={cn(headCellClass, className)} {...props} />
}

export function DataTableBody({
  className,
  ...props
}: HTMLAttributes<HTMLTableSectionElement>) {
  return <TBody className={className} {...props} />
}

export function DataTableRow({
  className,
  ...props
}: HTMLAttributes<HTMLTableRowElement>) {
  return <TR className={cn('even:bg-transparent hover:bg-surface-2', className)} {...props} />
}

export function DataTableCell({
  className,
  ...props
}: TdHTMLAttributes<HTMLTableCellElement>) {
  return <TD className={cn(bodyCellClass, className)} {...props} />
}

export function TableActions({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('app-table-actions', className)} {...props} />
}
