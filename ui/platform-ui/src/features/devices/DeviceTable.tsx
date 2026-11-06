import { useEffect, useRef } from 'react'
import { Play, Square } from 'lucide-react'
import type { DeviceDetail, DeviceStatusAction } from '@/api/types'
import { useI18n } from '@/i18n'
import { PROTOCOLS } from '@/lib/constants'
import { formatRelative } from '@/lib/format'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'

function isRunning(device: DeviceDetail): boolean {
  return device.collectStatus === 'collecting' || device.collectStatus === 'collectingError'
}

function SelectAllCheckbox({
  checked,
  indeterminate,
  onChange,
}: {
  checked: boolean
  indeterminate: boolean
  onChange: () => void
}) {
  const ref = useRef<HTMLInputElement>(null)
  const { t } = useI18n()
  useEffect(() => {
    if (ref.current) ref.current.indeterminate = indeterminate
  }, [indeterminate])
  return (
    <input
      ref={ref}
      type="checkbox"
      aria-label={t('devices.table.selectAll')}
      checked={checked}
      onChange={onChange}
      className="size-3.5 cursor-pointer accent-[var(--primary)]"
    />
  )
}

export function DeviceTable({
  rows,
  page,
  pageSize,
  selected,
  allSelected,
  busy = false,
  onToggle,
  onToggleAll,
  onView,
  onEdit,
  onDelete,
  onStatusAction,
}: {
  rows: DeviceDetail[]
  page: number
  pageSize: number
  selected: Set<string>
  allSelected: boolean
  busy?: boolean
  onToggle: (id: string) => void
  onToggleAll: () => void
  onView: (device: DeviceDetail) => void
  onEdit: (device: DeviceDetail) => void
  onDelete: (device: DeviceDetail) => void
  onStatusAction: (device: DeviceDetail, action: DeviceStatusAction) => void
}) {
  const { t } = useI18n()
  const partiallySelected = rows.some((device) => selected.has(device.id))

  return (
    <div className="min-w-[1180px]">
      <Table>
        <THead>
          <tr>
            <TH className="w-8">
              <SelectAllCheckbox
                checked={allSelected}
                indeterminate={!allSelected && partiallySelected}
                onChange={onToggleAll}
              />
            </TH>
            <TH className="w-12">{t('devices.table.index')}</TH>
            <TH className="min-w-44">{t('devices.table.deviceCode')}</TH>
            <TH className="min-w-36">{t('devices.table.deviceName')}</TH>
            <TH className="w-20">{t('devices.table.protocol')}</TH>
            <TH className="w-32">{t('devices.table.model')}</TH>
            <TH className="min-w-44">{t('devices.table.address')}</TH>
            <TH className="w-16">{t('devices.table.variables')}</TH>
            <TH className="w-24">{t('devices.table.status')}</TH>
            <TH className="w-24">{t('devices.table.updatedAt')}</TH>
            <TH className="min-w-32">{t('devices.table.remark')}</TH>
            <TH className="w-52">{t('common.field.actions')}</TH>
          </tr>
        </THead>
        <TBody>
          {rows.map((device, index) => {
            const running = isRunning(device)
            return (
              <TR key={device.id}>
                <TD>
                  <input
                    type="checkbox"
                    aria-label={t('devices.table.selectDevice', { name: device.name })}
                    checked={selected.has(device.id)}
                    onChange={() => onToggle(device.id)}
                    className="size-3.5 cursor-pointer accent-[var(--primary)]"
                  />
                </TD>
                <TD className="text-xs text-muted tabular">
                  {(page - 1) * pageSize + index + 1}
                </TD>
                <TD>
                  <button
                    type="button"
                    onClick={() => onView(device)}
                    title={device.deviceCode}
                    className="block max-w-56 truncate text-left font-mono text-xs text-primary hover:underline"
                  >
                    {device.deviceCode}
                  </button>
                </TD>
                <TD className="max-w-48 truncate" title={device.name}>
                  {device.name}
                </TD>
                <TD className="text-xs text-muted">
                  {PROTOCOLS[device.deviceType]?.label ?? device.deviceType}
                </TD>
                <TD className="text-xs text-muted" title={device.deviceModel}>
                  {device.deviceModel}
                </TD>
                <TD
                  className="max-w-52 truncate font-mono text-xs text-muted"
                  title={device.address?.location}
                >
                  {device.address?.location || '—'}
                </TD>
                <TD className="text-xs text-muted tabular">{device.variables.length}</TD>
                <TD>
                  <StatusBadge status={device.collectStatus} />
                </TD>
                <TD className="text-xs whitespace-nowrap text-muted">
                  {formatRelative(device.modTime)}
                </TD>
                <TD
                  className="max-w-40 truncate text-xs text-muted"
                  title={device.topic || t('devices.defaultTopic')}
                >
                  {device.topic || t('devices.defaultTopic')}
                </TD>
                <TD className="whitespace-nowrap">
                  <div className="flex flex-nowrap items-center gap-2">
                    <button
                      type="button"
                      onClick={() => onView(device)}
                      disabled={busy}
                      className="text-xs text-primary hover:underline disabled:pointer-events-none disabled:opacity-50"
                    >
                      {t('common.action.view')}
                    </button>
                    <button
                      type="button"
                      onClick={() => onEdit(device)}
                      disabled={busy}
                      className="text-xs text-primary hover:underline disabled:pointer-events-none disabled:opacity-50"
                    >
                      {t('common.action.edit')}
                    </button>
                    <button
                      type="button"
                      onClick={() => onStatusAction(device, running ? 'stop' : 'start')}
                      disabled={busy}
                      className="inline-flex items-center gap-0.5 text-xs text-primary hover:underline disabled:pointer-events-none disabled:opacity-50"
                    >
                      {running ? <Square className="size-3" /> : <Play className="size-3" />}
                      {running ? t('common.action.stop') : t('common.action.start')}
                    </button>
                    <button
                      type="button"
                      onClick={() => onDelete(device)}
                      disabled={busy}
                      className="text-xs text-danger hover:underline disabled:pointer-events-none disabled:opacity-50"
                    >
                      {t('common.action.delete')}
                    </button>
                  </div>
                </TD>
              </TR>
            )
          })}
        </TBody>
      </Table>
    </div>
  )
}
