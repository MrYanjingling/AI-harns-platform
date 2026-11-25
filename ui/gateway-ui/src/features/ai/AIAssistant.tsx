import { Sparkles } from 'lucide-react'
import { useI18n } from '@harns/core'

export function AIAssistant({ onOpen }: { onOpen: () => void }) {
  const { t } = useI18n()
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label={t('gateway.ai.assistant.title')}
      title={t('gateway.ai.assistant.title')}
      className="flex size-8 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
    >
      <Sparkles className="size-4 text-primary" />
    </button>
  )
}
