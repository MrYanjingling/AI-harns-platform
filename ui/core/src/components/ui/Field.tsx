import type {
  InputHTMLAttributes,
  LabelHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '../../lib/cn'

const controlBase =
  'w-full rounded border border-border-strong bg-surface px-2.5 text-sm text-fg placeholder:text-faint ' +
  'transition-colors hover:border-faint focus:border-primary focus:outline-none ' +
  'focus:ring-2 focus:ring-primary/15 disabled:cursor-not-allowed disabled:opacity-60'

export function Label({ className, ...props }: LabelHTMLAttributes<HTMLLabelElement>) {
  return (
    <label className={cn('block text-xs font-medium text-muted', className)} {...props} />
  )
}

export function Input({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cn(controlBase, 'h-8', className)} {...props} />
}

export function Textarea({ className, ...props }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cn(controlBase, 'min-h-20 py-2', className)} {...props} />
}

export function Select({ className, children, ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <div className="relative">
      <select
        className={cn(controlBase, 'h-8 cursor-pointer appearance-none pr-8', className)}
        {...props}
      >
        {children}
      </select>
      <ChevronDown className="pointer-events-none absolute top-1/2 right-2.5 size-3.5 -translate-y-1/2 text-faint" />
    </div>
  )
}

export interface FieldProps {
  label?: ReactNode
  htmlFor?: string
  labelClassName?: string
  labelWidthClassName?: string
  labelAccessory?: ReactNode
  hint?: ReactNode
  hintClassName?: string
  error?: string
  required?: boolean
  inline?: boolean
  className?: string
  children: ReactNode
}

export function Field({
  label,
  htmlFor,
  labelClassName,
  labelWidthClassName,
  labelAccessory,
  hint,
  hintClassName,
  error,
  required,
  inline,
  className,
  children,
}: FieldProps) {
  if (inline) {
    return (
      <div
        className={cn(
          'grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-1 gap-y-1',
          className,
        )}
      >
        {label ? (
          <div className={cn('flex items-center gap-0.5 whitespace-nowrap', labelWidthClassName)}>
            <Label htmlFor={htmlFor} className={labelClassName}>
              {label}
              {required ? (
                <span className="ml-0.5 align-super text-[10px] leading-none text-danger">
                  *
                </span>
              ) : null}
            </Label>
            {labelAccessory}
          </div>
        ) : (
          <span />
        )}
        <div className="min-w-0">{children}</div>
        {error ? (
          <p className="col-start-2 text-xs text-danger">{error}</p>
        ) : hint ? (
          <p className={cn('col-start-2 text-xs text-faint', hintClassName)}>{hint}</p>
        ) : null}
      </div>
    )
  }

  return (
    <div className={cn('space-y-1.5', className)}>
      {label ? (
        <div className="flex items-center gap-1">
          <Label htmlFor={htmlFor} className={labelClassName}>
            {label}
            {required ? (
              <span className="ml-0.5 align-super text-[10px] leading-none text-danger">*</span>
            ) : null}
          </Label>
          {labelAccessory}
        </div>
      ) : null}
      {children}
      {error ? (
        <p className="text-xs text-danger">{error}</p>
      ) : hint ? (
        <p className="text-xs text-faint">{hint}</p>
      ) : null}
    </div>
  )
}
