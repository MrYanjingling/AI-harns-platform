import { useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Boxes, Plus, RadioTower, Search, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteThings, useThingModels, useThings } from '@/api/hooks'
import { useI18n } from '@harns/core'
import { cn } from '@harns/core'
import { formatRelative } from '@harns/core'
import { errorMessage } from '@harns/core'
import { Badge } from '@harns/core'
import { Button } from '@harns/core'
import { Card } from '@harns/core'
import { ConfirmDialog } from '@harns/core'
import { EmptyState } from '@harns/core'
import { Field, Input } from '@harns/core'
import { Skeleton, TableSkeleton } from '@harns/core'
import { TBody, TD, TH, THead, TR, Table } from '@harns/core'
import { ErrorState } from '@harns/core'
import { PageHeader } from '@harns/core'
import { Pagination } from '@harns/core'

/** 左侧筛选列表一次取满物模型,后端单页上限 100。 */
const MODEL_LOOKUP_SIZE = 100
const MODEL_FILTER_PARAM = 'thingModelId'

export function ThingsPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [deleteOpen, setDeleteOpen] = useState(false)

  const selectedModelId = searchParams.get(MODEL_FILTER_PARAM) ?? ''

  const things = useThings({
    page,
    pageSize,
    keyword: keyword || undefined,
    thingModelId: selectedModelId || undefined,
  })
  const models = useThingModels({ page: 1, pageSize: MODEL_LOOKUP_SIZE })
  const deleteMutation = useDeleteThings()

  const modelNames = useMemo(
    () => new Map((models.data?.list ?? []).map((model) => [model.id, model.name])),
    [models.data],
  )

  const list = things.data?.list ?? []
  const total = things.data?.total ?? 0
  const allSelected = list.length > 0 && list.every((thing) => selected.has(thing.id))
  const hasFilter = Boolean(keyword || selectedModelId)

  const createPath = selectedModelId
    ? `/platform/things/new?${MODEL_FILTER_PARAM}=${selectedModelId}`
    : '/platform/things/new'

  const clearSelection = () => {
    setPage(1)
    setSelected(new Set())
  }

  const selectModel = (modelId: string) => {
    clearSelection()
    setSearchParams(modelId ? { [MODEL_FILTER_PARAM]: modelId } : {}, { replace: true })
  }

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    clearSelection()
  }

  const reset = () => {
    setKeywordInput('')
    setKeyword('')
    clearSelection()
    setSearchParams({}, { replace: true })
  }

  const toggle = (id: string) => {
    setSelected((current) => {
      const next = new Set(current)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleAll = () => {
    setSelected((current) => {
      const next = new Set(current)
      if (allSelected) list.forEach((thing) => next.delete(thing.id))
      else list.forEach((thing) => next.add(thing.id))
      return next
    })
  }

  const confirmDelete = async () => {
    const ids = Array.from(selected)
    try {
      const result = await deleteMutation.mutateAsync(ids)
      toast.success(t('things.deletedToast', { count: result.deleted }))
      setSelected(new Set())
      setDeleteOpen(false)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const modelList = models.data?.list ?? []

  return (
    <div>
      <PageHeader
        title={t('things.title')}
        description={t('things.description')}
        actions={
          <Button
            variant="primary"
            icon={<Plus className="size-4" />}
            onClick={() => navigate(createPath)}
          >
            {t('things.create')}
          </Button>
        }
      />

      <div className="grid gap-4 lg:grid-cols-[17rem_minmax(0,1fr)]">
        <Card className="h-fit overflow-hidden lg:sticky lg:top-4">
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            <p className="text-sm font-medium">{t('things.filter.title')}</p>
            {models.isSuccess ? (
              <span className="text-[11px] text-faint tabular">{models.data?.total ?? 0}</span>
            ) : null}
          </div>

          {models.isLoading ? (
            <div className="space-y-2 p-3">
              {Array.from({ length: 4 }).map((_, index) => (
                <Skeleton key={index} className="h-9 w-full" />
              ))}
            </div>
          ) : models.isError ? (
            <div className="p-3">
              <ErrorState error={models.error} onRetry={models.refetch} compact />
            </div>
          ) : modelList.length === 0 ? (
            <div className="p-4">
              <EmptyState
                icon={<Boxes className="size-5" />}
                title={t('things.filter.empty')}
                description={t('things.filter.emptyDescription')}
                action={
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => navigate('/platform/thing-models/new')}
                  >
                    {t('thingModels.create')}
                  </Button>
                }
              />
            </div>
          ) : (
            <ul className="max-h-[calc(100dvh-14rem)] overflow-y-auto p-1.5">
              <li>
                <ModelFilterItem
                  active={!selectedModelId}
                  label={t('things.filter.all')}
                  onClick={() => selectModel('')}
                />
              </li>
              {modelList.map((model) => (
                <li key={model.id}>
                  <ModelFilterItem
                    active={selectedModelId === model.id}
                    label={model.name}
                    hint={model.thingModelKey}
                    onClick={() => selectModel(model.id)}
                  />
                </li>
              ))}
            </ul>
          )}
        </Card>

        <div className="min-w-0 space-y-4">
          <Card className="flex flex-wrap items-end gap-3 p-4">
            <Field label={t('common.action.search')} className="min-w-56 flex-1">
              <div className="relative">
                <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
                <Input
                  value={keywordInput}
                  placeholder={t('things.searchPlaceholder')}
                  className="pl-9"
                  onChange={(event) => setKeywordInput(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') applySearch()
                  }}
                />
              </div>
            </Field>
            <Button variant="outline" onClick={applySearch}>
              {t('common.action.query')}
            </Button>
            <Button variant="ghost" onClick={reset}>
              {t('common.action.reset')}
            </Button>
          </Card>

          <Card className="overflow-hidden">
            <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-5 py-3">
              <p className="text-xs text-muted">
                {t('things.count', { count: total })}
                {selected.size > 0 ? (
                  <span className="ml-2 text-primary">
                    {t('things.selected', { count: selected.size })}
                  </span>
                ) : null}
              </p>
              {selected.size > 0 ? (
                <Button
                  size="sm"
                  variant="outline"
                  icon={<Trash2 className="size-3.5 text-danger" />}
                  onClick={() => setDeleteOpen(true)}
                >
                  {t('things.batchDelete')}
                </Button>
              ) : null}
            </div>

            {things.isLoading ? (
              <TableSkeleton rows={6} cols={6} />
            ) : things.isError ? (
              <div className="p-5">
                <ErrorState error={things.error} onRetry={things.refetch} compact />
              </div>
            ) : list.length === 0 ? (
              <EmptyState
                icon={<RadioTower className="size-5" />}
                title={
                  selectedModelId && !keyword
                    ? t('things.emptyModel')
                    : hasFilter
                      ? t('things.emptyMatch')
                      : t('things.emptyNone')
                }
                description={
                  selectedModelId && !keyword
                    ? t('things.emptyModelDescription')
                    : hasFilter
                      ? t('things.emptyMatchDescription')
                      : t('things.emptyNoneDescription')
                }
                action={
                  hasFilter ? (
                    <div className="flex flex-wrap justify-center gap-2">
                      <Button variant="outline" onClick={reset}>
                        {t('things.clearFilter')}
                      </Button>
                      {selectedModelId ? (
                        <Button
                          variant="primary"
                          icon={<Plus className="size-4" />}
                          onClick={() => navigate(createPath)}
                        >
                          {t('things.create')}
                        </Button>
                      ) : null}
                    </div>
                  ) : (
                    <Button
                      variant="primary"
                      icon={<Plus className="size-4" />}
                      onClick={() => navigate(createPath)}
                    >
                      {t('things.create')}
                    </Button>
                  )
                }
              />
            ) : (
              <>
                <Table>
                  <THead>
                    <tr>
                      <TH className="w-10">
                        <input
                          type="checkbox"
                          aria-label={t('things.selectPageAria')}
                          checked={allSelected}
                          onChange={toggleAll}
                          className="size-4 cursor-pointer accent-[var(--primary)]"
                        />
                      </TH>
                      <TH>{t('things.table.name')}</TH>
                      <TH>{t('things.table.model')}</TH>
                      <TH>{t('common.field.status')}</TH>
                      <TH>{t('things.table.lastActive')}</TH>
                      <TH>{t('things.table.upTopic')}</TH>
                      <TH className="text-right">{t('common.field.actions')}</TH>
                    </tr>
                  </THead>
                  <TBody>
                    {list.map((thing) => {
                      const online = thing.enable === 1 && thing.onlineStatus === 1
                      return (
                        <TR
                          key={thing.id}
                          className="cursor-pointer"
                          onClick={() => navigate(`/platform/things/${thing.id}`)}
                        >
                          <TD onClick={(event) => event.stopPropagation()}>
                            <input
                              type="checkbox"
                              aria-label={t('things.selectAria', { name: thing.deviceName })}
                              checked={selected.has(thing.id)}
                              onChange={() => toggle(thing.id)}
                              className="size-4 cursor-pointer accent-[var(--primary)]"
                            />
                          </TD>
                          <TD>
                            <p className="font-medium">{thing.deviceName}</p>
                            <p className="font-mono text-[11px] text-faint">{thing.deviceKey}</p>
                          </TD>
                          <TD className="max-w-44 truncate text-xs text-muted">
                            {modelNames.get(thing.thingModelId) ?? thing.thingModelId}
                          </TD>
                          <TD>
                            <div className="flex flex-wrap items-center gap-1.5">
                              {thing.enable === 1 ? (
                                <Badge tone="success" dot>
                                  {t('things.enabled')}
                                </Badge>
                              ) : (
                                <Badge tone="neutral" dot>
                                  {t('things.disabled')}
                                </Badge>
                              )}
                              {online ? (
                                <Badge tone="info" dot>
                                  {t('things.online')}
                                </Badge>
                              ) : (
                                <Badge tone="neutral">{t('things.offline')}</Badge>
                              )}
                            </div>
                          </TD>
                          <TD className="text-xs text-muted">
                            {thing.lastActiveTime ? formatRelative(thing.lastActiveTime) : '—'}
                          </TD>
                          <TD
                            className="max-w-48 truncate font-mono text-[11px] text-muted"
                            title={thing.upTopic}
                          >
                            {thing.upTopic}
                          </TD>
                          <TD>
                            <div
                              className="flex justify-end gap-1"
                              onClick={(event) => event.stopPropagation()}
                            >
                              <Button
                                size="sm"
                                variant="ghost"
                                onClick={() => navigate(`/platform/things/${thing.id}`)}
                              >
                                {t('common.action.view')}
                              </Button>
                              <Button
                                size="sm"
                                variant="ghost"
                                aria-label={t('things.deleteAria', { name: thing.deviceName })}
                                onClick={() => {
                                  setSelected(new Set([thing.id]))
                                  setDeleteOpen(true)
                                }}
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
                <Pagination
                  page={page}
                  pageSize={pageSize}
                  total={total}
                  onPageChange={(next) => {
                    setPage(next)
                    setSelected(new Set())
                  }}
                  onPageSizeChange={(size) => {
                    setPageSize(size)
                    setPage(1)
                    setSelected(new Set())
                  }}
                  disabled={things.isFetching}
                />
              </>
            )}
          </Card>
        </div>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        title={t('things.deleteTitle')}
        description={t('things.deleteDescription', { count: selected.size })}
        confirmText={t('common.action.delete')}
        danger
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
        onClose={() => setDeleteOpen(false)}
      />
    </div>
  )
}

function ModelFilterItem({
  active,
  label,
  hint,
  onClick,
}: {
  active: boolean
  label: string
  hint?: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      title={hint ? `${label} · ${hint}` : label}
      onClick={onClick}
      className={cn(
        'w-full rounded px-2.5 py-2 text-left transition-colors',
        active
          ? 'bg-primary-soft font-medium text-primary'
          : 'text-muted hover:bg-surface-2 hover:text-fg',
      )}
    >
      <span className="block truncate text-xs">{label}</span>
      {hint ? (
        <span className="mt-0.5 block truncate font-mono text-[10px] text-faint">{hint}</span>
      ) : null}
    </button>
  )
}
