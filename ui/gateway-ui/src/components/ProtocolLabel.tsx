import { useI18n } from '@harns/core'
import { deviceModelMeta, deviceTypeMeta } from '@/lib/constants'

/** 协议与型号:技术名称按原样展示,描述走字典。 */
export function ProtocolLabel({ type, model }: { type: string; model: string }) {
  const { t } = useI18n()
  const typeMeta = deviceTypeMeta(type)
  const modelMeta = deviceModelMeta(type, model)
  return (
    <div className="min-w-0">
      <p className="truncate text-sm text-fg">{typeMeta?.label ?? (type || '—')}</p>
      <p
        className="truncate text-xs text-faint"
        title={modelMeta ? t(modelMeta.descriptionKey) : model}
      >
        {modelMeta?.label ?? (model || '—')}
      </p>
    </div>
  )
}
