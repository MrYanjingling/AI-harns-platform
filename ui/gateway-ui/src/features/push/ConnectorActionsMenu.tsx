import { Play, Plus, RotateCw, Square } from 'lucide-react'
import { ActionMenu, useI18n } from '@harns/core'
import type { PushConnector } from '@/api/types'

export type ConnectorOperation = 'start' | 'stop' | 'restart'

interface ConnectorActionsMenuProps {
  connector: PushConnector
  disabled?: boolean
  onAction: (kind: ConnectorOperation) => void
  onCreateGroup: () => void
}

export function ConnectorActionsMenu({
  connector,
  disabled = false,
  onAction,
  onCreateGroup,
}: ConnectorActionsMenuProps) {
  const { t } = useI18n()

  return (
    <ActionMenu
      label={t('gateway.push.action.more')}
      disabled={disabled}
      items={[
        {
          key: 'create-group',
          label: t('gateway.push.action.createGroup'),
          icon: <Plus />,
          onSelect: onCreateGroup,
        },
        {
          key: 'start',
          label: t('common.action.start'),
          icon: <Play />,
          disabled: connector.runState !== 'stopped',
          onSelect: () => onAction('start'),
        },
        {
          key: 'stop',
          label: t('common.action.stop'),
          icon: <Square />,
          disabled: connector.status === 'stopped',
          onSelect: () => onAction('stop'),
        },
        {
          key: 'restart',
          label: t('common.action.restart'),
          icon: <RotateCw />,
          disabled: connector.runState !== 'running',
          onSelect: () => onAction('restart'),
        },
      ]}
    />
  )
}
