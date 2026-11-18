import { Play, RotateCw, Square } from 'lucide-react'
import { ActionMenu, useI18n } from '@harns/core'
import type { Collection } from '@/api/types'
import {
  canRestartCollection,
  canStartCollection,
  canStopCollection,
} from '@/lib/collection-actions'

export type CollectionOperation = 'start' | 'stop' | 'restart'

interface CollectionActionsMenuProps {
  collection: Collection
  onAction: (kind: CollectionOperation) => void
}

export function CollectionActionsMenu({
  collection,
  onAction,
}: CollectionActionsMenuProps) {
  const { t } = useI18n()

  return (
    <ActionMenu
      label={t('gateway.collection.action.more')}
      menuWidth={128}
      items={[
        {
          key: 'start',
          label: t('common.action.start'),
          icon: <Play />,
          disabled: !canStartCollection(collection),
          onSelect: () => onAction('start'),
        },
        {
          key: 'stop',
          label: t('common.action.stop'),
          icon: <Square />,
          disabled: !canStopCollection(collection),
          onSelect: () => onAction('stop'),
        },
        {
          key: 'restart',
          label: t('common.action.restart'),
          icon: <RotateCw />,
          disabled: !canRestartCollection(collection),
          onSelect: () => onAction('restart'),
        },
      ]}
    />
  )
}
