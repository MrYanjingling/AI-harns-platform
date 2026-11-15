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
  cn,
} from '@harns/core'

const headCellClass =
  "relative border-0 border-b border-border px-4 py-4 text-sm font-medium text-fg after:absolute after:top-1/2 after:right-0 after:h-[1.6em] after:w-px after:-translate-y-1/2 after:bg-border after:content-[''] last:after:hidden"

const bodyCellClass =
  'border-0 border-b border-border px-4 py-4 align-middle text-xs leading-5 text-fg'

export function GatewayTable({ className, ...props }: HTMLAttributes<HTMLTableElement>) {
  return <Table className={cn('text-xs', className)} {...props} />
}

export function GatewayTHead({ className, ...props }: HTMLAttributes<HTMLTableSectionElement>) {
  return (
    <THead
      className={cn('sticky top-0 z-[3] bg-surface-2 text-sm font-medium text-fg', className)}
      {...props}
    />
  )
}

export function GatewayTH({ className, ...props }: ThHTMLAttributes<HTMLTableCellElement>) {
  return <TH className={cn(headCellClass, className)} {...props} />
}

export function GatewayTBody({ className, ...props }: HTMLAttributes<HTMLTableSectionElement>) {
  return <TBody className={className} {...props} />
}

export function GatewayTR({ className, ...props }: HTMLAttributes<HTMLTableRowElement>) {
  return <TR className={cn('even:bg-transparent hover:bg-surface-2', className)} {...props} />
}

export function GatewayTD({ className, ...props }: TdHTMLAttributes<HTMLTableCellElement>) {
  return <TD className={cn(bodyCellClass, className)} {...props} />
}
