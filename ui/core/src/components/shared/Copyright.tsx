import { cn } from '../../lib/cn'

const COPYRIGHT_TEXT = 'Copyright © 2026-2027 成都尤利卡科技有限公司版权'

export function Copyright({ className }: { className?: string }) {
  return <span className={cn(className)}>{COPYRIGHT_TEXT}</span>
}
