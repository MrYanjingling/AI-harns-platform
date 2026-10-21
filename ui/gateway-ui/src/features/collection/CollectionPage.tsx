import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Eye,
  Pencil,
  Play,
  Plus,
  RefreshCw,
  RotateCw,
  Search,
  Square,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Button,
  ConfirmDialog,
  CopyButton,
  EmptyState,
  ErrorState,
  Input,
  PageHeader,
  Select,
  StatCard,
  Table,
  TableSkeleton,
  TBody,
  TD,
  TH,
  THead,
  TR,
  errorMessage,
  formatRelative,
  useI18n,
} from '@harns/core'
import { useDeleteDevice, useDevices, useGatewayMeta, useSwitchDeviceStatus } from '@/api/hooks'
import type { DeviceOperator, GatewayDeviceDetail } from '@/api/types'
import { DeviceStatusBadge } from '@/components/DeviceStatusBadge'
import { ProtocolLabel } from '@/components/ProtocolLabel'
import {
  COLLECT_STATUS,
  DEVICE_TYPES,
  defaultTopic,
  isAbnormalStatus,
} from '@/lib/constants'

type PendingAction = { kind: 'delete' | DeviceOperator; device: GatewayDeviceDetail }

export function CollectionPage() {
  const { t } = useI18n()
  const devices = useDevices()
  const meta = useGatewayMeta()
  const deleteDevice = useDeleteDevice()
  const switchStatus = useSwitchDeviceStatus()
  const [keyword, setKeyword] = useState('')
  const [typeFilter, setTypeFilter] = useState('all')
  const [statusFilter, setStatusFilter] = useState('all')
  const [pending, setPending] = useState<PendingAction | null>(null)

  const list = devices.data ?? []

  const stats = useMemo(() => {
    const collecting = list.filter((item) => item.collectStatus === 'collecting').length
    const abnormal = list.filter((item) => isAbnormalStatus(item.collectStatus)).length
    const stopped = list.filter((item) => item.collectStatus === 'stopped').length
    const variables = list.reduce((sum, item) => sum + (item.variables?.length ?? 0), 0)
    return { total: list.length, collecting, abnormal, stopped, variables }
  }, [list])

  const filtered = useMemo(() => {
    const query = keyword.trim().toLowerCase()
    return list.filter((item) => {
      if (typeFilter !== 'all' && item.deviceType !== typeFilter) return false
      if (statusFilter !== 'all' && item.collectStatus !== statusFilter) return false
      if (!query) return true
      const endpoint = [item.address?.location, item.address?.option?.port]
        .filter((value) => value !== undefined && value !== '')
        .join(':')
      return [item.name, item.deviceCode, item.id, item.topic, endpoint]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
        .includes(query)
    })
  }, [list, keyword, typeFilter, statusFilter])

  const refresh = () => {
    void devices.refetch()
    void meta.refetch()
  }

  const confirm = async () => {
    if (!pending) return
    const { kind, device } = pending
    try {
      if (kind === 'delete') {
        await deleteDevice.mutateAsync({ id: device.id, eTag: device.eTag, name: device.name })
        toast.success(t('gateway.collection.toast.deleted'))
      } else {
        await switchStatus.mutateAsync({ id: device.id, name: device.name, status: kind })
        toast.success(
          kind === 'start'
            ? t('gateway.collection.toast.started')
            : kind === 'stop'
              ? t('gateway.collection.toast.stopped')
              : t('gateway.collection.toast.restarted'),
        )
      }
      setPending(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const pendingLoading = deleteDevice.isPending || switchStatus.isPending

  return (
    <div className="space-y-4">
      <PageHeader
        title={t('gateway.collection.title')}
        description={t('gateway.collection.description')}
        actions={
          <>
            <Button
              variant="outline"
              loading={devices.isFetching}
              onClick={refresh}
              icon={<RefreshCw className="size-4" />}
            >
              {t('common.action.refresh')}
            </Button>
            <Link
              to="/gateway/collection/new"
              className="inline-flex h-8 items-center gap-1.5 rounded bg-primary px-3 text-sm font-medium text-primary-fg transition-colors hover:bg-primary-hover"
            >
              <Plus className="size-4" />
              {t('gateway.collection.create')}
            </Link>
          </>
        }
      />

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatCard
          label={t('gateway.collection.stat.total')}
          value={stats.total}
          hint={t('gateway.collection.variableCount', { count: stats.variables })}
        />
        <StatCard label={t('gateway.collection.stat.collecting')} value={stats.collecting} />
        <StatCard label={t('gateway.collection.stat.abnormal')} value={stats.abnormal} />
        <StatCard label={t('gateway.collection.stat.stopped')} value={stats.stopped} />
      </div>

      <div className="overflow-hidden rounded border border-border bg-surface">
        <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
          <div className="relative min-w-56 flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-faint" />
            <Input
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              placeholder={t('gateway.collection.searchPlaceholder')}
              aria-label={t('common.action.search')}
              className="pl-8"
            />
          </div>
          <Select
            value={typeFilter}
            onChange={(event) => setTypeFilter(event.target.value)}
            aria-label={t('gateway.collection.filter.type')}
            className="w-40"
          >
            <option value="all">{t('gateway.collection.filter.allTypes')}</option>
            {DEVICE_TYPES.map((item) => (
              <option key={item.value} value={item.value}>
                {item.label}
              </option>
            ))}
          </Select>
          <Select
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value)}
            aria-label={t('gateway.collection.filter.status')}
            className="w-40"
          >
            <option value="all">{t('gateway.collection.filter.allStatuses')}</option>
            {Object.entries(COLLECT_STATUS).map(([value, item]) => (
              <option key={value} value={value}>
                {t(item.labelKey)}
              </option>
            ))}
          </Select>
        </div>

        {devices.isLoading ? (
          <TableSkeleton rows={6} cols={6} />
        ) : devices.isError ? (
          <div className="p-3">
            <ErrorState error={devices.error} onRetry={() => void devices.refetch()} />
          </div>
        ) : list.length === 0 ? (
          <EmptyState
            title={t('gateway.collection.empty.title')}
            description={t('gateway.collection.empty.description')}
            action={
              <Link
                to="/gateway/collection/new"
                className="inline-flex h-8 items-center gap-1.5 rounded bg-primary px-3 text-sm font-medium text-primary-fg transition-colors hover:bg-primary-hover"
              >
                <Plus className="size-4" />
                {t('gateway.collection.create')}
              </Link>
            }
          />
        ) : filtered.length === 0 ? (
          <EmptyState
            title={t('gateway.collection.empty.filteredTitle')}
            description={t('gateway.collection.empty.filteredDescription')}
          />
        ) : (
          <Table>
            <THead>
              <TR>
                <TH>{t('gateway.collection.columns.device')}</TH>
                <TH>{t('gateway.collection.columns.protocol')}</TH>
                <TH>{t('gateway.collection.columns.endpoint')}</TH>
                <TH>{t('gateway.collection.columns.cycle')}</TH>
                <TH>{t('gateway.collection.columns.status')}</TH>
                <TH>{t('gateway.collection.columns.topic')}</TH>
                <TH className="text-right">{t('gateway.collection.columns.actions')}</TH>
              </TR>
            </THead>
            <TBody>
              {filtered.map((device) => {
                const topic =
                  device.topic ||
                  (meta.data?.id ? defaultTopic(meta.data.id, device.id) : '')
                const option = device.address?.option
                const endpoint = [device.address?.location, option?.port]
                  .filter((value) => value !== undefined && value !== '')
                  .join(':')
                return (
                  <TR key={device.id}>
                    <TD>
                      <Link
                        to={`/gateway/collection/${device.id}`}
                        className="block max-w-56 truncate text-sm font-medium text-fg hover:text-primary"
                      >
                        {device.name}
                      </Link>
                      <span className="block max-w-56 truncate font-mono text-xs text-faint">
                        {device.deviceCode || device.id}
                      </span>
                    </TD>
                    <TD>
                      <ProtocolLabel type={device.deviceType} model={device.deviceModel} />
                    </TD>
                    <TD>
                      <span className="block max-w-48 truncate font-mono text-xs text-muted">
                        {endpoint || '—'}
                      </span>
                    </TD>
                    <TD className="whitespace-nowrap text-xs text-muted tabular">
                      {t('gateway.common.cycle', { seconds: device.collectorCycle })}
                    </TD>
                    <TD>
                      <DeviceStatusBadge status={device.collectStatus} />
                    </TD>
                    <TD>
                      <div className="flex items-center gap-1">
                        <span
                          className="block max-w-56 truncate font-mono text-xs text-muted"
                          title={topic || undefined}
                        >
                          {topic || t('gateway.collection.noTopic')}
                        </span>
                        {topic ? (
                          <CopyButton value={topic} label={t('gateway.common.topicCopied')} />
                        ) : null}
                      </div>
                      <span className="block text-[11px] text-faint">
                        {formatRelative(device.modTime)}
                      </span>
                    </TD>
                    <TD>
                      <div className="flex items-center justify-end gap-0.5">
                        <Link
                          to={`/gateway/collection/${device.id}`}
                          title={t('gateway.collection.action.detail')}
                          aria-label={t('gateway.collection.action.detail')}
                          className="flex size-7 items-center justify-center rounded text-faint transition-colors hover:bg-surface-2 hover:text-fg"
                        >
                          <Eye className="size-3.5" />
                        </Link>
                        <Link
                          to={`/gateway/collection/${device.id}/edit`}
                          title={t('gateway.collection.action.edit')}
                          aria-label={t('gateway.collection.action.edit')}
                          className="flex size-7 items-center justify-center rounded text-faint transition-colors hover:bg-surface-2 hover:text-fg"
                        >
                          <Pencil className="size-3.5" />
                        </Link>
                        {device.collectStatus === 'collecting' ? (
                          <Button
                            variant="ghost"
                            size="icon"
                            title={t('gateway.collection.action.stop')}
                            aria-label={t('gateway.collection.action.stop')}
                            onClick={() => setPending({ kind: 'stop', device })}
                          >
                            <Square className="size-3.5" />
                          </Button>
                        ) : (
                          <Button
                            variant="ghost"
                            size="icon"
                            title={t('gateway.collection.action.start')}
                            aria-label={t('gateway.collection.action.start')}
                            onClick={() => setPending({ kind: 'start', device })}
                          >
                            <Play className="size-3.5" />
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="icon"
                          title={t('gateway.collection.action.restart')}
                          aria-label={t('gateway.collection.action.restart')}
                          onClick={() => setPending({ kind: 'restart', device })}
                        >
                          <RotateCw className="size-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          title={t('gateway.collection.action.delete')}
                          aria-label={t('gateway.collection.action.delete')}
                          onClick={() => setPending({ kind: 'delete', device })}
                        >
                          <Trash2 className="size-3.5 text-danger" />
                        </Button>
                      </div>
                    </TD>
                  </TR>
                )
              })}
            </TBody>
          </Table>
        )}
      </div>

      <ConfirmDialog
        open={Boolean(pending)}
        onClose={() => setPending(null)}
        onConfirm={() => void confirm()}
        loading={pendingLoading}
        danger={pending?.kind === 'delete'}
        title={
          pending?.kind === 'delete'
            ? t('gateway.collection.confirm.deleteTitle')
            : pending?.kind === 'start'
              ? t('gateway.collection.confirm.startTitle')
              : pending?.kind === 'stop'
                ? t('gateway.collection.confirm.stopTitle')
                : t('gateway.collection.confirm.restartTitle')
        }
        description={
          pending?.kind === 'delete'
            ? t('gateway.collection.confirm.deleteDescription', { name: pending.device.name })
            : pending?.kind === 'start'
              ? t('gateway.collection.confirm.startDescription')
              : pending?.kind === 'stop'
                ? t('gateway.collection.confirm.stopDescription')
                : t('gateway.collection.confirm.restartDescription')
        }
        confirmText={
          pending?.kind === 'delete'
            ? t('common.action.delete')
            : pending?.kind === 'start'
              ? t('common.action.start')
              : pending?.kind === 'stop'
                ? t('common.action.stop')
                : t('common.action.restart')
        }
      />
    </div>
  )
}
