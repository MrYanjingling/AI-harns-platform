import { useRef } from 'react'
import { ImagePlus, Trash2, Upload } from 'lucide-react'
import { toast } from 'sonner'
import { useUploadImage } from '@/api/hooks'
import { useI18n } from '@/i18n'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Spinner } from '@/components/ui/Spinner'

const MAX_IMAGE_BYTES = 10 * 1024 * 1024
const ACCEPTED_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp', 'image/gif'])
const PUBLIC_BASE = import.meta.env.VITE_MINIO_PUBLIC_BASE

export function ImageUploadField({
  value,
  onChange,
}: {
  value: string
  onChange: (key: string) => void
}) {
  const { t } = useI18n()
  const inputRef = useRef<HTMLInputElement>(null)
  const upload = useUploadImage()
  const previewUrl = value && PUBLIC_BASE ? `${PUBLIC_BASE.replace(/\/$/, '')}/${value}` : null

  const pick = async (file: File | undefined) => {
    if (!file) return
    if (!ACCEPTED_TYPES.has(file.type)) {
      toast.error(t('image.invalidType'))
      return
    }
    if (file.size > MAX_IMAGE_BYTES) {
      toast.error(t('image.tooLarge'))
      return
    }
    try {
      const result = await upload.mutateAsync(file)
      onChange(result.key)
      toast.success(t('image.uploaded'))
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div className="flex flex-wrap items-start gap-4">
      <div className="flex size-24 shrink-0 items-center justify-center overflow-hidden rounded border border-border bg-surface-2">
        {previewUrl ? (
          <img src={previewUrl} alt={t('image.alt')} className="size-full object-cover" />
        ) : (
          <ImagePlus className="size-6 text-faint" />
        )}
      </div>

      <div className="min-w-0 flex-1 space-y-2">
        <p className="truncate font-mono text-[11px] text-muted" title={value}>
          {value || t('image.unset')}
        </p>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            icon={
              upload.isPending ? <Spinner className="size-3.5" /> : <Upload className="size-3.5" />
            }
            disabled={upload.isPending}
            onClick={() => inputRef.current?.click()}
          >
            {upload.isPending
              ? t('image.uploading')
              : value
                ? t('image.change')
                : t('image.upload')}
          </Button>
          {value ? (
            <Button
              size="sm"
              variant="ghost"
              icon={<Trash2 className="size-3.5" />}
              onClick={() => onChange('')}
            >
              {t('image.remove')}
            </Button>
          ) : null}
        </div>
        <p className="text-[11px] leading-relaxed text-faint">
          {t('image.hint')}
          {PUBLIC_BASE ? '' : ` ${t('image.noPublicBase')}`}
        </p>
      </div>

      <input
        ref={inputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp,image/gif"
        className="hidden"
        onChange={(event) => {
          void pick(event.target.files?.[0])
          event.target.value = ''
        }}
      />
    </div>
  )
}
