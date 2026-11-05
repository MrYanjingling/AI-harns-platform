import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, RadioTower, Search, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteThings, useThingModels, useThings } from '@/api/hooks'
import { formatRelative } from '@/lib/format'
import { errorMessage } from '@/lib/http'
import { Badge } from '@/components/ui/Badge'
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

const MODEL_LOOKUP_SIZE = 100

export function ThingsPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [deleteOpen, setDeleteOpen] = useState(false)

  const things = useThings({ page, pageSize, keyword: keyword || undefined })
  const models = useThingModels({ page: 1, pageSize: MODEL_LOOKUP_SIZE })
  const deleteMutation = useDeleteThings()

  const modelNames = useMemo(
    () => new Map((models.data?.list ?? []).map((model) => [model.id, model.name])),
    [models.data],
  )

  const list = things.data?.list ?? []
  const total = things.data?.total ?? 0
  const allSelected = list.length > 0 && list.every((thing) => selected.has(thing.id))

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    setPage(1)
    setSelected(new Set())
  }

  const reset = () => {
    setKeywordInput('')
    setKeyword('')
    setPage(1)
    setSelected(new Set())
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
      toast.success(`已删除 ${result.deleted} 台设备`)
      setSelected(new Set())
      setDeleteOpen(false)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div>
      <PageHeader
        title="平台设备"
        description="基于物模型接入的设备(Thing),凭据由后端生成,支持 MQTT 与 HTTP 上报。"
        actions={
          <Button
            variant="primary"
            icon={<Plus className="size-4" />}
            onClick={() => navigate('/platform/things/new')}
          >
            批量新建设备
          </Button>
        }
      />

      <Card className="mb-4 flex flex-wrap items-end gap-3 p-4">
        <Field label="搜索" className="min-w-56 flex-1">
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
            <Input
              value={keywordInput}
              placeholder="设备名称 / Device Key"
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
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-5 py-3">
          <p className="text-xs text-muted">
            共 <span className="font-medium text-fg tabular">{total}</span> 台设备
            {selected.size > 0 ? (
              <span className="ml-2 text-primary">已选 {selected.size} 台</span>
            ) : null}
          </p>
          {selected.size > 0 ? (
            <Button
              size="sm"
              variant="outline"
              icon={<Trash2 className="size-3.5 text-danger" />}
              onClick={() => setDeleteOpen(true)}
            >
              批量删除
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
            title={keyword ? '没有匹配的设备' : '还没有平台设备'}
            description={
              keyword
                ? '尝试更换关键词,或清空搜索条件。'
                : '先创建物模型,再批量创建设备并获取接入凭证。'
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
                  onClick={() => navigate('/platform/things/new')}
                >
                  批量新建设备
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
                      aria-label="全选当前页设备"
                      checked={allSelected}
                      onChange={toggleAll}
                      className="size-4 cursor-pointer accent-[var(--primary)]"
                    />
                  </TH>
                  <TH>设备</TH>
                  <TH>物模型</TH>
                  <TH>状态</TH>
                  <TH>最后上报</TH>
                  <TH>上行 Topic</TH>
                  <TH className="text-right">操作</TH>
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
                          aria-label={`选择 ${thing.deviceName}`}
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
                              启用
                            </Badge>
                          ) : (
                            <Badge tone="neutral" dot>
                              禁用
                            </Badge>
                          )}
                          {online ? (
                            <Badge tone="info" dot>
                              在线
                            </Badge>
                          ) : (
                            <Badge tone="neutral">离线</Badge>
                          )}
                        </div>
                      </TD>
                      <TD className="text-xs text-muted">
                        {thing.lastActiveTime ? formatRelative(thing.lastActiveTime) : '—'}
                      </TD>
                      <TD className="max-w-48 truncate font-mono text-[11px] text-muted" title={thing.upTopic}>
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
                            详情
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            aria-label={`删除 ${thing.deviceName}`}
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

      <ConfirmDialog
        open={deleteOpen}
        title="删除设备"
        description={`确定删除选中的 ${selected.size} 台设备吗?删除后设备凭证立即失效,该操作不可恢复。`}
        confirmText="删除"
        danger
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
        onClose={() => setDeleteOpen(false)}
      />
    </div>
  )
}
