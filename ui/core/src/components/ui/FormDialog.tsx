import { cn } from '../../lib/cn'
import { Dialog, type DialogProps } from './Dialog'

export interface FormDialogProps
  extends Omit<DialogProps, 'size' | 'className' | 'bodyClassName'> {
  width?: 'default' | 'wide'
  className?: string
  bodyClassName?: string
}

export function FormDialog({
  width = 'default',
  className,
  bodyClassName,
  ...props
}: FormDialogProps) {
  return (
    <Dialog
      size="xl"
      className={cn(
        'rounded-lg',
        width === 'wide'
          ? 'max-w-[var(--dialog-wide-width)]'
          : 'max-w-[var(--dialog-form-width)]',
        className,
      )}
      bodyClassName={cn('app-dialog-body-scroll space-y-3 p-4', bodyClassName)}
      {...props}
    />
  )
}
