import { Search } from 'lucide-react'
import type { InputHTMLAttributes } from 'react'
import { cn } from '../../lib/cn'
import { Input } from './Field'

export interface SearchInputProps extends InputHTMLAttributes<HTMLInputElement> {
  wrapperClassName?: string
  inputClassName?: string
}

export function SearchInput({
  wrapperClassName,
  inputClassName,
  className,
  ...props
}: SearchInputProps) {
  return (
    <div className={cn('relative', wrapperClassName)}>
      <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
      <Input
        className={cn('h-9 pl-9 text-sm', inputClassName, className)}
        {...props}
      />
    </div>
  )
}
