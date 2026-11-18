import { Play, Radio, RotateCw, Square } from 'lucide-react'
import { ActionMenu, useI18n } from '@harns/core'
import type { PushGroup } from '@/api/types'

interface PushGroupActionsMenuProps {
  group: PushGroup
  connectorStatus?: string
  disabled?: boolean
  onStart: () => void
  onStop: () => void
  onRestart: () => void
  onViewPayloads: () => void
}

export function PushGroupActionsMenu({
  group,
  connectorStatus,
  disabled = false,
  onStart,
  onStop,
  onRestart,
  onViewPayloads,
}: PushGroupActionsMenuProps) {
  const { t } = useI18n()
  const canStart = group.runState === 'stopped'
  const canStop = group.status !== 'stopped'
  const canRestart = group.runState === 'running'
  const canViewPayloads = group.status === 'running' && connectorStatus === 'running'

  return (
    <ActionMenu
      label={t('gateway.push.action.more')}
      disabled={disabled}
      items={[
        {
          key: 'payloads',
          label: t('gateway.push.payload.view'),
          icon: <Radio />,
          disabled: !canViewPayloads,
          onSelect: onViewPayloads,
        },
        {
          key: 'start',
          label: t('common.action.start'),
          icon: <Play />,
          disabled: !canStart,
          onSelect: onStart,
        },
        {
          key: 'stop',
          label: t('common.action.stop'),
          icon: <Square />,
          disabled: !canStop,
          onSelect: onStop,
        },
        {
          key: 'restart',
          label: t('common.action.restart'),
          icon: <RotateCw />,
          disabled: !canRestart,
          onSelect: onRestart,
        },
      ]}
    />
  )
}
