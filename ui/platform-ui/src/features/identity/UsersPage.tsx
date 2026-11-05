import { useState } from 'react'
import { KeyRound, Plus, Search, Trash2, UserRound } from 'lucide-react'
import { toast } from 'sonner'
import { useDeleteUser, useResetUserPassword, useUsers } from '@/api/hooks'
import type { UserInfo } from '@/api/types'
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
import { UserFormDialog } from './UserFormDialog'

export function UsersPage() {
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<UserInfo | undefined>(undefined)
  const [resetTarget, setResetTarget] = useState<UserInfo | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<UserInfo | null>(null)

  const users = useUsers({ page, pageSize, keyword: keyword || undefined })
  const resetMutation = useResetUserPassword()
  const deleteMutation = useDeleteUser()

  const list = users.data?.list ?? []
  const total = users.data?.total ?? 0

  const applySearch = () => {
    setKeyword(keywordInput.trim())
    setPage(1)
  }

  const resetSearch = () => {
    setKeywordInput('')
    setKeyword('')
    setPage(1)
  }

  const openCreate = () => {
    setEditing(undefined)
    setFormOpen(true)
  }

  const openEdit = (user: UserInfo) => {
    setEditing(user)
    setFormOpen(true)
  }

  const confirmReset = async () => {
    if (!resetTarget) return
    try {
      await resetMutation.mutateAsync(resetTarget.id)
      toast.success(`密码已重置为 ${resetTarget.userName}@qwer`)
      setResetTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteMutation.mutateAsync(deleteTarget.id)
      toast.success(`用户「${deleteTarget.userName}」已删除`)
      setDeleteTarget(null)
    } catch (error) {
      toast.error(errorMessage(error))
    }
  }

  return (
    <div>
      <PageHeader
        title="用户管理"
        description="平台账号与角色、组织绑定。禁用或重置密码会立即失效该用户的会话。"
        actions={
          <Button variant="primary" icon={<Plus className="size-4" />} onClick={openCreate}>
            新建用户
          </Button>
        }
      />

      <Card className="mb-4 flex flex-wrap items-end gap-3 p-4">
        <Field label="搜索" className="min-w-56 flex-1">
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-faint" />
            <Input
              value={keywordInput}
              placeholder="用户名 / 工号 / 昵称"
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
        {users.isLoading ? (
          <TableSkeleton rows={6} cols={5} />
        ) : users.isError ? (
          <div className="p-5">
            <ErrorState error={users.error} onRetry={users.refetch} compact />
          </div>
        ) : list.length === 0 ? (
          <EmptyState
            icon={<UserRound className="size-5" />}
            title={keyword ? '没有匹配的用户' : '还没有用户'}
            description={keyword ? '尝试更换关键词。' : '创建用户并分配角色与组织。'}
            action={
              keyword ? (
                <Button variant="outline" onClick={resetSearch}>
                  清除搜索
                </Button>
              ) : (
                <Button variant="primary" onClick={openCreate}>
                  新建用户
                </Button>
              )
            }
          />
        ) : (
          <>
            <Table>
              <THead>
                <tr>
                  <TH>用户</TH>
                  <TH>工号</TH>
                  <TH>角色</TH>
                  <TH>状态</TH>
                  <TH>最后登录</TH>
                  <TH className="text-right">操作</TH>
                </tr>
              </THead>
              <TBody>
                {list.map((user) => (
                  <TR key={user.id}>
                    <TD>
                      <p className="font-medium">{user.nickName}</p>
                      <p className="font-mono text-[11px] text-faint">{user.userName}</p>
                    </TD>
                    <TD className="text-xs text-muted">{user.userNo}</TD>
                    <TD className="max-w-56">
                      <div className="flex flex-wrap gap-1">
                        {(user.roles ?? []).length > 0 ? (
                          user.roles.map((role) => (
                            <Badge key={role.id} tone="primary">
                              {role.name}
                            </Badge>
                          ))
                        ) : (
                          <span className="text-xs text-faint">未分配</span>
                        )}
                      </div>
                    </TD>
                    <TD>
                      <PlatformStatusBadge status={user.status} />
                    </TD>
                    <TD className="text-xs text-muted">
                      {user.lastLoginTime ? formatDateTime(user.lastLoginTime) : '从未登录'}
                    </TD>
                    <TD>
                      <div className="flex justify-end gap-1">
                        <Button size="sm" variant="ghost" onClick={() => openEdit(user)}>
                          编辑
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={`重置 ${user.userName} 的密码`}
                          title="重置密码"
                          onClick={() => setResetTarget(user)}
                        >
                          <KeyRound className="size-3.5" />
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label={`删除 ${user.userName}`}
                          onClick={() => setDeleteTarget(user)}
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
              disabled={users.isFetching}
            />
          </>
        )}
      </Card>

      {formOpen ? (
        <UserFormDialog open={formOpen} user={editing} onClose={() => setFormOpen(false)} />
      ) : null}

      <ConfirmDialog
        open={Boolean(resetTarget)}
        title="重置密码"
        description={
          resetTarget
            ? `将「${resetTarget.userName}」的密码重置为 ${resetTarget.userName}@qwer,并使其当前会话失效。`
            : undefined
        }
        confirmText="重置"
        loading={resetMutation.isPending}
        onConfirm={confirmReset}
        onClose={() => setResetTarget(null)}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除用户"
        description={
          deleteTarget
            ? `确定删除用户「${deleteTarget.userName}」吗?角色与组织绑定会一并清理,该操作不可恢复。`
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
