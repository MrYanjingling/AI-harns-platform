import { useRef } from 'react'
import { ImagePlus, Trash2, Upload } from 'lucide-react'
import { toast } from 'sonner'
import { useUploadImage } from '@/api/hooks'
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
  const inputRef = useRef<HTMLInputElement>(null)
  const upload = useUploadImage()
  const previewUrl = value && PUBLIC_BASE ? `${PUBLIC_BASE.replace(/\/$/, '')}/${value}` : null

  const pick = async (file: File | undefined) => {
    if (!file) return
    if (!ACCEPTED_TYPES.has(file.type)) {
      toast.error('仅支持 JPEG、PNG、WebP、GIF 图片')
      return
    }
    if (file.size > MAX_IMAGE_BYTES) {
      toast.error('图片不能超过 10 MiB')
      return
    }
    try {
      const result = await upload.mutateAsync(file)
      onChange(result.key)
      toast.success('图片已上传')
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div className="flex flex-wrap items-start gap-4">
      <div className="flex size-24 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-border bg-surface-2">
        {previewUrl ? (
          <img src={previewUrl} alt="设备图片" className="size-full object-cover" />
        ) : (
          <ImagePlus className="size-6 text-faint" />
        )}
      </div>

      <div className="min-w-0 flex-1 space-y-2">
        <p className="truncate font-mono text-[11px] text-muted" title={value}>
          {value || '未设置图片'}
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
            {upload.isPending ? '上传中…' : value ? '更换图片' : '上传图片'}
          </Button>
          {value ? (
            <Button
              size="sm"
              variant="ghost"
              icon={<Trash2 className="size-3.5" />}
              onClick={() => onChange('')}
            >
              移除
            </Button>
          ) : null}
        </div>
        <p className="text-[11px] leading-relaxed text-faint">
          支持 JPEG / PNG / WebP / GIF,单张不超过 10 MiB。
          {PUBLIC_BASE ? '' : ' 未配置 VITE_MINIO_PUBLIC_BASE 时仅显示对象 Key。'}
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
