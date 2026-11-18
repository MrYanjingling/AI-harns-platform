import { cn } from '../../lib/cn'

export function Copyright({
  brand = 'AI-harns',
  year = new Date().getFullYear(),
  className,
}: {
  brand?: string
  year?: number
  className?: string
}) {
  return <span className={cn(className)}>© {year} {brand}</span>
}
