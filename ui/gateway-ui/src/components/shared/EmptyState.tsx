import { Inbox } from 'lucide-react'

export function EmptyState({ text }: { text: string }) {
  return (
    <div className="flex min-h-56 flex-col items-center justify-center gap-2 text-muted">
      <Inbox className="size-7 text-faint" />
      <span className="text-sm">{text}</span>
    </div>
  )
}
