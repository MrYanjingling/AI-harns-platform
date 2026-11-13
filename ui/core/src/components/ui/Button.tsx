import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { cn } from '../../lib/cn'
import { Spinner } from './Spinner'

type Variant = 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger' | 'dangerSoft'
type Size = 'sm' | 'md' | 'lg' | 'icon'

const variants: Record<Variant, string> = {
  primary: 'bg-primary text-primary-fg hover:bg-primary-hover active:translate-y-px',
  secondary:
    'bg-surface-2 text-fg border border-border hover:bg-surface-3 active:translate-y-px',
  outline:
    'border border-border-strong bg-surface text-fg hover:bg-surface-2 active:translate-y-px',
  ghost: 'text-muted hover:bg-surface-2 hover:text-fg',
  danger: 'bg-danger text-white hover:opacity-90 active:translate-y-px',
  dangerSoft:
    'border border-danger/25 bg-danger/5 text-danger hover:bg-danger/10 active:translate-y-px',
}

const sizes: Record<Size, string> = {
  sm: 'h-7 px-2.5 text-xs gap-1 rounded',
  md: 'h-8 px-3 text-sm gap-1.5 rounded',
  lg: 'h-9 px-4 text-sm gap-2 rounded',
  icon: 'size-8 rounded',
}

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  loading?: boolean
  icon?: ReactNode
}

export function Button({
  className,
  variant = 'secondary',
  size = 'md',
  loading = false,
  icon,
  children,
  disabled,
  type = 'button',
  ...props
}: ButtonProps) {
  return (
    <button
      type={type}
      disabled={disabled || loading}
      className={cn(
        'inline-flex shrink-0 items-center justify-center font-medium whitespace-nowrap transition-colors',
        'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
        'disabled:pointer-events-none disabled:opacity-50',
        variants[variant],
        sizes[size],
        className,
      )}
      {...props}
    >
      {loading ? <Spinner className="size-4" /> : icon}
      {children}
    </button>
  )
}
