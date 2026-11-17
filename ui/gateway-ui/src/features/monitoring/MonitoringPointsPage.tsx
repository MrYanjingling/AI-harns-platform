import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { Gauge, Play, RefreshCw, RotateCw, ScanSearch, Search } from 'lucide-react'
import {
  Badge,
  Button,
  Card,
  Dialog,
  EmptyState,
  ErrorState,
  Input,
  Pagination,
  SearchSelect,
  Select,
  TableSkeleton,
  showErrorToast,
  useI18n,
  type SearchSelectOption,
  type Translate,
} from '@harns/core'
import {
  useCollectionRuntimeAction,
  useCollectionVariables,
  useCollections,
  useDeliverCollectionAction,
} from '@/api/hooks'
import type { AccessMode, CollectionVariable } from '@/api/types'
import { CollectStatusBadge } from '@/components/StatusBadge'
import {
  GatewayTBody,
  GatewayTD,
  GatewayTH,
  GatewayTHead,
  GatewayTR,
  GatewayTable,
} from '@/components/GatewayTable'
import { ColumnResizeHandle, useColumnResize } from '@/components/useColumnResize'

function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function variableAddress(variable: CollectionVariable): string {
  if (variable.isCalculated) return variable.expression || '—'
  return variable.address || '—'
}

function displayNumber(value: number | null | undefined): string {
  return value === undefined || value === null ? '—' : String(value)
}

function accessModeLabel(mode: AccessMode, t: (key: string) => string): string {
  if (mode === 'r') return t('gateway.monitoring.access.read')
  if (mode === 'w') return t('gateway.monitoring.access.write')
  return t('gateway.monitoring.access.readWrite')
}

const integerRanges: Record<string, [string, string]> = {
  int16: ['-32768', '32767'],
  uint16: ['0', '65535'],
  int32: ['-2147483648', '2147483647'],
  int64: ['-9223372036854775808', '9223372036854775807'],
}

function validateControlValue(
  variable: CollectionVariable | null,
  value: string,
  t: Translate,
): string {
  if (!variable) return ''
  const text = value.trim()
  if (!text) return ''
  const dataType = variable.dataType.trim().toLowerCase()
  if (dataType === 'bool' || dataType === 'bit') {
    return text === '0' || text === '1' ? '' : t('gateway.monitoring.valueBinary')
  }
  if (dataType === 'string') return ''
  if (!(dataType in integerRanges) && !['float32', 'float64', 'number'].includes(dataType)) {
    return t('gateway.monitoring.valueUnsupported')
  }
  if (!/^-?\d+(\.\d+)?$/.test(text)) return t('gateway.monitoring.valueNumber')

  const rate = variable.rate ?? 1
  const offset = variable.offset ?? 0
  if (!Number.isFinite(rate) || rate === 0 || !Number.isFinite(offset)) {
    return t('gateway.monitoring.valueUnsupported')
  }

  const range = integerRanges[dataType]
  if (range && rate === 1 && offset === 0) {
    if (!/^-?\d+$/.test(text)) return t('gateway.monitoring.valueInteger')
    const integer = BigInt(text)
    if (integer < BigInt(range[0]) || integer > BigInt(range[1])) {
      return t('gateway.monitoring.valueRange', { type: variable.dataType })
    }
    return ''
  }

  const number = Number(text)
  const rawValue = (number - offset) / rate
  if (!Number.isFinite(number) || !Number.isFinite(rawValue)) {
    return t('gateway.monitoring.valueRange', { type: variable.dataType })
  }
  if (range) {
    if (!Number.isInteger(rawValue)) return t('gateway.monitoring.valueInteger')
    if (rawValue < Number(range[0]) || rawValue > Number(range[1])) {
      return t('gateway.monitoring.valueRange', { type: variable.dataType })
    }
    return ''
  }
  if (dataType === 'float32' && Math.abs(rawValue) > 3.4028234663852886e38) {
    return t('gateway.monitoring.valueRange', { type: variable.dataType })
  }
  return ''
}

export function MonitoringPointsPage() {
  const { t } = useI18n()
  const collections = useCollections({ page: 1, pageSize: 100 })
  const collectionList = collections.data?.list ?? []
  const [collectionId, setCollectionId] = useState('')
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [controlVariable, setControlVariable] = useState<CollectionVariable | null>(null)
  const [controlValue, setControlValue] = useState('')
  const [autoRefreshOpen, setAutoRefreshOpen] = useState(false)
  const [autoRefreshInput, setAutoRefreshInput] = useState('10')
  const [autoRefreshSeconds, setAutoRefreshSeconds] = useState(10)
  const [autoRefreshActive, setAutoRefreshActive] = useState(false)
  const [autoRefreshError, setAutoRefreshError] = useState('')
  const runtimeAction = useCollectionRuntimeAction()
  const deliver = useDeliverCollectionAction()
  const variables = useCollectionVariables(collectionId || undefined, {
    page,
    pageSize,
    keyword: keyword || undefined,
  })
  const list = variables.data?.list ?? []
  const total = variables.data?.total ?? 0
  const currentCollection = collectionList.find((item) => item.id === collectionId)
  const frameAnalysisSupported =
    Boolean(currentCollection) && currentCollection?.driverType !== 'opcUa'
  const frameAnalysisReady =
    frameAnalysisSupported &&
    ['collecting', 'collectingError'].includes(currentCollection?.collectStatus ?? '')
  const { widths, totalWidth, resize } = useColumnResize({
    order: 64,
    name: 180,
    dataType: 100,
    address: 240,
    bits: 64,
    precision: 64,
    rate: 64,
    offset: 64,
    accessMode: 90,
    value: 160,
    actions: 90,
  })

  useEffect(() => {
    if (collectionList.length === 0) {
      if (collectionId) setCollectionId('')
      return
    }
    if (!collectionList.some((item) => item.id === collectionId)) {
      // 默认优先选择支持报文分析的非 OPC UA 实例。
      const preferred =
        collectionList.find((item) => item.driverType !== 'opcUa') ?? collectionList[0]
      setCollectionId(preferred.id)
      setPage(1)
    }
  }, [collectionList, collectionId])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setKeyword(keywordInput.trim())
      setPage(1)
    }, 300)
    return () => window.clearTimeout(timer)
  }, [keywordInput])

  useEffect(() => {
    if (!autoRefreshActive) return
    const timer = window.setInterval(() => {
      void variables.refetch()
      void collections.refetch()
    }, autoRefreshSeconds * 1000)
    return () => window.clearInterval(timer)
  }, [autoRefreshActive, autoRefreshSeconds, variables.refetch, collections.refetch])

  const collectionOptions = useMemo<SearchSelectOption[]>(
    () => collectionList.map((item) => ({ value: item.id, label: item.name })),
    [collectionList],
  )
  const isBoolControl =
    controlVariable?.dataType.trim().toLowerCase() === 'bool'
  const isIntegerControl = ['int16', 'uint16', 'int32', 'int64'].includes(
    controlVariable?.dataType.trim().toLowerCase() ?? '',
  )
  const controlError = useMemo(
    () => validateControlValue(controlVariable, controlValue, t),
    [controlVariable, controlValue, t],
  )

  const openFrameAnalysis = () => {
    if (!currentCollection || !frameAnalysisReady) return
    const params = new URLSearchParams({ collectionId: currentCollection.id })
    window.open(`/monitoring/points/analysis?${params.toString()}`, '_blank', 'noopener,noreferrer')
  }

  const openAutoRefresh = () => {
    setAutoRefreshInput(String(autoRefreshSeconds))
    setAutoRefreshError('')
    setAutoRefreshOpen(true)
  }

  const closeAutoRefresh = () => {
    setAutoRefreshOpen(false)
    setAutoRefreshError('')
  }

  const stopAutoRefresh = () => {
    setAutoRefreshActive(false)
    closeAutoRefresh()
  }

  const confirmAutoRefresh = () => {
    const seconds = Number(autoRefreshInput.trim())
    if (!Number.isInteger(seconds) || seconds < 1 || seconds > 3600) {
      setAutoRefreshError(t('gateway.monitoring.autoRefreshInvalid'))
      return
    }
    setAutoRefreshSeconds(seconds)
    setAutoRefreshActive(true)
    setAutoRefreshOpen(false)
    setAutoRefreshError('')
    void variables.refetch()
    void collections.refetch()
  }

  const runRuntimeAction = async (action: 'start' | 'restart') => {
    if (!currentCollection) return
    try {
      await runtimeAction.mutateAsync({
        id: currentCollection.id,
        name: currentCollection.name,
        action,
      })
      toast.success(
        action === 'start'
          ? t('gateway.collection.toast.started')
          : t('gateway.collection.toast.restarted'),
      )
      await collections.refetch()
    } catch (error) {
      showErrorToast(error)
    }
  }

  const submitControl = async () => {
    if (!currentCollection || !controlVariable || !controlValue.trim() || controlError) return
    try {
      await deliver.mutateAsync({
        id: currentCollection.id,
        name: currentCollection.name,
        actions: [{ name: controlVariable.name, value: controlValue.trim() }],
      })
      toast.success(t('gateway.detail.control.toast'))
      setControlVariable(null)
      setControlValue('')
      await variables.refetch()
    } catch (error) {
      showErrorToast(error)
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="mb-3 flex shrink-0 flex-wrap items-center gap-2">
        <label htmlFor="monitoring-collection" className="text-sm text-muted">
          {t('gateway.monitoring.collection')}
        </label>
        <SearchSelect
          id="monitoring-collection"
          value={collectionId}
          options={collectionOptions}
          onChange={(next) => {
            setCollectionId(next)
            setKeywordInput('')
            setKeyword('')
            setPage(1)
            setControlVariable(null)
          }}
          ariaLabel={t('gateway.monitoring.collection')}
          emptyText={t('common.state.empty')}
          className="w-[240px]"
          panelClassName="w-[240px]"
        />
        <div className="relative w-[220px]">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
          <Input
            value={keywordInput}
            placeholder={t('gateway.monitoring.searchPlaceholder')}
            className="h-9 pl-9 text-sm"
            onChange={(event) => setKeywordInput(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                setKeyword(keywordInput.trim())
                setPage(1)
              }
            }}
          />
        </div>
        {currentCollection ? (
          <>
            <CollectStatusBadge
              status={currentCollection.collectStatus}
              error={currentCollection.lastError}
              className="border-[1.5px] px-2 py-1"
            />
            {['stopped', 'emptyVariable'].includes(currentCollection.collectStatus) ? (
              <Button
                variant="outline"
                size="sm"
                loading={runtimeAction.isPending}
                icon={<Play className="size-3.5" />}
                onClick={() => void runRuntimeAction('start')}
              >
                {t('common.action.start')}
              </Button>
            ) : ['collecting', 'collectingError', 'unconnected', 'error'].includes(
                currentCollection.collectStatus,
              ) ? (
              <Button
                variant="outline"
                size="sm"
                loading={runtimeAction.isPending}
                icon={<RotateCw className="size-3.5" />}
                onClick={() => void runRuntimeAction('restart')}
              >
                {t('common.action.restart')}
              </Button>
            ) : null}
          </>
        ) : null}
        <div className="ml-auto flex items-center gap-2">
          <Button
            variant={autoRefreshActive ? 'primary' : 'outline'}
            className="h-9"
            icon={<RefreshCw className="size-3.5" />}
            onClick={openAutoRefresh}
          >
            {autoRefreshActive
              ? t('gateway.monitoring.autoRefreshActive', { seconds: autoRefreshSeconds })
              : t('gateway.monitoring.autoRefresh')}
          </Button>
        {frameAnalysisSupported ? (
          <Button
            variant="primary"
            className="ml-auto h-9"
            disabled={!frameAnalysisReady}
            title={frameAnalysisReady ? undefined : t('gateway.monitoring.analysisUnavailable')}
            icon={<ScanSearch className="size-3.5" />}
            onClick={openFrameAnalysis}
          >
            {t('gateway.monitoring.analysis')}
          </Button>
        ) : null}
        </div>
      </div>

      <Card className="flex min-h-0 flex-1 flex-col overflow-hidden">
        <div className="min-h-0 flex-1 overflow-auto">
          {collections.isLoading || variables.isLoading ? (
            <TableSkeleton rows={6} cols={11} />
          ) : collections.isError ? (
            <div className="p-5">
              <ErrorState
                error={collections.error}
                onRetry={() => void collections.refetch()}
                compact
              />
            </div>
          ) : variables.isError ? (
            <div className="p-5">
              <ErrorState
                error={variables.error}
                onRetry={() => void variables.refetch()}
                compact
              />
            </div>
          ) : collectionList.length === 0 ? (
            <EmptyState
              icon={<Gauge className="size-5" />}
              title={t('gateway.monitoring.emptyCollectionTitle')}
              description={t('gateway.monitoring.emptyCollectionDescription')}
            />
          ) : list.length === 0 ? null : (
            <GatewayTable
              style={{
                tableLayout: 'fixed',
                width: `max(100%, ${totalWidth}px)`,
              }}
            >
              <colgroup>
                <col style={{ width: widths.order }} />
                <col style={{ width: widths.name }} />
                <col style={{ width: widths.dataType }} />
                <col style={{ width: widths.address }} />
                <col style={{ width: widths.bits }} />
                <col style={{ width: widths.precision }} />
                <col style={{ width: widths.rate }} />
                <col style={{ width: widths.offset }} />
                <col style={{ width: widths.accessMode }} />
                <col style={{ width: widths.value }} />
                <col style={{ width: widths.actions }} />
              </colgroup>
              <GatewayTHead>
                <tr>
                  <GatewayTH className="text-center">
                    {t('common.field.order')}
                    <ColumnResizeHandle column="order" width={widths.order} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.variable.name')}
                    <ColumnResizeHandle column="name" width={widths.name} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.variable.dataType')}
                    <ColumnResizeHandle
                      column="dataType"
                      width={widths.dataType}
                      onResize={resize}
                    />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.monitoring.addressExpression')}
                    <ColumnResizeHandle
                      column="address"
                      width={widths.address}
                      onResize={resize}
                    />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.variable.bits')}
                    <ColumnResizeHandle column="bits" width={widths.bits} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.variable.precision')}
                    <ColumnResizeHandle
                      column="precision"
                      width={widths.precision}
                      onResize={resize}
                    />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.variable.rate')}
                    <ColumnResizeHandle column="rate" width={widths.rate} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.variable.offset')}
                    <ColumnResizeHandle column="offset" width={widths.offset} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.variable.accessMode')}
                    <ColumnResizeHandle
                      column="accessMode"
                      width={widths.accessMode}
                      onResize={resize}
                    />
                  </GatewayTH>
                  <GatewayTH>
                    {t('gateway.variable.value')}
                    <ColumnResizeHandle column="value" width={widths.value} onResize={resize} />
                  </GatewayTH>
                  <GatewayTH>
                    {t('common.field.actions')}
                    <ColumnResizeHandle
                      column="actions"
                      width={widths.actions}
                      onResize={resize}
                    />
                  </GatewayTH>
                </tr>
              </GatewayTHead>
              <GatewayTBody>
                {list.map((variable, index) => (
                  <GatewayTR key={variable.name}>
                    <GatewayTD className="text-center text-muted">
                      {(page - 1) * pageSize + index + 1}
                    </GatewayTD>
                    <GatewayTD>
                      <div className="flex items-center gap-1.5">
                        <span className="truncate text-fg">{variable.name}</span>
                        {variable.isCalculated ? (
                          <Badge tone="info">{t('gateway.variable.calculated')}</Badge>
                        ) : null}
                      </div>
                    </GatewayTD>
                    <GatewayTD>
                      <span className="font-mono text-muted">{variable.dataType}</span>
                    </GatewayTD>
                    <GatewayTD>
                      <span
                        className="block truncate font-mono text-muted"
                        title={variableAddress(variable)}
                      >
                        {variableAddress(variable)}
                      </span>
                    </GatewayTD>
                    <GatewayTD>{displayNumber(variable.bits)}</GatewayTD>
                    <GatewayTD>{displayNumber(variable.precision)}</GatewayTD>
                    <GatewayTD>{displayNumber(variable.rate)}</GatewayTD>
                    <GatewayTD>{displayNumber(variable.offset)}</GatewayTD>
                    <GatewayTD>
                      <Badge
                        tone={variable.accessMode === 'r' ? 'neutral' : 'primary'}
                        className="border-[1.5px] px-2 py-1"
                      >
                        {accessModeLabel(variable.accessMode, t)}
                      </Badge>
                    </GatewayTD>
                    <GatewayTD>
                      <span className="block truncate">{formatValue(variable.value)}</span>
                    </GatewayTD>
                    <GatewayTD>
                      {/* 调试期间暂时放开空变量/停止/未连接状态的下控限制。 */}
                      {variable.accessMode === 'w' || variable.accessMode === 'rw' ? (
                        <button
                          type="button"
                          onClick={() => {
                            setControlValue('')
                            setControlVariable(variable)
                          }}
                          className="text-sm text-primary transition-colors hover:text-primary-hover"
                        >
                          {t('gateway.monitoring.control')}
                        </button>
                      ) : null}
                    </GatewayTD>
                  </GatewayTR>
                ))}
              </GatewayTBody>
            </GatewayTable>
          )}
        </div>
        <div className="shrink-0">
          <Pagination
            page={page}
            pageSize={pageSize}
            total={total}
            onPageChange={setPage}
            onPageSizeChange={(size) => {
              setPageSize(size)
              setPage(1)
            }}
            disabled={variables.isFetching || variables.isLoading}
          />
        </div>
      </Card>

      <Dialog
        open={autoRefreshOpen}
        onClose={closeAutoRefresh}
        title={t('gateway.monitoring.autoRefreshTitle')}
        description={t('gateway.monitoring.autoRefreshDescription')}
        className="w-[320px] max-w-[calc(100vw-2rem)] rounded-lg"
        footer={
          <>
            {autoRefreshActive ? (
              <Button variant="outline" onClick={stopAutoRefresh}>
                {t('gateway.monitoring.autoRefreshStop')}
              </Button>
            ) : null}
            <Button variant="outline" onClick={closeAutoRefresh}>
              {t('common.action.cancel')}
            </Button>
            <Button variant="primary" onClick={confirmAutoRefresh}>
              {t('common.action.confirm')}
            </Button>
          </>
        }
      >
        <div className="space-y-2">
          <label
            htmlFor="monitoring-auto-refresh-seconds"
            className="block text-sm text-fg"
          >
            {t('gateway.monitoring.autoRefreshSeconds')}
          </label>
          <Input
            id="monitoring-auto-refresh-seconds"
            type="number"
            min={1}
            max={3600}
            step={1}
            autoFocus
            value={autoRefreshInput}
            aria-invalid={Boolean(autoRefreshError)}
            className={
              autoRefreshError
                ? 'border-danger hover:border-danger focus:border-danger focus:ring-danger/15'
                : undefined
            }
            onChange={(event) => {
              setAutoRefreshInput(event.target.value)
              setAutoRefreshError('')
            }}
          />
          {autoRefreshError ? (
            <p className="text-xs text-danger">{autoRefreshError}</p>
          ) : null}
        </div>
      </Dialog>

      <Dialog
        open={Boolean(controlVariable)}
        onClose={() => setControlVariable(null)}
        title={t('gateway.detail.control.title')}
        description={
          controlVariable?.name ? (
            <span className="text-sm font-medium leading-5 text-muted">
              {controlVariable.name}
            </span>
          ) : undefined
        }
        className="w-[280px] max-w-[calc(100vw-2rem)] rounded-lg"
        headerClassName="items-center pt-3"
        footerClassName="min-h-10 items-center py-1"
        footer={
          <>
            <Button variant="outline" onClick={() => setControlVariable(null)}>
              {t('common.action.cancel')}
            </Button>
            <Button
              variant="primary"
              loading={deliver.isPending}
              disabled={!controlValue.trim() || Boolean(controlError)}
              onClick={() => void submitControl()}
              className="self-center"
            >
              {t('gateway.detail.control.submit')}
            </Button>
          </>
        }
      >
        <div className="space-y-3">
          <dl className="divide-y divide-border rounded border border-border bg-surface-2">
            <div className="flex items-center justify-between gap-4 px-3 py-2">
              <dt className="text-sm leading-5 text-fg">{t('gateway.variable.dataType')}</dt>
              <dd className="font-mono text-sm text-fg">{controlVariable?.dataType ?? '—'}</dd>
            </div>
            <div className="flex items-center justify-between gap-4 px-3 py-2">
              <dt className="text-sm leading-5 text-fg">{t('gateway.variable.address')}</dt>
              <dd className="truncate font-mono text-sm text-fg">
                {controlVariable?.address || '—'}
              </dd>
            </div>
            <div className="flex items-center justify-between gap-4 px-3 py-2">
              <dt className="text-sm leading-5 text-fg">{t('gateway.variable.value')}</dt>
              <dd className="truncate font-mono text-sm text-fg">
                {formatValue(controlVariable?.value)}
              </dd>
            </div>
            <div className="flex items-center justify-between gap-4 px-3 py-2">
              <dt className="shrink-0 text-sm leading-5 text-fg">{t('gateway.monitoring.writeValue')}</dt>
              <dd className="min-w-0 flex-1">
                {isBoolControl ? (
                  <Select
                    id="monitoring-control-value"
                    value={controlValue}
                    autoFocus
                    aria-invalid={Boolean(controlError)}
                    className={
                      controlError
                        ? 'border-danger hover:border-danger focus:border-danger focus:ring-danger/15'
                        : undefined
                    }
                    onChange={(event) => setControlValue(event.target.value)}
                  >
                    <option value="" disabled>
                      {t('gateway.form.selectPlaceholder')}
                    </option>
                    <option value="0">0</option>
                    <option value="1">1</option>
                  </Select>
                ) : (
                  <Input
                    id="monitoring-control-value"
                    value={controlValue}
                    autoFocus
                    inputMode={isIntegerControl ? 'numeric' : 'decimal'}
                    placeholder={t('gateway.detail.control.valuePlaceholder')}
                    autoComplete="off"
                    spellCheck={false}
                    aria-invalid={Boolean(controlError)}
                    className={
                      controlError
                        ? 'border-danger hover:border-danger focus:border-danger focus:ring-danger/15'
                        : undefined
                    }
                    onChange={(event) => setControlValue(event.target.value)}
                  />
                )}
                {controlError ? (
                  <p className="mt-1 text-xs text-danger">{controlError}</p>
                ) : null}
              </dd>
            </div>
          </dl>
        </div>
      </Dialog>
    </div>
  )
}
