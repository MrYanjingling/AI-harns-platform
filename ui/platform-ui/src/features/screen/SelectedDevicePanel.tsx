import { Link } from 'react-router-dom'
import { ArrowUpRight, MousePointerClick } from 'lucide-react'
import type { DeviceDetail } from '@/api/types'
import { PROTOCOLS } from '@/lib/constants'
import { formatRelative, formatNumber } from '@/lib/format'
import { DeviceTypeBadge } from '@/components/shared/DeviceTypeBadge'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { deviceAddress, formatVariableValue, readVariables } from './variables'

export function SelectedDevicePanel({ device }: { device: DeviceDetail | null }) {
  if (!device) {
    return (
      <aside className="flex min-h-64 flex-col items-center justify-center gap-3 rounded-2xl border border-border bg-surface/70 p-6 text-center backdrop-blur lg:h-full">
        <div className="flex size-11 items-center justify-center rounded-2xl border border-border bg-surface-2 text-faint">
          <MousePointerClick className="size-5" />
        </div>
        <div>
          <p className="text-sm font-medium">选择一台设备</p>
          <p className="mt-1 text-xs leading-relaxed text-muted">
            点击 3D 场景或左侧列表中的设备,查看实时采集变量。
          </p>
        </div>
      </aside>
    )
  }

  const variables = readVariables(device)
  const meta = PROTOCOLS[device.deviceType]

  return (
    <aside className="flex min-h-0 flex-col overflow-hidden rounded-2xl border border-border bg-surface/70 backdrop-blur lg:h-full">
      <header className="border-b border-border px-4 py-3">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-xs font-semibold tracking-wide">设备实时数据</h2>
          <StatusBadge status={device.collectStatus} />
        </div>
        <p className="mt-2 truncate text-base font-semibold tracking-tight" title={device.name}>
          {device.name}
        </p>
        <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
          <DeviceTypeBadge type={device.deviceType} model={device.deviceModel} />
          <span className="truncate font-mono text-[10px] text-faint">{device.deviceCode}</span>
        </div>
      </header>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-3">
        <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-2 text-[11px]">
          <dt className="text-faint">连接地址</dt>
          <dd className="truncate text-right font-mono text-fg" title={deviceAddress(device)}>
            {deviceAddress(device)}
          </dd>
          <dt className="text-faint">采集周期</dt>
          <dd className="text-right text-fg tabular">{device.collectorCycle} 秒</dd>
          <dt className="text-faint">变量数量</dt>
          <dd className="text-right text-fg tabular">{variables.length}</dd>
          <dt className="text-faint">最后更新</dt>
          <dd className="text-right text-fg">{formatRelative(device.modTime)}</dd>
          <dt className="text-faint">协议</dt>
          <dd className="truncate text-right text-fg">{meta?.description ?? device.deviceType}</dd>
        </dl>

        <div>
          <div className="mb-2 flex items-center justify-between">
            <h3 className="text-[10px] font-medium tracking-wider text-faint uppercase">
              采集变量
            </h3>
            <span className="text-[10px] text-faint tabular">{variables.length} 点</span>
          </div>

          {variables.length === 0 ? (
            <p className="rounded-lg border border-border bg-surface-2/50 px-3 py-4 text-center text-xs text-muted">
              该设备尚未配置变量
            </p>
          ) : (
            <ul className="space-y-1.5">
              {variables.map((variable) => {
                const numeric = typeof variable.value === 'number'
                return (
                  <li
                    key={variable.name}
                    className="rounded-lg border border-border bg-surface-2/40 px-2.5 py-2"
                  >
                    <div className="flex items-baseline justify-between gap-2">
                      <span className="min-w-0 flex-1 truncate text-[11px] text-muted" title={variable.name}>
                        {variable.name}
                      </span>
                      <span
                        className="shrink-0 text-sm font-semibold text-fg tabular"
                        title={formatVariableValue(variable.value, variable.dataType)}
                      >
                        {numeric
                          ? formatNumber(variable.value as number, 2)
                          : formatVariableValue(variable.value, variable.dataType)}
                      </span>
                    </div>
                    <div className="mt-1 flex items-center gap-1.5 text-[10px] text-faint">
                      <span className="rounded bg-surface-3 px-1 py-px font-mono">
                        {variable.dataType}
                      </span>
                      <span>{variable.accessMode === 'rw' ? '可读写' : '只读'}</span>
                    </div>
                  </li>
                )
              })}
            </ul>
          )}
        </div>
      </div>

      <footer className="border-t border-border p-3">
        <Link
          to={`/devices/${device.id}`}
          className="flex h-9 items-center justify-center gap-1.5 rounded-lg border border-border bg-surface text-xs font-medium transition-colors hover:bg-surface-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
        >
          查看设备详情
          <ArrowUpRight className="size-3.5" />
        </Link>
      </footer>
    </aside>
  )
}
