import {
  forwardRef,
  type ButtonHTMLAttributes,
  type ReactNode,
} from 'react'
import { cn } from '../../lib/cn'
import { Tooltip } from '../ui/Tooltip'

export interface TableActionButtonProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  label: string
  icon: ReactNode
}

export const TableActionButton = forwardRef<HTMLButtonElement, TableActionButtonProps>(
  function TableActionButton({ label, icon, className, disabled, ...props }, forwardedRef) {
    return (
      <Tooltip content={label} disabled={disabled} className="shrink-0">
        <button
          ref={forwardedRef}
          type="button"
          disabled={disabled}
          aria-label={label}
          title={label}
          className={cn(
            'inline-flex size-[var(--table-action-size)] shrink-0 items-center justify-center rounded text-muted transition-colors',
            'hover:bg-surface-2 hover:text-fg disabled:pointer-events-none disabled:opacity-40',
            '[&_svg]:size-4',
            className,
          )}
          {...props}
        >
          {icon}
        </button>
      </Tooltip>
    )
  },
)
