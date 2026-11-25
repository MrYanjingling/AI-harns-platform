import { Sparkles } from 'lucide-react'
import { RightDrawer, useI18n } from '@harns/core'
import { AIAssistantChat } from './AIAssistantChat'

export function AIAssistantDrawer({
  open,
  onClose,
}: {
  open: boolean
  onClose: () => void
}) {
  const { t } = useI18n()
  return (
    <RightDrawer
      open={open}
      onClose={onClose}
      title={t('gateway.ai.assistant.title')}
      resizeLabel={t('gateway.ai.assistant.resize')}
      icon={<Sparkles className="size-4 text-primary" />}
      bodyClassName="p-3"
    >
      <AIAssistantChat active={open} className="h-full" />
    </RightDrawer>
  )
}
