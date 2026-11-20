import { Sparkles } from 'lucide-react'
import { useI18n } from '@harns/core'

export function AIAssistant() {
  const { t } = useI18n()
  const openAssistant = () => {
    window.open('/ai/assistant', '_blank', 'noopener,noreferrer')
  }

  return (
    <button
      type="button"
      onClick={openAssistant}
      aria-label={t('gateway.ai.assistant.title')}
      title={t('gateway.ai.assistant.title')}
      className="flex size-8 items-center justify-center rounded text-muted transition-colors hover:bg-surface-2 hover:text-fg"
    >
      <Sparkles className="size-4" />
    </button>
  )
}
