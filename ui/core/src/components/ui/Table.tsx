import type { HTMLAttributes, TdHTMLAttributes, ThHTMLAttributes } from 'react'
import { cn } from '../../lib/cn'

export function Table({ className, ...props }: HTMLAttributes<HTMLTableElement>) {
  return (
    <div className="w-full overflow-x-auto">
      <table className={cn('w-full border-collapse text-sm', className)} {...props} />
    </div>
  )
}

export function THead({ className, ...props }: HTMLAttributes<HTMLTableSectionElement>) {
  return (
    <thead
      className={cn(
        'sticky top-0 z-[3] bg-surface-2 text-left text-[length:var(--table-head-font-size)] font-semibold text-fg',
        className,
      )}
      {...props}
    />
  )
}

export function TBody({ className, ...props }: HTMLAttributes<HTMLTableSectionElement>) {
  return <tbody className={className} {...props} />
}

export function TR({ className, ...props }: HTMLAttributes<HTMLTableRowElement>) {
  return (
    <tr
      className={cn('transition-colors hover:bg-surface-2', className)}
      {...props}
    />
  )
}

export function TH({ className, ...props }: ThHTMLAttributes<HTMLTableCellElement>) {
  return (
    <th
      className={cn(
        "relative border-0 border-b border-border-strong px-[var(--table-cell-padding-x)] py-4 text-[length:var(--table-head-font-size)] font-semibold whitespace-nowrap text-fg after:absolute after:top-1/2 after:right-0 after:h-[1.6em] after:w-px after:-translate-y-1/2 after:bg-border after:content-[''] last:after:hidden",
        className,
      )}
      {...props}
    />
  )
}

export function TD({ className, ...props }: TdHTMLAttributes<HTMLTableCellElement>) {
  return (
    <td
      className={cn(
        'h-[var(--table-row-height)] border-0 border-b border-border px-[var(--table-cell-padding-x)] py-2 align-middle text-[length:var(--table-data-font-size)] leading-5 font-normal font-table text-table-fg',
        className,
      )}
      {...props}
    />
  )
}
