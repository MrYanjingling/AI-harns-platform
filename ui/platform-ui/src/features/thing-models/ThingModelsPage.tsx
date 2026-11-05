import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Boxes, Plus, Search, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteThingModel, useThingModels } from '@/api/hooks'
import { formatDateTime } from '@/lib/format'
import { errorMessage } from '@/lib/http'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState } from '@/components/ui/EmptyState'
import { Field, Input } from '@/components/ui/Field'
import { TableSkeleton } from '@/components/ui/Skeleton'
import { TBody, TD, TH, THead, TR, Table } from '@/components/ui/Table'
import { ErrorState } from '@/components/shared/ErrorState'
import { PageHeader } from '@/components/shared/PageHeader'
import { Pagination } from '@/components/shared/Pagination'

export function ThingModelsPage() {
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
      toast.success(`物模型「${deleteTarget.name}」已删除`)
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
        title="物模型"
        description="定义设备属性与特征的静态模板。属性是动态数据通道,特征是静态描述。"
        actions={
          <Button
            variant="primary"
            icon={<Plus className="size-4" />}
            onClick={() => navigate('/platform/thing-models/new')}
          >
            新建物模型
          </Button>
        }
      />

      <Card className="mb-4 flex flex-wrap items-end gap-3 p-4">
        <Field label="搜索" className="min-w-56 flex-1">
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
            <Input
              value={keywordInput}
              placeholder="物模型名称 / Key"
              className="pl-9"
              onChange={(event) => setKeywordInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') applySearch()
              }}
            />
          </div>
        </Field>
        <Button variant="outline" onClick={applySearch}>
          查询
        </Button>
        <Button variant="ghost" onClick={reset}>
          重置
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
            title={keyword ? '没有匹配的物模型' : '还没有物模型'}
            description={
              keyword
                ? '尝试更换关键词,或清空搜索条件。'
                : '先创建物模型,再基于它为设备定义属性与特征。'
            }
            action={
              keyword ? (
                <Button variant="outline" onClick={reset}>
                  清除搜索
                </Button>
              ) : (
                <Button
                  variant="primary"
                  icon={<Plus className="size-4" />}
                  onClick={() => navigate('/platform/thing-models/new')}
                >
                  新建物模型
                </Button>
              )
            }
          />
        ) : (
          <>
            <Table>
              <THead>
                <tr>
                  <TH>物模型</TH>
                  <TH>Key</TH>
                  <TH>描述</TH>
                  <TH>属性数</TH>
                  <TH>更新时间</TH>
                  <TH className="text-right">操作</TH>
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
                          aria-label={`删除 ${model.name}`}
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
        title="删除物模型"
        description={
          deleteTarget
            ? `确定删除「${deleteTarget.name}」及其全部属性吗?该操作不可恢复。`
            : undefined
        }
        confirmText="删除"
        danger
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
        onClose={() => setDeleteTarget(null)}
      />
    </div>
  )
}
