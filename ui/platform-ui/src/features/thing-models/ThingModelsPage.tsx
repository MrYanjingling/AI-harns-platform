import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Boxes, Plus, Search, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteThingModel, useThingModels } from '@/api/hooks'
import { useI18n } from '@harns/core'
import { formatDateTime } from '@harns/core'
import { errorMessage } from '@harns/core'
import { Button } from '@harns/core'
import { Card } from '@harns/core'
import { ConfirmDialog } from '@harns/core'
import { EmptyState } from '@harns/core'
import { Field, Input } from '@harns/core'
import { TableSkeleton } from '@harns/core'
import { TBody, TD, TH, THead, TR, Table } from '@harns/core'
import { ErrorState } from '@harns/core'
import { PageHeader } from '@harns/core'
import { Pagination } from '@harns/core'

export function ThingModelsPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null)

  const query = { page, pageSize, keyword: keyword || undefined }
  const models = useThingModels(query)
  const deleteMutation = useDeleteThingModel()

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    setPage(1)
  }

  const reset = () => {
    setKeywordInput('')
    setKeyword('')
    setPage(1)
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success(t('thingModels.deletedToast', { name: deleteTarget.name }))
      setDeleteTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const list = models.data?.list ?? []
  const total = models.data?.total ?? 0

  return (
    <div>
      <PageHeader
        title={t('thingModels.title')}
        description={t('thingModels.description')}
        actions={
          <Button
            variant="primary"
            icon={<Plus className="size-4" />}
            onClick={() => navigate('/platform/thing-models/new')}
          >
            {t('thingModels.create')}
          </Button>
        }
      />

      <Card className="mb-4 flex flex-wrap items-end gap-3 p-4">
        <Field label={t('common.action.search')} className="min-w-56 flex-1">
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
            <Input
              value={keywordInput}
              placeholder={t('thingModels.searchPlaceholder')}
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
        {models.isLoading ? (
          <TableSkeleton rows={6} cols={5} />
        ) : models.isError ? (
          <div className="p-5">
            <ErrorState error={models.error} onRetry={models.refetch} compact />
          </div>
        ) : list.length === 0 ? (
          <EmptyState
            icon={<Boxes className="size-5" />}
            title={keyword ? t('thingModels.emptyMatch') : t('thingModels.emptyNone')}
            description={
              keyword
                ? t('thingModels.emptyMatchDescription')
                : t('thingModels.emptyNoneDescription')
            }
            action={
              keyword ? (
                <Button variant="outline" onClick={reset}>
                  {t('thingModels.clearSearch')}
                </Button>
              ) : (
                <Button
                  variant="primary"
                  icon={<Plus className="size-4" />}
                  onClick={() => navigate('/platform/thing-models/new')}
                >
                  {t('thingModels.create')}
                </Button>
              )
            }
          />
        ) : (
          <>
            <Table>
              <THead>
                <tr>
                  <TH>{t('thingModels.table.model')}</TH>
                  <TH>{t('thingModels.table.key')}</TH>
                  <TH>{t('common.field.description')}</TH>
                  <TH>{t('thingModels.table.propertyCount')}</TH>
                  <TH>{t('common.field.updatedAt')}</TH>
                  <TH className="text-right">{t('common.field.actions')}</TH>
                </tr>
              </THead>
              <TBody>
                {list.map((model) => (
                  <TR
                    key={model.id}
                    className="cursor-pointer"
                    onClick={() => navigate(`/platform/thing-models/${model.id}`)}
                  >
                    <TD>
                      <p className="font-medium">{model.name}</p>
                      <p className="font-mono text-[11px] text-faint">{model.id}</p>
                    </TD>
                    <TD className="font-mono text-xs text-muted">{model.thingModelKey}</TD>
                    <TD className="max-w-64 truncate text-xs text-muted" title={model.description}>
                      {model.description || '—'}
                    </TD>
                    <TD className="text-xs text-muted tabular">{model.propertyCount}</TD>
                    <TD className="text-xs text-muted">{formatDateTime(model.updatedTime)}</TD>
                    <TD>
                      <div
                        className="flex justify-end"
                        onClick={(event) => event.stopPropagation()}
                      >
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={t('thingModels.deleteAria', { name: model.name })}
                          onClick={() => setDeleteTarget({ id: model.id, name: model.name })}
                        >
                          <Trash2 className="size-3.5 text-danger" />
                        </Button>
                      </div>
                    </TD>
                  </TR>
                ))}
              </TBody>
            </Table>
            <Pagination
              page={page}
              pageSize={pageSize}
              total={total}
              onPageChange={setPage}
              onPageSizeChange={(size) => {
                setPageSize(size)
                setPage(1)
              }}
              disabled={models.isFetching}
            />
          </>
        )}
      </Card>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title={t('thingModels.deleteTitle')}
        description={
          deleteTarget
            ? t('thingModels.deleteDescription', { name: deleteTarget.name })
            : undefined
        }
        confirmText={t('common.action.delete')}
        danger
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
        onClose={() => setDeleteTarget(null)}
      />
    </div>
  )
}
