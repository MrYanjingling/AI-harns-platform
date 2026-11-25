import { Pause, Play, ScanSearch, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Badge, Button, RightDrawer, useI18n } from '@harns/core'
import {
  FrameAnalysisPanel,
  type FrameAnalysisHeaderState,
} from './FrameAnalysisPanel'

export function FrameAnalysisDrawer({
  open,
  collectionId,
  onClose,
}: {
  open: boolean
  collectionId: string
  onClose: () => void
}) {
  const { t } = useI18n()
  const [headerState, setHeaderState] = useState<FrameAnalysisHeaderState | null>(null)

  useEffect(() => {
    setHeaderState(null)
  }, [collectionId])

  return (
    <RightDrawer
      open={open}
      onClose={onClose}
      title={t('gateway.title.frameAnalysis')}
      resizeLabel={t('gateway.frames.resize')}
      icon={<ScanSearch className="size-4 text-primary" />}
      bodyClassName="p-3"
      headerActions={
        headerState ? (
          <>
            <Badge tone="neutral" className="border-[1.5px] px-2 py-1">
              {t('gateway.frames.items', { count: headerState.count })}
            </Badge>
            <Button
              variant="ghost"
              className="h-8"
              disabled={headerState.count === 0}
              icon={<Trash2 className="size-3.5" />}
              onClick={headerState.clear}
            >
              {t('gateway.frames.clear')}
            </Button>
            <Button
              variant="outline"
              className="h-8"
              disabled={!headerState.ready}
              icon={
                headerState.paused ? (
                  <Play className="size-3.5" />
                ) : (
                  <Pause className="size-3.5" />
                )
              }
              onClick={() => headerState.setPaused(!headerState.paused)}
            >
              {headerState.paused ? t('gateway.frames.resume') : t('gateway.frames.pause')}
            </Button>
          </>
        ) : null
      }
    >
      <FrameAnalysisPanel
        collectionId={collectionId}
        active={open}
        onHeaderStateChange={setHeaderState}
      />
    </RightDrawer>
  )
}
