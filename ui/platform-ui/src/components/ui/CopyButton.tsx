import { useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { toast } from 'sonner'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/cn'

export function CopyButton({
  value,
  label,
  className,
}: {
  value: string
  label?: string
  className?: string
}) {
  const { t } = useI18n()
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      toast.success(label ?? t('components.copy.success'))
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      toast.error(t('components.copy.failed'))
    }
  }

  return (
    <button
      type="button"
      onClick={copy}
      aria-label={t('components.copy.aria')}
      className={cn(
        'inline-flex size-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-surface-2 hover:text-fg',
        'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
        className,
      )}
    >
      {copied ? <Check className="size-3.5 text-primary" /> : <Copy className="size-3.5" />}
    </button>
  )
}
