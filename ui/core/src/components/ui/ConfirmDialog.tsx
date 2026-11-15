import type { ReactNode } from 'react'
import { CircleAlert } from 'lucide-react'
import { useI18n } from '../../i18n'
import { cn } from '../../lib/cn'
import { Button } from './Button'
import { Dialog } from './Dialog'

export interface ConfirmDialogProps {
  open: boolean
  title: ReactNode
  description?: ReactNode
  confirmText?: string
  cancelText?: string
  danger?: boolean
  loading?: boolean
  onConfirm: () => void
  onClose: () => void
}

export function ConfirmDialog({
  open,
  description,
  confirmText,
  cancelText,
  danger = false,
  loading = false,
  onConfirm,
  onClose,
}: ConfirmDialogProps) {
  const { t } = useI18n()
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={
        <span className="flex items-center gap-2 text-lg font-medium text-fg">
          <CircleAlert
            className={cn('size-7 shrink-0', danger ? 'text-danger' : 'text-warning')}
          />
          {t('components.confirm.title')}
        </span>
      }
      size="sm"
      className="rounded-lg"
      headerClassName="items-center px-7 pt-4 pb-0"
      bodyClassName="px-7 pt-2 pb-4"
      footerClassName="gap-3 px-7 pt-0 pb-5"
      footer={
        <>
          <Button
            variant="outline"
            size="sm"
            className="h-7 rounded-lg px-2.5 text-sm"
            onClick={onClose}
            disabled={loading}
          >
            {cancelText ?? t('components.confirm.cancel')}
          </Button>
          <Button
            variant={danger ? 'danger' : 'warning'}
            size="sm"
            className="h-7 rounded-lg px-2.5 text-sm"
            onClick={onConfirm}
            loading={loading}
          >
            {confirmText ?? t('components.confirm.confirm')}
          </Button>
        </>
      }
    >
      {description ? <p className="text-[13px] leading-relaxed text-muted">{description}</p> : null}
    </Dialog>
  )
}
