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
  disabledReason?: string
}

export const TableActionButton = forwardRef<HTMLButtonElement, TableActionButtonProps>(
  function TableActionButton(
    { label, icon, disabledReason, className, disabled, ...props },
    forwardedRef,
  ) {
    const tooltip = disabled && disabledReason ? disabledReason : label

    return (
      <Tooltip
        content={tooltip}
        disabled={Boolean(disabled && !disabledReason)}
        focusable={Boolean(disabled && disabledReason)}
        ariaLabel={disabled && disabledReason ? disabledReason : undefined}
        className="shrink-0"
      >
        <button
          ref={forwardedRef}
          type="button"
          disabled={disabled}
          aria-label={label}
          className={cn('app-table-action-button', className)}
          {...props}
        >
          {icon}
        </button>
      </Tooltip>
    )
  },
)
