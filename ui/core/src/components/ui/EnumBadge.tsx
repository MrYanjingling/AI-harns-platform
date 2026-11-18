import { cn } from '../../lib/cn'
import { Badge, type BadgeProps } from './Badge'

export type EnumBadgeProps = BadgeProps

export function EnumBadge({ className, children, ...props }: EnumBadgeProps) {
  return (
    <Badge className={cn('enum-badge', className)} {...props}>
      {children}
    </Badge>
  )
}
