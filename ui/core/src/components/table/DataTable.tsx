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

export function DataTable(props: HTMLAttributes<HTMLTableElement>) {
  return <Table {...props} />
}

export function DataTableHead(props: HTMLAttributes<HTMLTableSectionElement>) {
  return <THead {...props} />
}

export function DataTableHeadCell(props: ThHTMLAttributes<HTMLTableCellElement>) {
  return <TH {...props} />
}

export function DataTableBody(props: HTMLAttributes<HTMLTableSectionElement>) {
  return <TBody {...props} />
}

export function DataTableRow(props: HTMLAttributes<HTMLTableRowElement>) {
  return <TR {...props} />
}

export function DataTableCell(props: TdHTMLAttributes<HTMLTableCellElement>) {
  return <TD {...props} />
}

export function TableActions({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('app-table-actions', className)} {...props} />
}
