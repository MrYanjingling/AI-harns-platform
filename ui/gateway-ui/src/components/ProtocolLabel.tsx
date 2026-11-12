import { driverModelLabel, driverTypeLabel } from '@/lib/constants'

/** 协议与型号:技术名称按原样展示,不做翻译。 */
export function ProtocolLabel({ type, model }: { type: string; model: string }) {
  return (
    <div className="min-w-0">
      <p className="truncate text-[13px] text-fg">{driverTypeLabel(type) || '—'}</p>
      <p className="truncate text-[11px] text-faint" title={model}>
        {driverModelLabel(model) || '—'}
      </p>
    </div>
  )
}
