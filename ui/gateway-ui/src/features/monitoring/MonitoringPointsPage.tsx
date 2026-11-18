import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { Play, RotateCw, ScanSearch, SlidersHorizontal } from 'lucide-react'
import {
  AutoRefreshButton,
  Badge,
  Button,
  Dialog,
  EnumBadge,
  ErrorState,
  Input,
  ListTableCard,
  SearchInput,
  Pagination,
  SearchSelect,
  Select,
  TableSkeleton,
  Tooltip,
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
  DataTableBody,
  DataTableCell,
  DataTableHeadCell,
  DataTableHead,
  DataTableRow,
  DataTable,
} from '@harns/core'
import { ColumnResizeHandle, useColumnResize } from '@harns/core'
import { TableActionButton } from '@harns/core'

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
  const canControl = ['collecting', 'collectingError'].includes(
    currentCollection?.collectStatus ?? '',
  )
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
    actions: 64,
  })

  useEffect(() => {
    if (collectionList.length === 0) {
      if (collectionId) setCollectionId('')
      return
    }
    if (!collectionList.some((item) => item.id === collectionId)) {
      setCollectionId(collectionList[0].id)
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
      <div className="app-page-toolbar">
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
          ariaLabel={t('gateway.monitoring.selectCollection')}
          emptyText={t('common.state.empty')}
          className="w-[240px]"
        />
        <SearchInput
          wrapperClassName="w-[220px]"
          value={keywordInput}
          placeholder={t('gateway.monitoring.searchPlaceholder')}
          onChange={(event) => setKeywordInput(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              setKeyword(keywordInput.trim())
              setPage(1)
            }
          }}
        />
        {currentCollection ? (
          <>
            <CollectStatusBadge
              status={currentCollection.collectStatus}
              error={currentCollection.lastError}
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
          {collectionId ? (
            <AutoRefreshButton
              onRefresh={async () => { await Promise.all([variables.refetch(), collections.refetch()]) }}
            />
          ) : null}
        {frameAnalysisSupported ? (
          <Tooltip
            content={t('gateway.monitoring.analysisUnavailable')}
            disabled={frameAnalysisReady}
            focusable={!frameAnalysisReady}
            ariaLabel={!frameAnalysisReady ? t('gateway.monitoring.analysisUnavailable') : undefined}
            className="shrink-0"
          >
            <Button
              variant="primary"
              className="h-9"
              disabled={!frameAnalysisReady}
              icon={<ScanSearch className="size-3.5" />}
              onClick={openFrameAnalysis}
            >
              {t('gateway.monitoring.analysis')}
            </Button>
          </Tooltip>
        ) : null}
        </div>
      </div>

      <ListTableCard
        pagination={
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
        }
      >
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
          ) : collectionList.length === 0 || !collectionId || list.length === 0 ? null : (
            <DataTable
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
              <DataTableHead>
                <tr>
                  <DataTableHeadCell className="text-center">
                    {t('common.field.order')}
                    <ColumnResizeHandle column="order" width={widths.order} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.variable.name')}
                    <ColumnResizeHandle column="name" width={widths.name} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.variable.dataType')}
                    <ColumnResizeHandle
                      column="dataType"
                      width={widths.dataType}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.monitoring.addressExpression')}
                    <ColumnResizeHandle
                      column="address"
                      width={widths.address}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.variable.bits')}
                    <ColumnResizeHandle column="bits" width={widths.bits} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.variable.precision')}
                    <ColumnResizeHandle
                      column="precision"
                      width={widths.precision}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.variable.rate')}
                    <ColumnResizeHandle column="rate" width={widths.rate} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.variable.offset')}
                    <ColumnResizeHandle column="offset" width={widths.offset} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.variable.accessMode')}
                    <ColumnResizeHandle
                      column="accessMode"
                      width={widths.accessMode}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('gateway.variable.value')}
                    <ColumnResizeHandle column="value" width={widths.value} onResize={resize} />
                  </DataTableHeadCell>
                  <DataTableHeadCell>
                    {t('common.field.actions')}
                    <ColumnResizeHandle
                      column="actions"
                      width={widths.actions}
                      onResize={resize}
                    />
                  </DataTableHeadCell>
                </tr>
              </DataTableHead>
              <DataTableBody>
                {list.map((variable, index) => (
                  <DataTableRow key={variable.name}>
                    <DataTableCell className="text-center">
                      {(page - 1) * pageSize + index + 1}
                    </DataTableCell>
                    <DataTableCell>
                      <div className="flex items-center gap-1.5">
                        <span className="truncate">{variable.name}</span>
                        {variable.isCalculated ? (
                          <Badge tone="info">{t('gateway.variable.calculated')}</Badge>
                        ) : null}
                      </div>
                    </DataTableCell>
                    <DataTableCell>
                      <span>{variable.dataType}</span>
                    </DataTableCell>
                    <DataTableCell>
                      <span
                        className="block truncate"
                        title={variableAddress(variable)}
                      >
                        {variableAddress(variable)}
                      </span>
                    </DataTableCell>
                    <DataTableCell>{displayNumber(variable.bits)}</DataTableCell>
                    <DataTableCell>{displayNumber(variable.precision)}</DataTableCell>
                    <DataTableCell>{displayNumber(variable.rate)}</DataTableCell>
                    <DataTableCell>{displayNumber(variable.offset)}</DataTableCell>
                    <DataTableCell>
                      <EnumBadge tone={variable.accessMode === 'r' ? 'neutral' : 'primary'}>
                        {accessModeLabel(variable.accessMode, t)}
                      </EnumBadge>
                    </DataTableCell>
                    <DataTableCell>
                      <span className="block truncate">{formatValue(variable.value)}</span>
                    </DataTableCell>
                    <DataTableCell>
                      {variable.accessMode === 'w' || variable.accessMode === 'rw' ? (
                        <TableActionButton
                          label={t('gateway.monitoring.control')}
                          icon={<SlidersHorizontal />}
                          disabled={!canControl}
                          disabledReason={
                            !canControl ? t('gateway.monitoring.controlUnavailable') : undefined
                          }
                          onClick={() => {
                            setControlValue('')
                            setControlVariable(variable)
                          }}
                        />
                      ) : null}
                    </DataTableCell>
                  </DataTableRow>
                ))}
              </DataTableBody>
            </DataTable>
          )}
      </ListTableCard>

      <Dialog
        open={Boolean(controlVariable)}
        onClose={() => setControlVariable(null)}
        title={t('gateway.detail.control.title')}
        className="app-control-dialog"
        headerClassName="app-control-dialog-header"
        bodyClassName="app-control-dialog-body"
        footerClassName="app-control-dialog-footer"
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
            >
              {t('gateway.detail.control.submit')}
            </Button>
          </>
        }
      >
        <div className="app-control-dialog-content">
          <p
            role={controlError ? 'alert' : undefined}
            aria-live="polite"
            className={
              controlError ? 'app-dialog-error app-dialog-error--shake' : 'app-dialog-error'
            }
          >
            {controlError}
          </p>
          <dl className="app-control-dialog-summary">
            <div className="app-control-dialog-row">
              <dt className="app-control-dialog-label">{t('gateway.variable.dataType')}</dt>
              <dd className="app-control-dialog-value">{controlVariable?.dataType ?? '—'}</dd>
            </div>
            <div className="app-control-dialog-row">
              <dt className="app-control-dialog-label">{t('gateway.variable.address')}</dt>
              <dd className="app-control-dialog-value app-control-dialog-value--truncate">
                {controlVariable?.address || '—'}
              </dd>
            </div>
            <div className="app-control-dialog-row">
              <dt className="app-control-dialog-label">{t('gateway.variable.value')}</dt>
              <dd className="app-control-dialog-value app-control-dialog-value--truncate">
                {formatValue(controlVariable?.value)}
              </dd>
            </div>
            <div className="app-control-dialog-row">
              <dt className="app-control-dialog-label">{t('gateway.monitoring.writeValue')}</dt>
              <dd className="app-control-dialog-value-field">
                <div className="app-control-dialog-field">
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
                </div>
              </dd>
            </div>
          </dl>
        </div>
      </Dialog>
    </div>
  )
}
