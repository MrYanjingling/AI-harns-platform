import { useState } from 'react'
import { Plus, Search, ShieldCheck, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteRole, useRoles } from '@/api/hooks'
import type { RoleInfo } from '@/api/types'
import { formatDateTime } from '@/lib/format'
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
import { PlatformStatusBadge } from '@/components/shared/PlatformStatusBadge'
import { RoleFormDialog } from './RoleFormDialog'

export function RolesPage() {
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<RoleInfo | undefined>(undefined)
  const [deleteTarget, setDeleteTarget] = useState<RoleInfo | null>(null)

  const roles = useRoles({ page, pageSize, keyword: keyword || undefined })
  const deleteMutation = useDeleteRole()

  const list = roles.data?.list ?? []
  const total = roles.data?.total ?? 0

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    setPage(1)
  }

  const resetSearch = () => {
    setKeywordInput('')
    setKeyword('')
    setPage(1)
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success(`角色「${deleteTarget.name}」已删除`)
      setDeleteTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div>
      <PageHeader
        title="角色管理"
        description="角色用于组织权限集合,删除角色会同时清理用户绑定。"
        actions={
          <Button
            variant="primary"
            icon={<Plus className="size-4" />}
            onClick={() => {
              setEditing(undefined)
              setFormOpen(true)
            }}
          >
            新建角色
          </Button>
        }
      />

      <Card className="mb-4 flex flex-wrap items-end gap-3 p-4">
        <Field label="搜索" className="min-w-56 flex-1">
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
            <Input
              value={keywordInput}
              placeholder="角色名称 / 编码"
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
        <Button variant="ghost" onClick={resetSearch}>
          重置
        </Button>
      </Card>

      <Card className="overflow-hidden">
        {roles.isLoading ? (
          <TableSkeleton rows={6} cols={5} />
        ) : roles.isError ? (
          <div className="p-5">
            <ErrorState error={roles.error} onRetry={roles.refetch} compact />
          </div>
        ) : list.length === 0 ? (
          <EmptyState
            icon={<ShieldCheck className="size-5" />}
            title={keyword ? '没有匹配的角色' : '还没有角色'}
            description={keyword ? '尝试更换关键词。' : '创建角色后可在新建用户时绑定。'}
            action={
              keyword ? (
                <Button variant="outline" onClick={resetSearch}>
                  清除搜索
                </Button>
              ) : (
                <Button
                  variant="primary"
                  onClick={() => {
                    setEditing(undefined)
                    setFormOpen(true)
                  }}
                >
                  新建角色
                </Button>
              )
            }
          />
        ) : (
          <>
            <Table>
              <THead>
                <tr>
                  <TH>角色</TH>
                  <TH>编码</TH>
                  <TH>描述</TH>
                  <TH>状态</TH>
                  <TH>更新时间</TH>
                  <TH className="text-right">操作</TH>
                </tr>
              </THead>
              <TBody>
                {list.map((role) => (
                  <TR key={role.id}>
                    <TD>
                      <p className="font-medium">{role.name}</p>
                      {role.code === 'admin' ? (
                        <Badge tone="violet" className="mt-1">
                          内置超管
                        </Badge>
                      ) : null}
                    </TD>
                    <TD className="font-mono text-xs text-muted">{role.code}</TD>
                    <TD className="max-w-64 truncate text-xs text-muted" title={role.description}>
                      {role.description || '—'}
                    </TD>
                    <TD>
                      <PlatformStatusBadge status={role.status} />
                    </TD>
                    <TD className="text-xs text-muted">{formatDateTime(role.updatedTime)}</TD>
                    <TD>
                      <div className="flex justify-end gap-1">
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setEditing(role)
                            setFormOpen(true)
                          }}
                        >
                          编辑
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={`删除 ${role.name}`}
                          onClick={() => setDeleteTarget(role)}
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
              disabled={roles.isFetching}
            />
          </>
        )}
      </Card>

      {formOpen ? (
        <RoleFormDialog open={formOpen} role={editing} onClose={() => setFormOpen(false)} />
      ) : null}

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除角色"
        description={
          deleteTarget
            ? `确定删除角色「${deleteTarget.name}」吗?用户绑定会一并清理,该操作不可恢复。`
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
